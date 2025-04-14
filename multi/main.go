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

	// Extract and log metadata before streaming
	metadataString := extractMetadataString(filePath)
	log.Printf("Now playing: %s", metadataString)

	outputDir := "./static"
	playlistFile := filepath.Join(outputDir, "stream.m3u8")

	// Check if intro file (output.wav) exists
	introFile := "output.wav"
	_, err := os.Stat(introFile)
	hasIntro := err == nil

	var cmd *exec.Cmd

	if hasIntro {
		log.Printf("Found intro file, playing intro overlay with music")

		// Use FFmpeg filter complex to mix the intro with the music
		// The filter will overlay the intro on top of the music, with the intro at normal volume and the music at reduced volume during the intro
		cmd = exec.Command("ffmpeg", "-re", "-y",
			"-i", filePath, // Input 0: main MP3 file
			"-i", introFile, // Input 1: intro WAV file
			"-filter_complex",
			// Complex filter to mix both audio streams:
			// 1. Take full music track
			// 2. Take intro audio
			// 3. Mix them, lowering music volume during intro and then restore
			"[0:a]volume=1[music];"+
				"[1:a]aformat=sample_rates=44100:channel_layouts=stereo,volume=5,adelay=1000|1000[intro];"+
				"[music][intro]amix=inputs=2:duration=first:dropout_transition=3[aout]",
			"-map", "[aout]", // Map the output of the filter
			"-c:a", "aac",
			"-b:a", "128k",
			"-f", "hls",
			"-hls_time", "2",
			"-hls_list_size", "5",
			"-hls_segment_filename", filepath.Join(outputDir, "segment_%01d.ts"),
			"-hls_flags", "delete_segments+append_list+omit_endlist",
			playlistFile)
	} else {
		log.Printf("No intro file found, playing music only")

		// Original command for just the MP3 file
		cmd = exec.Command("ffmpeg", "-re", "-y", "-vn",
			"-i", filePath,
			"-c:a", "aac",
			"-b:a", "128k",
			"-f", "hls",
			"-hls_time", "2",
			"-hls_list_size", "5",
			"-hls_segment_filename", filepath.Join(outputDir, "segment_%01d.ts"),
			"-hls_flags", "delete_segments+append_list+omit_endlist",
			playlistFile)
	}

	processLock.Lock()
	currentProcess = cmd
	processLock.Unlock()

	log.Printf("Starting FFmpeg streaming for file: %s", filePath)

	// Start the FFmpeg process
	err = cmd.Start()
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
		for i := range mp3Files {
			currentFile := mp3Files[i]
			var nextFile string
			if i+1 < len(mp3Files) {
				nextFile = mp3Files[i+1]

				// Preload next track's intro
				go func(file string) {
					meta := extractMetadataString(file)
					intro := createIntroText(meta)
					err := textToSpeechAndSave(intro, "next_intro.wav")
					if err != nil {
						log.Printf("Error preparing next intro: %v", err)
					} else {
						log.Printf("Preloaded intro for next track: %s", file)
					}
				}(nextFile)
			}

			wg.Add(1)
			streamMP3(currentFile, &wg)

			// Check if the stream was skipped
			processLock.Lock()
			wasKilled := (currentProcess == nil)
			processLock.Unlock()

			if wasKilled {
				log.Println("Streaming was interrupted. Moving to the next file.")
			}

			// Rename preloaded intro to be used by FFmpeg
			if _, err := os.Stat("next_intro.wav"); err == nil {
				os.Rename("next_intro.wav", "output.wav")
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
