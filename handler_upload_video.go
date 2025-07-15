package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	uploadLimit := 1 << 30 // 1GB
	r.Body = http.MaxBytesReader(w, r.Body, int64(uploadLimit))

	vId, err := uuid.Parse(r.PathValue("videoID"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error(), err)
		return
	}

	apiKey, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error(), err)
		return
	}
	uId, err := auth.ValidateJWT(apiKey, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error(), err)
		return
	}

	dbVideo, err := cfg.db.GetVideo(vId)
	if err != nil {
		respondWithError(w, http.StatusNotFound, err.Error(), err)
		return
	}
	if dbVideo.UserID != uId {
		respondWithError(w, http.StatusUnauthorized, "not the owner of the video", errors.New("not the owner of the video"))
		return
	}

	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error(), err)
		return
	}
	defer file.Close()
	contentType := header.Header.Get("Content-Type")
	_, _, _, err = getContentType(contentType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error(), err)
		return
	}
	tmpFile, err := os.CreateTemp("", vId.String()+"*"+".mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error(), err)
		return
	}
	defer os.Remove(tmpFile.Name())

	_, err = io.Copy(tmpFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error(), err)
		return
	}
	_, err = tmpFile.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error(), err)
		return
	}

	processedFile, err := processVideoForFastStart(tmpFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error(), err)
		return
	}
	tmpFile.Close()
	os.Remove(tmpFile.Name())
	tmpFile, err = os.Open(processedFile)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error(), err)
		return
	}
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	ratio, err := getVideoAspectRatio(tmpFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error(), err)
		return
	}

	b := make([]byte, 32)
	rand.Read(b)

	fileName := base64.RawURLEncoding.EncodeToString(b) + ".mp4"
	switch ratio {
	case "16:9":
		fileName = "landscape" + "/" + fileName
	case "9:16":
		fileName = "portrait" + "/" + fileName
	default:
		fileName = ratio + "/" + fileName
	}
	_, err = cfg.s3Client.PutObject(
		context.Background(),
		&s3.PutObjectInput{
			Bucket:      &cfg.s3Bucket, // aws.String(cfg.s3Bucket),
			Key:         &fileName,
			Body:        tmpFile,
			ContentType: &contentType,
		},
	)
	if err != nil {
		respondWithError(w, http.StatusBadGateway, err.Error(), err)
		return
	}
	url := fmt.Sprintf("%s,%s", cfg.s3Bucket, fileName)
	dbVideo.VideoURL = &url
	err = cfg.db.UpdateVideo(dbVideo)
	if err != nil {
		respondWithError(w, http.StatusBadGateway, err.Error(), err)
		return
	}

	signedVideo, err := cfg.dbVideoToSignedVideo(dbVideo)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error(), err)
		return
	}
	fmt.Printf("New video %s uploaded !\n", *signedVideo.VideoURL)

}
