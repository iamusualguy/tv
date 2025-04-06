package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

var (
	currentProcess *exec.Cmd
	processLock    sync.Mutex
)

func skipHandler(w http.ResponseWriter, r *http.Request) {
	processLock.Lock()
	defer processLock.Unlock()

	if currentProcess != nil && currentProcess.Process != nil {
		// Kill the current FFmpeg process
		err := currentProcess.Process.Kill()
		if err != nil {
			log.Printf("Error killing FFmpeg process: %v", err)
			http.Error(w, "Failed to kill FFmpeg process", http.StatusInternalServerError)
			return
		}
		log.Println("FFmpeg process killed via skip request")
		currentProcess = nil
	}

	fmt.Fprintln(w, "Skip signal received! FFmpeg process terminated.")
}

func streamMP3(filePath string, wg *sync.WaitGroup) {
	defer wg.Done()

	outputDir := "./static"
	playlistFile := filepath.Join(outputDir, "stream.m3u8")

	// Build FFmpeg command to convert MP3 to HLS stream
	cmd := exec.Command("ffmpeg", "-re", "-y", "-vn",
		"-i", filePath,
		"-c:a", "aac",
		"-b:a", "128k",
		"-f", "hls",
		"-hls_time", "2",
		"-hls_list_size", "5",
		"-hls_segment_filename", filepath.Join(outputDir, "segment_%01d.ts"),
		"-hls_flags", "delete_segments+append_list+omit_endlist",
		playlistFile)

	processLock.Lock()
	currentProcess = cmd
	processLock.Unlock()

	log.Printf("Starting FFmpeg streaming for file: %s", filePath)

	// Start the FFmpeg process
	err := cmd.Start()
	if err != nil {
		log.Printf("Error starting FFmpeg: %v", err)
		return
	}

	// Wait for the process to complete (either naturally or by being killed)
	err = cmd.Wait()
	if err != nil {
		// Check if the error is due to the process being killed by the skip handler
		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			log.Printf("FFmpeg process terminated: %v", err)
		} else {
			log.Printf("FFmpeg process error: %v", err)
		}
	} else {
		log.Printf("FFmpeg process completed successfully for file: %s", filePath)
	}

	processLock.Lock()
	if currentProcess == cmd {
		currentProcess = nil
	}
	processLock.Unlock()
}

func shuffleArray(arr []string) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(arr), func(i, j int) {
		arr[i], arr[j] = arr[j], arr[i]
	})
}
func startStreamingLoop(mp3Files []string) {
	// Initial shuffle
	for {
		shuffleArray(mp3Files)

		var wg sync.WaitGroup

		// Process MP3 files one by one
		for _, mp3File := range mp3Files {
			wg.Add(1)
			// Start streaming the current MP3 file and wait for it to complete or be skipped
			streamMP3(mp3File, &wg)

			// Check if we need to stop processing due to a shutdown signal
			processLock.Lock()
			wasKilled := (currentProcess == nil)
			processLock.Unlock()

			if wasKilled {
				log.Println("Streaming was interrupted. Moving to the next file.")
			}
		}

		// Wait for all processes to complete
		wg.Wait()
		log.Println("All MP3 files have been processed")
	}
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

	// Start HTTP server to listen for /skip requests
	http.HandleFunc("/skip", skipHandler)

	// Serve static files (HLS stream) from /static/
	fs := http.FileServer(http.Dir("./static/"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	go func() {
		log.Println("HTTP server listening on :8582")
		log.Fatal(http.ListenAndServe(":8582", nil))
	}()

	// Wait a bit for the HTTP server to start
	time.Sleep(500 * time.Millisecond)

	var mp3Files []string

	// Walk through the directory and subdirectories
	err = filepath.Walk("./", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Println(err)
			return err
		}
		// Check if the file is an MP3
		if !info.IsDir() && filepath.Ext(path) == ".mp3" {
			mp3Files = append(mp3Files, path)
		}
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

	// Print the found MP3 files
	fmt.Println("Found MP3 files:")
	for _, file := range mp3Files {
		fmt.Println(file)
	}

	if len(mp3Files) == 0 {
		log.Println("No MP3 files found in current directory")
	} else {
		log.Printf("Found %d MP3 files. Starting streaming service...", len(mp3Files))
		go startStreamingLoop(mp3Files)
	}

	// Keep the program running indefinitely
	select {}
}

