package main

import (
	"fmt"
	"io"
	"net/http"

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

	MFile, _, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusFailedDependency, "Couldn't form file", err)
		return
	}
	ImageData, err := io.ReadAll(MFile)
	if err != nil {
		respondWithError(w, http.StatusFailedDependency, "Couldn't read image data", err)
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
	TN := thumbnail{
		data:      ImageData,
		mediaType: "image",
	}

	videoThumbnails[videoID] = TN
	videoURL := fmt.Sprintf("http://localhost:<port>/api/thumbnails/%s", videoID)

	DBVideoMeta.ThumbnailURL = &videoURL
	if err := cfg.db.UpdateVideo(DBVideoMeta); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not update video meta data", err)
		return
	}

	respondWithJSON(w, http.StatusOK, DBVideoMeta)
}
