package manga

import (
	"cmp"
	"errors"
	"fmt"
	"math"
	"os"
	"seanime/internal/api/anilist"
	"seanime/internal/extension"
	hibikemanga "seanime/internal/extension/hibike/manga"
	"seanime/internal/hook"
	"seanime/internal/util"
	"seanime/internal/util/comparison"
	"seanime/internal/util/result"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"github.com/adrg/strutil/metrics"
	"github.com/samber/lo"
)

// Cache bucket types for manga
const (
	bucketTypeChapter = "chapter"
	bucketTypeVolume  = "volume"
)

// Common errors
var (
	ErrNoChapters = errors.New("no chapters found")
	ErrNoVolumes  = errors.New("no volumes found")
)

type (
	// ChapterContainer is used to display the list of chapters from a provider in the client.
	// It is cached in a unique file cache bucket with a key of the format: {provider}${mediaId}
	ChapterContainer struct {
		MediaId  int                             `json:"mediaId"`
		Provider string                         `json:"provider"`
		Chapters []*hibikemanga.ChapterDetails `json:"chapters"`
	}

	// VolumeContainer is used to display the list of volumes from a provider in the client.
	// It is cached in a unique file cache bucket with a key of the format: {provider}${mediaId}
	VolumeContainer struct {
		MediaId  int                          `json:"mediaId"`
		Provider string                       `json:"provider"`
		Volumes []*hibikemanga.VolumeDetails `json:"volumes"`
	}
)

