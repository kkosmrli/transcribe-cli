package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

const MaxChunkSizeMB int64 = 25

// splitAudioFile splits an audio file into chunks if it exceeds MaxChunkSizeMB.
// It returns a slice of file paths, which will contain the original path if no splitting was needed,
// or the paths to the created chunks if splitting occurred.
func splitAudioFile(filePath string) ([]string, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info for %s: %w", filePath, err)
	}

	if fileInfo.Size() <= MaxChunkSizeMB*1024*1024 {
		log.Printf("File %s is within size limit (%dMB), no splitting needed.", filePath, MaxChunkSizeMB)
		return []string{filePath}, nil
	}

	log.Printf("File %s (%d bytes) exceeds %dMB, splitting into chunks...", filePath, fileInfo.Size(), MaxChunkSizeMB)

	dir := filepath.Dir(filePath)
	ext := filepath.Ext(filePath)
	baseName := strings.TrimSuffix(filepath.Base(filePath), ext)

	outputPattern := filepath.Join(dir, fmt.Sprintf("%s_chunk_%%03d%s", baseName, ext))
	log.Printf("Output pattern for chunks: %s", outputPattern)

	// Using a fixed segment time of 600 seconds (10 minutes)
	// ffmpeg -i inputfile -f segment -segment_time 600 -c copy -reset_timestamps 1 -map 0 outputfile_chunk_%03d.ext
	err = ffmpeg.Input(filePath).
		Output(outputPattern, ffmpeg.KwArgs{
			"f":                "segment",
			"segment_time":     "600",
			"c":                "copy",
			"reset_timestamps": "1", // Ensures timestamps start from 0 for each chunk
			"map":              "0",   // Selects all streams from the first input (0)
		}).
		OverWriteOutput(). // Allows overwriting existing chunk files if any
		Run()

	if err != nil {
		return nil, fmt.Errorf("ffmpeg splitting failed for %s: %w", filePath, err)
	}

	log.Printf("FFmpeg splitting process completed for %s.", filePath)

	// Glob for the created chunk files
	globPattern := filepath.Join(dir, fmt.Sprintf("%s_chunk_*%s", baseName, ext))
	chunkFilePaths, err := filepath.Glob(globPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob for chunk files with pattern %s: %w", globPattern, err)
	}

	if len(chunkFilePaths) == 0 {
		log.Printf("No chunks found for %s with pattern %s. This might indicate an issue with ffmpeg splitting or the glob pattern.", filePath, globPattern)
		// It's possible ffmpeg didn't create chunks if the file was just over the limit but less than one segment_time.
		// In such a case, it might produce a single output file that isn't numbered, or it might behave unexpectedly.
		// For now, we'll return the original file path if no chunks are found, assuming ffmpeg handled it or it was a small file.
		// A more robust solution might check ffmpeg's output or ensure at least one chunk is always created if splitting was attempted.
		// However, ffmpeg with -f segment should create at least one chunk even if input duration < segment_time.
		// If it truly created no files, it's an error state.
		return nil, fmt.Errorf("ffmpeg splitting was expected for %s, but no chunk files were found with pattern %s", filePath, globPattern)
	}

	// Sort the chunk file paths to ensure they are in the correct order
	sort.Strings(chunkFilePaths)

	log.Printf("Successfully split %s into %d chunks:", filePath, len(chunkFilePaths))
	for _, chunkPath := range chunkFilePaths {
		log.Printf("  - %s", chunkPath)
	}

	return chunkFilePaths, nil
}
