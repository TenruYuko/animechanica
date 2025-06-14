package manga_providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	hibikemanga "seanime/internal/extension/hibike/manga"
	"github.com/rs/zerolog"
	"github.com/adrg/strutil/metrics"
)

// Internal manga provider implementation

// mangaMeta holds cached metadata for a manga directory
// Used for efficient fuzzy search
type mangaMeta struct {
	dirName     string
	romajiTitle string
	matchTarget string // lowercased romajiTitle or dirName
}

// InternalStorageProvider implements a provider that reads from internal storage/downloads
type InternalStorageProvider struct {
	logger         *zerolog.Logger
	baseDir        string

	mangaCache     []mangaMeta
	mangaCacheInit bool
	mangaScanTotal int
	mangaScanDone  int
	mangaScanMutex sync.Mutex
}

// NewInternalStorage creates a new internal storage manga provider
func NewInternalStorage(ctx context.Context, logger *zerolog.Logger, baseDir string) hibikemanga.Provider {
	provider := &InternalStorageProvider{
		logger:  logger,
		baseDir: baseDir,
	}

	// Build manga cache at startup, but do it in a goroutine for progress tracking
	dirs, err := os.ReadDir(baseDir)
	if err == nil {
		provider.mangaScanTotal = len(dirs)
		provider.mangaScanDone = 0
		go func() {
			defer func() {
				if r := recover(); r != nil {
					logger.Error().Msgf("Panic during manga scan: %v", r)
					logger.Error().Msg("Manga scan stopped due to panic. Please check the previous log for details.")
				}
			}()
			cache := make([]mangaMeta, 0, len(dirs))
			var stoppedEarly bool
			var stoppedReason string
			for i, entry := range dirs {
				select {
				case <-ctx.Done():
					logger.Warn().Msg("Manga scan cancelled by context, stopping early.")
					stoppedEarly = true
					stoppedReason = "context cancelled"
					break
				default:
				}
				if stoppedEarly {
					logger.Error().Msgf("Manga scan stopped early at %d/%d: %s", i, len(dirs), stoppedReason)
					break
				}
				if entry.IsDir() {
					mangaName := entry.Name()
					mangaDir := filepath.Join(baseDir, mangaName)
					if (i+1)%1000 == 0 || i == len(dirs)-1 {
						logger.Info().Msgf("Scanning progress: %s/%s (currently: %s)", commafy(i+1), commafy(len(dirs)), mangaName)
					}
					logger.Debug().Msgf("Scanning manga directory: %s", mangaDir)
					romajiTitle := ""
					infoPath := filepath.Join(mangaDir, "info.json")
					// Add timeout for reading/parsing info.json
					done := make(chan struct{})
					go func() {
						defer func() {
							if r := recover(); r != nil {
								logger.Warn().Msgf("Panic reading info.json for %s: %v", mangaDir, r)
								stoppedEarly = true
								stoppedReason = "panic reading info.json for " + mangaDir
								close(done)
							}
						}()
						if data, err := os.ReadFile(infoPath); err == nil {
							var meta struct {
								Romaji string `json:"romaji"`
							}
							if err := json.Unmarshal(data, &meta); err == nil && meta.Romaji != "" {
								romajiTitle = meta.Romaji
							} else if err != nil {
								logger.Warn().Err(err).Msgf("Failed to parse info.json for %s", mangaDir)
							}
						} else if !os.IsNotExist(err) {
							logger.Warn().Err(err).Msgf("Failed to read info.json for %s", mangaDir)
						}
						close(done)
					}()
					select {
					case <-done:
						// finished in time
					case <-time.After(5 * time.Second):
						logger.Warn().Msgf("Timeout reading/parsing info.json for %s, skipping directory", mangaDir)
						continue
					case <-ctx.Done():
						logger.Warn().Msgf("Manga scan cancelled by context during info.json read for %s", mangaDir)
						stoppedEarly = true
						stoppedReason = "context cancelled during info.json read"
						break
					}
					if stoppedEarly {
						break
					}
					matchTarget := strings.ToLower(romajiTitle)
					if matchTarget == "" {
						matchTarget = strings.ToLower(mangaName)
					}
					cache = append(cache, mangaMeta{
						dirName:     mangaName,
						romajiTitle: romajiTitle,
						matchTarget: matchTarget,
					})
				}
				// Update progress every 100 directories, and always on the last one
				if (i+1)%100 == 0 || i == len(dirs)-1 {
					provider.mangaScanMutex.Lock()
					provider.mangaScanDone = i + 1
					provider.mangaScanMutex.Unlock()
				}
			}
			if stoppedEarly {
				logger.Error().Msgf("Manga scan did not complete. Last processed: %d/%d. Reason: %s", provider.mangaScanDone, len(dirs), stoppedReason)
			} else {
				provider.mangaCache = cache
				provider.mangaCacheInit = true
				logger.Info().Msg("Internal Storage Scanned! You may now use downloads!")
			}
		}()
	} else {
		logger.Error().Err(err).Msg("internal-storage: Failed to scan manga directory at startup")
	}

	return provider
}

