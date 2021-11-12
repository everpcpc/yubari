package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func getFileName(file string) string {
	tokens := strings.Split(file, string(os.PathSeparator))
	return tokens[len(tokens)-1]
}

func downloadFile(url string, path string) (string, error) {
	tokens := strings.Split(url, "/")
	fileName := tokens[len(tokens)-1]
	// support for twitter https://pbs.twimg.com/media/DdFIBS0VQAMhpmv.png:orig
	if subToken := strings.Split(fileName, ":"); len(subToken) == 2 {
		fileName = subToken[0]
	}
	// support for twitter https://pbs.twimg.com/media/xxxx.mp4?tag=3
	if subToken := strings.Split(fileName, "?"); len(subToken) == 2 {
		fileName = subToken[0]
	}

	fullPath := path + string(os.PathSeparator) + fileName

	if _, err := os.Stat(fullPath); err == nil {
		logger.Infof("%s exists", fullPath)
		return fullPath, nil
	}

	output, err := os.Create(fullPath)
	if err != nil {
		logger.Errorf("%+v", err)
		return "", err
	}
	defer output.Close()

	logger.Debugf("--> Downloading %s", url)
	response, err := http.Get(url)
	if err != nil {
		logger.Errorf("%+v", err)
		return "", err
	}
	defer response.Body.Close()

	n, err := io.Copy(output, response.Body)
	if err != nil {
		logger.Errorf("%+v", err)
		return "", err
	}
	logger.Debugf("%s: %s", fullPath, ByteCountIEC(n))
	return fullPath, nil
}

func removeFile(url string, path string) error {
	tokens := strings.Split(url, "/")
	fileName := tokens[len(tokens)-1]
	logger.Debugf("--> Deleting %s", fileName)
	fullPath := path + string(os.PathSeparator) + fileName
	err := os.Remove(fullPath)
	if err != nil {
		logger.Errorf("%+v", err)
		return err
	}
	logger.Debugf("--> Deleted %s", fullPath)
	return nil
}

func ByteCountIEC(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB",
		float64(b)/float64(div), "KMGTPE"[exp])
}
