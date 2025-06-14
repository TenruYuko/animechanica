package manga

import (
	"bytes"
	"errors"
	"image"
	_ "image/jpeg" // Register JPEG format
	_ "image/png"  // Register PNG format
	"net/http"
	"seanime/internal/database/db"
	"seanime/internal/events"
	"seanime/internal/extension"
	manga_providers "seanime/internal/manga/providers"
	"seanime/internal/util/filecache"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	_ "golang.org/x/image/bmp"  // Register BMP format
	_ "golang.org/x/image/tiff" // Register Tiff format
	_ "golang.org/x/image/webp" // Register WebP format
)

var (
	ErrNoResults            = errors.New("no results found for this media")
	ErrNoChapters           = errors.New("no manga chapters found")
	ErrChapterNotFound      = errors.New("chapter not found")
	ErrChapterNotDownloaded = errors.New("chapter not downloaded")
	ErrNoTitlesProvided     = errors.New("no titles provided")
)

type (
	Repository struct {
		logger                *zerolog.Logger
		fileCacher            *filecache.Cacher
		cacheDir              string
		providerExtensionBank *extension.UnifiedBank
		serverUri             string
		wsEventManager        events.WSEventManagerInterface
		mu                    sync.Mutex
		downloadDir           string
		db                    *db.Database
	}

	NewRepositoryOptions struct {
		Logger         *zerolog.Logger
		CacheDir       string
		FileCacher     *filecache.Cacher
		ServerURI      string
		WsEventManager events.WSEventManagerInterface
		DownloadDir    string
		Database       *db.Database
	}
)

func NewRepository(opts *NewRepositoryOptions) *Repository {
	r := &Repository{
		logger:                opts.Logger,
		fileCacher:            opts.FileCacher,
		cacheDir:              opts.CacheDir,
		serverUri:             opts.ServerURI,
		wsEventManager:        opts.WsEventManager,
		downloadDir:           opts.DownloadDir,
		providerExtensionBank: extension.NewUnifiedBank(),
		db:                    opts.Database,
	}
	return r
}

func (r *Repository) InitExtensionBank(bank *extension.UnifiedBank) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providerExtensionBank = bank
	r.logger.Debug().Msg("manga: Initialized provider extension bank")
}

func (r *Repository) RemoveProvider(id string) {
	r.providerExtensionBank.Delete(id)
}

func (r *Repository) GetProviderExtensionBank() *extension.UnifiedBank {
	return r.providerExtensionBank
}

// GetInternalStorageProvider returns the internal storage provider for handling local manga files
// Returns nil if the internal storage provider is not found or not available
func (r *Repository) GetInternalStorageProvider() *manga_providers.InternalStorageProvider {
	if r.providerExtensionBank == nil {
		return nil
	}
	
	// Get the internal storage extension from the bank
	providerExtension, ok := extension.GetExtension[extension.MangaProviderExtension](r.providerExtensionBank, "internal-storage")
	if !ok {
		r.logger.Debug().Msg("manga: Internal storage provider extension not found")
		return nil
	}
	
	// Extract the underlying provider implementation
	provider := providerExtension.GetProvider()
	
	// Type assert to the internal storage provider type
	if internalProvider, ok := provider.(*manga_providers.InternalStorageProvider); ok {
		return internalProvider
	}
	
	r.logger.Debug().Msg("manga: Internal storage provider found but type assertion failed")
	return nil
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// File Cache
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type bucketType string

const (
	bucketTypeChapterKey                = "1"
	bucketTypeChapter        bucketType = "chapters"
	bucketTypePage           bucketType = "pages"
	bucketTypePageDimensions bucketType = "page-dimensions"
)

// getFcProviderBucket returns a bucket for the provider and mediaId.
//
//	e.g., manga_comick_chapters_123, manga_mangasee_pages_456
//
// Note: Each bucket contains only 1 key-value pair.
func (r *Repository) getFcProviderBucket(provider string, mediaId int, bucketType bucketType) filecache.Bucket {
	return filecache.NewBucket("manga_"+provider+"_"+string(bucketType)+"_"+strconv.Itoa(mediaId), time.Hour*24*7)
}

// EmptyMangaCache deletes all manga buckets associated with the given mediaId.
func (r *Repository) EmptyMangaCache(mediaId int) (err error) {
	// Empty the manga cache
	err = r.fileCacher.RemoveAllBy(func(filename string) bool {
		return strings.HasPrefix(filename, "manga_") && strings.Contains(filename, strconv.Itoa(mediaId))
	})
	return
}

func ParseChapterContainerFileName(filename string) (provider string, bucketType bucketType, mediaId int, ok bool) {
	filename = strings.TrimSuffix(filename, ".json")
	filename = strings.TrimSuffix(filename, ".cache")
	filename = strings.TrimSuffix(filename, ".txt")
	parts := strings.Split(filename, "_")
	if len(parts) != 4 {
		return "", "", 0, false
	}

	provider = parts[1]
	var err error
	mediaId, err = strconv.Atoi(parts[3])
	if err != nil {
		return "", "", 0, false
	}

	switch parts[2] {
	case "chapters":
		bucketType = bucketTypeChapter
	case "pages":
		bucketType = bucketTypePage
	case "page-dimensions":
		bucketType = bucketTypePageDimensions
	default:
		return "", "", 0, false
	}

	ok = true
	return
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func getImageNaturalSize(url string) (int, int, error) {
	// Fetch the image
	resp, err := http.Get(url)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	// Decode the image
	img, _, err := image.DecodeConfig(resp.Body)
	if err != nil {
		return 0, 0, err
	}

	// Return the natural size
	return img.Width, img.Height, nil
}

func getImageNaturalSizeB(data []byte) (int, int, error) {
	// Decode the image
	img, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return 0, 0, err
	}

	// Return the natural size
	return img.Width, img.Height, nil
}
