const express = require('express');
const fs = require('fs');
const path = require('path');
const { spawn, exec } = require('child_process');

const app = express();

const videoFolder = './video';
const adsVideoFolder = './video/ads';
const series = './video/series';
const adsFreq = 2; // how often to show ads 
const resolution = '720:480';
const tvName = "usual tv";
let videoQueue = [];
let adsQueue = [];
let currentIndex = 0;
let currentProcess = null;

let queue = [];
let currentVideoCount = 0;

async function getVideosFromFolder(folder) {
  const fileExtensionRegex = /\.(mp4|webm)$/i;

  const newVideoQueue = getFilesRecursively(folder)
    .filter(file => fileExtensionRegex.test(file))
    .map(async (video) => {
      const duration = (await getVideoDuration(video))
      let videoName = path.parse(video).name.replace(/[^ a-zA-Z0-9-\u0400-\u04FF]/g, '');
      const folderName = path.basename(path.dirname(video));
      return {
        path: video,
        name: videoName,
        duration: duration,
        folderName: folderName,
      }
    });


  return await Promise.all(newVideoQueue);
}

async function refillVideosFromFolder(folder = videoFolder) {
  console.log("start refill folder: ", folder);
  currentVideoCount = 0;
  queue = await getVideosFromFolder(folder);
  console.log("hehehehehehehe", queue);
}

app.listen(3001, async () => {
  await refillVideosFromFolder();
  console.log('Server running on port 3001');
  startStream();
});

function startStream() {
  const staticFolder = 'static';
  if (fs.existsSync(staticFolder)) {
    fs.rmSync(staticFolder, { recursive: true });
  }
  fs.mkdirSync(staticFolder);
  playNextVideo();
}

function playNextVideo() {
  if (queue.length > 0) {

    const indx = (currentVideoCount + 1) % queue.length
    const currentVideo = queue[indx];

    const command = getStreamCommand(currentVideo);
    currentProcess = spawn('ffmpeg', command);

    currentProcess.stderr.on('data', (data) => {
      console.error(`FFmpeg error: ${data}`);
    });

    // console.log(currentIndex, videoFile);
    currentProcess.on('close', (code) => {
      if (code !== 0) {
        console.log(">> [ SKIP ] >>", currentVideo.name, code, currentVideo.duration)
      }

      removeOldTSFiles("./static/");
      currentVideoCount++;
      playNextVideo();
    });
  }
}

