
sooooo how to start it

you need opentts in docker: docker run -it -p 5500:5500 synesthesiam/opentts:ru  

new setup uses https://github.com/twirapp/silero-tts-api-server 
docker run --rm -p 8000:8000 twirapp/silero-tts-api-server 

ollama  OLLAMA_HOST=0.0.0.0 ollama serve; 
ffmpeg
golang

run ollama with model of your choice
run opentts with language of your choice
use correct adress and names in host.go
run go programm (or build and run)

# how to run

go run . ./path/to/music

# how to build
go build -o radioHost

# todo
run with -i as path to music
add config to set the adresses for opentts and ollama and model names




 ffmpeg -re -i "Аватарка.mp3" \
 -vn -c:a aac \
  -b:a 128k \
  -f hls \
  -hls_time 2 \
  -hls_list_size 5 -hls_flags delete_segments+append_list+omit_endlist \
  -hls_segment_filename "static/segment_%01d.ts" \
  static/stream.m3u8

         | 
try this v

ffmpeg -re -i ".mp4" \
  # Video + Audio Stream
  -map 0:v:0 -map 0:a:0 -c:v h264 -b:v 1500k -c:a aac -b:a 128k -f hls -hls_time 2 -hls_list_size 5 -hls_flags delete_segments+append_list+omit_endlist \
  -hls_segment_filename "static/video_audio_segment_%01d.ts" \
  static/video_audio_stream.m3u8 \
  
  # Audio Only Stream
  -map 0:a:0 -c:a aac -b:a 128k -f hls -hls_time 2 -hls_list_size 5 -hls_flags delete_segments+append_list+omit_endlist \
  -hls_segment_filename "static/audio_segment_%01d.ts" \
  static/audio_stream.m3u8

