const express = require("express");
const fs = require("fs");
const path = require("path");
const { spawn, exec } = require("child_process");
const ICAL = require("node-ical");
const {
	isWithinInterval,
	addDays,
	parse,
	getTime,
	addMilliseconds,
} = require("date-fns");

const {
	getCurrentEventForDate,
	getStreamCommand,
} = require("./utils.js")

const app = express();

const videoFolder = "./video/";
const adsFreq = 4; // how often to show ads
const resolution = "720:480";
const tvName = "usual tv";
let videoQueue = [];
let adsQueue = [];
let currentIndex = 0;
let currentProcess = null;

let schedule = [];
let currentVideoCount = 0;
async function getVideos() {
	const libraryObject = {}
	const contentTypes = ["music", "series", "ads"];
	for (let cont of contentTypes) {
		libraryObject[cont] = {
			playedCount: 0,
			videos: await getVideosFromFolder(videoFolder, cont),
		};
	}
	return libraryObject;
}

async function refillSchedule() {
	currentVideoCount = 0;
	const icalFilePath = "tv-cal.ics";
	schedule = [];
	getWeatherString();
	const libraryObject = await getVideos();


	let kindaCurrentDate = Date.now(); // probably better to add some seconds to adjust with schedule  generation time
	console.log(libraryObject);
	kindaCurrentDate = addMilliseconds(
		kindaCurrentDate,
		0,
	);
	for (let i = 0; i < 15; i++) {
		const currentEvent = getCurrentEventForDate(icalFilePath, kindaCurrentDate);
		if (currentEvent) {
			const bibrary = (i % adsFreq == 0) ? // we might show ads 
				libraryObject["ads"] :
				libraryObject[currentEvent.summary];

			if (bibrary) {
				const anotherIndex = (bibrary.playedCount++) % bibrary.videos.length;
				const newVideo =
					bibrary.videos[anotherIndex];
				if (newVideo) {
				
					schedule.push({ ...newVideo, start: kindaCurrentDate });
					kindaCurrentDate = addMilliseconds(
						kindaCurrentDate,
						hmmssSSSToMS(newVideo.duration),
					);
					// increse the date by video duration
					

					console.log("::>>", anotherIndex, newVideo.folderName, kindaCurrentDate, newVideo.name, newVideo.type);
				}
			}
		}
	}
	console.log(libraryObject);
}

async function getVideosFromFolder(videoFolder, cont) {
	const folder = videoFolder + cont;
	const fileExtensionRegex = /\.(mp4|avi|mkv|webm)$/i;

	const newVideoQueue = getFilesRecursively(folder)
		.filter((file) => fileExtensionRegex.test(file))
		.map(async (video) => {
			const duration = await getVideoFormattedDuration(video);
			let videoName = path
				.parse(video)
				.name.replace(/[^ a-zA-Z0-9-\u0400-\u04FF]/g, "");
			const folderName = path.basename(path.dirname(video));
			return {
				type: cont,
				path: video,
				name: videoName,
				duration: duration,
				folderName: folderName,
			};
		}).sort(() => Math.random() - 0.5);

	return await Promise.all(newVideoQueue);
}

app.listen(3001, async () => {
	await refillSchedule();
	// await refillVideosFromFolder();
	console.log('Server running on port 3001');
	startStream();
});

function startStream() {
	// clean the previous stream files
	const staticFolder = "static";
	if (fs.existsSync(staticFolder)) {
		fs.rmSync(staticFolder, { recursive: true });
	}
	fs.mkdirSync(staticFolder);

	playNextVideo();
}

function playNextVideo() {
	if (schedule.length > 0) {
		if (currentVideoCount - 2 > schedule.length) {
			refillSchedule();
		}
		const indx = (currentVideoCount) % schedule.length;
		const currentVideo = schedule[indx];
		console.log(currentVideoCount, "::", currentVideo.name, currentVideo.type)
		const command = getStreamCommand(currentVideo);
		currentProcess = spawn("ffmpeg", command);

		currentProcess.stderr.on("data", (data) => {
			console.error(`FFmpeg error: ${data}`);
		});

		currentProcess.on("close", (code) => {
			if (code !== 0) {
				console.log(
					">> [ SKIP ] >>",
					currentVideo.name,
					code,
					currentVideo.duration,
				);
			}

			removeOldTSFiles("./static/");
			currentVideoCount++;
			playNextVideo();
		});
	}
}

//-------------------

app.set("view engine", "ejs");
app.set("views", __dirname + "/views");

