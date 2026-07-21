package cmd

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"log/slog"
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
	client, err := CreateClient()
	if err != nil {
		return "", err
	}
	reader, _, err := client.Repositories.DownloadReleaseAsset(context.Background(), pkg.Owner, pkg.Repo, asset.GetID(), http.DefaultClient)
	if err != nil {
		return "", fmt.Errorf("download release asset %s: %w", asset.GetName(), err)
	}
	defer logClose(reader)

	// Create a directory
	path := fmt.Sprintf(DefaultTmpDirPattern+"%s/", generateRandomString(6))
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.Mkdir(path, os.ModePerm); err != nil {
			return "", fmt.Errorf("create temp directory %s: %w", path, err)
		}
	}

	destination := path + asset.GetName()
	var out io.Writer
	f, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", fmt.Errorf("create %s: %w", destination, err)
	}
	out = f
	defer logClose(f)

	if isTTY && !rootNoProgress {
		bar := progressbar.NewOptions(
			asset.GetSize(),
			progressbar.OptionSetBytes(asset.GetSize()),
		)
		out = io.MultiWriter(out, bar)
	}
	if _, err := io.Copy(out, reader); err != nil {
		return "", fmt.Errorf("write %s: %w", destination, err)
	}
	if isTTY && !rootNoProgress {
		fmt.Println()
	}
	return destination, nil
}

func sudo() string {
	_, err := exec.LookPath("sudo")
	if err == nil {
		return "sudo "
	}
	return ""
}

func installBinary(filename string, renameBinaryTo string) (string, error) {
	slog.Debug("installing as a binary")
	tmpDir := getTmpDir(filename)

	installedBinaryName := renameBinaryTo
	if installedBinaryName == "" {
		installedBinaryName = filepath.Base(filename)
	}
	installedFile := fmt.Sprintf("%s%s", DefaultBinDir, installedBinaryName)

	msgStep("Installing %s %s %s", bold(strings.TrimPrefix(filename, tmpDir)), cyan(sArrow), bold(installedFile))

	if err := removeBinary(installedFile); err != nil {
		return "", err
	}
	cmdCp := exec.Command("/bin/sh", "-c", fmt.Sprintf("%scp %s %s", sudo(), filename, installedFile))
	if err := cmdCp.Run(); err != nil {
		return "", fmt.Errorf("copy %s to %s: %w", filename, installedFile, err)
	}
	cmdChmod := exec.Command("/bin/sh", "-c", sudo()+"chmod 755 "+installedFile)
	err := cmdChmod.Run()
	if err != nil {
		err = fmt.Errorf("chmod %s: %w", installedFile, err)
	}

	if strings.HasPrefix(tmpDir, DefaultTmpDirPattern) {
		slog.Debug("removing temp dir", "path", tmpDir)
		cmdRm := exec.Command("/bin/sh", "-c", fmt.Sprintf("rm -rf %s", tmpDir))
		if rmErr := cmdRm.Run(); rmErr != nil {
			slog.Log(context.Background(), LevelTrace, "remove temp dir", "path", tmpDir, "err", rmErr)
		}
	}
	return installedFile, err
}

func removeBinary(filename string) error {
	cmdRm := exec.Command("/bin/sh", "-c", fmt.Sprintf("%srm -f %s", sudo(), filename))
	if err := cmdRm.Run(); err != nil {
		return fmt.Errorf("remove %s: %w", filename, err)
	}
	return nil
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

// askForNumber prompts for a number between 1 and to (inclusive). If --yes
// was passed, it returns 1 without prompting.
func askForNumber(msg string, to int) (int, error) {
	if rootYes {
		return 1, nil
	}
	fmt.Printf("%s %s ", msg, dim(fmt.Sprintf("[1-%d]", to)))
	var response string
	if _, err := fmt.Scanln(&response); err != nil {
		return 0, fmt.Errorf("read response: %w", err)
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
	return responseInt, nil
}

// askForConfirmation prompts for a yes/no answer. If --yes was passed, it
// returns true without prompting.
func askForConfirmation(msg string) (bool, error) {
	if rootYes {
		return true, nil
	}
	fmt.Print(msg + " " + dim("[y/n]") + " ")
	var response string
	if _, err := fmt.Scanln(&response); err != nil {
		return false, fmt.Errorf("read response: %w", err)
	}
	okayResponses := []string{"y", "Y", "yes", "Yes", "YES"}
	nokayResponses := []string{"n", "N", "no", "No", "NO"}
	if containsString(okayResponses, response) {
		return true, nil
	} else if containsString(nokayResponses, response) {
		return false, nil
	}
	fmt.Println("  Please type yes or no and then press enter:")
	return askForConfirmation(msg)
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
	return posString(slice, element) != -1
}

// ProgressReader is a reader that prints progress
type ProgressReader struct {
	r   io.Reader
	bar *progressbar.ProgressBar
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.r.Read(p)
	if addErr := pr.bar.Add(n); addErr != nil {
		slog.Log(context.Background(), LevelTrace, "update progress bar", "err", addErr)
	}
	return n, err
}

// CreateClient creates github client instance. It will try to use GITHUB_TOKEN
// environment variable to create authenticated client (no rate limits)
func CreateClient() (*github.Client, error) {
	// First check if the token was provided as a flag
	token := rootToken
	if token != "" {
		slog.Debug("using token from flag")
	}
	if token == "" {
		// See if it is set in configuration
		config, err := ReadConfig(ConfigFile)
		if err != nil {
			return nil, err
		}
		token = config.Settings["token"]
		if token != "" {
			slog.Debug("using token from config")
		}
	}
	if token == "" {
		// Try to get it from environments
		token = os.Getenv("GITHUB_TOKEN")
		if token != "" {
			slog.Debug("using token from env")
		}
	}
	if token == "" {
		// Give up, use anonymous session
		slog.Debug("using anonymous client")
		return github.NewClient(nil), nil
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	return github.NewClient(tc), nil
}

func tomd5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", filePath, err)
	}
	defer logClose(file)
	h := md5.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", fmt.Errorf("hash %s: %w", filePath, err)
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// logClose closes c and logs any error at trace level. Deferred close
// errors are rarely actionable, so they don't propagate as return values.
func logClose(c io.Closer) {
	if err := c.Close(); err != nil {
		slog.Log(context.Background(), LevelTrace, "close", "err", err)
	}
}
