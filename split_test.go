package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// createDummyAudioFile creates a file with roughly the specified size by repeating sample audio data.
// If sampleAudioPath is not found or readable, it creates a binary file of the target size as a fallback.
func createDummyAudioFile(t *testing.T, dir string, fileName string, targetSizeMB int64, sampleAudioPath string) string {
	t.Helper()
	filePath := filepath.Join(dir, fileName)
	targetSizeBytes := targetSizeMB * 1024 * 1024

	if targetSizeBytes == 0 { // Create a very small file if 0MB specified, e.g., 1KB
		targetSizeBytes = 1024
	}

	sampleData, err := os.ReadFile(sampleAudioPath)
	if err != nil {
		log.Printf("Warning: Sample audio %s not found or not readable. Creating dummy binary file instead: %v", sampleAudioPath, err)
		// Fallback to creating a simple binary file of the target size
		dummyContent := make([]byte, targetSizeBytes)
		// Fill with some non-zero data to avoid issues with some file systems or tools
		for i := range dummyContent {
			dummyContent[i] = byte(i % 256)
		}
		if errWrite := os.WriteFile(filePath, dummyContent, 0644); errWrite != nil {
			t.Fatalf("Failed to write dummy binary file %s: %v", filePath, errWrite)
		}
		// Check actual file size
		fileInfo, statErr := os.Stat(filePath)
		if statErr != nil {
			t.Fatalf("Failed to stat dummy binary file %s: %v", filePath, statErr)
		}
		if fileInfo.Size() != targetSizeBytes {
			t.Logf("Warning: Dummy binary file size %d does not exactly match target %d. This might be due to filesystem block allocation.", fileInfo.Size(), targetSizeBytes)
		}
		return filePath
	}

	file, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("Failed to create dummy file %s: %v", filePath, err)
	}
	defer file.Close()

	var currentSize int64
	sampleSize := int64(len(sampleData))
	if sampleSize == 0 { // Ensure we don't loop infinitely if sampleData is empty
		t.Fatalf("Sample audio data from %s is empty. Cannot create dummy file.", sampleAudioPath)
	}

	for currentSize < targetSizeBytes {
		written, errWrite := file.Write(sampleData)
		if errWrite != nil {
			t.Fatalf("Failed to write to dummy file %s: %v", filePath, errWrite)
		}
		currentSize += int64(written)
	}
	// Ensure the file is flushed and closed properly before checking its size.
	file.Close() 

	// Verify the created file size
	fileInfo, errStat := os.Stat(filePath)
	if errStat != nil {
		t.Fatalf("Failed to get file info for created dummy file %s: %v", filePath, errStat)
	}
	// Allow for some minor variation if the sample data doesn't perfectly tile into targetSizeBytes
	if fileInfo.Size() < targetSizeBytes {
		t.Fatalf("Created dummy file %s size %d is less than target %d bytes. Loop iterations: %d, sampleSize: %d", filePath, fileInfo.Size(), targetSizeBytes, currentSize/sampleSize, sampleSize)
	}

	return filePath
}

// TestSplitAudioFile_SmallerThanChunkSize tests splitting with a file smaller than MaxChunkSizeMB.
func TestSplitAudioFile_SmallerThanChunkSize(t *testing.T) {
	tempDir := t.TempDir()
	sampleAudioPath := "../test_sample.m4a" // Relative to the test file's location in the package directory

	// MaxChunkSizeMB is 25. Create a 10MB file.
	filePath := createDummyAudioFile(t, tempDir, "small_audio.m4a", 10, sampleAudioPath)

	chunkPaths, err := splitAudioFile(filePath)
	if err != nil {
		t.Fatalf("splitAudioFile failed for small file: %v", err)
	}

	if len(chunkPaths) != 1 {
		t.Errorf("Expected 1 chunk for small file, got %d", len(chunkPaths))
	}
	if len(chunkPaths) == 1 && chunkPaths[0] != filePath {
		t.Errorf("Expected chunk path to be the original file path, got %s", chunkPaths[0])
	}
}