func getMangaChapterContainerCacheKey(provider string, mediaId int) string {
	return fmt.Sprintf("%s$%d", provider, mediaId)
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type GetMangaChapterContainerOptions struct {
	Provider string
	MediaId  int
	Titles   []*string
	Year     int
}

// GetMangaChapterContainer returns the ChapterContainer for a manga entry based on the provider.
// If it isn't cached, it will search for the manga, create a ChapterContainer and cache it.
func (r *Repository) GetMangaChapterContainer(opts *GetMangaChapterContainerOptions) (ret *ChapterContainer, err error) {
	defer util.HandlePanicInModuleWithError("manga/GetMangaChapterContainer", &err)

	provider := opts.Provider
	mediaId := opts.MediaId
	titles := opts.Titles

	r.logger.Trace().
		Str("provider", provider).
		Int("mediaId", mediaId).
		Msgf("manga: Getting chapters")

	chapterContainerKey := getMangaChapterContainerCacheKey(provider, mediaId)

	// +---------------------+
	// |     Hook event      |
	// +---------------------+

	// Trigger hook event
	reqEvent := &MangaChapterContainerRequestedEvent{
		Provider: provider,
		MediaId:  mediaId,
		Titles:   titles,
		Year:     opts.Year,
		ChapterContainer: &ChapterContainer{
			MediaId:  mediaId,
			Provider: provider,
			Chapters: []*hibikemanga.ChapterDetails{},
		},
	}
	err = hook.GlobalHookManager.OnMangaChapterContainerRequested().Trigger(reqEvent)
	if err != nil {
		r.logger.Error().Err(err).Msg("manga: Exception occurred while triggering hook event")
		return nil, fmt.Errorf("manga: Error in hook, %w", err)
	}

	// Default prevented, return the chapter container
	if reqEvent.DefaultPrevented {
		if reqEvent.ChapterContainer == nil {
			return nil, fmt.Errorf("manga: No chapter container returned by hook event")
		}
		return reqEvent.ChapterContainer, nil
	}

	// +---------------------+
	// |       Cache         |
	// +---------------------+

	var container *ChapterContainer
	containerBucket := r.getFcProviderBucket(provider, mediaId, bucketTypeChapter)

	// Check if the container is in the cache
	if found, _ := r.fileCacher.Get(containerBucket, chapterContainerKey, &container); found {
		r.logger.Info().Str("bucket", containerBucket.Name()).Msg("manga: Chapter Container Cache HIT")

		// Trigger hook event
		ev := &MangaChapterContainerEvent{
			ChapterContainer: container,
		}
		err = hook.GlobalHookManager.OnMangaChapterContainer().Trigger(ev)
		if err != nil {
			r.logger.Error().Err(err).Msg("manga: Exception occurred while triggering hook event")
		}
		container = ev.ChapterContainer

		return container, nil
	}

	// Delete the map cache
	mangaLatestChapterNumberMap.Delete(ChapterCountMapCacheKey)

	providerExtension, ok := extension.GetExtension[extension.MangaProviderExtension](r.providerExtensionBank, provider)
	if !ok {
		r.logger.Error().Str("provider", provider).Msg("manga: Provider not found")
		return nil, errors.New("manga: Provider not found")
	}

	var mangaId string

	// +---------------------+
	// |      Database       |
	// +---------------------+

	// Search for the mapping in the database
	mapping, found := r.db.GetMangaMapping(provider, mediaId)
	if found {
		r.logger.Debug().Str("mangaId", mapping.MangaID).Msg("manga: Using manual mapping")
		mangaId = mapping.MangaID
	}

	if mangaId == "" {
		// +---------------------+
		// |       Search        |
		// +---------------------+

		r.logger.Trace().Msg("manga: Searching for manga")

		if titles == nil {
			return nil, ErrNoTitlesProvided
		}

		titles = lo.Filter(titles, func(title *string, _ int) bool {
			return util.IsMostlyLatinString(*title)
		})

		var searchRes []*hibikemanga.SearchResult

		var err error
		for _, title := range titles {
			var _searchRes []*hibikemanga.SearchResult

			_searchRes, err = providerExtension.GetProvider().Search(hibikemanga.SearchOptions{
				Query: *title,
				Year:  opts.Year,
			})
			if err == nil {

				HydrateSearchResultSearchRating(_searchRes, title)

				searchRes = append(searchRes, _searchRes...)
			} else {
				r.logger.Warn().Err(err).Msg("manga: Search failed")
			}
		}

		if searchRes == nil || len(searchRes) == 0 {
			r.logger.Error().Msg("manga: No search results found")
			if err != nil {
				return nil, fmt.Errorf("%w, %w", ErrNoResults, err)
			} else {
				return nil, ErrNoResults
			}
		}

		// Overwrite the provider just in case
		for _, res := range searchRes {
			res.Provider = provider
		}

		bestRes := GetBestSearchResult(searchRes)

		mangaId = bestRes.ID
	}

	// +---------------------+
	// |    Get chapters     |
	// +---------------------+

	chapterList, err := providerExtension.GetProvider().FindChapters(mangaId)
	if err != nil {
		r.logger.Error().Err(err).Msg("manga: Failed to get chapters")
		return nil, ErrNoChapters
	}

	// Overwrite the provider just in case
	for _, chapter := range chapterList {
		chapter.Provider = provider
	}

	container = &ChapterContainer{
		MediaId:  mediaId,
		Provider: provider,
		Chapters: chapterList,
	}

	// Trigger hook event
	ev := &MangaChapterContainerEvent{
		ChapterContainer: container,
	}
	err = hook.GlobalHookManager.OnMangaChapterContainer().Trigger(ev)
	if err != nil {
		r.logger.Error().Err(err).Msg("manga: Exception occurred while triggering hook event")
	}
	container = ev.ChapterContainer

	// Cache the container only if it has chapters
	if len(container.Chapters) > 0 {
		err = r.fileCacher.Set(containerBucket, chapterContainerKey, container)
		if err != nil {
			r.logger.Warn().Err(err).Msg("manga: Failed to populate cache")
		}
	}

	r.logger.Info().Str("bucket", containerBucket.Name()).Msg("manga: Retrieved chapters")
	return container, nil
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// getMangaId retrieves the manga ID for the given provider, mediaId, titles, and year.
// It first checks for a manual mapping in the database, then searches if no mapping is found.
func (r *Repository) getMangaId(provider string, mediaId int, titles []*string, year int) (string, error) {
	// Check for manual mapping in the database
	mapping, found := r.db.GetMangaMapping(provider, mediaId)
	if found {
		r.logger.Debug().Str("mangaId", mapping.MangaID).Msg("manga: Using manual mapping")
		return mapping.MangaID, nil
	}

	// Get provider extension
	baseExt, found := r.providerExtensionBank.Get(provider)
	if !found {
		return "", fmt.Errorf("manga: Provider extension not found: %s", provider)
	}

	providerExtension, ok := baseExt.(extension.MangaProviderExtension)
	if !ok {
		return "", fmt.Errorf("manga: Invalid extension type for provider: %s", provider)
	}

	// Search for manga if no mapping found
	r.logger.Trace().Msg("manga: Searching for manga")

	if titles == nil {
		return "", ErrNoTitlesProvided
	}

	titles = lo.Filter(titles, func(title *string, _ int) bool {
		return util.IsMostlyLatinString(*title)
	})

	var searchRes []*hibikemanga.SearchResult
	var err error

	for _, title := range titles {
		var _searchRes []*hibikemanga.SearchResult

		_searchRes, err = providerExtension.GetProvider().Search(hibikemanga.SearchOptions{
			Query: *title,
			Year:  year,
		})
		if err == nil {
			HydrateSearchResultSearchRating(_searchRes, title)
			searchRes = append(searchRes, _searchRes...)
		} else {
			r.logger.Warn().Err(err).Msg("manga: Search failed")
		}
	}

	if searchRes == nil || len(searchRes) == 0 {
		r.logger.Error().Msg("manga: No search results found")
		if err != nil {
			return "", fmt.Errorf("%w, %w", ErrNoResults, err)
		} else {
			return "", ErrNoResults
		}
	}

	// Overwrite the provider just in case
	for _, res := range searchRes {
		res.Provider = provider
	}

	bestRes := GetBestSearchResult(searchRes)
	return bestRes.ID, nil
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// GetMangaVolumeContainer returns the VolumeContainer for a manga entry based on the provider.
// Like ChapterContainer, it is cached in a file cache bucket.
func (r *Repository) GetMangaVolumeContainer(opts *GetMangaChapterContainerOptions) (ret *VolumeContainer, err error) {
	defer util.HandlePanicInModuleWithError("manga/GetMangaVolumeContainer", &err)

	provider := opts.Provider
	mediaId := opts.MediaId
	titles := opts.Titles

	r.logger.Trace().
		Str("provider", provider).
		Int("mediaId", mediaId).
		Msgf("manga: Getting volumes")

	volumeContainerKey := getMangaChapterContainerCacheKey(provider, mediaId)
	containerBucket := r.getFcProviderBucket(provider, mediaId, bucketTypeVolume)

	// Check cache first
	var container *VolumeContainer
	found, err := r.fileCacher.Get(containerBucket, volumeContainerKey, &container)
	if err == nil && found && container != nil {
		r.logger.Trace().Str("bucket", containerBucket.Name()).Msg("manga: Retrieved volumes from cache")
		return container, nil
	}

	// Get extension
	baseExt, found := r.providerExtensionBank.Get(provider)
	if !found || baseExt == nil {
		r.logger.Error().Str("provider", provider).Msg("manga: Extension not found")
		return nil, fmt.Errorf("manga: Extension not found")
	}

	providerExtension, ok := baseExt.(extension.MangaProviderExtension)
	if !ok {
		r.logger.Error().Str("provider", provider).Msg("manga: Invalid extension type")
		return nil, fmt.Errorf("manga: Invalid extension type for provider: %s", provider)
	}

	// Search for manga ID first if not mapped
	mangaId, err := r.getMangaId(provider, mediaId, titles, opts.Year)
	if err != nil {
		r.logger.Error().Err(err).Msg("manga: Failed to get manga ID")
		return nil, err
	}

	// Get volumes from provider
	volumeList, err := providerExtension.GetProvider().FindVolumes(mangaId)
	if err != nil {
		r.logger.Error().Err(err).Msg("manga: Failed to get volumes")
		return nil, ErrNoVolumes
	}

	// Overwrite the provider just in case
	for _, volume := range volumeList {
		volume.Provider = provider
	}

	container = &VolumeContainer{
		MediaId:  mediaId,
		Provider: provider,
		Volumes:  volumeList,
	}

	// Cache the container only if it has volumes
	if len(container.Volumes) > 0 {
		err = r.fileCacher.Set(containerBucket, volumeContainerKey, container)
		if err != nil {
			r.logger.Warn().Err(err).Msg("manga: Failed to populate volume cache")
		}
	}

	r.logger.Info().Str("bucket", containerBucket.Name()).Msg("manga: Retrieved volumes")
	return container, nil
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// RefreshChapterContainers deletes all cached chapter containers and refetches them based on the selected provider map.
func (r *Repository) RefreshChapterContainers(mangaCollection *anilist.MangaCollection, selectedProviderMap map[int]string) (err error) {
	defer util.HandlePanicInModuleWithError("manga/RefreshChapterContainers", &err)

	// Read the cache directory
	entries, err := os.ReadDir(r.cacheDir)
	if err != nil {
		return err
	}

	removedMediaIds := make(map[int]struct{})
	mu := sync.Mutex{}

	wg := sync.WaitGroup{}
	wg.Add(len(entries))
	for _, entry := range entries {
		go func(entry os.DirEntry) {
			defer wg.Done()

			if entry.IsDir() {
				return
			}

			provider, bucketType, mediaId, ok := ParseChapterContainerFileName(entry.Name())
			if !ok {
				return
			}
			// If the bucket type is not chapter, skip
			if bucketType != bucketTypeChapter {
				return
			}

			r.logger.Trace().Str("provider", provider).Int("mediaId", mediaId).Msg("manga: Refetching chapter container")

			mu.Lock()
			// Remove the container from the cache if it hasn't been removed yet
			if _, ok := removedMediaIds[mediaId]; !ok {
				r.EmptyMangaCache(mediaId)
				removedMediaIds[mediaId] = struct{}{}
			}
			mu.Unlock()

			// If a selectedProviderMap is provided, check if the provider is in the map
			if selectedProviderMap != nil {
				// If the manga is not in the map, continue
				if _, ok := selectedProviderMap[mediaId]; !ok {
					return
				}

				// If the provider is not the one selected, continue
				if selectedProviderMap[mediaId] != provider {
					return
				}
			}

			// Get the manga from the collection
			mangaEntry, found := mangaCollection.GetListEntryFromMangaId(mediaId)
			if !found {
				return
			}

			// If the manga is not currently reading or repeating, continue
			if *mangaEntry.GetStatus() != anilist.MediaListStatusCurrent && *mangaEntry.GetStatus() != anilist.MediaListStatusRepeating {
				return
			}

			// Refetch the container
			_, err = r.GetMangaChapterContainer(&GetMangaChapterContainerOptions{
				Provider: provider,
				MediaId:  mediaId,
				Titles:   mangaEntry.GetMedia().GetAllTitles(),
				Year:     mangaEntry.GetMedia().GetStartYearSafe(),
			})
			if err != nil {
				r.logger.Error().Err(err).Msg("manga: Failed to refetch chapter container")
				return
			}

			r.logger.Trace().Str("provider", provider).Int("mediaId", mediaId).Msg("manga: Refetched chapter container")
		}(entry)
	}
	wg.Wait()

	return nil
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

const ChapterCountMapCacheKey = 1

var mangaLatestChapterNumberMap = result.NewResultMap[int, map[int][]MangaLatestChapterNumberItem]()

type MangaLatestChapterNumberItem struct {
	Provider  string `json:"provider"`
	Scanlator string `json:"scanlator"`
	Language  string `json:"language"`
	Number    int    `json:"number"`
}

// GetMangaLatestChapterNumbersMap retrieves the latest chapter number for all manga entries.
// It scans the cache directory for chapter containers and counts the number of chapters fetched from the provider for each manga.
//
// Unlike [GetMangaLatestChapterNumberMap], it will segregate the chapter numbers by scanlator and language.
func (r *Repository) GetMangaLatestChapterNumbersMap() (ret map[int][]MangaLatestChapterNumberItem, err error) {
	defer util.HandlePanicInModuleThen("manga/GetMangaLatestChapterNumbersMap", func() {})
	ret = make(map[int][]MangaLatestChapterNumberItem)

	if m, ok := mangaLatestChapterNumberMap.Get(ChapterCountMapCacheKey); ok {
		ret = m
		return
	}

	// Go through all chapter container caches
	entries, err := os.ReadDir(r.cacheDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Get the provider and mediaId from the file cache name
		provider, mediaId, ok := parseChapterFileName(entry.Name())
		if !ok {
			continue
		}

		containerBucket := r.getFcProviderBucket(provider, mediaId, bucketTypeChapter)

		// Get the container from the file cache
		var container *ChapterContainer
		chapterContainerKey := getMangaChapterContainerCacheKey(provider, mediaId)
		if found, _ := r.fileCacher.Get(containerBucket, chapterContainerKey, &container); !found {
			continue
		}

		// Create groups
		groupByScanlator := lo.GroupBy(container.Chapters, func(c *hibikemanga.ChapterDetails) string {
			return c.Scanlator
		})

		for scanlator, chapters := range groupByScanlator {
			groupByLanguage := lo.GroupBy(chapters, func(c *hibikemanga.ChapterDetails) string {
				return c.Language
			})

			for language, chapters := range groupByLanguage {
				lastChapter := slices.MaxFunc(chapters, func(a *hibikemanga.ChapterDetails, b *hibikemanga.ChapterDetails) int {
					return cmp.Compare(a.Index, b.Index)
				})

				chapterNumFloat, _ := strconv.ParseFloat(lastChapter.Chapter, 32)
				chapterCount := int(math.Floor(chapterNumFloat))

				if _, ok := ret[mediaId]; !ok {
					ret[mediaId] = []MangaLatestChapterNumberItem{}
				}

				ret[mediaId] = append(ret[mediaId], MangaLatestChapterNumberItem{
					Provider:  provider,
					Scanlator: scanlator,
					Language:  language,
					Number:    chapterCount,
				})
			}
		}
	}

	// Trigger hook event
	ev := &MangaLatestChapterNumbersMapEvent{
		LatestChapterNumbersMap: ret,
	}
	err = hook.GlobalHookManager.OnMangaLatestChapterNumbersMap().Trigger(ev)
	if err != nil {
		r.logger.Error().Err(err).Msg("manga: Exception occurred while triggering hook event")
	}
	ret = ev.LatestChapterNumbersMap

	mangaLatestChapterNumberMap.Set(ChapterCountMapCacheKey, ret)
	return
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func parseChapterFileName(dirName string) (provider string, mId int, ok bool) {
	if !strings.HasPrefix(dirName, "manga_") {
		return "", 0, false
	}
	dirName = strings.TrimSuffix(dirName, ".cache")
	parts := strings.Split(dirName, "_")
	if len(parts) != 4 {
		return "", 0, false
	}

	provider = parts[1]
	mId, err := strconv.Atoi(parts[3])
	if err != nil {
		return "", 0, false
	}

	return provider, mId, true
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func GetBestSearchResult(searchRes []*hibikemanga.SearchResult) *hibikemanga.SearchResult {
	bestRes := searchRes[0]
	for _, res := range searchRes {
		if res.SearchRating > bestRes.SearchRating {
			bestRes = res
		}
	}
	return bestRes
}

// HydrateSearchResultSearchRating rates the search results based on the provided title
// It checks if all search results have a rating of 0 and if so, it calculates ratings
// using the Sorensen-Dice
func HydrateSearchResultSearchRating(_searchRes []*hibikemanga.SearchResult, title *string) {
	// Rate the search results if all ratings are 0
	if noRatings := lo.EveryBy(_searchRes, func(res *hibikemanga.SearchResult) bool {
		return res.SearchRating == 0
	}); noRatings {
		wg := sync.WaitGroup{}
		wg.Add(len(_searchRes))
		
		// Normalize the input title once for more accurate comparisons
		normalizedInputTitle := normalizeTitle(*title)
		
		for _, res := range _searchRes {
			go func(res *hibikemanga.SearchResult) {
				defer wg.Done()

				// First try to find romaji title that closely matches the normalized input
				var bestMatch float64
				res.SearchRating = 0
				var romajiTitles []string
				var otherTitles []string
				
				// Collect and normalize all titles
				if res.Synonyms != nil {
					for _, syn := range res.Synonyms {
						normalizedSyn := normalizeTitle(syn)
						if isRomaji(syn) {
							romajiTitles = append(romajiTitles, normalizedSyn)
						} else {
							otherTitles = append(otherTitles, normalizedSyn)
						}
					}
				}

				// Convert string slices to *string slices for compatibility
				var compTitles []*string
				
				// Prioritize romaji titles first
				for _, rt := range romajiTitles {
					t := rt // Create a new variable to get its address
					compTitles = append(compTitles, &t)
				}
				
				// Add other titles as fallback
				for _, ot := range otherTitles {
					t := ot // Create a new variable to get its address
					compTitles = append(compTitles, &t)
				}
				
				// Compare the normalized input title with each candidate title
				for _, titlePtr := range compTitles {
					if titlePtr != nil {
						similarity := metrics.NewJaroWinkler().Compare(normalizedInputTitle, *titlePtr)
						if similarity > bestMatch {
							bestMatch = similarity
							res.SearchRating = similarity // JaroWinkler already returns a value between 0 and 1
						}
					}
				}

				compRes, ok := comparison.FindBestMatchWithSorensenDice(title, compTitles)
				if !ok {
					return
				}

				res.SearchRating = compRes.Rating
				return
			}(res)
		}
		wg.Wait()
	}
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// isRomaji checks if a string contains only basic latin characters and common punctuation
func isRomaji(s string) bool {
	for _, r := range s {
		if unicode.In(r, unicode.Hiragana) || unicode.In(r, unicode.Katakana) || unicode.In(r, unicode.Han) {
			return false
		}
		// Allow basic latin chars, numbers, spaces and common punctuation
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) && !unicode.IsSpace(r) && 
		   !strings.ContainsRune("'-!?,.:", r) {
			return false
		}
	}
	return true
}

// normalizeTitle normalizes a manga title for comparison by:
// - Converting to lowercase
// - Removing special characters except spaces
// - Normalizing whitespace
func normalizeTitle(s string) string {
	// Convert to lowercase
	s = strings.ToLower(s)
	
	// Replace multiple spaces with single space
	s = strings.Join(strings.Fields(s), " ")
	
	// Remove special characters except spaces
	normalized := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || r == ' ' {
			return r
		}
		return -1
	}, s)
	
	return strings.TrimSpace(normalized)
}