// BuildCacheSync builds the manga cache synchronously (for startup)
func (p *InternalStorageProvider) BuildCacheSync() {
	dirs, err := os.ReadDir(p.baseDir)
	if (err != nil) {
		p.logger.Error().Err(err).Msg("internal-storage: Failed to scan manga directory at startup (sync)")
		return
	}
	p.mangaScanTotal = len(dirs)
	p.mangaScanDone = 0
	cache := make([]mangaMeta, 0, len(dirs))
	var stoppedEarly bool
	var stoppedReason string
	for i, entry := range dirs {
		if stoppedEarly {
			p.logger.Error().Msgf("Manga scan stopped early at %d/%d: %s", i, len(dirs), stoppedReason)
			break
		}
		if entry.IsDir() {
			mangaName := entry.Name()
			mangaDir := filepath.Join(p.baseDir, mangaName)
			if (i+1)%1000 == 0 || i == len(dirs)-1 {
				p.logger.Info().Msgf("Scanning progress: %s/%s (currently: %s)", commafy(i+1), commafy(len(dirs)), mangaName)
			}
			p.logger.Debug().Msgf("Scanning manga directory: %s", mangaDir)
			romajiTitle := ""
			infoPath := filepath.Join(mangaDir, "info.json")
			// Add timeout for reading/parsing info.json
			done := make(chan struct{})
			go func() {
				defer func() {
					if r := recover(); r != nil {
						p.logger.Warn().Msgf("Panic reading info.json for %s: %v", mangaDir, r)
						stoppedEarly = true
						stoppedReason = "panic reading info.json for " + mangaDir
						close(done)
					}
				}()
				if data, err := os.ReadFile(infoPath); err == nil {
					var meta struct {
						Romaji string `json:"romaji"`
					}
					if err := json.Unmarshal(data, &meta); err == nil && meta.Romaji != "" {
						romajiTitle = meta.Romaji
					} else if err != nil {
						p.logger.Warn().Err(err).Msgf("Failed to parse info.json for %s", mangaDir)
					}
				} else if !os.IsNotExist(err) {
					p.logger.Warn().Err(err).Msgf("Failed to read info.json for %s", mangaDir)
				}
				close(done)
			}()
			select {
			case <-done:
				// finished in time
			case <-time.After(5 * time.Second):
				p.logger.Warn().Msgf("Timeout reading/parsing info.json for %s, skipping directory", mangaDir)
				continue
			}
			if stoppedEarly {
				break
			}
			matchTarget := strings.ToLower(romajiTitle)
			if matchTarget == "" {
				matchTarget = strings.ToLower(mangaName)
			}
			cache = append(cache, mangaMeta{
				dirName:     mangaName,
				romajiTitle: romajiTitle,
				matchTarget: matchTarget,
			})
		}
		// Update progress every 100 directories, and always on the last one
		if (i+1)%100 == 0 || i == len(dirs)-1 {
			p.mangaScanMutex.Lock()
			p.mangaScanDone = i + 1
			p.mangaScanMutex.Unlock()
		}
	}
	if stoppedEarly {
		p.logger.Error().Msgf("Manga scan did not complete. Last processed: %d/%d. Reason: %s", p.mangaScanDone, len(dirs), stoppedReason)
	} else {
		p.mangaCache = cache
		p.mangaCacheInit = true
		p.logger.Info().Msg("Internal Storage Scanned! You may now use downloads!")
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

	if !p.mangaCacheInit {
		p.mangaScanMutex.Lock()
		done := p.mangaScanDone
		total := p.mangaScanTotal
		p.mangaScanMutex.Unlock()
		msg := "Still scanning.. Please wait"
		if total > 0 {
			msg = msg + " " + commafy(done) + "/" + commafy(total)
		}
		return nil, fmt.Errorf(msg)
	}

	query := strings.ToLower(opts.Query)
	dice := metrics.NewSorensenDice()
	dice.CaseSensitive = false

	results := make([]*hibikemanga.SearchResult, 0)
	debugMatches := make([]struct {
		dirName     string
		romajiTitle string
		matchTarget string
		score       float64
	}, 0, len(p.mangaCache))

	// Create timeout context for search operation
	searchCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Parallel similarity calculation with timeout
	type scoredResult struct {
		idx   int
		score float64
	}
	scored := make([]scoredResult, len(p.mangaCache))
	var wg sync.WaitGroup
	searchDone := make(chan struct{})
	
	go func() {
		for i, meta := range p.mangaCache {
			wg.Add(1)
			go func(i int, meta mangaMeta) {
				defer wg.Done()
				score := 1.0
				if query != "" {
					score = dice.Compare(query, meta.matchTarget)
				}
				scored[i] = scoredResult{idx: i, score: score}
			}(i, meta)
		}
		wg.Wait()
		close(searchDone)
	}()

	// Wait for search completion or timeout
	select {
	case <-searchDone:
		// Search completed successfully
	case <-searchCtx.Done():
		p.logger.Warn().Msgf("Manga search timed out after 5 seconds for query: %s", opts.Query)
		return nil, fmt.Errorf("Search timed out after 5 seconds. Please try a more specific search term")
	}

	for i, meta := range p.mangaCache {
		score := scored[i].score
		debugMatches = append(debugMatches, struct {
			dirName     string
			romajiTitle string
			matchTarget string
			score       float64
		}{meta.dirName, meta.romajiTitle, meta.matchTarget, score})
		if query != "" && score < 0.85 {
			continue
		}
		title := meta.dirName
		if meta.romajiTitle != "" {
			title = meta.romajiTitle
		}
		results = append(results, &hibikemanga.SearchResult{
			ID:    meta.dirName,
			Title: title,
			Image: "",
			Year:  0,
		})
	}

	// Debug output: log all potential matches and their scores
	for _, m := range debugMatches {
		p.logger.Debug().Msgf("internal-storage: Potential match: dir='%s' romaji='%s' matchTarget='%s' score=%.3f", m.dirName, m.romajiTitle, m.matchTarget, m.score)
	}

	// Check if no results found for non-empty query
	if len(results) == 0 && query != "" {
		p.logger.Info().Msgf("No manga found for query: %s", opts.Query)
		return nil, fmt.Errorf("Cannot find manga matching '%s'. Please check your search term or try a different title", opts.Query)
	}

	return results, nil
}

// FindChapters implements chapter finding for internal storage
func (p *InternalStorageProvider) FindChapters(mangaID string) ([]*hibikemanga.ChapterDetails, error) {
	mangaPath := filepath.Join(p.baseDir, mangaID)
	entries, err := os.ReadDir(mangaPath)
	if err != nil {
		return nil, err
	}

	chapters := make([]*hibikemanga.ChapterDetails, 0)
	volNum := 1
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".cbz") {
			chapterID := entry.Name()
			chapter := &hibikemanga.ChapterDetails{
				ID:       mangaID + "/" + chapterID, // Always use '/' as separator
				Title:    "Volume " + strconv.Itoa(volNum),
				Chapter:  "Volume " + strconv.Itoa(volNum), // Use 'Volume' label for internal storage
				Index:    uint(volNum - 1),
				Provider: "internal-storage",
			}
			chapters = append(chapters, chapter)
			volNum++
		}
	}
	return chapters, nil
}

