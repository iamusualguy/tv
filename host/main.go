package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const ollamaURL = "http://localhost:11434/api/generate"
const model = "llama3.2" // change to your model

type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type OllamaResponse struct {
	Response string `json:"response"`
}

func main() {
	prompt, err := os.ReadFile("prompt.txt")
	fmt.Println("You says:", string(prompt))

	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	reqBody := OllamaRequest{
		Model:  model,
		Prompt: string(prompt),
		Stream: false,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return
	}

	resp, err := http.Post(ollamaURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		fmt.Println("Request failed:", err)
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response:", err)
		return
	}
	fmt.Println("--- says:", string(respBody))

	var ollamaResp OllamaResponse
	json.Unmarshal(respBody, &ollamaResp)

	fmt.Println("Ollama says:", ollamaResp.Response)
}
