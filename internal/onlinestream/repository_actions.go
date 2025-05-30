package onlinestream

import (
	"errors"
	"fmt"
	"seanime/internal/api/anilist"
	"seanime/internal/extension"
	hibikeonlinestream "seanime/internal/extension/hibike/onlinestream"
	onlinestream_providers "seanime/internal/onlinestream/providers"
	"seanime/internal/util/comparison"
	"strings"
)

var (
	ErrNoAnimeFound         = errors.New("no anime found")
	ErrNoEpisodes           = errors.New("no episodes found")
	errNoEpisodeSourceFound = errors.New("no source found for episode")
)

type (
	// episodeContainer contains results of fetching the episodes from the provider.
	episodeContainer struct {
		Provider string
		// List of episode details from the provider.
		// It is used to get the episode servers.
		ProviderEpisodeList []*hibikeonlinestream.EpisodeDetails
		// List of episodes with their servers.
		Episodes []*episodeData
	}

	// episodeData contains some details about a provider episode and all available servers.
	episodeData struct {
		Provider string
		ID       string
		Number   int
		Title    string
		Servers  []*hibikeonlinestream.EpisodeServer
	}
)

// getEpisodeContainer gets the episode details and servers from the specified provider.
// It takes the media ID, titles in order to fetch the episode details.
//   - This function can be used to only get the episode details by setting 'from' and 'to' to 0.
//
// Since the episode details are cached, we can request episode servers multiple times without fetching the episode details again.
func (r *Repository) getEpisodeContainer(provider string, media *anilist.BaseAnime, from int, to int, dubbed bool, year int) (*episodeContainer, error) {

	r.logger.Debug().
		Str("provider", provider).
		Int("mediaId", media.ID).
		Int("from", from).
		Int("to", to).
		Bool("dubbed", dubbed).
		Msg("onlinestream: Getting episode container")

	// Key identifying the provider episode list in the file cache.
	// It includes "dubbed" because Gogoanime has a different entry for dubbed anime.
	// e.g. 1$provider$true
	providerEpisodeListKey := fmt.Sprintf("%d$%s$%v", media.ID, provider, dubbed)

	// Create the episode container
	ec := &episodeContainer{
		Provider:            provider,
		Episodes:            make([]*episodeData, 0),
		ProviderEpisodeList: make([]*hibikeonlinestream.EpisodeDetails, 0),
	}

	// Get the episode details from the provider.
	r.logger.Debug().
		Str("key", providerEpisodeListKey).
		Msgf("onlinestream: Fetching %s episode list", provider)

	// Buckets for caching the episode list and episode data.
	fcEpisodeListBucket := r.getFcEpisodeListBucket(provider, media.ID)
	fcEpisodeDataBucket := r.getFcEpisodeDataBucket(provider, media.ID)

	// Check if the episode list is cached to avoid fetching it again.
	var providerEpisodeList []*hibikeonlinestream.EpisodeDetails
	if found, _ := r.fileCacher.Get(fcEpisodeListBucket, providerEpisodeListKey, &providerEpisodeList); !found {
		var err error
		providerEpisodeList, err = r.getProviderEpisodeList(provider, media, dubbed, year)
		if err != nil {
			r.logger.Error().Err(err).Msg("onlinestream: Failed to get provider episodes")
			return nil, err // ErrNoAnimeFound or ErrNoEpisodes
		}
		_ = r.fileCacher.Set(fcEpisodeListBucket, providerEpisodeListKey, providerEpisodeList)
	} else {
		r.logger.Debug().
			Str("key", providerEpisodeListKey).
			Msg("onlinestream: Cache HIT for episode list")
	}

	ec.ProviderEpisodeList = providerEpisodeList

	for _, episodeDetails := range providerEpisodeList {

		if episodeDetails.Number >= from && episodeDetails.Number <= to {

			// Check if the episode is cached to avoid fetching the sources again.
			key := fmt.Sprintf("%d$%s$%d$%v", media.ID, provider, episodeDetails.Number, dubbed)

			r.logger.Debug().
				Str("key", key).
				Msgf("onlinestream: Fetching episode '%d' servers", episodeDetails.Number)

			// Check episode cache
			var cached *episodeData
			if found, _ := r.fileCacher.Get(fcEpisodeDataBucket, key, &cached); found {
				ec.Episodes = append(ec.Episodes, cached)

				r.logger.Debug().
					Str("key", key).
					Msgf("onlinestream: Cache HIT for episode '%d' servers", episodeDetails.Number)

				continue
			}

			// Zoro dubs
			if provider == onlinestream_providers.ZoroProvider && dubbed {
				// If the episode details have both sub and dub, we need to get the dub episode.
				if !strings.HasSuffix(episodeDetails.ID, string(hibikeonlinestream.SubAndDub)) {
					// Skip sub-only episodes
					continue
				}
				// Replace "both" with "dub" so that [getProviderEpisodeServers] can find the dub episode.
				episodeDetails.ID = strings.Replace(episodeDetails.ID, string(hibikeonlinestream.SubAndDub), string(hibikeonlinestream.Dub), 1)
			}

			// Fetch episode servers
			servers, err := r.getProviderEpisodeServers(provider, episodeDetails)
			if err != nil {
				r.logger.Error().Err(err).Msgf("onlinestream: failed to get episode '%d' servers", episodeDetails.Number)
				continue
			}

			episode := &episodeData{
				ID:      episodeDetails.ID,
				Number:  episodeDetails.Number,
				Title:   episodeDetails.Title,
				Servers: servers,
			}
			ec.Episodes = append(ec.Episodes, episode)

			r.logger.Debug().
				Str("key", key).
				Msgf("onlinestream: Found %d servers for episode '%d'", len(servers), episodeDetails.Number)

			_ = r.fileCacher.Set(fcEpisodeDataBucket, key, episode)

		}

	}

	if from > 0 && to > 0 && len(ec.Episodes) == 0 {
		r.logger.Error().Msg("onlinestream: No episodes found")
		return nil, ErrNoEpisodes
	}

	if len(ec.ProviderEpisodeList) == 0 {
		r.logger.Error().Msg("onlinestream: No episodes found")
		return nil, ErrNoEpisodes
	}

	return ec, nil
}