// FindChapterPages - Internal storage manga reading is disabled
func (p *InternalStorageProvider) FindChapterPages(chapterID string) ([]*hibikemanga.ChapterPage, error) {
	// Internal storage manga reading has been disabled
	// The internal storage provider remains available as a source but pages cannot be read
	p.logger.Debug().Msgf("FindChapterPages: Internal storage manga reading is disabled for chapterID: %s", chapterID)
	
	// Return an error to indicate that page reading is not available
	return nil, fmt.Errorf("internal-storage: manga reading functionality has been disabled")
}

// GetPage - Internal storage manga reading is disabled
// CONTRACT: This function expects 'file' and 'page' parameters to be passed as exact, literal strings (not percent-encoded, not decoded, not altered in any way).
// All symbols, including UTF-8/unicode and special characters, are preserved and used as-is.
// The HTTP/router/proxy/frontend layer MUST pass the exact symbols as they appear in the filesystem and CBZ, with no encoding or decoding.
func (p *InternalStorageProvider) GetPage(file string, page string) (io.ReadCloser, error) {
	// Internal storage manga reading has been disabled
	// The internal storage provider remains available as a source but pages cannot be read
	p.logger.Debug().Msgf("GetPage: Internal storage manga reading is disabled for file: %s, page: %s", file, page)
	
	// Return an error to indicate that page reading is not available
	return nil, fmt.Errorf("internal-storage: manga reading functionality has been disabled")
}

// commafy formats an integer with commas (e.g., 12345 -> "12,345")
func commafy(n int) string {
	s := strconv.Itoa(n)
	if len(s) <= 3 {
		return s
	}
	var out strings.Builder
	pre := len(s) % 3
	if pre > 0 {
		out.WriteString(s[:pre])
		if len(s) > pre {
			out.WriteByte(',')
		}
	}
	for i := pre; i < len(s); i += 3 {
		out.WriteString(s[i : i+3])
		if i+3 < len(s) {
			out.WriteByte(',')
		}
	}
	return out.String()
}