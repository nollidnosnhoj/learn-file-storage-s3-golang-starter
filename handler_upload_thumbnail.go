package main

import (
	"fmt"
	"io"
	"mime"
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

	// TODO: implement the upload here
	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	formFile, fileHeader, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer formFile.Close()
	mediaType := fileHeader.Header.Get("Content-Type")

	// Get file extension based on content type
	var fileExtension string
	validExtensions, err := mime.ExtensionsByType(mediaType)
	if err != nil || len(validExtensions) == 0 {
		fileExtension = "application/octet-stream"
	}
	fileExtension = validExtensions[0]
	fileName := fmt.Sprintf("%s%s", videoID, fileExtension)
	thumbnailPath := filepath.Join(cfg.assetsRoot, fileName)
	thumbnailFile, err := os.Create(thumbnailPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to upload thumbnail", err)
		return
	}
	defer thumbnailFile.Close()
	_, err = io.Copy(thumbnailFile, formFile)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to upload thumbnail", err)
		return
	}

	videoMetadata, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to get video metadata", err)
		return
	}
	if videoMetadata.UserID != userID {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	thumbnailURL := fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, fileName)
	videoMetadata.ThumbnailURL = &thumbnailURL
	err = cfg.db.UpdateVideo(videoMetadata)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to save video metadata", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoMetadata)
}
