package cmd

import (
	"os"
	"testing"
)

func Test_FindBinaryFile_empty_dir(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("/tmp/", "test")
	defer os.RemoveAll(tmpDir)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
		return
	}

	_, err = findBinaryFile(tmpDir)
	if err.Error() != "unable to find a binary file in archive" {
		t.Errorf("Unexpected error: %s", err)
	}
}

func Test_FindBinaryFile_simple_binary(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("/tmp/", "test")
	defer os.RemoveAll(tmpDir)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
		return
	}

	// Create a binary file
	binaryFilename := tmpDir + "/test"
	f, err := os.Create(binaryFilename)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
		return
	}
	defer f.Close()

	// Write binary data to the file
	binaryData := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09}
	_, err = f.Write(binaryData)
	if err != nil {
		t.Errorf("Error writing to file: %s", err)
		return
	}

	foundFilename, err := findBinaryFile(tmpDir)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
		return
	}
	if foundFilename != binaryFilename {
		t.Errorf("Expected %s, got %s", binaryFilename, foundFilename)
	}
}

func Test_FindBinaryFile_simple_binary_nested(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("/tmp/", "test")
	defer os.RemoveAll(tmpDir)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
		return
	}

	// Create a nested directory
	nestedDir := tmpDir + "/nested"
	err = os.Mkdir(nestedDir, os.ModePerm)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
		return
	}

	// Create a binary file
	binaryFilename := nestedDir + "/test"
	f, err := os.Create(binaryFilename)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
		return
	}
	defer f.Close()

	// Write binary data to the file
	binaryData := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09}
	_, err = f.Write(binaryData)
	if err != nil {
		t.Errorf("Error writing to file: %s", err)
		return
	}

	foundFilename, err := findBinaryFile(tmpDir)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
		return
	}
	if foundFilename != binaryFilename {
		t.Errorf("Expected %s, got %s", binaryFilename, foundFilename)
	}
}