// getProviderEpisodeServers gets all the available servers for the episode.
// It returns errNoEpisodeSourceFound if no sources are found.
//
// Example:
//
//	episodeDetails, _ := getProviderEpisodeListFromTitles(provider, titles, dubbed)
//	episodeServers, err := getProviderEpisodeServers(provider, episodeDetails[0])
func (r *Repository) getProviderEpisodeServers(provider string, episodeDetails *hibikeonlinestream.EpisodeDetails) ([]*hibikeonlinestream.EpisodeServer, error) {
	var providerServers []*hibikeonlinestream.EpisodeServer

	providerExtension, ok := extension.GetExtension[extension.OnlinestreamProviderExtension](r.providerExtensionBank, provider)
	if !ok {
		return nil, fmt.Errorf("provider extension '%s' not found", provider)
	}

	for _, episodeServer := range providerExtension.GetProvider().GetSettings().EpisodeServers {
		res, err := providerExtension.GetProvider().FindEpisodeServer(episodeDetails, episodeServer)
		if err == nil {
			// Add the server to the list for the episode
			providerServers = append(providerServers, res)
		}
	}

	if len(providerServers) == 0 {
		return nil, errNoEpisodeSourceFound
	}

	return providerServers, nil
}

// getProviderEpisodeList gets all the hibikeonlinestream.EpisodeDetails from the provider based on the anime's titles.
// It returns ErrNoAnimeFound if the anime is not found or ErrNoEpisodes if no episodes are found.
func (r *Repository) getProviderEpisodeList(provider string, media *anilist.BaseAnime, dubbed bool, year int) ([]*hibikeonlinestream.EpisodeDetails, error) {
	var ret []*hibikeonlinestream.EpisodeDetails
	// romajiTitle := strings.ReplaceAll(media.GetEnglishTitleSafe(), ":", "")
	// englishTitle := strings.ReplaceAll(media.GetRomajiTitleSafe(), ":", "")

	romajiTitle := media.GetRomajiTitleSafe()
	englishTitle := media.GetEnglishTitleSafe()

	providerExtension, ok := extension.GetExtension[extension.OnlinestreamProviderExtension](r.providerExtensionBank, provider)
	if !ok {
		return nil, fmt.Errorf("provider extension '%s' not found", provider)
	}

	mId := media.ID

	var matchId string

	// +---------------------+
	// |      Database       |
	// +---------------------+

	// Search for the mapping in the database
	mapping, found := r.db.GetOnlinestreamMapping(provider, mId)
	if found {
		r.logger.Debug().Str("animeId", mapping.AnimeID).Msg("onlinestream: Using manual mapping")
		matchId = mapping.AnimeID
	}

	if matchId == "" {
		// +---------------------+
		// |       Search        |
		// +---------------------+

		// Get search results.
		var searchResults []*hibikeonlinestream.SearchResult

		queryMedia := hibikeonlinestream.Media{
			ID:           media.ID,
			IDMal:        media.GetIDMal(),
			Status:       string(*media.GetStatus()),
			Format:       string(*media.GetFormat()),
			EnglishTitle: media.GetTitle().GetEnglish(),
			RomajiTitle:  media.GetRomajiTitleSafe(),
			EpisodeCount: media.GetTotalEpisodeCount(),
			Synonyms:     media.GetSynonymsContainingSeason(),
			IsAdult:      *media.GetIsAdult(),
			StartDate: &hibikeonlinestream.FuzzyDate{
				Year:  *media.GetStartDate().GetYear(),
				Month: media.GetStartDate().GetMonth(),
				Day:   media.GetStartDate().GetDay(),
			},
		}

		added := make(map[string]struct{})

		if romajiTitle != "" {
			// Search by romaji title
			res, err := providerExtension.GetProvider().Search(hibikeonlinestream.SearchOptions{
				Media: queryMedia,
				Query: romajiTitle,
				Dub:   dubbed,
				Year:  year,
			})
			if err == nil && len(res) > 0 {
				searchResults = append(searchResults, res...)
				for _, r := range res {
					added[r.ID] = struct{}{}
				}
			}
			if err != nil {
				r.logger.Error().Err(err).Msg("onlinestream: Failed to search for romaji title")
			}
			r.logger.Debug().
				Int("romajiTitleResults", len(res)).
				Msg("onlinestream: Found results for romaji title")
		}

		if englishTitle != "" {
			// Search by english title
			res, err := providerExtension.GetProvider().Search(hibikeonlinestream.SearchOptions{
				Media: queryMedia,
				Query: englishTitle,
				Dub:   dubbed,
				Year:  year,
			})
			if err == nil && len(res) > 0 {
				for _, r := range res {
					if _, ok := added[r.ID]; !ok {
						searchResults = append(searchResults, r)
					}
				}
			}
			if err != nil {
				r.logger.Error().Err(err).Msg("onlinestream: Failed to search for english title")
			}
			r.logger.Debug().
				Int("englishTitleResults", len(res)).
				Msg("onlinestream: Found results for english title")
		}

		if len(searchResults) == 0 {
			return nil, ErrNoAnimeFound
		}

		bestResult, found := GetBestSearchResult(searchResults, media.GetAllTitles())
		if !found {
			return nil, ErrNoAnimeFound
		}
		matchId = bestResult.ID
	}

	// Fetch episodes.
	ret, err := providerExtension.GetProvider().FindEpisodes(matchId)
	if err != nil {
		return nil, err
	}

	if len(ret) == 0 {
		return nil, ErrNoEpisodes
	}

	return ret, nil
}

func GetBestSearchResult(searchResults []*hibikeonlinestream.SearchResult, titles []*string) (*hibikeonlinestream.SearchResult, bool) {
	// Filter results to get the best match.
	compBestResults := make([]*comparison.LevenshteinResult, 0, len(searchResults))
	for _, r := range searchResults {
		// Compare search result title with all titles.
		compBestResult, found := comparison.FindBestMatchWithLevenshtein(&r.Title, titles)
		if found {
			compBestResults = append(compBestResults, compBestResult)
		}
	}

	if len(compBestResults) == 0 {
		return nil, false
	}

	compBestResult := compBestResults[0]
	for _, r := range compBestResults {
		if r.Distance < compBestResult.Distance {
			compBestResult = r
		}
	}

	// Get most accurate search result.
	var bestResult *hibikeonlinestream.SearchResult
	for _, r := range searchResults {
		if r.Title == *compBestResult.OriginalValue {
			bestResult = r
			break
		}
	}
	return bestResult, true
}
