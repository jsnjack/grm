package cmd

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/schollz/progressbar/v2"
)

// DefaultBinDir is the default location for binary files
const DefaultBinDir = "/usr/local/bin/"

func downloadFile(url string, filename string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Create a directory
	path := fmt.Sprintf("/tmp/grm.%s/", generateRandomString(6))
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.Mkdir(path, os.ModePerm)
		if err != nil {
			return "", err
		}
	}

	var out io.Writer
	f, err := os.OpenFile(path+filename, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", err
	}
	out = f
	defer f.Close()

	bar := progressbar.NewOptions(
		int(resp.ContentLength),
		progressbar.OptionSetBytes(int(resp.ContentLength)),
	)
	out = io.MultiWriter(out, bar)
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}
	fmt.Println("")
	return path + filename, nil
}

func installBinary(filename string) (string, error) {
	fmt.Printf("Installing in %s...\n", DefaultBinDir)
	cmdCp := exec.Command("/bin/sh", "-c", fmt.Sprintf("sudo cp %s %s", filename, DefaultBinDir))
	err := cmdCp.Run()
	if err != nil {
		return "", err
	}
	installedFile := fmt.Sprintf("%s%s", DefaultBinDir, filepath.Base(filename))
	cmd := exec.Command("/bin/sh", "-c", "sudo chmod +x "+installedFile)
	err = cmd.Run()
	return installedFile, err
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var src = rand.NewSource(time.Now().UnixNano())

// generateRandomString generates random string of requested length
func generateRandomString(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}
