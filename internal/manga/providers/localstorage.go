package manga_providers

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	hibikemanga "seanime/internal/extension/hibike/manga"
)

// LocalStorageProvider implements a manga provider that reads from local storage
type LocalStorageProvider struct {
	baseDir string
}

// NewLocalStorageProvider creates a new local storage manga provider
func NewLocalStorageProvider(baseDir string) hibikemanga.Provider {
	return &LocalStorageProvider{
		baseDir: baseDir,
	}
}

// GetSettings returns the provider settings
func (p *LocalStorageProvider) GetSettings() hibikemanga.Settings {
	return hibikemanga.Settings{
		SupportsMultiScanlator: false,
		SupportsMultiLanguage:  false,
	}
}

// Search implements manga search - for local storage we just return all manga
func (p *LocalStorageProvider) Search(opts hibikemanga.SearchOptions) ([]*hibikemanga.SearchResult, error) {
	results, err := p.GetMangaList()
	if err != nil {
		return nil, err
	}

	// Filter by search query if provided
	if opts.Query != "" {
		filtered := make([]*hibikemanga.SearchResult, 0)
		for _, manga := range results {
			if strings.Contains(strings.ToLower(manga.Title), strings.ToLower(opts.Query)) {
				filtered = append(filtered, manga)
			}
		}
		return filtered, nil
	}

	return results, nil
}

// GetMangaList returns a list of manga from local storage
func (p *LocalStorageProvider) GetMangaList() ([]*hibikemanga.SearchResult, error) {
	// Walk the base directory to find manga
	entries, err := os.ReadDir(p.baseDir)
	if err != nil {
		return nil, err
	}

	results := make([]*hibikemanga.SearchResult, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			// Each directory is a manga
			mangaID := entry.Name()
			title := mangaID // Use directory name as title for now

			// Create manga result
			manga := &hibikemanga.SearchResult{
				ID:       mangaID,
				Title:    title,
				Provider: "local-storage",
			}

			results = append(results, manga)
		}
	}

	return results, nil
}

// FindChapters lists chapters for a specific manga
func (p *LocalStorageProvider) FindChapters(mangaID string) ([]*hibikemanga.ChapterDetails, error) {
	// Path to manga directory
	mangaPath := filepath.Join(p.baseDir, mangaID)
	entries, err := os.ReadDir(mangaPath)
	if err != nil {
		return nil, err
	}

	chapters := make([]*hibikemanga.ChapterDetails, 0)
	// Each .cbz file is a chapter
	for idx, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".cbz") {
			name := strings.TrimSuffix(entry.Name(), ".cbz")
			chapterID := entry.Name()

			_, chapterNum := p.parseChapterFromFilename(name)

			chapter := &hibikemanga.ChapterDetails{
				ID:       chapterID,
				Title:    name,
				Chapter:  chapterNum,
				Index:    uint(idx),
				Provider: "local-storage",
			}
			chapters = append(chapters, chapter)
		}
	}

	// Sort chapters by their numeric value
	sort.Slice(chapters, func(i, j int) bool {
		numI, _ := strconv.ParseFloat(chapters[i].Chapter, 64)
		numJ, _ := strconv.ParseFloat(chapters[j].Chapter, 64)
		return numI < numJ
	})

	return chapters, nil
}

// FindVolumes lists volumes for a specific manga
func (p *LocalStorageProvider) FindVolumes(id string) ([]*hibikemanga.VolumeDetails, error) {
	// For local storage, we'll treat all chapters as part of a single volume
	// or group them by volume if the filename contains volume information
	chapters, err := p.FindChapters(id)
	if err != nil {
		return nil, err
	}

	// For simplicity, put all chapters in a single volume
	// This can be enhanced later to parse volume information from filenames
	volume := &hibikemanga.VolumeDetails{
		ID:       "vol-1",
		Number:   "1",
		Title:    "Volume 1",
		Provider: "local-storage",
		Chapters: chapters,
	}

	return []*hibikemanga.VolumeDetails{volume}, nil
}

// FindChapterPages implements the Provider interface for retrieving chapter pages
func (p *LocalStorageProvider) FindChapterPages(id string) ([]*hibikemanga.ChapterPage, error) {
	// Assume id is in format mangaID/chapterID
	parts := strings.Split(id, "/")
	if len(parts) != 2 {
		return nil, os.ErrInvalid
	}

	mangaID, chapterID := parts[0], parts[1]

	// Open and read the CBZ file
	filePath := filepath.Join(p.baseDir, mangaID, chapterID)
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	// Get and sort all image files
	pages := make([]*hibikemanga.ChapterPage, 0)
	for idx, f := range r.File {
		if isImageFile(f.Name) {
			pages = append(pages, &hibikemanga.ChapterPage{
				URL:      filepath.Join("/media/manga", mangaID, chapterID, f.Name),
				Index:    idx,
				Provider: "local-storage",
			})
		}
	}

	// Sort pages by name to maintain order
	sort.SliceStable(pages, func(i, j int) bool {
		return filepath.Base(pages[i].URL) < filepath.Base(pages[j].URL)
	})

	return pages, nil
}

// GetPage extracts and returns a specific page from a CBZ file
func (p *LocalStorageProvider) GetPage(mangaID string, chapterID string, pagePath string) (io.ReadCloser, error) {
	filePath := filepath.Join(p.baseDir, mangaID, chapterID)

	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, err
	}

	// Find the specific page in the zip
	pageName := filepath.Base(pagePath)
	for _, f := range r.File {
		if filepath.Base(f.Name) == pageName {
			return f.Open()
		}
	}

	r.Close()
	return nil, os.ErrNotExist
}

// parseChapterFromFilename tries to extract chapter number from filename
func (p *LocalStorageProvider) parseChapterFromFilename(filename string) (title string, chapterNum string) {
	// Try to extract chapter number from filename
	// Common formats: Chapter_001.cbz, Ch.1.cbz, Ch 1.cbz, etc.
	parts := strings.Fields(strings.Map(func(r rune) rune {
		switch r {
		case '_', '.', '-':
			return ' '
		default:
			return r
		}
	}, filename))

	for i, part := range parts {
		if strings.HasPrefix(strings.ToLower(part), "ch") || strings.HasPrefix(strings.ToLower(part), "chapter") {
			if i+1 < len(parts) {
				if num, err := strconv.ParseFloat(parts[i+1], 64); err == nil {
					return strings.Join(parts[:i], " "), strconv.FormatFloat(num, 'f', -1, 64)
				}
			}
		}
		if num, err := strconv.ParseFloat(part, 64); err == nil {
			return strings.Join(parts[:i], " "), strconv.FormatFloat(num, 'f', -1, 64)
		}
	}
	return filename, "0"
}

// isImageFile checks if a filename represents an image
func isImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return true
	default:
		return false
	}
}