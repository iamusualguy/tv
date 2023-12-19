const express = require('express');
const fs = require('fs');
const path = require('path');
const { spawn } = require('child_process');

const app = express();

const videoFolder = './video';
const resolution = '320:240';
const tvName = "usual tv";
let videoQueue = [];
let currentIndex = 0;
let currentProcess = null;


function refillQueue() {
  console.log("refill queue");
  videoQueue = fs.readdirSync(videoFolder).filter(file => file.endsWith('.mp4'));
}

function startNextVideo() {
  if (videoQueue.length === 0 || currentIndex == 0) {
    refillQueue();
  }
  if (videoQueue.length > 0) {
    const videoFile = videoQueue[currentIndex];
    const command = [
      '-nostdin',
      //'-re',
      '-i',
      path.join(videoFolder, videoFile),
      '-c:v',
      'libx264',
      // "-preset",
      // "ultrafast",
      "-loglevel",
      "error", 
      '-vf',
      `[in]scale=${resolution}:force_original_aspect_ratio=decrease,pad=${resolution}:(ow-iw)/2:(oh-ih)/2,drawtext=fontsize=25:fontcolor=white:text='${tvName}':x=25:y=25,drawtext=fontsize=18:fontcolor=white:text='%{localtime\\:%T}':x=25:y=55[out]`,
      '-hls_time',
      '1',
      '-hls_list_size',
      '5',
      '-f',
      'hls',
      '-segment_wrap',
      '6',
      '-hls_flags',
      'delete_segments+append_list+omit_endlist',
      'static/stream.m3u8',
    ];
    currentProcess = spawn('ffmpeg', command);
    console.log(currentIndex, videoFile);
    currentProcess.on('close', (code) => {
      if (code !== 0) {
        console.log(">>>>", videoFile, code)
      }
      // if (code === 0 || code === 255) {
        currentIndex = (currentIndex + 1) % videoQueue.length;
        removeOldTSFiles("./static/");
        startNextVideo();
      // }
    });
  }
}



app.get('/', (req, res) => {
  res.sendFile(__dirname + '/index.html');
})

function start() {
  const staticFolder = 'static';
  if (fs.existsSync(staticFolder)) {
    fs.rmSync(staticFolder, { recursive: true });
  }
  fs.mkdirSync(staticFolder);
  startNextVideo();

}

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


function removeOldTSFiles(directoryPath) {
  const playlistFile = path.join(directoryPath, 'stream.m3u8');

  // Read the content of the playlist file
  fs.readFile(playlistFile, 'utf8', (err, data) => {
    if (err) {
      console.error('Error reading playlist file:', err);
      return;
    }

    // Extract TS file names from the playlist
    const tsFiles = data.match(/stream\d+\.ts/g);

    if (!tsFiles || tsFiles.length === 0) {
      console.log('No TS files found in the playlist.');
      return;
    }

    // Get all TS files in the directory
    fs.readdir(directoryPath, (err, files) => {
      if (err) {
        console.error('Error reading directory:', err);
        return;
      }

      // Filter out TS files that are not listed in the playlist
      const obsoleteTSFiles = files.filter(file => file.startsWith('stream') && file.endsWith('.ts') && !tsFiles.includes(file));

      // Remove obsolete TS files
      obsoleteTSFiles.forEach(file => {
        fs.unlink(path.join(directoryPath, file), err => {
          if (err) {
            console.error(`Error removing ${file}:`, err);
          } else {
            // console.log(`${file} has been removed.`);
          }
        });
      });
    });
  });
}