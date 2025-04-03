package main

import (
    "log"
    "os/exec"
)

func main() {
    // Generate playlist.txt
    cmd := exec.Command("bash", "-c", `ls *.mp3 | sed "s/^/file '/; s/$/'/" > playlist.txt`)
    if err := cmd.Run(); err != nil {
        log.Fatalf("Error generating playlist: %v", err)
    }

    // FFmpeg command
    ffmpegCmd := exec.Command("ffmpeg", "-re", "-stream_loop", "-1", 
        "-f", "concat", "-safe", "0", "-i", "playlist.txt",
        "-acodec", "libmp3lame", "-b:a", "128k", "-bufsize", "32k",
        "-fflags", "nobuffer", "-flags", "low_delay", "-flush_packets", "1", 
        "-max_delay", "0", "-f", "mp3", "udp://127.0.0.1:1234")

    // Redirect output for debugging
    ffmpegCmd.Stderr = log.Writer()

    log.Println("Starting the audio stream...")
    if err := ffmpegCmd.Run(); err != nil {
        log.Fatalf("FFmpeg error: %v", err)
    }
}