// app.get("/stick", (req, res) => {
// 	const getName = (file) =>
// 		path.parse(file).name.replace(/[^ a-zA-Z0-9-\u0400-\u04FF]/g, "");
// 	const status = {
// 		current: getName(videoQueue[currentIndex]),
// 		next: getName(videoQueue[currentIndex + 1] ?? " "),
// 	};
// 	res.send(status);
// });

app.get("/", (req, res) => {
	const myVariable = "idx";
	const items = schedule.map((x, idx) =>
	currentVideoCount === idx ? `> ${idx}: ${x.name} ${x.start}` : `   ${idx} ${x.name}  ${x.start}`,
	);
	res.render("index", { items, myVariable });
});

// app.get("/c", (req, res) => {
// 	const items = videoQueue.map((x, idx) =>
// 		currentIndex === idx ? `> ${idx}: ${x}` : `   ${idx} ${x}`,
// 	);
// 	res.render("control", { items });
// });

// serve stream
app.use("/static", express.static("static"));

app.get("/next/:number", (req, res) => {
	serverVariable = parseInt(req.params.number);
	currentVideoCount = serverVariable - 1;
	console.log("next track is", serverVariable, schedule[serverVariable]);
	if (currentProcess) {
		currentProcess.kill();
	}
});

app.get("/skip", (req, res) => {
	if (currentProcess) {
		currentProcess.kill();
	}
	res.send("Skipped to the next video");
});

app.get("/refill", (req, res) => {
	refillSchedule();
	res.send("refill the queue");
});


// ----------- garbage ⬇️

function removeOldTSFiles(directoryPath) {
	const playlistFile = path.join(directoryPath, "stream.m3u8");

	// Read the content of the playlist file
	fs.readFile(playlistFile, "utf8", (err, data) => {
		if (err) {
			console.error("Error reading playlist file:", err);
			return;
		}

		// Extract TS file names from the playlist
		const tsFiles = data.match(/stream\d+\.ts/g);

		if (!tsFiles || tsFiles.length === 0) {
			console.log("No TS files found in the playlist.");
			return;
		}

		// Get all TS files in the directory
		fs.readdir(directoryPath, (err, files) => {
			if (err) {
				console.error("Error reading directory:", err);
				return;
			}

			// Filter out TS files that are not listed in the playlist
			const obsoleteTSFiles = files.filter(
				(file) =>
					file.startsWith("stream") &&
					file.endsWith(".ts") &&
					!tsFiles.includes(file),
			);

			// Remove obsolete TS files
			obsoleteTSFiles.forEach((file) => {
				fs.unlink(path.join(directoryPath, file), (err) => {
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
	const curlCommand = "curl -s https://wttr.in/Amsterdam?T0";
	exec(curlCommand, (error, stdout, stderr) => {
		if (error) {
			console.error(`exec error: ${error}`);
			return;
		}
		console.log(`stdout: ${stdout}`);
		fs.writeFile("weather.txt", stdout.replace(/\\/g, "\\\\"), (err) => {
			if (err) throw err;
			console.log("Weather data saved to weather.txt");
		});
	});
}


function getVideoFormattedDuration(videoPath) {
	return new Promise((resolve, reject) => {
		const ffprobe = spawn("ffprobe", [
			"-v",
			"error",
			"-show_entries",
			"format=duration",
			"-of",
			"default=noprint_wrappers=1:nokey=1",
			"-sexagesimal",
			videoPath,
		]);

		let duration = "";

		ffprobe.stdout.on("data", (data) => {
			duration += data.toString();
		});

		ffprobe.on("error", (err) => {
			reject(err);
		});

		ffprobe.on("close", (code) => {
			if (code === 0) {
				const formattedDuration = duration.toString().slice(0, 11);
				resolve(formattedDuration);
			} else {
				reject(new Error(`ffprobe exited with code ${code}`));
			}
		});
	});
}

function getFilesRecursively(dir, fileList = []) {
	const files = fs.readdirSync(dir);

	files.forEach((file) => {
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

function hmmssSSSToMS(timeString) {
	const [hours, minutes, seconds] = timeString.split(":");
	const milliseconds = seconds.includes(".") ? seconds.split(".")[1] : "0";
	const secondsWithoutMilliseconds = seconds.includes(".")
		? seconds.split(".")[0]
		: seconds;

	// Step 2: Convert to milliseconds
	const hoursInMilliseconds = parseInt(hours) * 60 * 60 * 1000;
	const minutesInMilliseconds = parseInt(minutes) * 60 * 1000;
	const secondsInMilliseconds = parseInt(secondsWithoutMilliseconds) * 1000;
	const totalMilliseconds =
		hoursInMilliseconds +
		minutesInMilliseconds +
		secondsInMilliseconds +
		parseInt(milliseconds);

	// console.log(totalMilliseconds);
	return totalMilliseconds;
}
