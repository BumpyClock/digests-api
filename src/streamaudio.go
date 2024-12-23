// Package main provides the main functionality for the web server.
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
	"github.com/sirupsen/logrus"
)

var (
	ttsClient *texttospeech.Client
	once      sync.Once
)

/**
 * @function initTTSClient
 * @description Initializes the Google Cloud Text-to-Speech client.
 * @returns {void}
 * @dependencies texttospeech.NewClient, log
 */
func initTTSClient() {
	var err error
	ttsClient, err = texttospeech.NewClient(context.Background())
	if err != nil {
		log.WithFields(logrus.Fields{
			"error": err,
		}).Fatal("Failed to create TTS client")
	}
}

/**
 * @function splitTextIntoChunks
 * @description Splits a text into chunks of a specified maximum size,
 *              ensuring that words are not split across chunks.
 * @param {string} text The text to split.
 * @param {int} maxChunkSize The maximum size of each chunk.
 * @returns {[]string} A slice of text chunks.
 */
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

/**
 * @function streamAudioHandler
 * @description Handles HTTP requests to the /streamaudio endpoint.
 *              It expects a POST request with a JSON body containing text or a URL to synthesize.
 *              It synthesizes the text to speech using the Google Cloud Text-to-Speech API,
 *              caches the audio content, and streams it to the client.
 * @param {http.ResponseWriter} w The HTTP response writer.
 * @param {*http.Request} r The HTTP request.
 * @returns {void}
 * @dependencies cache, log, once, initTTSClient, ttsClient, splitTextIntoChunks
 */
func streamAudioHandler(w http.ResponseWriter, r *http.Request) {
	log.Info("Received request to stream audio")

	// Ensure it's a POST request
	if r.Method != http.MethodPost {
		log.WithFields(logrus.Fields{
			"method": r.Method,
		}).Warn("[streamAudioHandler] Invalid method")
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read and parse the request body
	var ttsReq TTSRequest
	err := json.NewDecoder(r.Body).Decode(&ttsReq)
	if err != nil {
		log.WithFields(logrus.Fields{
			"error": err,
		}).Error("[streamAudioHandler] Error decoding request body")
		http.Error(w, "Bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	log.WithFields(logrus.Fields{
		"text": ttsReq.Text,
		"url":  ttsReq.Url,
	}).Debug("[streamAudioHandler] Request received")

	// Check if text is provided
	if ttsReq.Text == "" {
		log.Warn("[streamAudioHandler] No text provided")
		http.Error(w, "No text provided", http.StatusBadRequest)
		return
	} else if ttsReq.Url != "" {
		// Check if the URL is valid
		if !(strings.HasPrefix(ttsReq.Url, "http://") || strings.HasPrefix(ttsReq.Url, "https://")) {
			log.WithFields(logrus.Fields{
				"url": ttsReq.Url,
			}).Warn("[streamAudioHandler] Invalid URL provided")
			http.Error(w, "Invalid URL provided", http.StatusBadRequest)
			return
		}
	}

	cacheKey := ttsReq.Url
	if cacheKey == "" {
		cacheKey = createHash(ttsReq.Text)
	}

	var cachedAudio []byte
	// Check if the audio content is cached
	err = cache.Get(audio_prefix, cacheKey, &cachedAudio)
	if err == nil {
		log.WithFields(logrus.Fields{
			"key": cacheKey,
		}).Debug("[streamAudioHandler] Audio content found in cache")
		// Set the headers and write the audio content to the response
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Header().Set("Content-Length", fmt.Sprint(len(cachedAudio)))

		// Write the audio content to the response
		_, err = w.Write(cachedAudio)
		if err != nil {
			log.WithFields(logrus.Fields{
				"error": err,
			}).Error("[streamAudioHandler] Failed to write audio content to response")
		}
		return
	}

	// Initialize the TTS client once
	once.Do(initTTSClient)

	log.WithFields(logrus.Fields{
		"text": ttsReq.Text,
	}).Debug("[streamAudioHandler] Text to be synthesized")
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
				AudioEncoding: *texttospeechpb.AudioEncoding_MP3.Enum(),
			},
		}
		// Perform the text-to-speech request
		resp, err := ttsClient.SynthesizeSpeech(context.Background(), &req)
		if err != nil {
			log.WithFields(logrus.Fields{
				"error": err,
			}).Error("[streamAudioHandler] Failed to synthesize speech")
			http.Error(w, "Failed to synthesize speech", http.StatusInternalServerError)
			return
		} else {
			log.Debug("[streamAudioHandler] Speech synthesized successfully")
		}

		// Append the audio content to the buffer
		audioContent.Write(resp.AudioContent)
	}

	// Cache the audio content
	if err := cache.Set(audio_prefix, cacheKey, audioContent.Bytes(), 7*24*time.Hour); err != nil {
		log.WithFields(logrus.Fields{
			"key":   cacheKey,
			"error": err,
		}).Error("[streamAudioHandler] Failed to cache audio content")
	} else {
		log.WithFields(logrus.Fields{
			"key": cacheKey,
		}).Debug("[streamAudioHandler] Audio content cached successfully")
	}

	// Set the headers and write the audio content to the response
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Content-Length", fmt.Sprint(audioContent.Len()))

	// Write the audio content to the response
	_, err = w.Write(audioContent.Bytes())
	if err != nil {
		log.WithFields(logrus.Fields{
			"error": err,
		}).Error("[streamAudioHandler] Failed to write audio content to response")
	}
}
