package cmd

import (
	"os"
	"path"
	"testing"
)

func Test_FindBinaryFile_empty_dir(t *testing.T) {
	tmpDir := t.TempDir() + "/"

	_, err := findBinaryFile(tmpDir)
	if err == nil {
		t.Errorf("Expected error, got <nil>")
		return
	}
	if want := "unable to find a binary file in archive"; err.Error() != want {
		t.Errorf("Unexpected error: %s", err)
	}
}

func Test_FindBinaryFile_simple_binary(t *testing.T) {
	tmpDir := t.TempDir() + "/"

	binaryFilename := tmpDir + "test"
	binaryData := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09}
	if err := os.WriteFile(binaryFilename, binaryData, 0644); err != nil {
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
	tmpDir := t.TempDir() + "/"

	nestedDir := tmpDir + "nested"
	if err := os.Mkdir(nestedDir, os.ModePerm); err != nil {
		t.Errorf("Unexpected error: %s", err)
		return
	}

	binaryFilename := nestedDir + "/test"
	binaryData := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09}
	if err := os.WriteFile(binaryFilename, binaryData, 0644); err != nil {
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

func Test_FindBinaryFile_macos_binary(t *testing.T) {
	tmpDir := t.TempDir() + "/"

	// Create a binary file and a second one with the reserved macOS
	// "._" prefix, which is not executable.
	// https://github.com/jsnjack/grm/issues/12
	binaryFilename := path.Join(tmpDir, "test")
	binaryFilename2 := path.Join(tmpDir, "._test")
	binaryData := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09}
	if err := os.WriteFile(binaryFilename, binaryData, 0644); err != nil {
		t.Errorf("Error writing to file: %s", err)
		return
	}
	if err := os.WriteFile(binaryFilename2, binaryData, 0644); err != nil {
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
