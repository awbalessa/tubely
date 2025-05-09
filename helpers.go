package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os/exec"
)

func getVideoAspectRatio(filepath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filepath)
	var buffer bytes.Buffer
	cmd.Stdout = &buffer
	if err := cmd.Run(); err != nil {
		log.Print("Error running exec command")
		return "", err
	}

	type Stream struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	}

	type FFProbeOutput struct {
		Streams []Stream `json:"streams"`
	}

	var output FFProbeOutput
	if err := json.Unmarshal(buffer.Bytes(), &output); err != nil {
		log.Print("Error unmarshalling cmd stdout")
		return "", err
	}

	if len(output.Streams) == 0 {
		log.Print("Empty output stream slice")
		return "", fmt.Errorf("Empty output stream slice")
	}

	aspect := float64(output.Streams[0].Width) / float64(output.Streams[0].Height)
	if math.Abs(aspect-16.0/9.0) < 0.1 {
		return "16:9", nil
	} else if math.Abs(aspect-9.0/16.0) < 0.1 {
		return "9:16", nil
	} else {
		return "other", nil
	}
}

func processVideoForFastStart(filepath string) (string, error) {
	outputPath := fmt.Sprintf("%s.processing", filepath)
	cmd := exec.Command("ffmpeg", "-i", filepath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outputPath)
	if err := cmd.Run(); err != nil {
		log.Print("Error running exec command")
		return "", err
	}

	return outputPath, nil
}
