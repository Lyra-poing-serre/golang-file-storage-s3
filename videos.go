package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
	"path/filepath"
)

func getVideoAspectRatio(filePath string) (string, error) {
	const cmdName = "ffprobe"
	cmdArgs := []string{
		"-v", "error", "-print_format", "json", "-show_streams", filePath,
	}
	var b bytes.Buffer
	type cmdOutput struct {
		Streams []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"streams"`
	}
	const (
		ratio16by9 = 16.0 / 9.0
		ratio9by16 = 9.0 / 16.0
		tolerance  = 0.02 // tolérance de 2 %
	)
	cmd := exec.Command(cmdName, cmdArgs...)
	cmd.Stdout = &b
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	var output cmdOutput
	err = json.Unmarshal(b.Bytes(), &output)
	if err != nil {
		return "", err
	}
	//thx boots
	width, height := output.Streams[0].Width, output.Streams[0].Height
	ratio := float64(width) / float64(height)
	if math.Abs(ratio-ratio16by9) < tolerance {
		return "16:9", nil
	} else if math.Abs(ratio-ratio9by16) < tolerance {
		return "9:16", nil
	} else {
		return "other", nil
	}
}

func processVideoForFastStart(filePath string) (string, error) {
	const cmdName = "ffmpeg"
	name := filepath.Base(filePath)
	suffix := filepath.Ext(filePath)
	outFile := filepath.Join(
		filepath.Dir(filePath),
		fmt.Sprintf("%s.processing%s", name[:len(name)-len(suffix)], suffix),
	)
	fmt.Println(filePath, outFile)
	cmdArgs := []string{
		"-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outFile,
	}
	cmd := exec.Command(cmdName, cmdArgs...)
	fmt.Println(cmd.String())
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return outFile, nil
}
