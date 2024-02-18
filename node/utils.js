const ICAL = require("node-ical");
const axios = require('axios');
const fs = require('fs');

const {
    isWithinInterval,
    addDays,
    parse,
    getTime,
    addMilliseconds,
} = require("date-fns");
const fs = require("fs");
function getCurrentEventForDate(icalFilePath, providedDate) {
    try {
        const data = require("fs").readFileSync(icalFilePath, "utf8");
        const parsedData = ICAL.parseICS(data);

        for (const eventId in parsedData) {
            if (parsedData.hasOwnProperty(eventId)) {
                const event = parsedData[eventId];
                const startDate = event.start.toISOString();
                const endDate = event.end.toISOString();
                if (event.rrule && typeof event.rrule === "object") {
                    // Generate future occurrences within a reasonable time frame
                    for (let i = 0; i < 365; i++) {
                        const occurrenceDate = addDays(startDate, i);
                        const occurrenceEndDate = addDays(endDate, i);
                        if (
                            isWithinInterval(providedDate, {
                                start: occurrenceDate,
                                end: occurrenceEndDate,
                            })
                        ) {
                            // console.log("yo",
                            //   occurrenceDate,
                            //   providedDate,
                            //   event.rrule.options.interval,
                            //   event.summary,
                            // );
                            return event;
                        }
                    }
                } else {
                    if (
                        isWithinInterval(providedDate, { start: startDate, end: endDate })
                    ) {
                        // If no recurrence rule or invalid RRULE, check if it's a one-time event
                        return event;
                    }
                }
            }
        }

        // If no current event is found for the provided date, return undefined
        return undefined;
    } catch (error) {
        console.error("Error reading or parsing the iCal file:", error);
        return undefined;
    }
}

const resolution = "720:480";
const tvName = "usual tv";
function getStreamCommand(video) {
    const nextVideo = "next";
    return [
        "-nostdin",
        "-re",
        "-i",
        video.path,
        "-i",
        "overlay.png",
        "-c:v",
        "libx264",
        "-c:a",
        "copy", // Add this line to copy the audio codec
        "-loglevel",
        "error",
        "-filter_complex",
        `scale=${resolution}:force_original_aspect_ratio=decrease,pad=${resolution}:(ow-iw)/2:(oh-ih)/2,` +
        "overlay=0:0," +
        `drawtext=fontsize=18:fontfile=font.ttf:fontcolor=white:textfile=weather.txt:x=w-tw+20:y=(-35),` +
        `drawtext=fontsize=25:fontcolor=white:text='${tvName}':x=25:y=25,` +
        `drawtext=fontsize=11:fontcolor=white:text='%{pts\\:hms}':x=(6):y=h-th-13,` +
        `drawtext=fontsize=11:fontcolor=white:text='${"  0" + video.duration.replace(/:/g, "\\:")
        }':x=(10):y=h-th-2,` +
        `drawtext=fontsize=16:fontcolor=white:text='${video.name}':x=(w-tw-25):y=h-th-35,` +
        // `drawtext=fontsize=13:fontcolor=white:text='${nextVideo}':x=(w-tw-25):y=h-th-19,` +
        `drawtext=fontsize=13:fontcolor=white:text='${video.type}':x=(w-tw-25):y=h-th-19,` +
        `drawtext=fontsize=18:fontcolor=white:text='%{localtime\\:%T}':x=35:y=83[v]`,
        "-map",
        "[v]",
        "-map",
        "0:a",
        "-hls_time",
        "0.25",
        "-hls_list_size",
        "5",
        "-f",
        "hls",
        "-segment_wrap",
        "6",
        "-hls_flags",
        "delete_segments+append_list+omit_endlist",
        "static/stream.m3u8",
    ];
}

async function downloadICSFile(url, path) {
    try {
        const response = await axios({
            method: 'GET',
            url: url,
            responseType: 'stream',
        });

        const writer = fs.createWriteStream(path);

        response.data.pipe(writer);

        return new Promise((resolve, reject) => {
            writer.on('finish', resolve);
            writer.on('error', reject);
        });
        
    } catch (error) {
        console.error('Error downloading the .ics file:', error);
    }
}

module.exports = { getCurrentEventForDate, getStreamCommand, downloadICSFile };