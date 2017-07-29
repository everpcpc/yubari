package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func downloadFile(url string, path string) {
	tokens := strings.Split(url, "/")
	fileName := tokens[len(tokens)-1]
	fullPath := fmt.Sprintf("%s%s%s", path, string(os.PathSeparator), fileName)

	if _, err := os.Stat(fullPath); err == nil {
		logger.Noticef("%s exists", fileName)
		return
	}

	output, err := os.Create(fullPath)
	if err != nil {
		logger.Error(err)
		return
	}
	defer output.Close()

	logger.Infof("--> Downloading %s", url)
	response, err := http.Get(url)
	if err != nil {
		logger.Error(err)
		return
	}
	defer response.Body.Close()

	n, err := io.Copy(output, response.Body)
	if err != nil {
		logger.Error(err)
		return
	}
	logger.Debugf("%s: %d bytes", fileName, n)
}

func removeFile(url string, path string) {
	tokens := strings.Split(url, "/")
	fileName := tokens[len(tokens)-1]
	logger.Infof("--> Deleting %s", fileName)
	fullPath := fmt.Sprintf("%s%s%s", path, string(os.PathSeparator), fileName)
	err := os.Remove(fullPath)
	if err != nil {
		logger.Error(err)
		return
	}
	logger.Debugf("--> Deleted %s", fileName)
}
