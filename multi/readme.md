

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