function getStreamCommand(video) {
  const nextVideo = "next";
  return [
    '-nostdin',
    '-re',
    '-i',
    video.path,
    '-i',
    'overlay.png',
    '-c:v',
    'libx264',
    '-c:a',
    'copy', // Add this line to copy the audio codec
    "-loglevel",
    "error",
    '-filter_complex',
    `scale=${resolution}:force_original_aspect_ratio=decrease,pad=${resolution}:(ow-iw)/2:(oh-ih)/2,` +
    'overlay=0:0,' +
    `drawtext=fontsize=18:fontfile=font.ttf:fontcolor=white:textfile=weather.txt:x=w-tw+20:y=(-35),` +
    `drawtext=fontsize=25:fontcolor=white:text='${tvName}':x=25:y=25,` +
    `drawtext=fontsize=11:fontcolor=white:text='%{pts\\:hms}':x=(6):y=h-th-13,` +
    `drawtext=fontsize=11:fontcolor=white:text='${"  0"+ video.duration.replace(/:/g, '\\:')}':x=(10):y=h-th-2,` +
    `drawtext=fontsize=16:fontcolor=white:text='${video.name}':x=(w-tw-25):y=h-th-35,` +
    `drawtext=fontsize=13:fontcolor=white:text='${nextVideo}':x=(w-tw-25):y=h-th-19,` +
    `drawtext=fontsize=18:fontcolor=white:text='%{localtime\\:%T}':x=35:y=83[v]`,
    '-map',
    '[v]',
    '-map',
    '0:a',
    '-hls_time',
    '0.25',
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
}


//-------------------

function refillAds() {
  console.log("refill ads");
  adsQueue = fs.readdirSync(adsVideoFolder)
    .filter(file => file.endsWith('.mp4'));
}

function refillQueue(folder = videoFolder) {
  console.log("refill queue: ", folder);
  getWeatherString();
  currentIndex = 0;
  videoQueue = getFilesRecursively(folder)
    .filter(file => file.endsWith('.mp4'))
    .sort(() => Math.random() > 0.5 ? 1 : -1);
}



function startNextVideo(showAd = false) {
  if (videoQueue.length === 0) {
    refillQueue();
  }
  if (videoQueue.length > 0) {

    let videoFile = videoQueue[currentIndex];
    let videoPath = path.join(videoFile);

    let videoName = path.parse(videoFile).name.replace(/[^ a-zA-Z0-9-\u0400-\u04FF]/g, '');
    if (showAd) {
      const randomIndex = Math.floor(Math.random() * adsQueue.length);
      videoFile = adsQueue[randomIndex];
      console.log("show add: ", videoFile);
      videoPath = path.join(adsVideoFolder, videoFile);
      videoName = "[AD] " + path.parse(videoFile).name.replace(/[^ a-zA-Z0-9-\u0400-\u04FF]/g, '') + " [AD]";
    }

    const nextVideo = path.parse(videoQueue[(currentIndex + 1) % videoQueue.length]).name.replace(/[^ a-zA-Z0-9-\u0400-\u04FF]/g, '') + " >>";

    const command = getFfmpegCommand(videoPath, videoName, nextVideo);
    currentProcess = spawn('ffmpeg', command);

    currentProcess.stderr.on('data', (data) => {
      console.error(`FFmpeg error: ${data}`);
    });

    console.log(currentIndex, videoFile);
    currentProcess.on('close', (code) => {
      if (code !== 0) {
        console.log(">> SKIP >>", videoFile, code)
      }
      removeOldTSFiles("./static/");

      if (showAd !== true && currentIndex % adsFreq == 0) {
        startNextVideo(true);
      } else {
        currentIndex = (currentIndex + 1) % videoQueue.length;
        startNextVideo();
      }
    });
  }
}

function start() {
  const staticFolder = 'static';
  if (fs.existsSync(staticFolder)) {
    fs.rmSync(staticFolder, { recursive: true });
  }
  fs.mkdirSync(staticFolder);
  startNextVideo();
}


app.set('view engine', 'ejs');
app.set('views', __dirname + '/views');

app.get("/stick", (req, res) => {
  const getName = (file) => path.parse(file).name.replace(/[^ a-zA-Z0-9-\u0400-\u04FF]/g, '');
  const status = {
    current: getName(videoQueue[currentIndex]),
    next: getName(videoQueue[currentIndex + 1] ?? " ")
  }
  res.send(status);
})

app.get('/', (req, res) => {
  const myVariable = 'idx';
  const items = videoQueue
    .map((x, idx) => currentIndex === idx ? `> ${idx}: ${x}` : `   ${idx} ${x}`);
  res.render('index', { items, myVariable });
});

app.get('/c', (req, res) => {
  const items = videoQueue
    .map((x, idx) => currentIndex === idx ? `> ${idx}: ${x}` : `   ${idx} ${x}`);
  res.render('control', { items });
});

// serve stream
app.use('/static', express.static('static'));

app.get('/next/:number', (req, res) => {
  serverVariable = parseInt(req.params.number);
  currentIndex = serverVariable - 1;
  console.log("next track is", serverVariable, videoQueue[serverVariable]);
  if (currentProcess) {
    currentProcess.kill();
  }
})

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

// app.listen(3000, () => {
//   refillAds();
//   refillQueue();
//   console.log('Server running on port 3000');
//   start();
// });

// ----------- garbage ⬇️


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

function getWeatherString() {
  const curlCommand = 'curl -s https://wttr.in/Amsterdam?T0';
  exec(curlCommand, (error, stdout, stderr) => {
    if (error) {
      console.error(`exec error: ${error}`);
      return;
    }
    console.log(`stdout: ${stdout}`);
    fs.writeFile('weather.txt', stdout.replace(/\\/g, "\\\\"), (err) => {
      if (err) throw err;
      console.log('Weather data saved to weather.txt');
    });
  });
}


function getFfmpegCommand(videoPath, videoName, nextVideo) {
  const formattedDate = new Date().toLocaleDateString('en-US', { month: '2-digit', day: '2-digit' });

  return [
    '-nostdin',
    '-re',
    '-i',
    videoPath,
    '-i',
    'overlay.png',
    // '-i',
    // 'weather.png',
    '-c:v',
    'libx264',
    '-c:a',
    'copy', // Add this line to copy the audio codec
    "-loglevel",
    "error",
    '-filter_complex',
    `scale=${resolution}:force_original_aspect_ratio=decrease,pad=${resolution}:(ow-iw)/2:(oh-ih)/2,` +
    'overlay=0:0,' +
    // 'overlay=(w+90):(-30),' +
    `drawtext=fontsize=25:fontcolor=white:text='${tvName}':x=25:y=25,` +
    `drawtext=fontsize=18:fontfile=font.ttf:fontcolor=white:textfile=weather.txt:x=w-tw+20:y=(-35),` +
    `drawtext=fontsize=11:fontcolor=white:text='%{pts\\:hms}':x=(10):y=h-th-2,` +
    `drawtext=fontsize=16:fontcolor=white:text='${videoName}':x=(w-tw-25):y=h-th-35,` +
    `drawtext=fontsize=13:fontcolor=white:text='${nextVideo}':x=(w-tw-25):y=h-th-19,` +
    `drawtext=fontsize=18:fontcolor=white:text='%{localtime\\:%T}':x=35:y=83,` +
    `drawtext=fontsize=18:fontcolor=white:text='${formattedDate + ""}':x=15:y=55[v]`,
    '-map',
    '[v]',
    '-map',
    '0:a',
    '-hls_time',
    '0.25',
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
}


function getVideoDuration(videoPath) {
  return new Promise((resolve, reject) => {
    const ffprobe = spawn('ffprobe', [
      '-v',
      'error',
      '-show_entries',
      'format=duration',
      '-of',
      'default=noprint_wrappers=1:nokey=1',
      "-sexagesimal",
      videoPath,
    ]);

    let duration = '';

    ffprobe.stdout.on('data', (data) => {
      duration += data.toString();
    });

    ffprobe.on('error', (err) => {
      reject(err);
    });

    ffprobe.on('close', (code) => {
      if (code === 0) {

        const formattedDuration = duration.toString().slice(0, 11)
        resolve(formattedDuration);
      } else {
        reject(new Error(`ffprobe exited with code ${code}`));
      }
    });
  });
}

function getFilesRecursively(dir, fileList = []) {
  const files = fs.readdirSync(dir);

  files.forEach(file => {
    const filePath = path.join(dir, file);
    const stat = fs.statSync(filePath);

    if (stat.isDirectory()) {
      getFilesRecursively(filePath, fileList);
    } else {
      fileList.push(filePath);
    }
  });

  return fileList;
}