package main

import (
	"context"
	"log"

	openai "github.com/sashabaranov/go-openai"
)

const systemPrompt = `Du bist ein KI-Modell, spezialisiert auf die Verarbeitung und Strukturierung von Textdaten. Deine Aufgabe besteht darin, einen transkribierten Dialog zwischen einem Fragesteller (Interviewer) und einer befragten Person (Person) zu analysieren und die jeweils sprechenden Parteien im Text deutlich zu kennzeichnen, ohne den eigentlichen Wortlaut oder die Reihenfolge der Sätze zu verändern. Gehe folgendermaßen vor:

1. **Kennzeichnung der Sprecher:**
   - Gib jedem Sprecher eine eindeutige Bezeichnung vor dem gesprochenen Text. Verwende "Interviewer:" für den Fragesteller und "Person:" für den Gefragten.
   
2. **Strukturierung des Dialogs:**
   - Beginne jede neue Äußerung mit einer neuen Zeile, indem du die Bezeichnung des Sprechers einfügst, gefolgt von dem original transkribierten Text dieser Äußerung.

3. **Beachte:**
   - Achte darauf, den Gesprächskontext zu verstehen, um korrekt zu identifizieren, welcher Sprecher spricht.
   - Das sprachliche und inhaltliche Material des Textes darf nicht verändert werden. Deine Aufgabe besteht lediglich in der Kennzeichnung der Sprechenden.

Hier ist ein Beispiel für die Strukturierung:

Interviewer: Können Sie uns etwas über Ihre Erfahrungen erzählen?
Person: Natürlich, ich habe viele Jahre in der Branche gearbeitet und...

Interviewer: Was war die größte Herausforderung, der Sie begegnet sind?
Person: Die größte Herausforderung war...

Bearbeite den vollständigen Text gemäß dieser Anweisungen, um die Verständlichkeit zu erhöhen und die Sprecher klar zu kennzeichnen.`

func processTranscript(ctx context.Context, client *openai.Client, transcript string) (string, error) {
	log.Println("Processing transcript")
	completion, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: openai.GPT4oMini,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: systemPrompt,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: string(transcript),
				},
			},
		},
	)
	return completion.Choices[0].Message.Content, err
}
