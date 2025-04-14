package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// Metadata structure to hold song information
type Metadata struct {
	Artist string `json:"artist"`
	Title  string `json:"title"`
	Album  string `json:"album"`
	Year   string `json:"year"`
}

// Function to extract metadata and return a formatted string
func extractMetadataString(filePath string) string {
	// Use FFprobe (from FFmpeg suite) specifically for metadata extraction
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		filePath)

	output, err := cmd.Output()
	if err != nil {
		return filepath.Base(filePath)
	}

	// Parse the JSON output
	var result struct {
		Format struct {
			Tags map[string]string `json:"tags"`
		} `json:"format"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return filepath.Base(filePath)
	}

	// Initialize metadata values
	var artist, title, album, year string

	// Handle case-insensitive tag names (different files might have different cases)
	if result.Format.Tags != nil {
		for key, value := range result.Format.Tags {
			lowerKey := strings.ToLower(key)
			switch lowerKey {
			case "artist", "albumartist":
				if artist == "" {
					artist = value
				}
			case "title":
				title = value
			case "album":
				album = value
			case "date", "year", "originaldate":
				if year == "" {
					year = value
				}
			}
		}
	}

	// If metadata is missing, try to extract from filename
	if title == "" || artist == "" {
		filename := filepath.Base(filePath)
		nameWithoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))

		// Try to split on common separators like " - " to extract artist and title
		parts := strings.SplitN(nameWithoutExt, " - ", 2)
		if len(parts) == 2 {
			if artist == "" {
				artist = strings.TrimSpace(parts[0])
			}
			if title == "" {
				title = strings.TrimSpace(parts[1])
			}
		} else {
			// If no separator, just use filename as title
			if title == "" {
				title = nameWithoutExt
			}
		}
	}

	// Format metadata into a string based on what's available
	if artist != "" && title != "" {
		result := fmt.Sprintf("%s - %s", artist, title)

		if album != "" {
			result += fmt.Sprintf(" [%s", album)
			if year != "" {
				result += fmt.Sprintf(", %s", year)
			}
			result += "]"
		} else if year != "" {
			result += fmt.Sprintf(" [%s]", year)
		}

		return result
	} else if title != "" {
		if year != "" {
			return fmt.Sprintf("%s [%s]", title, year)
		}
		return title
	}

	// If we couldn't extract anything useful, return the filename
	return filepath.Base(filePath)
}
