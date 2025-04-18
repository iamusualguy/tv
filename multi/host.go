package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"text/template"
	"time"
)

type Config struct {
	OllamaURL      string `json:"ollamaURL"`
	Model          string `json:"model"`
	TTSURL         string `json:"ttsURL"`
	PromptFileName string `json:"promptFileName"`
}

const configFile = "config.json"

var config Config

func loadConfig() error {
	// Read the config file using os.ReadFile
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal the config JSON into the Config struct
	err = json.Unmarshal(data, &config)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return nil
}

type OllamaRequest struct {
	Model   string                 `json:"model"`
	Prompt  string                 `json:"prompt"`
	Stream  bool                   `json:"stream"`
	Options map[string]interface{} `json:"options"`
}

type OllamaResponse struct {
	Response string `json:"response"`
}

// Create a local random generator with a random seed
var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

// GenerateRandomValue returns a float64 between 0.2 and 1.0
func GenerateRandomValue() float64 {
	return 0.2 + rng.Float64()*(1.0-0.2)
}

func createIntroText(trackInfo string) string {
	// Read the config file using os.ReadFile
	data, err := os.ReadFile(configFile)
	if err != nil {
		fmt.Println("failed to read config file: %w", err)
	}

	// Unmarshal the config JSON into the Config struct
	err = json.Unmarshal(data, &config)
	if err != nil {
		fmt.Println("failed to unmarshal config: %w", err)
	}
	prompt, err := os.ReadFile(config.PromptFileName)
	if err != nil {
		fmt.Println("file name is:", string(config.PromptFileName))

		fmt.Println("Error reading prompt file:", err)
		return trackInfo
	}

	result, err := fillTemplate(string(prompt), trackInfo)
	if err != nil {
		fmt.Println("Error filling template:", err)
		return trackInfo
	}

	// fmt.Println("You says:", string(result))

	reqBody := OllamaRequest{
		Model:  config.Model,
		Prompt: string(result),
		Stream: false,
		Options: map[string]interface{}{
			"temperature": GenerateRandomValue(),
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return trackInfo
	}

	resp, err := http.Post(config.OllamaURL, "application/json", bytes.NewBuffer(body))
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

	// fmt.Println("--- says:", string(respBody))

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
	text = transliterate(text)
	fmt.Println(text)
	// URL encode the text parameter
	encodedText := url.QueryEscape(text)

	// Construct the full URL with all parameters
	requestURL := fmt.Sprintf("%s?speaker=baya&sample_rate=48000&pitch=50&rate=50&text=%s", config.TTSURL, encodedText)

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
