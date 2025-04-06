package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func skipHandler(w http.ResponseWriter, r *http.Request) {
	// Replace with your actual skip logic
	fmt.Fprintln(w, "Skip signal received!")
}
func main() {

	// Create static folder
	os.MkdirAll("static", os.ModePerm)

	// Remove existing static stream files
files, err := filepath.Glob("static/*")
	if err == nil {
		for _, f := range files {
			os.Remove(f)
		}
	}

	// Start HTTP server to listen for /skip requests.
	http.HandleFunc("/skip", skipHandler)
	// Serve static files (HLS stream) from /static/
	fs := http.FileServer(http.Dir("./static/"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	go func() {
		log.Println("HTTP server listening on :8080")
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

	select {}
}
