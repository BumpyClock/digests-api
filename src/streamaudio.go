package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
)

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

	// Check if text is provided
	if ttsReq.Text == "" {
		http.Error(w, "No text provided", http.StatusBadRequest)
		return
	}

	// Create context and client
	ctx := context.Background()

	client, err := texttospeech.NewClient(ctx)
	if err != nil {
		log.Printf("Failed to create TTS client: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer client.Close()
	log.Print("TTS client created")
	log.Print("Text to be synthesized: ", ttsReq.Text)

	// req := texttospeechpb.SynthesizeSpeechRequest{
	// 	Input: &texttospeechpb.SynthesisInput{
	// 		InputSource: &texttospeechpb.SynthesisInput_Text{Text: ttsReq.Text},
	// 	},

	// 	Voice: &texttospeechpb.VoiceSelectionParams{
	// 		LanguageCode: "en-US",
	// 		SsmlGender:   texttospeechpb.SsmlVoiceGender_NEUTRAL,
	// 	},
	// 	// Select the type of audio file you want returned.
	// 	AudioConfig: &texttospeechpb.AudioConfig{
	// 		AudioEncoding: texttospeechpb.AudioEncoding_MP3,
	// 	},
	// }
	req := texttospeechpb.SynthesizeSpeechRequest{
		// Set the text input to be synthesized.
		Input: &texttospeechpb.SynthesisInput{
			InputSource: &texttospeechpb.SynthesisInput_Text{Text: "Hello, World!"},
		},
		// Build the voice request, select the language code ("en-US") and the SSML
		// voice gender ("neutral").
		Voice: &texttospeechpb.VoiceSelectionParams{
			LanguageCode: "en-US",
			SsmlGender:   texttospeechpb.SsmlVoiceGender_NEUTRAL,
		},
		// Select the type of audio file you want returned.
		AudioConfig: &texttospeechpb.AudioConfig{
			AudioEncoding: texttospeechpb.AudioEncoding_MP3,
		},
	}
	// Perform the text-to-speech request
	resp, err := client.SynthesizeSpeech(ctx, &req)
	if err != nil {
		log.Printf("Failed to synthesize speech: %v", err)
		http.Error(w, "Failed to synthesize speech", http.StatusInternalServerError)
		return
	} else {
		log.Print("Speech synthesized successfully")
	}

	// Set the headers and write the audio content to the response
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Content-Length", fmt.Sprint(len(resp.AudioContent)))

	// Write the audio content to the response
	_, err = w.Write(resp.AudioContent)
	if err != nil {
		log.Printf("Failed to write audio content to response: %v", err)
	}
}
