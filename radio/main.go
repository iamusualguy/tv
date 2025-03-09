package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

type Player struct {
	playlist     []Track
	currentIndex int
	mu           sync.Mutex
	ffmpegCmd    *exec.Cmd
	isPlaying    bool
}

type Track struct {
	ID       int    `json:"id"`
	Path     string `json:"path"`
	Filename string `json:"filename"`
	Current  bool   `json:"current"`
}

func NewPlayer() *Player {
	return &Player{
		playlist:     []Track{},
		currentIndex: 0,
		isPlaying:    false,
	}
}

func (p *Player) ScanDirectory(root string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.playlist = []Track{}
	id := 1

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && (strings.HasSuffix(strings.ToLower(path), ".mp3") ||
			strings.HasSuffix(strings.ToLower(path), ".m4a") ||
			strings.HasSuffix(strings.ToLower(path), ".flac") ||
			strings.HasSuffix(strings.ToLower(path), ".ogg") ||
			strings.HasSuffix(strings.ToLower(path), ".wav")) {
			p.playlist = append(p.playlist, Track{
				ID:       id,
				Path:     path,
				Filename: filepath.Base(path),
				Current:  false,
			})
			id++
		}
		return nil
	})

	// Sort playlist by filename for consistency
	// You could implement a sort function here if needed

	return err
}

func (p *Player) GenerateM3U(outputPath string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.playlist) == 0 {
		return fmt.Errorf("playlist is empty")
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Write M3U header
	_, err = f.WriteString("#EXTM3U\n")
	if err != nil {
		return err
	}

	for _, track := range p.playlist {
		_, err = f.WriteString(fmt.Sprintf("#EXTINF:-1,%s\n%s\n", track.Filename, track.Path))
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Player) GetCurrentTrack() *Track {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.playlist) == 0 || p.currentIndex >= len(p.playlist) {
		return nil
	}

	// Set all tracks to not current
	for i := range p.playlist {
		p.playlist[i].Current = false
	}

	// Set current track
	p.playlist[p.currentIndex].Current = true

	return &p.playlist[p.currentIndex]
}

func (p *Player) GetPlaylist() []Track {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Return a copy of the playlist
	playlistCopy := make([]Track, len(p.playlist))
	copy(playlistCopy, p.playlist)

	// Update the 'Current' flag
	for i := range playlistCopy {
		playlistCopy[i].Current = (i == p.currentIndex)
	}

	return playlistCopy
}

func (p *Player) Skip() {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Stop current playback if any
	if p.ffmpegCmd != nil && p.ffmpegCmd.Process != nil {
		p.ffmpegCmd.Process.Kill()
		p.ffmpegCmd = nil
	}

	// Move to next track
	if len(p.playlist) > 0 {
		p.currentIndex = (p.currentIndex + 1) % len(p.playlist)
	}

	// If we're playing, start the next track
	if p.isPlaying {
		go p.startPlayback()
	}
}

func (p *Player) startPlayback() {
	p.mu.Lock()

	// Stop any existing playback
	if p.ffmpegCmd != nil && p.ffmpegCmd.Process != nil {
		p.ffmpegCmd.Process.Kill()
		p.ffmpegCmd = nil
	}

	if len(p.playlist) == 0 {
		p.isPlaying = false
		p.mu.Unlock()
		return
	}

	p.isPlaying = true
	currentTrack := p.playlist[p.currentIndex]
	p.mu.Unlock()

	log.Printf("Now playing: %s\n", currentTrack.Filename)

	// Use ffmpeg to convert and stream to a "null" output
	// This is just to keep track of playback and advance playlist
	cmd := exec.Command("ffmpeg", "-nostdin", "-y", "-i", currentTrack.Path, "-f", "null", "-")
	p.ffmpegCmd = cmd

	// Run ffmpeg
	err := cmd.Run()
	if err != nil {
		log.Printf("Error playing track: %v", err)
	}

	// After track finishes, move to next track
	p.mu.Lock()
	if p.isPlaying {
		p.currentIndex = (p.currentIndex + 1) % len(p.playlist)
		p.mu.Unlock()
		go p.startPlayback() // Start next track
	} else {
		p.mu.Unlock()
	}
}

func (p *Player) Start() {
	p.mu.Lock()
	if !p.isPlaying && len(p.playlist) > 0 {
		p.isPlaying = true
		p.mu.Unlock()
		go p.startPlayback()
	} else {
		p.mu.Unlock()
	}
}

func (p *Player) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.isPlaying = false
	if p.ffmpegCmd != nil && p.ffmpegCmd.Process != nil {
		p.ffmpegCmd.Process.Kill()
		p.ffmpegCmd = nil
	}
}

