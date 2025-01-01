package main

import (
	"log"

	ffmpeg_go "github.com/u2takey/ffmpeg-go"
)

func compressFile(filePath string) string {
	log.Println("Compressing audio file")
	err := ffmpeg_go.Input(filePath).Output(filePath+"compressed.m4a", ffmpeg_go.KwArgs{"b:a": "48k"}).OverWriteOutput().Run()
	if err != nil {
		log.Fatalf("error: %s", err)
	}
	log.Println("Audio file compressed")
	return filePath + "compressed.m4a"
}
