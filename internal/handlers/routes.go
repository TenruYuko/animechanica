package handlers

import (
	"net/http"
	"path/filepath"
	"seanime/internal/core"
	util "seanime/internal/util/proxies"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"github.com/ziflex/lecho/v3"
)

type Handler struct {
	App *core.App
}

func InitRoutes(app *core.App, e *echo.Echo) {
	// CORS middleware
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Cookie", "Authorization"},
		AllowCredentials: true,
	}))

	lechoLogger := lecho.From(*app.Logger)

	urisToSkip := []string{
		"/internal/metrics",
		"/_next",
		"/icons",
		"/events",
		"/api/v1/image-proxy",
		"/api/v1/mediastream/transcode/",
		"/api/v1/torrent-client/list",
		"/api/v1/proxy",
	}

	// Logging middleware
	e.Use(lecho.Middleware(lecho.Config{
		Logger: lechoLogger,
		Skipper: func(c echo.Context) bool {
			path := c.Request().URL.RequestURI()
			if filepath.Ext(c.Request().URL.Path) == ".txt" ||
				filepath.Ext(c.Request().URL.Path) == ".png" ||
				filepath.Ext(c.Request().URL.Path) == ".ico" {
				return true
			}
			for _, uri := range urisToSkip {
				if uri == path || strings.HasPrefix(path, uri) {
					return true
				}
			}
			return false
		},
		Enricher: func(c echo.Context, logger zerolog.Context) zerolog.Context {
			// Add which file the request came from
			return logger.Str("file", c.Path())
		},
	}))

	// Recovery middleware
	e.Use(middleware.Recover())

	// Client ID middleware
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Check if the client has a UUID cookie
			cookie, err := c.Cookie("Seanime-Client-Id")

			if err != nil || cookie.Value == "" {
				// Generate a new UUID for the client
				u := uuid.New().String()

				// Create a cookie with the UUID
				newCookie := new(http.Cookie)
				newCookie.Name = "Seanime-Client-Id"
				newCookie.Value = u
				newCookie.HttpOnly = false // Make the cookie accessible via JS
				newCookie.Expires = time.Now().Add(24 * time.Hour)
				newCookie.Path = "/"
				newCookie.Domain = ""
				newCookie.SameSite = http.SameSiteDefaultMode
				newCookie.Secure = false

				// Set the cookie
				c.SetCookie(newCookie)

				// Store the UUID in the context for use in the request
				c.Set("Seanime-Client-Id", u)
			} else {
				// Store the existing UUID in the context for use in the request
				c.Set("Seanime-Client-Id", cookie.Value)
			}

			return next(c)
		}
	})

	e.Use(headMethodMiddleware)

	h := &Handler{App: app}

	e.GET("/events", h.webSocketEventHandler)

	v1 := e.Group("/api").Group("/v1") // Base API group

	imageProxy := &util.ImageProxy{}
	v1.GET("/image-proxy", imageProxy.ProxyImage)

	v1.GET("/proxy", util.VideoProxy)
	v1.HEAD("/proxy", util.VideoProxy)

	v1.GET("/status", h.HandleGetStatus)
	v1.GET("/log/*", h.HandleGetLogContent)
	v1.GET("/logs/filenames", h.HandleGetLogFilenames)
	v1.DELETE("/logs", h.HandleDeleteLogs)
	v1.GET("/logs/latest", h.HandleGetLatestLogContent)
	// Auth endpoints (no session required)
	v1.POST("/auth/login", h.HandleLogin)
	v1.POST("/auth/logout", h.HandleLogout)
	v1.GET("/auth/check-session", h.HandleCheckSession)
	v1.GET("/auth/test-session", h.HandleTestSession)

	// Create a group for protected routes that require authentication
	protected := v1.Group("", h.SessionMiddleware)

	// Settings - protected routes
	protected.GET("/settings", h.HandleGetSettings)
	protected.PATCH("/settings", h.HandleSaveSettings)
	protected.POST("/start", h.HandleGettingStarted)
	protected.PATCH("/settings/auto-downloader", h.HandleSaveAutoDownloaderSettings)

	// Auto Downloader - protected routes
	protected.POST("/auto-downloader/run", h.HandleRunAutoDownloader)
	protected.GET("/auto-downloader/rule/:id", h.HandleGetAutoDownloaderRule)
	protected.GET("/auto-downloader/rule/anime/:id", h.HandleGetAutoDownloaderRulesByAnime)
	protected.GET("/auto-downloader/rules", h.HandleGetAutoDownloaderRules)
	protected.POST("/auto-downloader/rule", h.HandleCreateAutoDownloaderRule)
	protected.PATCH("/auto-downloader/rule", h.HandleUpdateAutoDownloaderRule)
	protected.DELETE("/auto-downloader/rule/:id", h.HandleDeleteAutoDownloaderRule)

	protected.GET("/auto-downloader/items", h.HandleGetAutoDownloaderItems)
	protected.DELETE("/auto-downloader/item", h.HandleDeleteAutoDownloaderItem)

	// Other - protected routes
	protected.POST("/test-dump", h.HandleTestDump)

	protected.POST("/directory-selector", h.HandleDirectorySelector)

	protected.POST("/open-in-explorer", h.HandleOpenInExplorer)

	protected.POST("/media-player/start", h.HandleStartDefaultMediaPlayer)

	//
	// AniList - protected routes
	//

	v1Anilist := protected.Group("/anilist")

	v1Anilist.GET("/collection", h.HandleGetAnimeCollection)
	v1Anilist.POST("/collection", h.HandleGetAnimeCollection)

	v1Anilist.GET("/collection/raw", h.HandleGetRawAnimeCollection)
	v1Anilist.POST("/collection/raw", h.HandleGetRawAnimeCollection)

	v1Anilist.GET("/media-details/:id", h.HandleGetAnilistAnimeDetails)

	v1Anilist.GET("/studio-details/:id", h.HandleGetAnilistStudioDetails)

	v1Anilist.POST("/list-entry", h.HandleEditAnilistListEntry)

	v1Anilist.DELETE("/list-entry", h.HandleDeleteAnilistListEntry)

	v1Anilist.POST("/list-anime", h.HandleAnilistListAnime)

	v1Anilist.POST("/list-recent-anime", h.HandleAnilistListRecentAiringAnime)

	v1Anilist.GET("/list-missed-sequels", h.HandleAnilistListMissedSequels)

	v1Anilist.GET("/stats", h.HandleGetAniListStats)

	//
	// MAL - protected routes
	//

	protected.POST("/mal/auth", h.HandleMALAuth)

	protected.POST("/mal/logout", h.HandleMALLogout)

	//
	// Library - protected routes
	//

	v1Library := protected.Group("/library")

	v1Library.POST("/scan", h.HandleScanLocalFiles)

	v1Library.DELETE("/empty-directories", h.HandleRemoveEmptyDirectories)

	v1Library.GET("/local-files", h.HandleGetLocalFiles)
	v1Library.POST("/local-files", h.HandleLocalFileBulkAction)
	v1Library.PATCH("/local-files", h.HandleUpdateLocalFiles)
	v1Library.DELETE("/local-files", h.HandleDeleteLocalFiles)
	v1Library.GET("/local-files/dump", h.HandleDumpLocalFilesToFile)
	v1Library.POST("/local-files/import", h.HandleImportLocalFiles)
	v1Library.PATCH("/local-file", h.HandleUpdateLocalFileData)

	v1Library.GET("/collection", h.HandleGetLibraryCollection)

	v1Library.GET("/scan-summaries", h.HandleGetScanSummaries)

	v1Library.GET("/missing-episodes", h.HandleGetMissingEpisodes)

	v1Library.GET("/anime-entry/:id", h.HandleGetAnimeEntry)
	v1Library.POST("/anime-entry/suggestions", h.HandleFetchAnimeEntrySuggestions)
	v1Library.POST("/anime-entry/manual-match", h.HandleAnimeEntryManualMatch)
	v1Library.PATCH("/anime-entry/bulk-action", h.HandleAnimeEntryBulkAction)
	v1Library.POST("/anime-entry/open-in-explorer", h.HandleOpenAnimeEntryInExplorer)
	v1Library.POST("/anime-entry/update-progress", h.HandleUpdateAnimeEntryProgress)
	v1Library.POST("/anime-entry/update-repeat", h.HandleUpdateAnimeEntryRepeat)
	v1Library.GET("/anime-entry/silence/:id", h.HandleGetAnimeEntrySilenceStatus)
	v1Library.POST("/anime-entry/silence", h.HandleToggleAnimeEntrySilenceStatus)

	v1Library.POST("/unknown-media", h.HandleAddUnknownMedia)

	//
	// Torrent / Torrent Client
	//

	v1.POST("/torrent/search", h.HandleSearchTorrent)
	v1.POST("/torrent-client/download", h.HandleTorrentClientDownload)
	v1.GET("/torrent-client/list", h.HandleGetActiveTorrentList)
	v1.POST("/torrent-client/action", h.HandleTorrentClientAction)
	v1.POST("/torrent-client/rule-magnet", h.HandleTorrentClientAddMagnetFromRule)

	//
	// Download
	//

	v1.POST("/download-torrent-file", h.HandleDownloadTorrentFile)

	//
	// Updates
	//

	v1.GET("/latest-update", h.HandleGetLatestUpdate)
	v1.GET("/changelog", h.HandleGetChangelog)
	v1.POST("/install-update", h.HandleInstallLatestUpdate)
	v1.POST("/download-release", h.HandleDownloadRelease)

	//
	// Theme
	//

	v1.GET("/theme", h.HandleGetTheme)
	v1.PATCH("/theme", h.HandleUpdateTheme)

	//
	// Playback Manager
	//

	v1.POST("/playback-manager/sync-current-progress", h.HandlePlaybackSyncCurrentProgress)
	v1.POST("/playback-manager/start-playlist", h.HandlePlaybackStartPlaylist)
	v1.POST("/playback-manager/playlist-next", h.HandlePlaybackPlaylistNext)
	v1.POST("/playback-manager/cancel-playlist", h.HandlePlaybackCancelCurrentPlaylist)
	v1.POST("/playback-manager/next-episode", h.HandlePlaybackPlayNextEpisode)
	v1.GET("/playback-manager/next-episode", h.HandlePlaybackGetNextEpisode)
	v1.POST("/playback-manager/autoplay-next-episode", h.HandlePlaybackAutoPlayNextEpisode)
	v1.POST("/playback-manager/play", h.HandlePlaybackPlayVideo)
	v1.POST("/playback-manager/play-random", h.HandlePlaybackPlayRandomVideo)
	//------------
	v1.POST("/playback-manager/manual-tracking/start", h.HandlePlaybackStartManualTracking)
	v1.POST("/playback-manager/manual-tracking/cancel", h.HandlePlaybackCancelManualTracking)

	//
	// Playlists
	//

	v1.GET("/playlists", h.HandleGetPlaylists)
	v1.POST("/playlist", h.HandleCreatePlaylist)
	v1.PATCH("/playlist", h.HandleUpdatePlaylist)
	v1.DELETE("/playlist", h.HandleDeletePlaylist)
	v1.GET("/playlist/episodes/:id/:progress", h.HandleGetPlaylistEpisodes)

	//
	// Onlinestream
	//

	v1.POST("/onlinestream/episode-source", h.HandleGetOnlineStreamEpisodeSource)
	v1.POST("/onlinestream/episode-list", h.HandleGetOnlineStreamEpisodeList)
	v1.DELETE("/onlinestream/cache", h.HandleOnlineStreamEmptyCache)

	v1.POST("/onlinestream/search", h.HandleOnlinestreamManualSearch)
	v1.POST("/onlinestream/manual-mapping", h.HandleOnlinestreamManualMapping)
	v1.POST("/onlinestream/get-mapping", h.HandleGetOnlinestreamMapping)
	v1.POST("/onlinestream/remove-mapping", h.HandleRemoveOnlinestreamMapping)

	//
	// Metadata Provider
	//

	v1.POST("/metadata-provider/tvdb-episodes", h.HandlePopulateTVDBEpisodes)
	v1.DELETE("/metadata-provider/tvdb-episodes", h.HandleEmptyTVDBEpisodes)

	v1.POST("/metadata-provider/filler", h.HandlePopulateFillerData)
	v1.DELETE("/metadata-provider/filler", h.HandleRemoveFillerData)

	//
	// Manga
	//
	v1Manga := v1.Group("/manga")
	v1Manga.POST("/anilist/collection", h.HandleGetAnilistMangaCollection)
	v1Manga.GET("/anilist/collection/raw", h.HandleGetRawAnilistMangaCollection)
	v1Manga.POST("/anilist/collection/raw", h.HandleGetRawAnilistMangaCollection)
	v1Manga.POST("/anilist/list", h.HandleAnilistListManga)
	v1Manga.GET("/collection", h.HandleGetMangaCollection)
	v1Manga.GET("/latest-chapter-numbers", h.HandleGetMangaLatestChapterNumbersMap)
	v1Manga.POST("/refetch-chapter-containers", h.HandleRefetchMangaChapterContainers)
	v1Manga.GET("/entry/:id", h.HandleGetMangaEntry)
	v1Manga.GET("/entry/:id/details", h.HandleGetMangaEntryDetails)
	v1Manga.DELETE("/entry/cache", h.HandleEmptyMangaEntryCache)
	v1Manga.POST("/chapters", h.HandleGetMangaEntryChapters)
	v1Manga.POST("/pages", h.HandleGetMangaEntryPages)
	v1Manga.POST("/update-progress", h.HandleUpdateMangaProgress)
	// DISABLED: Internal storage manga reading functionality has been disabled
	// v1Manga.GET("/internal/thumbnail/:mangaID/:volume", h.HandleGetInternalMangaVolumeThumbnail)
	// Raw Unicode/UTF-8 image/page serving routes - DISABLED
	// --- Unicode/UTF-8 path handling documentation ---
	// All /internal/page/* and /internal/image/* routes below were designed to accept and serve RAW Unicode/UTF-8 path segments.
	// No encoding/decoding or transformation was performed by the backend; all symbols were preserved as sent by the client.
	// If the client sent percent-encoded paths, the handler would decode them to raw Unicode before lookup.
	// This ensured robust support for manga/chapter/image names with any Unicode characters, including CJK, emoji, and symbols.
	// Example: /api/v1/manga/internal/page/【Oshi no Ko】/【Oshi no Ko】 - Volume 014.cbz/Chapter_141/0225.jpg
	// Example: /api/v1/manga/internal/page/Ai/Ai%20-%20Volume%20025.cbz/Chapter_150/0167.jpg
	//
	// If you need to serve images from short paths (e.g. /manga/Chapter_150/0167.jpg), you must implement a mapping or redirect to the full internal path.
	//
	// --- END Unicode/UTF-8 path handling documentation ---
	// DISABLED: Internal storage manga reading has been disabled
	// v1Manga.GET("/internal/page/*", h.HandleGetInternalMangaPage)
	// v1Manga.GET("/internal/image/*", h.HandleGetInternalMangaImage)

	v1Manga.GET("/downloaded-chapters/:id", h.HandleGetMangaEntryDownloadedChapters)
	v1Manga.GET("/downloads", h.HandleGetMangaDownloadsList)
	v1Manga.POST("/download-chapters", h.HandleDownloadMangaChapters)
	v1Manga.POST("/download-data", h.HandleGetMangaDownloadData)
	v1Manga.DELETE("/download-chapter", h.HandleDeleteMangaDownloadedChapters)
	v1Manga.GET("/download-queue", h.HandleGetMangaDownloadQueue)
	v1Manga.POST("/download-queue/start", h.HandleStartMangaDownloadQueue)
	v1Manga.POST("/download-queue/stop", h.HandleStopMangaDownloadQueue)
	v1Manga.DELETE("/download-queue", h.HandleClearAllChapterDownloadQueue)
	v1Manga.POST("/download-queue/reset-errored", h.HandleResetErroredChapterDownloadQueue)

	v1Manga.POST("/search", h.HandleMangaManualSearch)
	v1Manga.POST("/manual-mapping", h.HandleMangaManualMapping)
	v1Manga.POST("/get-mapping", h.HandleGetMangaMapping)
	v1Manga.POST("/remove-mapping", h.HandleRemoveMangaMapping)

	//
	// Character
	//

	v1Character := v1.Group("/character")
	v1Character.GET("/:id", h.HandleGetCharacterDetails)
	v1Character.POST("/:id/media", h.HandleGetCharacterMedia)

	//
	// File Cache
	//

	v1FileCache := v1.Group("/filecache")
	v1FileCache.GET("/total-size", h.HandleGetFileCacheTotalSize)
	v1FileCache.DELETE("/bucket", h.HandleRemoveFileCacheBucket)
	v1FileCache.GET("/mediastream/videofiles/total-size", h.HandleGetFileCacheMediastreamVideoFilesTotalSize)
	v1FileCache.DELETE("/mediastream/videofiles", h.HandleClearFileCacheMediastreamVideoFiles)

	//
	// Discord
	//

	v1Discord := protected.Group("/discord")
	v1Discord.POST("/presence/manga", h.HandleSetDiscordMangaActivity)
	v1Discord.POST("/presence/legacy-anime", h.HandleSetDiscordLegacyAnimeActivity)
	v1Discord.POST("/presence/anime", h.HandleSetDiscordAnimeActivityWithProgress)
	v1Discord.POST("/presence/anime-update", h.HandleUpdateDiscordAnimeActivityWithProgress)
	v1Discord.POST("/presence/cancel", h.HandleCancelDiscordActivity)

	//
	// Media Stream
	//
	protected.GET("/mediastream/settings", h.HandleGetMediastreamSettings)
	protected.PATCH("/mediastream/settings", h.HandleSaveMediastreamSettings)
	protected.POST("/mediastream/request", h.HandleRequestMediastreamMediaContainer)
	protected.POST("/mediastream/preload", h.HandlePreloadMediastreamMediaContainer)
	// Transcode
	protected.POST("/mediastream/shutdown-transcode", h.HandleMediastreamShutdownTranscodeStream)
	protected.GET("/mediastream/transcode/*", h.HandleMediastreamTranscode)
	protected.GET("/mediastream/subs/*", h.HandleMediastreamGetSubtitles)
	protected.GET("/mediastream/att/*", h.HandleMediastreamGetAttachments)
	protected.GET("/mediastream/direct", h.HandleMediastreamDirectPlay)
	protected.HEAD("/mediastream/direct", h.HandleMediastreamDirectPlay)
	protected.GET("/mediastream/file/*", h.HandleMediastreamFile)

	//
	// Torrent stream
	//
	protected.GET("/torrentstream/episodes/:id", h.HandleGetTorrentstreamEpisodeCollection)
	protected.GET("/torrentstream/settings", h.HandleGetTorrentstreamSettings)
	protected.PATCH("/torrentstream/settings", h.HandleSaveTorrentstreamSettings)
	protected.POST("/torrentstream/start", h.HandleTorrentstreamStartStream)
	protected.POST("/torrentstream/stop", h.HandleTorrentstreamStopStream)
	protected.POST("/torrentstream/drop", h.HandleTorrentstreamDropTorrent)
	protected.POST("/torrentstream/torrent-file-previews", h.HandleGetTorrentstreamTorrentFilePreviews)
	protected.POST("/torrentstream/batch-history", h.HandleGetTorrentstreamBatchHistory)
	protected.GET("/torrentstream/stream/*", echo.WrapHandler(h.HandleTorrentstreamServeStream()))

	//
	// Extensions
	//

	v1Extensions := protected.Group("/extensions")
	v1Extensions.POST("/playground/run", h.HandleRunExtensionPlaygroundCode)
	v1Extensions.POST("/external/fetch", h.HandleFetchExternalExtensionData)
	v1Extensions.POST("/external/install", h.HandleInstallExternalExtension)
	v1Extensions.POST("/external/uninstall", h.HandleUninstallExternalExtension)
	v1Extensions.POST("/external/edit-payload", h.HandleUpdateExtensionCode)
	v1Extensions.POST("/external/reload", h.HandleReloadExternalExtensions)
	v1Extensions.POST("/external/reload", h.HandleReloadExternalExtension)
	v1Extensions.POST("/all", h.HandleGetAllExtensions)
	v1Extensions.GET("/updates", h.HandleGetExtensionUpdateData)
	v1Extensions.GET("/list", h.HandleListExtensionData)
	v1Extensions.GET("/payload/:id", h.HandleGetExtensionPayload)
	v1Extensions.GET("/list/development", h.HandleListDevelopmentModeExtensions)
	v1Extensions.GET("/list/manga-provider", h.HandleListMangaProviderExtensions)
	v1Extensions.GET("/list/onlinestream-provider", h.HandleListOnlinestreamProviderExtensions)
	v1Extensions.GET("/list/anime-torrent-provider", h.HandleListAnimeTorrentProviderExtensions)
	v1Extensions.GET("/user-config/:id", h.HandleGetExtensionUserConfig)
	v1Extensions.POST("/user-config", h.HandleSaveExtensionUserConfig)
	v1Extensions.GET("/marketplace", h.HandleGetMarketplaceExtensions)
	v1Extensions.GET("/plugin-settings", h.HandleGetPluginSettings)
	v1Extensions.POST("/plugin-settings/pinned-trays", h.HandleSetPluginSettingsPinnedTrays)
	v1Extensions.POST("/plugin-permissions/grant", h.HandleGrantPluginPermissions)

	//
	// Continuity
	//
	v1Continuity := protected.Group("/continuity")
	v1Continuity.PATCH("/item", h.HandleUpdateContinuityWatchHistoryItem)
	v1Continuity.GET("/item/:id", h.HandleGetContinuityWatchHistoryItem)
	v1Continuity.GET("/history", h.HandleGetContinuityWatchHistory)

	//
	// Sync
	//
	v1Sync := protected.Group("/sync")
	v1Sync.GET("/track", h.HandleSyncGetTrackedMediaItems)
	v1Sync.POST("/track", h.HandleSyncAddMedia)
	v1Sync.DELETE("/track", h.HandleSyncRemoveMedia)
	v1Sync.GET("/track/:id/:type", h.HandleSyncGetIsMediaTracked)
	v1Sync.POST("/local", h.HandleSyncLocalData)
	v1Sync.GET("/queue", h.HandleSyncGetQueueState)
	v1Sync.POST("/anilist", h.HandleSyncAnilistData)
	v1Sync.POST("/updated", h.HandleSyncSetHasLocalChanges)
	v1Sync.GET("/updated", h.HandleSyncGetHasLocalChanges)
	v1Sync.GET("/storage/size", h.HandleSyncGetLocalStorageSize)

	//
	// Debrid
	//

	v1Debrid := protected.Group("/debrid")
	v1Debrid.GET("/settings", h.HandleGetDebridSettings)
	v1Debrid.PATCH("/settings", h.HandleSaveDebridSettings)
	v1Debrid.POST("/torrents", h.HandleDebridAddTorrents)
	v1Debrid.POST("/torrents/download", h.HandleDebridDownloadTorrent)
	v1Debrid.POST("/torrents/cancel", h.HandleDebridCancelDownload)
	v1Debrid.DELETE("/torrent", h.HandleDebridDeleteTorrent)
	v1Debrid.GET("/torrents", h.HandleDebridGetTorrents)
	v1Debrid.POST("/torrents/info", h.HandleDebridGetTorrentInfo)
	v1Debrid.POST("/torrents/file-previews", h.HandleDebridGetTorrentFilePreviews)
	v1Debrid.POST("/stream/start", h.HandleDebridStartStream)
	v1Debrid.POST("/stream/cancel", h.HandleDebridCancelStream)

	//
	// Report
	//

	v1.POST("/report/issue", h.HandleSaveIssueReport)
	v1.GET("/report/issue/download", h.HandleDownloadIssueReport)
}

func (h *Handler) JSON(c echo.Context, code int, i interface{}) error {
	return c.JSON(code, i)
}

func (h *Handler) RespondWithData(c echo.Context, data interface{}) error {
	return c.JSON(200, NewDataResponse(data))
}

func (h *Handler) RespondWithError(c echo.Context, err error) error {
	return c.JSON(500, NewErrorResponse(err))
}

func headMethodMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if c.Request().Method == http.MethodHead {
			// Set the method to GET temporarily to reuse the handler
			c.Request().Method = http.MethodGet

			defer func() {
				c.Request().Method = http.MethodHead
			}() // Restore method after

			// Call the next handler and then clear the response body
			if err := next(c); err != nil {
				if err.Error() == echo.ErrMethodNotAllowed.Error() {
					return c.NoContent(http.StatusOK)
				}

				return err
			}
		}

		return next(c)
	}
}
