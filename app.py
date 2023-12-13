import os
import shutil
import subprocess
from flask import Flask, render_template, request
from pytube import YouTube

app = Flask(__name__)

video_folder = "video"
video_queue = []
current_index = 0
current_process = None

def start_next_video():
    global current_process, current_index
    if not video_queue:
        refill_queue()
    if video_queue:
        video_file = video_queue[current_index]
        command = [
            "ffmpeg",
            "-re",
            "-i",
            os.path.join(video_folder, video_file),
            "-c:v",
            "libx264",
            "-preset",
            "ultrafast",  # Use ultrafast preset for faster encoding
            "-vf",
            "[in]scale=320:240:force_original_aspect_ratio=decrease,pad=320:240:(ow-iw)/2:(oh-ih)/2,drawtext=fontsize=25:fontcolor=white:text='пися палыч тв':x=25:y=25,drawtext=fontsize=18:fontcolor=white:text='%{localtime\\:%T}':x=25:y=55[out]",
            "-hls_time",
            "3",
            "-hls_list_size",
            "10",
            "-f",
            "hls",
            "-hls_flags",
            "delete_segments+append_list+omit_endlist",
            "-hls_delete_threshold",
            "10",
            "static/stream.m3u8",
        ]
        current_process = subprocess.Popen(command, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        current_index = (current_index + 1) % len(video_queue)
        current_process.wait()
        start_next_video()

def refill_queue():
    global video_queue
    video_queue = [file for file in os.listdir(video_folder) if file.endswith(".mp4")]

@app.route("/")
def index():
    return render_template("index.html", video_queue=video_queue, current_index=current_index)

@app.route("/start")
def start_stream():
    # Remove existing "static" folder and its contents if it exists
    static_folder = "static"
    if os.path.exists(static_folder):
        shutil.rmtree(static_folder)

    # Create a new "static" folder
    os.makedirs(static_folder)
    start_next_video()
    return "Streaming started"

@app.route("/skip")
def skip_video():
    global current_process
    if current_process:
        current_process.kill()
    # start_next_video()
    return "Skipped to the next video"

def download(link):
    yt = YouTube(link)
    video = yt.streams.filter(file_extension="mp4").first()
    video.download(video_folder)

@app.route("/download", methods=["POST"])
def download_video():
    video_url = request.form.get("video_url")
    if video_url:
        try:
            download(video_url)
            refill_queue()
            return "Video downloaded successfully!"
        except Exception as e:
            return f"Failed to download video: {str(e)}"
    return "No video URL provided."

if __name__ == "__main__":
    refill_queue()
    # start_stream()
    app.run(debug=False, host="0.0.0.0")
    start_stream()
