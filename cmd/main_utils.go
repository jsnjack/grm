package cmd

import (
	"fmt"
	"io"
	"log"
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
	installedFile := fmt.Sprintf("%s%s", DefaultBinDir, filepath.Base(filename))
	err := removeBinary(installedFile)
	if err != nil {
		return "", err
	}
	cmdCp := exec.Command("/bin/sh", "-c", fmt.Sprintf("sudo cp %s %s", filename, DefaultBinDir))
	err = cmdCp.Run()
	if err != nil {
		return "", err
	}
	cmd := exec.Command("/bin/sh", "-c", "sudo chmod +x "+installedFile)
	err = cmd.Run()
	return installedFile, err
}

func removeBinary(filename string) error {
	cmdRm := exec.Command("/bin/sh", "-c", fmt.Sprintf("sudo rm -f %s", filename))
	err := cmdRm.Run()
	return err
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

func askForConfirmation(msg string) bool {
	if rootYes {
		return true
	}
	fmt.Printf(msg + " [y/n] ")
	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		log.Fatal(err)
	}
	okayResponses := []string{"y", "Y", "yes", "Yes", "YES"}
	nokayResponses := []string{"n", "N", "no", "No", "NO"}
	if containsString(okayResponses, response) {
		return true
	} else if containsString(nokayResponses, response) {
		return false
	} else {
		fmt.Println("Please type yes or no and then press enter:")
		return askForConfirmation(msg)
	}
}

// posString returns the first index of element in slice.
// If slice does not contain element, returns -1.
func posString(slice []string, element string) int {
	for index, elem := range slice {
		if elem == element {
			return index
		}
	}
	return -1
}

// containsString returns true iff slice contains element
func containsString(slice []string, element string) bool {
	return !(posString(slice, element) == -1)
}
