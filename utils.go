package main

import (
	"io"
	"net/http"
	"os"
	"strings"
)

func downloadFile(url string, path string) (string, error) {
	tokens := strings.Split(url, "/")
	fileName := tokens[len(tokens)-1]
	fullPath := path + string(os.PathSeparator) + fileName

	if _, err := os.Stat(fullPath); err == nil {
		logger.Noticef("%s exists", fileName)
		return fileName, nil
	}

	output, err := os.Create(fullPath)
	if err != nil {
		logger.Error(err)
		return "", err
	}
	defer output.Close()

	logger.Debugf("--> Downloading %s", url)
	response, err := http.Get(url)
	if err != nil {
		logger.Error(err)
		return "", err
	}
	defer response.Body.Close()

	n, err := io.Copy(output, response.Body)
	if err != nil {
		logger.Error(err)
		return "", err
	}
	logger.Debugf("%s: %d bytes", fileName, n)
	return fileName, nil
}

func removeFile(url string, path string) error {
	tokens := strings.Split(url, "/")
	fileName := tokens[len(tokens)-1]
	logger.Debugf("--> Deleting %s", fileName)
	fullPath := path + string(os.PathSeparator) + fileName
	err := os.Remove(fullPath)
	if err != nil {
		logger.Error(err)
		return err
	}
	logger.Debugf("--> Deleted %s", fileName)
	return nil
}
