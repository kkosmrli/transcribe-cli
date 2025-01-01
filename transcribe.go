package main

import (
	"context"
	"fmt"
	"log"
	"os"

	openai "github.com/sashabaranov/go-openai"
)

const transcriptionPrompt = `Frage: Erzähle bitte etwas über dich und deine Position im Unternehmen. Antwort: Ja, mein Name ist XY, ich bin 40 Jahre alt und ich arbeite hier bei der X im Bereich Recht als Rechtsanwältin. Und ich kümmere mich um die Vertragsabwicklung und die rechtlichen Angelegenheiten im Bezug auf unsere Mitverlegungen. Genau.`

func transcribe(ctx context.Context, path string, client *openai.Client, audioFile *os.File) (string, error) {
	req := openai.AudioRequest{
		Model:    openai.Whisper1,
		FilePath: path,
		Language: "de",
		Format:   "text",
		Prompt:   transcriptionPrompt,
	}
	resp, err := client.CreateTranscription(ctx, req)
	if err != nil {
		fmt.Printf("Transcription error: %v\n", err)
		return "", fmt.Errorf("transcription error: %v", err)
	}

	err = os.WriteFile(audioFile.Name()+"_raw.txt", []byte(resp.Text), 0644)
	if err != nil {
		log.Printf("failed to write transcription to file: %v", err)
	}
	log.Println("Transcription saved to", audioFile.Name()+"_raw.txt")
	return resp.Text, nil
}
