package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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

	maxMemory := 10 << 20
	if err := r.ParseMultipartForm(int64(maxMemory)); err != nil {
		respondWithError(w, http.StatusFailedDependency, "Couldn't parse form", err)
		return
	}

	MFile, MFileHeader, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusFailedDependency, "Couldn't form file", err)
		return
	}

	Mime, _, err := mime.ParseMediaType(MFileHeader.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusFailedDependency, "Couldn't fetch mime", err)
		return
	}
	if Mime != "image/jpeg" && Mime != "image/png" {
		respondWithError(w, http.StatusBadRequest, "Wrong media type", err)
		return
	}
	extension := strings.Split(Mime, "/")[1]
	AssetString := fmt.Sprintf("/%s.%s", videoID.String(), extension)
	VideoPath := filepath.Join(cfg.assetsRoot, AssetString)
	VideoAddress, err := os.Create(VideoPath)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Could not create file", err)
		return
	}

	if _, err := io.Copy(VideoAddress, MFile); err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't write to file", err)
		return
	}

	DBVideoMeta, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get metadata", err)
		return
	}
	if DBVideoMeta.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "User not video owner", err)
		return
	}
	NewPath := fmt.Sprintf("http://localhost:%s/assets/%s.%s", cfg.port, videoID.String(), extension)
	DBVideoMeta.ThumbnailURL = &NewPath

	if err := cfg.db.UpdateVideo(DBVideoMeta); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not update video meta data", err)
		return
	}

	respondWithJSON(w, http.StatusOK, DBVideoMeta)
}
