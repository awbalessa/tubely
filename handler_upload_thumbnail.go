package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	const maxMemory = 10 << 20
	if err = r.ParseMultipartForm(maxMemory); err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse thumbnail", err)
		return
	}

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse thumbnail", err)
		return
	}
	defer file.Close()

	mediaType := header.Header.Get("Content-Type")
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to get video from database", err)
		return
	}

	if video.UserID != userID {
		http.Error(w, "Unauthorized to access this video", http.StatusUnauthorized)
		return
	}

	var extension string
	switch mediaType {
	case "image/jpeg":
		extension = ".jpg"
	case "image/png":
		extension = ".png"
	default:
		extension = ".bin"
	}

	path := filepath.Join(cfg.assetsRoot, videoIDString+extension)
	newFile, err := os.Create(path)
	if err != nil {
		respondWithError(w, 500, "Internal server error", err)
		return
	}

	defer newFile.Close()
	_, err = io.Copy(newFile, file)
	if err != nil {
		respondWithError(w, 500, "Internal server error", err)
		return
	}

	thumbStr := fmt.Sprintf("http://localhost:%s/assets/%s%s", cfg.port, videoIDString, extension)

	video.ThumbnailURL = &thumbStr
	if err = cfg.db.UpdateVideo(video); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
