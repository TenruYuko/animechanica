package manga_providers

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
	"io/ioutil"
	"strings"

	hibikemanga "seanime/internal/extension/hibike/manga"
	"github.com/stretchr/testify/assert"
)

func createTestCBZ(t *testing.T, dir string, name string, files map[string][]byte) string {
	cbzPath := filepath.Join(dir, name)
	f, err := os.Create(cbzPath)
	if err != nil {
		t.Fatalf("failed to create cbz: %v", err)
	}
	w := zip.NewWriter(f)
	for fname, data := range files {
		fw, err := w.Create(fname)
		if err != nil {
			t.Fatalf("failed to add file to cbz: %v", err)
		}
		_, err = fw.Write(data)
		if err != nil {
			t.Fatalf("failed to write file data: %v", err)
		}
	}
	w.Close()
	f.Close()
	return cbzPath
}

func TestInternalStorageProvider_FindChapterPages(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "cbztest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a CBZ with images in subfolders and non-ASCII names
	cbzName := "Test 漫画.cbz"
	files := map[string][]byte{
		"001.jpg": []byte("fakeimg1"),
		"subfolder/002.png": []byte("fakeimg2"),
		"サブ/003.jpeg": []byte("fakeimg3"),
		"004.txt": []byte("notimg"),
	}
	cbzPath := createTestCBZ(t, tmpDir, cbzName, files)

	provider := &InternalStorageProvider{baseDir: tmpDir}
	pages, err := provider.FindChapterPages(cbzName)
	assert.NoError(t, err)
	assert.Len(t, pages, 3)

	var names []string
	for _, p := range pages {
		names = append(names, p.URL)
	}
	joined := strings.Join(names, ",")
	assert.Contains(t, joined, "001.jpg")
	assert.Contains(t, joined, "subfolder/002.png")
	assert.Contains(t, joined, "サブ/003.jpeg")
}
