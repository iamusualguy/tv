package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	videoFolder   = "../video"
	videoQueue    []string
	currentIndex  int
	currentCmd    *exec.Cmd
	mutex         sync.Mutex
	isStreaming   bool
	streamingDone chan bool
)

func main() {
	// Create static folder
	os.MkdirAll("static", os.ModePerm)

	// Initialize video queue
	refillQueue()

	// Set up HTTP handlers
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/start", startStreamHandler)
	http.HandleFunc("/skip", skipVideoHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Start the server
	fmt.Println("Server started at http://0.0.0.0:8080")
	log.Fatal(http.ListenAndServe("0.0.0.0:8080", nil))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	html := `
<!DOCTYPE html>
<html>
<head>
    <title>Video Streaming Service</title>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/hls.js/1.1.5/hls.min.js"></script>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        video { width: 640px; height: 480px; background: #000; }
        button { margin: 10px 0; padding: 8px 16px; }
    </style>
</head>
<body>
    <h1>Video Streaming Service</h1>
    <video id="video" controls></video>
    <div>
        <button onclick="startStream()">Start Streaming</button>
        <button onclick="skipVideo()">Skip Video</button>
    </div>
    <script>
        function startStream() {
            fetch('/start')
                .then(response => response.text())
                .then(data => {
                    console.log(data);
                    setupPlayer();
                });
        }
        
        function skipVideo() {
            fetch('/skip')
                .then(response => response.text())
                .then(data => console.log(data));
        }
        
        function setupPlayer() {
            var video = document.getElementById('video');
            if (Hls.isSupported()) {
                var hls = new Hls();
                hls.loadSource('/static/stream.m3u8');
                hls.attachMedia(video);
                hls.on(Hls.Events.MANIFEST_PARSED, function() {
                    video.play();
                });
            } else if (video.canPlayType('application/vnd.apple.mpegurl')) {
                video.src = '/static/stream.m3u8';
                video.addEventListener('loadedmetadata', function() {
                    video.play();
                });
            }
        }
        
        // Auto setup player if stream is already running
        window.onload = function() {
            setupPlayer();
        };
    </script>
</body>
</html>
`
	fmt.Fprintf(w, html)
}

func startStreamHandler(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()

	if isStreaming {
		fmt.Fprintf(w, "Streaming is already running")
		return
	}

	// Remove existing static stream files
	files, _ := filepath.Glob("static/stream*.ts")
	for _, f := range files {
		os.Remove(f)
	}
	os.Remove("static/stream.m3u8")

	// Start streaming
	isStreaming = true
	streamingDone = make(chan bool)
	go startStreamProcess()

	fmt.Fprintf(w, "Streaming started")
}

func skipVideoHandler(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()

	if currentCmd != nil && currentCmd.Process != nil {
		// Kill the current process
		currentCmd.Process.Kill()
		fmt.Fprintf(w, "Skipped to the next video")
	} else {
		fmt.Fprintf(w, "No video is currently playing")
	}
}

func startStreamProcess() {
	for isStreaming {
		processNextVideo()
	}
	close(streamingDone)
}

func processNextVideo() {
	mutex.Lock()
	if len(videoQueue) == 0 {
		refillQueue()
	}

	if len(videoQueue) == 0 {
		mutex.Unlock()
		log.Println("No videos found in the queue")
		time.Sleep(5 * time.Second)
		return
	}

	videoFile := videoQueue[currentIndex]
	currentIndex = (currentIndex + 1) % len(videoQueue)
	mutex.Unlock()

	videoPath := filepath.Join(videoFolder, videoFile)
	log.Printf("Processing video: %s\n", videoPath)

	// Build FFmpeg command similar to the Python version
	cmd := exec.Command(
		"ffmpeg",
		"-re",
		"-i", videoPath,
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-vf", `[in]scale=320:240:force_original_aspect_ratio=decrease,pad=320:240:(ow-iw)/2:(oh-ih)/2,drawtext=fontsize=25:fontcolor=white:text='пися палыч тв':x=25:y=25,drawtext=fontsize=18:fontcolor=white:text='%{localtime\:%T}':x=25:y=55[out]`,
		"-hls_time", "3",
		"-hls_list_size", "10",
		"-f", "hls",
		"-hls_flags", "delete_segments+append_list+omit_endlist",
		"-hls_delete_threshold", "10",
		"static/stream.m3u8",
	)

	// Store current command to allow killing it later
	mutex.Lock()
	currentCmd = cmd
	mutex.Unlock()

	// Execute FFmpeg
	err := cmd.Run()
	if err != nil {
		// Check if the process was killed intentionally
		if strings.Contains(err.Error(), "killed") {
			log.Println("Video was skipped")
		} else {
			log.Printf("FFmpeg error: %v\n", err)
		}
	}
}

func refillQueue() {
	// Using os.ReadDir instead of ioutil.ReadDir
	files, err := os.ReadDir(videoFolder)
	if err != nil {
		log.Printf("Error reading video folder: %v\n", err)
		videoQueue = []string{}
		return
	}

	videoQueue = []string{}
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".mp4") {
			videoQueue = append(videoQueue, file.Name())
		}
	}

	log.Printf("Video queue refilled with %d videos\n", len(videoQueue))
}
