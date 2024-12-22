package main

import (
	"context"
	"strings"
	"sync"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
)

var (
	ttsClient *texttospeech.Client
	once      sync.Once
)

func initTTSClient() {
	var err error
	ttsClient, err = texttospeech.NewClient(context.Background())
	if err != nil {
		log.Fatalf("Failed to create TTS client: %v", err)
	}
}

func splitTextIntoChunks(text string, maxChunkSize int) []string {
	var chunks []string
	words := strings.Fields(text)
	var chunk string

	for _, word := range words {
		if len(chunk)+len(word)+1 > maxChunkSize {
			chunks = append(chunks, chunk)
			chunk = word
		} else {
			if chunk != "" {
				chunk += " "
			}
			chunk += word
		}
	}
	if chunk != "" {
		chunks = append(chunks, chunk)
	}

	return chunks
}
