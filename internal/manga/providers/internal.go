package manga_providers

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"

	hibikemanga "seanime/internal/extension/hibike/manga"
	"github.com/rs/zerolog"
)

// zipFileReadCloser is a helper struct that closes both the file and zip reader
type zipFileReadCloser struct {
	rc        io.ReadCloser
	zipReader *zip.ReadCloser
}

func (z *zipFileReadCloser) Read(p []byte) (n int, err error) {
	return z.rc.Read(p)
}

func (z *zipFileReadCloser) Close() error {
	z.rc.Close()
	return z.zipReader.Close()
}

// Internal manga provider implementation

// InternalStorageProvider implements a provider that reads from internal storage/downloads
type InternalStorageProvider struct {
	logger  *zerolog.Logger
	baseDir string
}

// NewInternalStorage creates a new internal storage manga provider
func NewInternalStorage(logger *zerolog.Logger, baseDir string) hibikemanga.Provider {
	return &InternalStorageProvider{
		logger:  logger,
		baseDir: baseDir,
	}
}

// GetSettings returns the provider settings
func (p *InternalStorageProvider) GetSettings() hibikemanga.Settings {
	return hibikemanga.Settings{
		SupportsMultiScanlator: false,
		SupportsMultiLanguage:  false,
	}
}

// Search implements manga search for internal storage
func (p *InternalStorageProvider) Search(opts hibikemanga.SearchOptions) ([]*hibikemanga.SearchResult, error) {
	if p.baseDir == "" {
		return []*hibikemanga.SearchResult{}, nil
	}

	// Check if base directory exists
	if _, err := os.Stat(p.baseDir); os.IsNotExist(err) {
		return []*hibikemanga.SearchResult{}, nil
	}

	results := make([]*hibikemanga.SearchResult, 0)

	// Walk through the directory to find manga folders
	err := filepath.Walk(p.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking even if there's an error
		}

		// Only process directories at the first level under baseDir
		if info.IsDir() && path != p.baseDir {
			relPath, _ := filepath.Rel(p.baseDir, path)
			if !strings.Contains(relPath, string(filepath.Separator)) {
				// This is a top-level directory, treat it as a manga
				mangaName := info.Name()
				
				// Filter by search query if provided
				if opts.Query != "" && !strings.Contains(strings.ToLower(mangaName), strings.ToLower(opts.Query)) {
					return nil
				}

				results = append(results, &hibikemanga.SearchResult{
					ID:    relPath,
					Title: mangaName,
					Image: "",
					Year:  0,
				})
			}
		}

		return nil
	})

	if err != nil {
		p.logger.Error().Err(err).Msg("manga: Error walking internal storage directory")
		return []*hibikemanga.SearchResult{}, err
	}

	return results, nil
}

// FindChapters implements chapter finding for internal storage
func (p *InternalStorageProvider) FindChapters(mangaID string) ([]*hibikemanga.ChapterDetails, error) {
	return []*hibikemanga.ChapterDetails{}, nil
}

// FindChapterPages implements page finding for internal storage
func (p *InternalStorageProvider) FindChapterPages(chapterID string) ([]*hibikemanga.ChapterPage, error) {
	return []*hibikemanga.ChapterPage{}, nil
}

// GetPage extracts and returns a specific page from a CBZ file using the internal storage format
func (p *InternalStorageProvider) GetPage(file string, page string) (io.ReadCloser, error) {
	// For internal storage, the file parameter is the relative path to the CBZ file
	cbzPath := filepath.Join(p.baseDir, file)
	
	// Open the CBZ file as a ZIP archive
	r, err := zip.OpenReader(cbzPath)
	if err != nil {
		return nil, err
	}
	
	// Find the specific page in the zip
	pageName := filepath.Base(page)
	for _, f := range r.File {
		if filepath.Base(f.Name) == pageName {
			rc, err := f.Open()
			if err != nil {
				r.Close()
				return nil, err
			}
			// Return a ReadCloser that also closes the zip reader
			return &zipFileReadCloser{rc: rc, zipReader: r}, nil
		}
	}
	
	r.Close()
	return nil, os.ErrNotExist
}