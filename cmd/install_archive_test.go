package cmd

import (
	"archive/tar"
	"archive/zip"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/mholt/archives"
)

func Test_FindBinaryFile_empty_dir(t *testing.T) {
	tmpDir := t.TempDir() + "/"

	_, err := findBinaryFiles(tmpDir)
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

	foundFilename, err := findBinaryFiles(tmpDir)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
		return
	}
	if len(foundFilename) != 1 || foundFilename[0].fullPath != binaryFilename {
		t.Errorf("Expected %s, got %v", binaryFilename, foundFilename)
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

	foundFilename, err := findBinaryFiles(tmpDir)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
		return
	}
	if len(foundFilename) != 1 || foundFilename[0].fullPath != binaryFilename {
		t.Errorf("Expected %s, got %v", binaryFilename, foundFilename)
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

	foundFilename, err := findBinaryFiles(tmpDir)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
		return
	}
	if len(foundFilename) != 1 || foundFilename[0].fullPath != binaryFilename {
		t.Errorf("Expected %s, got %v", binaryFilename, foundFilename)
	}
}

type archiveEntry struct {
	name     string
	data     []byte
	typeflag byte
	linkname string
}

func writeArchiveFile(t *testing.T, name string, write func(io.Writer) error) string {
	t.Helper()
	filename := filepath.Join(t.TempDir(), name)
	file, err := os.Create(filename)
	if err != nil {
		t.Fatal(err)
	}
	if err := write(file); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return filename
}

func tarWriter(entries ...archiveEntry) func(io.Writer) error {
	return func(w io.Writer) error {
		tw := tar.NewWriter(w)
		for _, entry := range entries {
			hdr := &tar.Header{Name: entry.name, Mode: 0755, Typeflag: entry.typeflag, Linkname: entry.linkname}
			if hdr.Typeflag == 0 {
				hdr.Typeflag = tar.TypeReg
				hdr.Size = int64(len(entry.data))
			}
			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}
			if _, err := tw.Write(entry.data); err != nil {
				return err
			}
		}
		return tw.Close()
	}
}

func zipWriter(entries ...archiveEntry) func(io.Writer) error {
	return func(w io.Writer) error {
		zw := zip.NewWriter(w)
		for _, entry := range entries {
			hdr := &zip.FileHeader{Name: entry.name, Method: zip.Deflate}
			hdr.SetMode(0755)
			data := entry.data
			if entry.typeflag == tar.TypeSymlink {
				hdr.SetMode(fs.ModeSymlink | 0777)
				data = []byte(entry.linkname)
			}
			member, err := zw.CreateHeader(hdr)
			if err != nil {
				return err
			}
			if _, err := member.Write(data); err != nil {
				return err
			}
		}
		return zw.Close()
	}
}

func compressedWriter(compressor archives.Compressor, write func(io.Writer) error) func(io.Writer) error {
	return func(w io.Writer) error {
		cw, err := compressor.OpenWriter(w)
		if err != nil {
			return err
		}
		if err := write(cw); err != nil {
			return err
		}
		return cw.Close()
	}
}

func rawWriter(data []byte) func(io.Writer) error {
	return func(w io.Writer) error {
		_, err := w.Write(data)
		return err
	}
}

func regularFiles(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	files := map[string][]byte{}
	err := filepath.WalkDir(dir, func(name string, entry fs.DirEntry, err error) error {
		if err != nil || !entry.Type().IsRegular() {
			return err
		}
		data, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, name)
		files[filepath.ToSlash(rel)] = data
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func TestUnpackArchiveFormats(t *testing.T) {
	readme := []byte("Readme text\n")
	nested := []archiveEntry{
		{name: "./tool-1.0/", typeflag: tar.TypeDir},
		{name: "./tool-1.0/bin/tool", data: fixtureBinary},
		{name: "./tool-1.0/README.md", data: readme},
	}
	wantNested := map[string][]byte{"tool-1.0/bin/tool": fixtureBinary, "tool-1.0/README.md": readme}
	tests := []struct {
		name  string
		file  string
		write func(io.Writer) error
		want  map[string][]byte
	}{
		{"tar.gz", "tool_linux_amd64.tar.gz", compressedWriter(archives.Gz{}, tarWriter(nested...)), wantNested},
		{"tar.xz", "tool_linux_amd64.tar.xz", compressedWriter(archives.Xz{}, tarWriter(nested...)), wantNested},
		{"zip", "tool_linux_amd64.zip", zipWriter(
			archiveEntry{name: "tool-1.0/bin/tool", data: fixtureBinary},
			archiveEntry{name: "tool-1.0/README.md", data: readme},
		), wantNested},
		{"gz", "tool_linux_amd64.gz", compressedWriter(archives.Gz{}, rawWriter(fixtureBinary)), map[string][]byte{"tool_linux_amd64": fixtureBinary}},
		{"bz2", "tool_linux_amd64.bz2", compressedWriter(archives.Bz2{}, rawWriter(fixtureBinary)), map[string][]byte{"tool_linux_amd64": fixtureBinary}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir, err := unpackArchive(writeArchiveFile(t, test.file, test.write))
			if err != nil {
				t.Fatal(err)
			}
			if got := regularFiles(t, dir); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("got files %q, want %q", got, test.want)
			}
		})
	}
}

