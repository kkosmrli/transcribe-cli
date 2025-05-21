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

	if compress {
		path = compressFile(path)
	}

	// Read audio file
	audioFile, err := os.Open(path)
	if err != nil {
		log.Fatalf("failed to open audio file: %v", err)
	}
	defer audioFile.Close()
	checkFileSize(audioFile)

	client := openai.NewClient(apiKey)
	transcript, err := transcribe(ctx, path, client, audioFile)
	if err != nil {
		log.Fatalf("failed to transcribe audio: %v", err)
	}

	result, err := processTranscript(ctx, client, transcript)

	if err != nil {
		log.Fatalf("failed to create chat completion: %v", err)
	}

	// Write structured transcription to file
	err = os.WriteFile(audioFile.Name()+"structured_transcription.txt", []byte(result), 0644)
	if err != nil {
		log.Fatalf("failed to write structured transcription to file: %v", err)
	}
}

func checkFileSize(file *os.File) {
	fileInfo, err := file.Stat()
	if err != nil {
		log.Fatalf("failed to get file info: %v", err)
	}
	if fileInfo.Size() > 25*1024*1024 {
		log.Fatalf("file size exceeds the limit of 25MB")
	}
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