const indexHTML = `
<!DOCTYPE html>
<html>
<head>
    <title>MP3 Player</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
        }
        h1 {
            color: #333;
        }
        .controls {
            margin: 20px 0;
            padding: 10px;
            background-color: #f5f5f5;
            border-radius: 5px;
        }
        button {
            padding: 8px 16px;
            margin-right: 10px;
            background-color: #4CAF50;
            color: white;
            border: none;
            border-radius: 4px;
            cursor: pointer;
        }
        button:hover {
            background-color: #45a049;
        }
        .playlist {
            list-style-type: none;
            padding: 0;
        }
        .playlist li {
            padding: 8px;
            border-bottom: 1px solid #ddd;
        }
        .current {
            background-color: #e7f3fe;
            font-weight: bold;
        }
        .player-container {
            margin: 20px 0;
        }
        audio {
            width: 100%;
        }
    </style>
    <script>
        // Auto refresh current track info every 5 seconds
        setInterval(function() {
            fetch('/api/current')
                .then(response => response.json())
                .then(data => {
                    if (data) {
                        document.getElementById('currentTrack').innerText = data.filename;
                    } else {
                        document.getElementById('currentTrack').innerText = "No track playing";
                    }
                });
        }, 5000);
    </script>
</head>
<body>
    <h1>MP3 Player</h1>
    
    <div class="player-container">
        <h2>Web Player</h2>
        <audio id="audioPlayer" controls autoplay>
            <source src="/stream" type="audio/mp3">
            Your browser does not support the audio element.
        </audio>
    </div>
    
    <div class="controls">
        <h2>Controls</h2>
        <button onclick="location.href='/api/skip'">Skip to Next Track</button>
        <button onclick="location.reload()">Refresh Page</button>
    </div>
    
    <div class="now-playing">
        <h2>Now Playing</h2>
        <p id="currentTrack">{{if .Current}}{{.Current.Filename}}{{else}}No track playing{{end}}</p>
    </div>
    
    <div class="playlist-container">
        <h2>Playlist</h2>
        <ul class="playlist">
            {{range .Playlist}}
            <li class="{{if .Current}}current{{end}}">
                {{.ID}}. {{.Filename}}
            </li>
            {{end}}
        </ul>
    </div>
</body>
</html>
`

func main() {
	// Define default music directory
	musicDir := "./music"

	// Allow overriding via environment variable
	if envDir := os.Getenv("MUSIC_DIR"); envDir != "" {
		musicDir = envDir
	}

	// Allow optional command line argument to override the directory
	if len(os.Args) > 1 {
		musicDir = os.Args[1]
	}

	// Get the port from environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Create output directory if it doesn't exist
	outputDir := "/data"
	if envOutputDir := os.Getenv("OUTPUT_DIR"); envOutputDir != "" {
		outputDir = envOutputDir
	}
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		os.MkdirAll(outputDir, 0755)
	}

	// Check if ffmpeg is installed
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		log.Fatalf("ffmpeg not found: %v. Please install ffmpeg to use this player.", err)
	}

	player := NewPlayer()

	// Scan directory for audio files
	log.Printf("Scanning directory: %s\n", musicDir)
	err = player.ScanDirectory(musicDir)
	if err != nil {
		log.Fatalf("Error scanning directory: %v", err)
	}

	// Generate M3U playlist
	m3uPath := filepath.Join(outputDir, "playlist.m3u")
	err = player.GenerateM3U(m3uPath)
	if err != nil {
		log.Printf("Error generating M3U: %v", err)
	} else {
		log.Printf("Generated playlist at %s\n", m3uPath)
	}

	log.Printf("Found %d audio files\n", len(player.GetPlaylist()))

	// Create templates
	indexTemplate := template.Must(template.New("index").Parse(indexHTML))

	// Setup HTTP server
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		data := struct {
			Current  *Track
			Playlist []Track
		}{
			Current:  player.GetCurrentTrack(),
			Playlist: player.GetPlaylist(),
		}

		indexTemplate.Execute(w, data)
	})

	// API endpoints
	http.HandleFunc("/api/skip", func(w http.ResponseWriter, r *http.Request) {
		player.Skip()
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	http.HandleFunc("/api/current", func(w http.ResponseWriter, r *http.Request) {
		track := player.GetCurrentTrack()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(track)
	})

	http.HandleFunc("/api/playlist", func(w http.ResponseWriter, r *http.Request) {
		playlist := player.GetPlaylist()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(playlist)
	})

	// Stream endpoint - uses ffmpeg to transcode on the fly
	http.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		track := player.GetCurrentTrack()
		if track == nil {
			http.Error(w, "No track available", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "audio/mp3")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Transfer-Encoding", "chunked")

		// Use ffmpeg to transcode the file to MP3 stream
		cmd := exec.Command("ffmpeg", "-nostdin", "-i", track.Path, "-f", "mp3", "-acodec", "libmp3lame", "-ac", "2", "-ab", "128k", "-")
		cmd.Stdout = w
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			log.Printf("Error streaming track: %v", err)
		}
	})

	// M3U playlist download
	http.HandleFunc("/playlist.m3u", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, m3uPath)
	})

	// Start the player (this starts playing the first track)
	if len(player.GetPlaylist()) > 0 {
		log.Println("Starting playback...")
		player.Start()
	} else {
		log.Println("No audio files found. Web server is running but nothing will play.")
	}

	// Start HTTP server
	log.Printf("HTTP server started on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
