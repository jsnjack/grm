package cmd

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v32/github"
	"github.com/schollz/progressbar/v2"
	"golang.org/x/oauth2"
)

// DefaultBinDir is the default location for binary files
const DefaultBinDir = "/usr/local/bin/"

// DefaultTmpDirPattern is the pattern that is used to generate tmp directory
// for packages during the installation
const DefaultTmpDirPattern = "/tmp/grm."

func downloadFile(asset *github.ReleaseAsset, pkg *Package) (string, error) {
	client := CreateClient()
	reader, _, err := client.Repositories.DownloadReleaseAsset(context.Background(), pkg.Owner, pkg.Repo, asset.GetID(), http.DefaultClient)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	// Create a directory
	path := fmt.Sprintf(DefaultTmpDirPattern+"%s/", generateRandomString(6))
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.Mkdir(path, os.ModePerm)
		if err != nil {
			return "", err
		}
	}

	var out io.Writer
	f, err := os.OpenFile(path+asset.GetName(), os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", err
	}
	out = f
	defer f.Close()

	bar := progressbar.NewOptions(
		asset.GetSize(),
		progressbar.OptionSetBytes(asset.GetSize()),
	)
	out = io.MultiWriter(out, bar)
	_, err = io.Copy(out, reader)
	if err != nil {
		return "", err
	}
	fmt.Println("")
	return path + asset.GetName(), nil
}

func installBinary(filename string, renameBinaryTo string, sudo string) (string, error) {
	logln("Installing as a binary")
	tmpDir := getTmpDir(filename)

	installedBinaryName := renameBinaryTo
	if installedBinaryName == "" {
		installedBinaryName = filepath.Base(filename)
	}
	installedFile := fmt.Sprintf("%s%s", DefaultBinDir, installedBinaryName)

	fmt.Printf("Installing %s to %s...\n", strings.TrimPrefix(filename, tmpDir), installedFile)

	err := removeBinary(installedFile, sudo)
	if err != nil {
		return "", err
	}
	cmdCp := exec.Command("/bin/sh", "-c", fmt.Sprintf("%scp %s %s", sudo, filename, installedFile))
	err = cmdCp.Run()
	if err != nil {
		return "", err
	}

	cmd := exec.Command("/bin/sh", "-c", sudo+"chmod +x "+installedFile)

	err = cmd.Run()

	if strings.HasPrefix(tmpDir, DefaultTmpDirPattern) {
		logf("Removing %s...\n", tmpDir)
		cmdRm := exec.Command("/bin/sh", "-c", fmt.Sprintf("rm -rf %s", tmpDir))
		err := cmdRm.Run()
		if err != nil {
			logln(err)
		}
	}
	return installedFile, err
}

func removeBinary(filename string, sudo string) error {
	cmdRm := exec.Command("/bin/sh", "-c", fmt.Sprintf("%srm -f %s", sudo, filename))
	err := cmdRm.Run()
	return err
}

func getTmpDir(path string) string {
	if strings.HasPrefix(path, DefaultTmpDirPattern) {
		split := strings.Split(path, "/")
		return "/" + split[1] + "/" + split[2] + "/"
	}
	return ""
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

func askForNumber(msg string, to int) int {
	if rootYes {
		return 1
	}
	fmt.Printf("%s [1-%d] ", msg, to)
	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		log.Fatal(err)
	}
	responseInt, err := strconv.Atoi(response)
	if err != nil {
		fmt.Printf("  Provide a number (%s)\n", err)
		return askForNumber(msg, to)
	}
	if responseInt > to || responseInt < 1 {
		fmt.Println("  Out of range")
		return askForNumber(msg, to)
	}
	return responseInt
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
		fmt.Println("  Please type yes or no and then press enter:")
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

// ProgressReader is a reader that prints progress
type ProgressReader struct {
	r   io.Reader
	bar *progressbar.ProgressBar
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.r.Read(p)
	pr.bar.Add(n)
	return n, err
}

// CreateClient creates github client instance. It will try to use GITHUB_TOKEN
// environment variable to create authenticated client (no rate limits)
func CreateClient() *github.Client {
	// First check if the token was provided as a flag
	token := rootToken
	if token != "" {
		logf("Token from flag: %s\n", token)
	}
	if token == "" {
		// See if it is set in configuration
		config, err := ReadConfig(ConfigFile)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		token = config.Settings["token"]
		if token != "" {
			logf("Token from config: %s\n", token)
		}
	}
	if token == "" {
		// Try to get it from environments
		token = os.Getenv("GITHUB_TOKEN")
		if token != "" {
			logf("Token from env: %s\n", token)
		}
	}
	if token == "" {
		// Give up, use anonymous session
		logln("Using anonymous client")
		return github.NewClient(nil)
	}
	logf("Using client with token: %s\n", token)

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	return github.NewClient(tc)
}

func tomd5(filePath string) (string, error) {
	var md5Value string
	file, err := os.Open(filePath)
	if err != nil {
		return md5Value, err
	}
	defer file.Close()
	h := md5.New()
	if _, err := io.Copy(h, file); err != nil {
		return md5Value, err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func logf(format string, a ...interface{}) {
	if rootVerbose {
		fmt.Printf(format, a...)
	}
}

func logln(a ...interface{}) {
	if rootVerbose {
		fmt.Println(a...)
	}
}
