package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os/exec"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
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

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s3Client)
	presignedHTTPRequest, err := presignClient.PresignGetObject(context.Background(), &s3.GetObjectInput{Bucket: &bucket, Key: &key}, s3.WithPresignExpires(expireTime))
	if err != nil {
		log.Printf("Error generating presigned HTTP request: %v", err)
		return "", err
	}

	return presignedHTTPRequest.URL, nil
}

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	if len(*video.VideoURL) == 0 {
		return database.Video{}, fmt.Errorf("Video URL of zero length")
	}

	parts := strings.Split(*video.VideoURL, ",")
	if len(parts) != 2 {
		return database.Video{}, fmt.Errorf("invalid video URL format: %s", *video.VideoURL)
	}
	bucket := parts[0]
	key := parts[1]

	presignedURL, err := generatePresignedURL(cfg.s3Client, bucket, key, time.Hour)
	if err != nil {
		log.Print(err)
		return database.Video{}, err
	}

	video.VideoURL = &presignedURL
	return video, nil
}
