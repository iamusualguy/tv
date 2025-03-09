package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

var (
	videoFolder    = "video"
	playlistFile   = "static/stream.m3u8"
	currentProcess *os.Process
	videoQueue     []string
	queueIndex     int
	mutex          sync.Mutex
)

func findVideos() {
	files, err := os.ReadDir(videoFolder)
	if err != nil {
		log.Fatal(err)
	}
	videoQueue = []string{}
	for _, file := range files {
		if !file.IsDir() {
			ext := filepath.Ext(file.Name())
			if ext == ".mp4" || ext == ".mkv" || ext == ".mp3" {
				videoQueue = append(videoQueue, file.Name())
			}
		}
	}
	if len(videoQueue) == 0 {
		log.Fatal("No media files found in folder.")
	}
}

func startNextVideo() {
	mutex.Lock()
	defer mutex.Unlock()

	if len(videoQueue) == 0 {
		findVideos()
	}

	videoFile := videoQueue[queueIndex]
	videoPath := filepath.Join(videoFolder, videoFile)
	queueIndex = (queueIndex + 1) % len(videoQueue)

	// Check if it's an MP3 file
	var ffmpegArgs []string
	if filepath.Ext(videoFile) == ".mp3" {
		ffmpegArgs = []string{
			"-re", "-loop", "1", "-i", "placeholder.jpg", // Placeholder image
			"-i", videoPath, "-c:v", "libx264", "-preset", "ultrafast",
			"-vf", "scale=320:240:force_original_aspect_ratio=decrease,pad=320:240:(ow-iw)/2:(oh-ih)/2,drawtext=fontsize=25:fontcolor=white:text='пися палыч тв':x=25:y=25,drawtext=fontsize=18:fontcolor=white:text='%{localtime\\:%T}':x=25:y=55",
			"-c:a", "aac", "-b:a", "128k",
			"-hls_time", "3", "-hls_list_size", "10", "-f", "hls",
			"-hls_flags", "delete_segments+append_list+omit_endlist",
			"-hls_delete_threshold", "10", playlistFile,
		}
	} else {
		ffmpegArgs = []string{
			"-re", "-i", videoPath, "-c:v", "libx264", "-preset", "ultrafast",
			"-vf", "scale=320:240:force_original_aspect_ratio=decrease,pad=320:240:(ow-iw)/2:(oh-ih)/2,drawtext=fontsize=25:fontcolor=white:text='пися палыч тв':x=25:y=25,drawtext=fontsize=18:fontcolor=white:text='%{localtime\\:%T}':x=25:y=55",
			"-hls_time", "3", "-hls_list_size", "10", "-f", "hls",
			"-hls_flags", "delete_segments+append_list+omit_endlist",
			"-hls_delete_threshold", "10", playlistFile,
		}
	}

	// Run FFmpeg
	cmd := exec.Command("ffmpeg", ffmpegArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	currentProcess = cmd.Process
	cmd.Wait()

	startNextVideo() // Start the next video when the current one finishes
}

func startStreamHandler(w http.ResponseWriter, r *http.Request) {
	go startNextVideo()
	fmt.Fprintln(w, "Streaming started!")
}

func skipVideoHandler(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()

	if currentProcess != nil {
		err := currentProcess.Kill()
		if err != nil {
			fmt.Fprintln(w, "Error skipping video:", err)
			return
		}
		fmt.Fprintln(w, "Skipped to the next video!")
	}
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()

	if len(videoQueue) == 0 {
		fmt.Fprintln(w, "No videos available.")
		return
	}

	currentVideo := videoQueue[(queueIndex-1+len(videoQueue))%len(videoQueue)]
	fmt.Fprintf(w, "Currently playing: %s\n", currentVideo)
}

func main() {
	findVideos()

	http.HandleFunc("/start", startStreamHandler)
	http.HandleFunc("/skip", skipVideoHandler)
	http.HandleFunc("/status", statusHandler)

	fmt.Println("Server running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
