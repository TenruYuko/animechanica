package manga_providers

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	hibikemanga "seanime/internal/extension/hibike/manga"
	"slices"
	"github.com/adrg/strutil/metrics"
)

var (
	ErrNoVolumes = fmt.Errorf("no volumes found")
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

// Search implements manga search - for local storage we just return all manga, but with fuzzy matching on romaji title
func (p *LocalStorageProvider) Search(opts hibikemanga.SearchOptions) ([]*hibikemanga.SearchResult, error) {
	results, err := p.GetMangaList()
	if err != nil {
		return nil, err
	}

	if opts.Query != "" {
		filtered := make([]*hibikemanga.SearchResult, 0)
		bestScore := 0.0
		var bestResult *hibikemanga.SearchResult
		for _, manga := range results {
			// Fuzzy match using Jaro-Winkler
			score := metrics.NewJaroWinkler().Compare(strings.ToLower(manga.Title), strings.ToLower(opts.Query))
			if score >= 0.7 {
				manga.SearchRating = score
				filtered = append(filtered, manga)
				if score > bestScore {
					bestScore = score
					bestResult = manga
				}
			}
		}
		// If we have a bestResult, return only that as the best match
		if bestResult != nil {
			return []*hibikemanga.SearchResult{bestResult}, nil
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

// FindVolumes returns a list of volumes for a manga, with padded numbers and cover art from first page
func (p *LocalStorageProvider) FindVolumes(mangaID string) ([]*hibikemanga.VolumeDetails, error) {
	mangaPath := filepath.Join(p.baseDir, mangaID)
	if _, err := os.Stat(mangaPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("manga directory not found: %s", mangaID)
	}

	volumeMap := make(map[float64]*hibikemanga.VolumeDetails)
	volumeCovers := make(map[float64]string)

	err := filepath.Walk(mangaPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(info.Name()), ".cbz") {
			return nil
		}
		var volumeNum float64 = 0
		var chapterNum float64
		name := strings.TrimSuffix(info.Name(), ".cbz")
		// Extract volume number
		if strings.Contains(strings.ToLower(name), "vol") {
			volParts := strings.Split(strings.ToLower(name), "vol")
			if len(volParts) > 1 {
				volStr := strings.TrimSpace(strings.Split(volParts[1], " ")[0])
				volStr = strings.TrimPrefix(volStr, ".")
				if v, err := strconv.ParseFloat(volStr, 64); err == nil {
					volumeNum = v
				}
			}
		}
		// Extract chapter number
		chParts := strings.Split(strings.ToLower(name), "ch")
		if len(chParts) > 1 {
			chStr := strings.TrimSpace(strings.Split(chParts[1], " ")[0])
			chStr = strings.TrimPrefix(chStr, ".")
			if ch, err := strconv.ParseFloat(chStr, 64); err == nil {
				chapterNum = ch
			}
		} else {
			for _, part := range strings.Fields(name) {
				if num, err := strconv.ParseFloat(part, 64); err == nil {
					chapterNum = num
					break
				}
			}
		}
		// Create or update volume
		vol, exists := volumeMap[volumeNum]
		if !exists {
			vol = &hibikemanga.VolumeDetails{
				Provider:   "local-storage",
				ID:         fmt.Sprintf("%03.0f", volumeNum), // Padded
				Number:     fmt.Sprintf("%03.0f", volumeNum), // Padded
				Title:      fmt.Sprintf("Volume %03.0f", volumeNum),
				Chapters:   make([]*hibikemanga.ChapterDetails, 0),
			}
			volumeMap[volumeNum] = vol
			// Extract cover from first page of first chapter in volume
			if _, exists := volumeCovers[volumeNum]; !exists {
				if r, err := zip.OpenReader(path); err == nil {
					defer r.Close()
					for _, f := range r.File {
						if isImageFile(f.Name) {
							volumeCovers[volumeNum] = fmt.Sprintf("file://%s#%s", path, f.Name)
							vol.CoverImage = volumeCovers[volumeNum]
							break
						}
					}
				}
			}
		}
		// Add chapter to volume
		chapter := &hibikemanga.ChapterDetails{
			ID:       filepath.Join(mangaID, info.Name()),
			Chapter:  p.normalizeChapterNumber(chapterNum),
			Title:    fmt.Sprintf("Chapter %s", p.normalizeChapterNumber(chapterNum)),
			Language: "en",
			Provider: "local-storage",
		}
		vol.Chapters = append(vol.Chapters, chapter)
		return nil
	})
	if err != nil {
		return nil, err
	}
	volumes := make([]*hibikemanga.VolumeDetails, 0, len(volumeMap))
	for _, vol := range volumeMap {
		// Sort chapters within volume by chapter number (numeric sort)
		slices.SortFunc(vol.Chapters, func(a, b *hibikemanga.ChapterDetails) int {
			numA, _ := strconv.ParseFloat(a.Chapter, 64)
			numB, _ := strconv.ParseFloat(b.Chapter, 64)
			if numA < numB {
				return -1
			}
			if numA > numB {
				return 1
			}
			return 0
		})
		volumes = append(volumes, vol)
	}
	// Sort volumes by padded number
	slices.SortFunc(volumes, func(a, b *hibikemanga.VolumeDetails) int {
		numA, _ := strconv.ParseFloat(a.Number, 64)
		numB, _ := strconv.ParseFloat(b.Number, 64)
		if numA < numB {
			return -1
		}
		if numA > numB {
			return 1
		}
		return 0
	})
	if len(volumes) == 0 {
		return nil, ErrNoVolumes
	}
	return volumes, nil
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

			_, chNumStr := p.parseChapterFromFilename(name)
			chNum, _ := strconv.ParseFloat(chNumStr, 64)
			chapter := &hibikemanga.ChapterDetails{
				ID:       chapterID,
				Title:    fmt.Sprintf("Chapter %s", p.normalizeChapterNumber(chNum)),
				Chapter:  p.normalizeChapterNumber(chNum),
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

// FindChapterPages implements the Provider interface for retrieving chapter pages
func (p *LocalStorageProvider) FindChapterPages(id string) ([]*hibikemanga.ChapterPage, error) {
	// Parse manga and chapter from ID (format: mangaID/chapter.cbz)
	parts := strings.Split(id, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid chapter ID format: %s", id)
	}
	mangaID, chapterFile := parts[0], parts[1]

	// Open the CBZ file
	filePath := filepath.Join(p.baseDir, mangaID, chapterFile)
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	// Extract all image files
	pages := make([]*hibikemanga.ChapterPage, 0)
	for idx, f := range r.File {
		if isImageFile(f.Name) {
			// Serve via HTTP endpoint for Internal
			pageURL := fmt.Sprintf("/api/v1/manga/local/page?mangaId=%s&chapterId=%s&pagePath=%s", mangaID, chapterFile, f.Name)
			pages = append(pages, &hibikemanga.ChapterPage{
				URL:      pageURL,
				Index:    idx,
				Provider: "local-storage",
			})
		}
	}

	// Sort pages by filename to maintain order
	sort.SliceStable(pages, func(i, j int) bool {
		nameI := filepath.Base(pages[i].URL)
		nameJ := filepath.Base(pages[j].URL)
		return nameI < nameJ
	})

	if len(pages) == 0 {
		return nil, ErrNoPages
	}

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

// normalizeChapterNumber converts a chapter number to a standardized string format
func (p *LocalStorageProvider) normalizeChapterNumber(num float64) string {
	// Format with one decimal place, but trim trailing .0
	str := fmt.Sprintf("%.1f", num)
	if strings.HasSuffix(str, ".0") {
		str = strings.TrimSuffix(str, ".0")
	}
	return str
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