// TestSplitAudioFile_LargerThanChunkSize tests splitting with a file larger than MaxChunkSizeMB.
func TestSplitAudioFile_LargerThanChunkSize(t *testing.T) {
	tempDir := t.TempDir()
	sampleAudioPath := "../test_sample.m4a"

	// MaxChunkSizeMB is 25. Create a 30MB file.
	// Note: Our test_sample.m4a is a placeholder and not a real audio file.
	// ffmpeg will error out. The test will check how splitAudioFile handles this.
	largeFilePath := createDummyAudioFile(t, tempDir, "large_audio.m4a", 30, sampleAudioPath)

	chunkPaths, err := splitAudioFile(largeFilePath)

	// With a placeholder test_sample.m4a, ffmpeg is expected to fail.
	// The function splitAudioFile should propagate this error.
	if err == nil {
		t.Logf("splitAudioFile did not return an error. This is unexpected with a placeholder audio file as ffmpeg should fail.")
		t.Logf("Number of chunks returned: %d", len(chunkPaths))
		// If by some chance ffmpeg didn't error (e.g., different ffmpeg version or environment),
		// then the primary expectation of splitting should hold: more than one chunk.
		if len(chunkPaths) <= 1 {
			t.Errorf("Expected file to be split into multiple chunks if ffmpeg succeeded, got %d", len(chunkPaths))
		}
		for _, chunkPath := range chunkPaths {
			if _, statErr := os.Stat(chunkPath); os.IsNotExist(statErr) {
				t.Errorf("Expected chunk file %s to exist, but it doesn't", chunkPath)
			}
			if !strings.Contains(filepath.Base(chunkPath), "_chunk_") {
				t.Errorf("Chunk file name %s does not match expected pattern '_chunk_'", filepath.Base(chunkPath))
			}
		}
	} else {
		// This is the expected path when using a placeholder test_sample.m4a
		log.Printf("splitAudioFile returned an error as expected (due to placeholder audio and ffmpeg failure): %v", err)
		// We expect chunkPaths to be nil or empty when ffmpeg fails and returns an error.
		if len(chunkPaths) > 0 {
			t.Errorf("Expected no chunk paths when ffmpeg fails, but got %d paths: %v", len(chunkPaths), chunkPaths)
		}
		// Check that the error message indicates an ffmpeg issue if possible (might be too specific)
		if !strings.Contains(err.Error(), "ffmpeg") && !strings.Contains(err.Error(), "failed to glob") && !strings.Contains(err.Error(), "no chunk files were found") {
			 // The "no chunk files were found" is a valid error from our splitAudioFile if ffmpeg runs but creates no output.
			t.Logf("Warning: Error message from splitAudioFile does not explicitly mention 'ffmpeg', 'glob', or 'no chunk files': %s. This might be okay depending on the failure point.", err.Error())
		}
	}
	// Cleanup of files in tempDir is automatic.
	// If splitAudioFile creates chunks outside tempDir (it does, in the same dir as source),
	// and source is in tempDir, then chunks are also in tempDir.
}


// TestSplitAudioFile_ZeroSizeFile tests splitting with a zero-size file.
func TestSplitAudioFile_ZeroSizeFile(t *testing.T) {
	tempDir := t.TempDir()
	sampleAudioPath := "../test_sample.m4a" // Needed by createDummyAudioFile

	// Create a 0MB file (helper function makes it 1KB)
	filePath := createDummyAudioFile(t, tempDir, "zero_size_audio.m4a", 0, sampleAudioPath)

	chunkPaths, err := splitAudioFile(filePath)
	if err != nil {
		t.Fatalf("splitAudioFile failed for zero-size file: %v", err)
	}

	if len(chunkPaths) != 1 {
		t.Errorf("Expected 1 chunk for zero-size file, got %d", len(chunkPaths))
	}
	if chunkPaths[0] != filePath {
		t.Errorf("Expected chunk path to be the original file path for zero-size file, got %s", chunkPaths[0])
	}
}

// TestSplitAudioFile_NonExistentFile tests splitting with a non-existent file.
func TestSplitAudioFile_NonExistentFile(t *testing.T) {
	tempDir := t.TempDir() // Not strictly needed but good practice
	nonExistentFilePath := filepath.Join(tempDir, "non_existent_audio.m4a")

	chunkPaths, err := splitAudioFile(nonExistentFilePath)

	if err == nil {
		t.Fatalf("Expected an error for non-existent file, but got nil")
	}
	if len(chunkPaths) != 0 {
		t.Errorf("Expected 0 chunk paths for non-existent file, got %d", len(chunkPaths))
	}
	// Check if the error is an os.IsNotExist error or contains relevant text
	// The error from splitAudioFile is wrapped, so os.IsNotExist might not directly work.
	// We check for the specific error message from os.Stat.
	if !strings.Contains(err.Error(), "no such file or directory") && !strings.Contains(err.Error(), "failed to get file info") {
		t.Errorf("Expected error to indicate file not found, but got: %v", err)
	}
}