func TestUnpackArchiveRejectsEntriesOutsideDirectory(t *testing.T) {
	absolute := filepath.Join(t.TempDir(), "absolute-evil")
	tests := []struct {
		name    string
		file    string
		write   func(io.Writer) error
		outside func(archive string) string
	}{
		{"tar parent", "tool.tar.gz", compressedWriter(archives.Gz{}, tarWriter(archiveEntry{name: "../evil", data: fixtureBinary})),
			func(archive string) string { return filepath.Join(filepath.Dir(archive), "evil") }},
		{"tar nested parent", "tool.tar.gz", compressedWriter(archives.Gz{}, tarWriter(archiveEntry{name: "tool/../../evil", data: fixtureBinary})),
			func(archive string) string { return filepath.Join(filepath.Dir(archive), "evil") }},
		{"tar absolute", "tool.tar.gz", compressedWriter(archives.Gz{}, tarWriter(archiveEntry{name: absolute, data: fixtureBinary})),
			func(string) string { return absolute }},
		{"zip parent", "tool.zip", zipWriter(archiveEntry{name: "../evil", data: fixtureBinary}),
			func(archive string) string { return filepath.Join(filepath.Dir(archive), "evil") }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			archive := writeArchiveFile(t, test.file, test.write)
			_, err := unpackArchive(archive)
			if err == nil || !strings.Contains(err.Error(), "outside the extraction directory") {
				t.Fatalf("expected an outside-directory error, got %v", err)
			}
			if _, err := os.Lstat(test.outside(archive)); !os.IsNotExist(err) {
				t.Fatalf("entry was written outside the extraction directory: %v", err)
			}
		})
	}
}

func TestUnpackArchiveSkipsLinks(t *testing.T) {
	outside := t.TempDir()
	tests := []struct {
		name  string
		file  string
		write func(io.Writer) error
	}{
		{"tar", "tool.tar.gz", compressedWriter(archives.Gz{}, tarWriter(
			archiveEntry{name: "tool", data: fixtureBinary},
			archiveEntry{name: "link", typeflag: tar.TypeSymlink, linkname: outside},
			archiveEntry{name: "link/evil", data: fixtureBinary},
			archiveEntry{name: "hard", typeflag: tar.TypeLink, linkname: "tool"},
		))},
		{"zip", "tool.zip", zipWriter(
			archiveEntry{name: "tool", data: fixtureBinary},
			archiveEntry{name: "link", typeflag: tar.TypeSymlink, linkname: outside},
			archiveEntry{name: "link/evil", data: fixtureBinary},
		)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir, err := unpackArchive(writeArchiveFile(t, test.file, test.write))
			if err != nil {
				t.Fatal(err)
			}
			if info, err := os.Lstat(filepath.Join(dir, "link")); err != nil || info.Mode()&fs.ModeSymlink != 0 {
				t.Fatalf("link entry was not skipped: %v %v", info, err)
			}
			if _, err := os.Lstat(filepath.Join(dir, "hard")); !os.IsNotExist(err) {
				t.Fatalf("hardlink entry was not skipped: %v", err)
			}
			if _, err := os.Lstat(filepath.Join(outside, "evil")); !os.IsNotExist(err) {
				t.Fatalf("entry was written through a link: %v", err)
			}
			want := map[string][]byte{"tool": fixtureBinary, "link/evil": fixtureBinary}
			if got := regularFiles(t, dir); !reflect.DeepEqual(got, want) {
				t.Fatalf("got files %q, want %q", got, want)
			}
		})
	}
}

func TestUnpackArchiveUnsupportedFormat(t *testing.T) {
	_, err := unpackArchive(writeArchiveFile(t, "notes.txt", rawWriter([]byte("plain text\n"))))
	if err == nil || !strings.Contains(err.Error(), "unsupported archive format") {
		t.Fatalf("expected unsupported format error, got %v", err)
	}
}

func TestPrepareAssetFromTarGzKeepsMemberPaths(t *testing.T) {
	withYes(t, true)
	archive := writeArchiveFile(t, "tool_linux_amd64.tar.gz", compressedWriter(archives.Gz{}, tarWriter(
		archiveEntry{name: "./tool-1.0/", typeflag: tar.TypeDir},
		archiveEntry{name: "./tool-1.0/tool", data: fixtureBinary},
		archiveEntry{name: "./tool-1.0/helper", data: fixtureBinary},
		archiveEntry{name: "./tool-1.0/README.md", data: []byte("Readme text\n")},
	)))
	destination := t.TempDir()
	plan, selection, err := prepareAsset(archive, filepath.Base(archive), "v1.0", destination, AssetSelection{Pattern: filepath.Base(archive)}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	var sources, paths []string
	for _, item := range plan {
		sources = append(sources, item.component.Source)
	}
	for _, binary := range selection.Binaries {
		paths = append(paths, binary.Path)
	}
	if want := []string{"tool-1.0/helper", "tool-1.0/tool"}; !reflect.DeepEqual(sources, want) {
		t.Fatalf("got sources %q, want %q", sources, want)
	}
	if want := []string{"tool-*/helper", "tool-*/tool"}; !reflect.DeepEqual(paths, want) {
		t.Fatalf("got remembered paths %q, want %q", paths, want)
	}
}
