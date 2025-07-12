package main

import (
	"errors"
	"mime"
	"strings"
)

func getContentType(mediaType string) (string, string, map[string]string, error) {
	mediaType, params, err := mime.ParseMediaType(mediaType)
	if err != nil {
		return "", "", map[string]string{}, err
	}
	switch contentType := strings.Split(mediaType, "/"); contentType[0] {
	case "image":
		if contentType[1] == "jpeg" || contentType[1] == "png" {
			return contentType[0], contentType[1], params, nil
		} else {
			return "", "", map[string]string{}, errors.New("allow only jpeg or png images")
		}
	case "video":
		if contentType[1] == "mp4" {
			return contentType[0], contentType[1], params, nil
		} else {
			return "", "", map[string]string{}, errors.New("allow onlymp4 videos")
		}
	default:
		return "", "", map[string]string{}, errors.New("allow only images or videos")
	}
}
