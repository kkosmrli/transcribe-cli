package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	openai "github.com/sashabaranov/go-openai"
)

func main() {
	ctx := context.Background()
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("failed to load .env file: %v", err)
	}
	apiKey := os.Getenv("API_KEY")

	var path string
	var compress bool
	flag.StringVar(&path, "input", "", "Path to the audio file")
	flag.BoolVar(&compress, "compress", false, "Compress the audio file before transcribing")
	flag.Parse()

	if path == "" {
		log.Fatal("input file is required")
	}

	CheckFileType(path)

	originalPath := path // Keep original path for final output naming and temp file logic
	processedPath := path // Path to the file that will be processed (original or compressed)

	if compress {
		log.Println("Compressing file:", originalPath)
		processedPath = compressFile(originalPath) // compressFile returns the new path of the compressed file
		log.Println("File compressed to:", processedPath)
	}

	log.Println("Checking if file needs splitting:", processedPath)
	audioFilePaths, err := splitAudioFile(processedPath) // MaxChunkSizeMB is used internally by splitAudioFile
	if err != nil {
		log.Fatalf("failed to split audio file: %v", err)
	}

	var tempFiles []string
	// Determine temporary files for cleanup
	if len(audioFilePaths) > 1 { // Splitting occurred
		tempFiles = append(tempFiles, audioFilePaths...)
		// If compression also happened and the compressed file is different from original, it's also temp
		if compress && processedPath != originalPath {
			tempFiles = append(tempFiles, processedPath)
		}
	} else if compress && processedPath != originalPath { // No splitting, but compression happened
		tempFiles = append(tempFiles, processedPath)
	}

	defer func() {
		if len(tempFiles) > 0 {
			log.Println("Cleaning up temporary files:", tempFiles)
			for _, fPath := range tempFiles {
				if err := os.Remove(fPath); err != nil {
					log.Printf("Warning: failed to remove temporary file %s: %v", fPath, err)
				}
			}
			log.Println("Temporary files cleaned up.")
		}
	}()

	var fullTranscript strings.Builder
	client := openai.NewClient(apiKey) // Initialize client once

	for i, chunkPath := range audioFilePaths {
		log.Printf("Processing chunk %d/%d: %s", i+1, len(audioFilePaths), chunkPath)

		audioFile, errOpen := os.Open(chunkPath)
		if errOpen != nil {
			log.Fatalf("failed to open audio chunk %s: %v", chunkPath, errOpen)
		}

		// Transcribe uses chunkPath for the API request and audioFile for some internal naming if needed.
		transcript, errTranscribe := transcribe(ctx, chunkPath, client, audioFile)
		audioFile.Close() // Close the chunk file immediately after transcription

		if errTranscribe != nil {
			log.Fatalf("failed to transcribe audio chunk %s: %v", chunkPath, errTranscribe)
		}
		fullTranscript.WriteString(transcript)
		if i < len(audioFilePaths)-1 { // Add newline separator between transcripts, but not after the last one.
			fullTranscript.WriteString("\n\n")
		}
	}

	log.Println("All chunks processed. Processing combined transcript...")
	result, err := processTranscript(ctx, client, fullTranscript.String())
	if err != nil {
		log.Fatalf("failed to create chat completion for combined transcript: %v", err)
	}

	// Save the final output based on the original input file name
	originalInputBaseName := strings.TrimSuffix(filepath.Base(originalPath), filepath.Ext(originalPath))
	outputFileName := filepath.Join(filepath.Dir(originalPath), originalInputBaseName+"_structured_transcription.txt")
	err = os.WriteFile(outputFileName, []byte(result), 0644)
	if err != nil {
		log.Fatalf("failed to write structured transcription to file %s: %v", outputFileName, err)
	}
	log.Println("Structured transcription saved to", outputFileName)
}

func CheckFileType(path string) {
	allowedExtensions := []string{"mp3", "mp4", "mpeg", "mpga", "m4a", "wav", "webm"}
	ext := filepath.Ext(path)

	if len(ext) > 0 && ext[0] == '.' {
		ext = ext[1:] // Remove the leading dot
	}
	ext = strings.ToLower(ext)

	for _, allowedExt := range allowedExtensions {
		if ext == allowedExt {
			return // Valid file type
		}
	}

	// If the loop finishes, the extension is not allowed
	allowedTypesStr := strings.Join(allowedExtensions, ", ")
	originalExtWithDot := filepath.Ext(path) // Get original extension with dot for the error message
	log.Fatalf("Invalid file type: %s. Allowed types are: %s", originalExtWithDot, allowedTypesStr)
}
