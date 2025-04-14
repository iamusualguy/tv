package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"text/template"
)

const ollamaURL = "http://localhost:11434/api/generate"
const model = "yandex/YandexGPT-5-Lite-8B-instruct-GGUF:latest" // change to your model like: llama3.2

type OllamaRequest struct {
	Model   string                 `json:"model"`
	Prompt  string                 `json:"prompt"`
	Stream  bool                   `json:"stream"`
	Options map[string]interface{} `json:"options"`
}

type OllamaResponse struct {
	Response string `json:"response"`
}

func createIntroText(trackInfo string) string {
	prompt, err := os.ReadFile("prompt-ru.txt")
	if err != nil {
		fmt.Println("Error reading prompt file:", err)
		return trackInfo
	}

	result, err := fillTemplate(string(prompt), trackInfo)
	if err != nil {
		fmt.Println("Error filling template:", err)
		return trackInfo
	}

	fmt.Println("You says:", string(result))

	reqBody := OllamaRequest{
		Model:  model,
		Prompt: string(result),
		Stream: false,
		Options: map[string]interface{}{
			"temperature": 1,
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return trackInfo
	}

	resp, err := http.Post(ollamaURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		fmt.Println("Request failed:", err)
		return trackInfo
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response:", err)
		return trackInfo
	}

	fmt.Println("--- says:", string(respBody))

	var ollamaResp OllamaResponse
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		fmt.Println("Error unmarshalling response:", err)
		return trackInfo
	}

	fmt.Println("Ollama says:", ollamaResp.Response)
	return ollamaResp.Response
}

func fillTemplate(templateStr, thing string) (string, error) {
	tmpl, err := template.New("template").Parse(templateStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	data := struct {
		Thing string
	}{
		Thing: thing,
	}

	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func textToSpeechAndSave(text string, outputFilePath string) error {
	// URL encode the text parameter
	encodedText := url.QueryEscape(text)

	// Construct the full URL with all parameters
	requestURL := fmt.Sprintf("http://localhost:5500/api/tts?voice=larynx:hajdurova-glow_tts&lang=en&vocoder=high&denoiserStrength=0.001&text=%s", encodedText)

	// Make the HTTP GET request
	resp, err := http.Get(requestURL)
	if err != nil {
		return fmt.Errorf("TTS request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check if the response is successful
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("TTS API returned non-OK status: %d %s", resp.StatusCode, resp.Status)
	}

	// Create a file to save the WAV data
	file, err := os.Create(outputFilePath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Copy the response body (WAV data) directly to the file
	bytesWritten, err := io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write WAV data to file: %w", err)
	}

	fmt.Printf("Successfully saved %d bytes of audio to %s\n", bytesWritten, outputFilePath)
	return nil
}
