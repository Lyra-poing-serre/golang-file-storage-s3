package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s3Client)
	presignRequest, err := presignClient.PresignGetObject(
		context.Background(),
		&s3.GetObjectInput{
			Bucket: &bucket,
			Key:    &key,
		},
		s3.WithPresignExpires(expireTime),
	)
	if err != nil {
		return "", err
	}
	return presignRequest.URL, nil
}

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	if video.VideoURL == nil {
		return video, nil
	}

	url := strings.SplitN(*video.VideoURL, ",", 2)
	if len(url) != 2 {
		return video, fmt.Errorf("malformed video url: %s", *video.VideoURL)
	}
	bucket, key := url[0], url[1]
	presignUrl, err := generatePresignedURL(cfg.s3Client, bucket, key, 2*time.Minute)
	if err != nil {
		return database.Video{}, err
	}
	video.VideoURL = &presignUrl
	return video, nil
}
