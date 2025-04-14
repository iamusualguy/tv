package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/template"
)

const ollamaURL = "http://localhost:11434/api/generate"
const model = "llama3.2" // "yandex/YandexGPT-5-Lite-8B-instruct-GGUF:latest" // change to your model like: llama3.2

type OllamaRequest struct {
	Model   string                 `json:"model"`
	Prompt  string                 `json:"prompt"`
	Stream  bool                   `json:"stream"`
	Options map[string]interface{} `json:"options"`
}

type OllamaResponse struct {
	Response string `json:"response"`
}

func main() {
	prompt, err := os.ReadFile("prompt.txt")
	result, err := fillTemplate(string(prompt), "Dance party. Dance! Dance! - Love Anthem")
	fmt.Println("You says:", string(result))

	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	reqBody := OllamaRequest{
		Model:  model,
		Prompt: string(result),
		Stream: false,
		Options: map[string]interface{}{
			"temperature": 0.51,
		},
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

