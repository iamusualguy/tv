const express = require('express');
const fs = require('fs');
const path = require('path');
const { spawn } = require('child_process');

const app = express();

// app.register('.html', require('jade'))

const videoFolder = '../video';
let videoQueue = [];
let currentIndex = 0;
let currentProcess = null;

function startNextVideo() {
  if (videoQueue.length === 0) {
    refillQueue();
  }
  if (videoQueue.length > 0) {
    const videoFile = videoQueue[currentIndex];
    console.log(videoFile);
    const command = [
      // 'ffmpeg',
      '-re',
      '-i',
      path.join(videoFolder, videoFile),
      '-c:v',
      'libx264',
      '-vf',
      "[in]scale=320:240:force_original_aspect_ratio=decrease,pad=320:240:(ow-iw)/2:(oh-ih)/2,drawtext=fontsize=25:fontcolor=white:text='пися палыч тв':x=25:y=25,drawtext=fontsize=18:fontcolor=white:text='%{localtime\\:%T}':x=25:y=55[out]",
      '-hls_time',
      '1',
      '-hls_list_size',
      '5',
      '-f',
      'hls',
      '-segment_wrap',
      '6',
      // '-hls_flags',
      // 'append_list+omit_endlist',
      '-hls_flags',
      'delete_segments+append_list+omit_endlist',
      // '-hls_delete_threshold',
      // '5',
      'static/stream.m3u8',
    ];
    currentProcess = spawn('ffmpeg', command);
    currentProcess.on('close', (code) => {

    currentIndex = (currentIndex + 1) % videoQueue.length;
      startNextVideo();
    });
    // currentProcess.on('exit', startNextVideo);
  }
}

function refillQueue() {
  videoQueue = fs.readdirSync(videoFolder).filter(file => file.endsWith('.mp4'));
}

// app.get('/', (req, res) => {
//   const data = {
//     videoQueue: videoQueue,
//     current_index: currentIndex,
//   };
//   res.render('index', data);
// });

app.get('/', (req, res) => {
  // res.writeHead(200, {'Content-Type': 'text/html'})
  res.sendFile(__dirname + '/index.html');
  // res.end()
})

function start() {
  const staticFolder = 'static';
  if (fs.existsSync(staticFolder)) {
    fs.rmSync(staticFolder, { recursive: true });
  }
  fs.mkdirSync(staticFolder);
  startNextVideo();
  
}

app.get('/start', (req, res) => {
  start();
  res.send('Streaming started');
});

app.use('/static', express.static('static'));

app.get('/skip', (req, res) => {
  if (currentProcess) {
    currentProcess.kill();
  }
  res.send('Skipped to the next video');
});

app.listen(3000, () => {
  refillQueue();
  console.log('Server running on port 3000');
  start();
});
