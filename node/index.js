const express = require('express');
const fs = require('fs');
const path = require('path');
const { spawn } = require('child_process');

const app = express();

const videoFolder = './video';
const resolution = '720:480';
const tvName = "usual tv";
let videoQueue = [];
let currentIndex = 0;
let currentProcess = null;


function refillQueue() {
  console.log("refill queue");
  videoQueue = fs.readdirSync(videoFolder)
    .filter(file => file.endsWith('.mp4'))
    .sort(() => Math.random() > 0.5 ? 1 : -1);
}

function startNextVideo() {
  if (videoQueue.length === 0 || currentIndex == 0) {
    refillQueue();
  }
  if (videoQueue.length > 0) {
    const formattedDate = new Date().toLocaleDateString('en-US', { month: '2-digit', day: '2-digit' });

    const videoFile = videoQueue[currentIndex];
    const videoName = path.parse(videoFile).name;
    const nextVideo = path.parse(videoQueue[(currentIndex +1)%videoQueue.length]).name + " >>";
    const command = [
      '-nostdin',
      '-re',
      '-i',
      path.join(videoFolder, videoFile),
      '-i',
      'overlay.png',
      '-c:v',
      'libx264',
      // "-preset",
      // "ultrafast",
      "-loglevel",
      "error",
      '-filter_complex',
      `scale=${resolution}:force_original_aspect_ratio=decrease,pad=${resolution}:(ow-iw)/2:(oh-ih)/2,` +
      'overlay=0:0,' +
      `drawtext=fontsize=25:fontcolor=white:text='${tvName}':x=25:y=25,` +
      `drawtext=fontsize=18:fontcolor=white:text='%{pts\\:hms}':x=(w-tw-10):y=25,` +
      `drawtext=fontsize=16:fontcolor=white:text='${videoName}':x=(w-tw-25):y=h-th-35,` +
      `drawtext=fontsize=13:fontcolor=white:text='${nextVideo}':x=(w-tw-25):y=h-th-19,` +
      `drawtext=fontsize=18:fontcolor=white:text='%{localtime\\:%T}':x=35:y=83,` +
      `drawtext=fontsize=18:fontcolor=white:text='${formattedDate + ""}':x=15:y=55[v]`,
      '-map',
      '[v]',
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

    currentProcess.stderr.on('data', (data) => {
      console.error(`FFmpeg error: ${data}`);
    });

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



// app.get('/', (req, res) => {
//   res.sendFile(__dirname + '/index.html');
// })

app.set('view engine', 'ejs');
app.set('views', __dirname + '/views');

app.get('/', (req, res) => {
  const myVariable = 'Hello from Node.js!';
  const items = videoQueue
    .map((x, idx) => currentIndex === idx ? `> ${idx}: ${x}` : `   ${idx} ${x}`);
  res.render('index', { items, myVariable });
});

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

app.get('/refill', (req, res) => {
  refillQueue();
  res.send('refill the queue');
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
