package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
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

func streamAudioHandler(w http.ResponseWriter, r *http.Request) {
	log.Print("Received request to stream audio")

	// Ensure it's a POST request
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read and parse the request body
	var ttsReq TTSRequest
	err := json.NewDecoder(r.Body).Decode(&ttsReq)
	if err != nil {
		http.Error(w, "Bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	log.Print("Request: ", ttsReq.Text)
	log.Print("Request URL: ", ttsReq.Url)

	// Check if text is provided
	if ttsReq.Text == "" {
		http.Error(w, "No text provided", http.StatusBadRequest)
		return
	} else if ttsReq.Url != "" {
		// Check if the URL is valid
		if !(strings.HasPrefix(ttsReq.Url, "http://") || strings.HasPrefix(ttsReq.Url, "https://")) {
			http.Error(w, "Invalid URL provided", http.StatusBadRequest)
			return
		}
	}

	cacheKey := ttsReq.Url

	var cachedAudio []byte
	// Check if the audio content is cached
	err = cache.Get(audio_prefix, cacheKey, &cachedAudio)
	if err == nil {
		log.Print("Audio content found in cache")
		// Set the headers and write the audio content to the response
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Header().Set("Content-Length", fmt.Sprint(len(cachedAudio)))

		// Write the audio content to the response
		_, err = w.Write(cachedAudio)

		if err != nil {
			log.Printf("Failed to write audio content to response: %v", err)
		}
		return
	}

	// Initialize the TTS client once
	once.Do(initTTSClient)

	log.Print("Text to be synthesized: ", ttsReq.Text)
	const maxChunkSize = 1000

	// Split text into chunks of up to 1000 characters
	chunks := splitTextIntoChunks(ttsReq.Text, maxChunkSize)

	var audioContent bytes.Buffer

	for _, chunk := range chunks {
		req := texttospeechpb.SynthesizeSpeechRequest{
			// Set the text input to be synthesized.
			Input: &texttospeechpb.SynthesisInput{
				InputSource: &texttospeechpb.SynthesisInput_Text{Text: chunk},
			},
			// Build the voice request, select the language code ("en-US") and the SSML
			// voice gender ("neutral").
			Voice: &texttospeechpb.VoiceSelectionParams{
				LanguageCode: "en-US",
				Name:         "en-US-Neural2-J",
			},
			// Select the type of audio file you want returned.
			AudioConfig: &texttospeechpb.AudioConfig{
				AudioEncoding: *texttospeechpb.AudioEncoding_OGG_OPUS.Enum(),
			},
		}
		// Perform the text-to-speech request
		resp, err := ttsClient.SynthesizeSpeech(context.Background(), &req)
		if err != nil {
			log.Printf("Failed to synthesize speech: %v", err)
			http.Error(w, "Failed to synthesize speech", http.StatusInternalServerError)
			return
		} else {
			log.Print("Speech synthesized successfully")
		}

		// Append the audio content to the buffer
		audioContent.Write(resp.AudioContent)
	}

	// Cache the audio content
	if err := cache.Set(audio_prefix, cacheKey, audioContent.Bytes(), 7*24*time.Hour); err != nil {
		log.Printf("Failed to cache audio content: %v", err)
	} else {
		log.Print("Audio content cached successfully for url: ", cacheKey)
	}

	// Set the headers and write the audio content to the response
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Content-Length", fmt.Sprint(audioContent.Len()))

	// Write the audio content to the response
	_, err = w.Write(audioContent.Bytes())
	if err != nil {
		log.Printf("Failed to write audio content to response: %v", err)
	}
}
