package main

import (
	"crypto/rand"
	"encoding/base64"
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
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	tn, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Could'nt to get the image data", err)
		return
	}
	defer tn.Close()

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "User isn't the video owner", err)
		return
	}
	_, cSuffix, _, err := getContentType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid content", err)
		return
	}

	fileSuffix := "." + cSuffix
	bufName := make([]byte, 32)
	rand.Read(bufName)
	tnFp := filepath.Join(cfg.assetsRoot, base64.RawURLEncoding.EncodeToString(bufName)+fileSuffix)
	file, err := os.Create(tnFp)
	if err != nil {
		fmt.Println(err)
		respondWithError(w, http.StatusInternalServerError, "Can't create local file", err)
		return
	}
	_, err = io.Copy(file, tn)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, " local file", err)
		return
	}
	fmt.Println("New thumbnail created at ", tnFp)

	tnUrl := fmt.Sprintf("http://localhost:%s/%s", cfg.port, tnFp)
	fmt.Println(tnFp) // todo RM
	video.ThumbnailURL = &tnUrl
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Can't upload video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
