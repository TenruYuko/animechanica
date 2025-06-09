package qbittorrent

import (
	"fmt"
	"io"
	"github.com/rs/zerolog"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"seanime/internal/torrent_clients/qbittorrent/application"
	"seanime/internal/torrent_clients/qbittorrent/log"
	"seanime/internal/torrent_clients/qbittorrent/rss"
	"seanime/internal/torrent_clients/qbittorrent/search"
	"seanime/internal/torrent_clients/qbittorrent/sync"
	"seanime/internal/torrent_clients/qbittorrent/torrent"
	"seanime/internal/torrent_clients/qbittorrent/transfer"
	"strings"

	"golang.org/x/net/publicsuffix"
)

type Client struct {
	baseURL          string
	logger           *zerolog.Logger
	client           *http.Client
	Username         string
	Password         string
	Port             int
	Host             string
	Path             string
	DisableBinaryUse bool
	Tags             string
	Application      qbittorrent_application.Client
	Log              qbittorrent_log.Client
	RSS              qbittorrent_rss.Client
	Search           qbittorrent_search.Client
	Sync             qbittorrent_sync.Client
	Torrent          qbittorrent_torrent.Client
	Transfer         qbittorrent_transfer.Client
}

type NewClientOptions struct {
	Logger           *zerolog.Logger
	Username         string
	Password         string
	Port             int
	Host             string
	Path             string
	DisableBinaryUse bool
	Tags             string
}

func NewClient(opts *NewClientOptions) *Client {
	baseURL := fmt.Sprintf("http://%s:%d/api/v2", opts.Host, opts.Port)

	if strings.HasPrefix(opts.Host, "https://") {
		opts.Host = strings.TrimPrefix(opts.Host, "https://")
		baseURL = fmt.Sprintf("https://%s:%d/api/v2", opts.Host, opts.Port)
	}

	client := &http.Client{}
	return &Client{
		baseURL:          baseURL,
		logger:           opts.Logger,
		client:           client,
		Username:         opts.Username,
		Password:         opts.Password,
		Port:             opts.Port,
		Path:             opts.Path,
		DisableBinaryUse: opts.DisableBinaryUse,
		Host:             opts.Host,
		Tags:             opts.Tags,
		Application: qbittorrent_application.Client{
			BaseUrl: baseURL + "/app",
			Client:  client,
			Logger:  opts.Logger,
		},
		Log: qbittorrent_log.Client{
			BaseUrl: baseURL + "/log",
			Client:  client,
			Logger:  opts.Logger,
		},
		RSS: qbittorrent_rss.Client{
			BaseUrl: baseURL + "/rss",
			Client:  client,
			Logger:  opts.Logger,
		},
		Search: qbittorrent_search.Client{
			BaseUrl: baseURL + "/search",
			Client:  client,
			Logger:  opts.Logger,
		},
		Sync: qbittorrent_sync.Client{
			BaseUrl: baseURL + "/sync",
			Client:  client,
			Logger:  opts.Logger,
		},
		Torrent: qbittorrent_torrent.Client{
			BaseUrl: baseURL + "/torrents",
			Client:  client,
			Logger:  opts.Logger,
		},
		Transfer: qbittorrent_transfer.Client{
			BaseUrl: baseURL + "/transfer",
			Client:  client,
			Logger:  opts.Logger,
		},
	}
}

func (c *Client) Login() error {
	// Create a new HTTP client with a cookie jar for each login attempt
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return fmt.Errorf("failed to create cookie jar: %w", err)
	}
	c.client = &http.Client{Jar: jar}
	
	// Construct the base URL with the correct host and port
	c.baseURL = fmt.Sprintf("http://%s:%d/api/v2", c.Host, c.Port)
	c.logger.Debug().Str("baseURL", c.baseURL).Msg("qBittorrent: Constructed base URL")
	
	// Construct the login endpoint
	endpoint := c.baseURL + "/auth/login"
	c.logger.Debug().Str("endpoint", endpoint).Msg("qBittorrent: Login endpoint")
	
	// Prepare login form data
	data := url.Values{}
	data.Add("username", c.Username)
	data.Add("password", c.Password)
	
	// Create the request
	request, err := http.NewRequest("POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	request.Header.Add("content-type", "application/x-www-form-urlencoded")
	
	// Execute the request
	c.logger.Debug().Msg("qBittorrent: Sending login request")
	resp, err := c.client.Do(request)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	
	// Read response body for debugging
	bodyBytes, _ := io.ReadAll(resp.Body)
	respBody := string(bodyBytes)
	
	// Close response body
	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.logger.Err(err).Msg("failed to close login response body")
		}
	}()
	
	// Check status code
	c.logger.Debug().Int("statusCode", resp.StatusCode).Str("status", resp.Status).Str("responseBody", respBody).Msg("qBittorrent: Login response")
	if resp.StatusCode != 200 {
		return fmt.Errorf("invalid status %s: %s", resp.Status, respBody)
	}
	
	// Check for cookies
	c.logger.Debug().Int("cookieCount", len(resp.Cookies())).Msg("qBittorrent: Cookies in response")
	for i, cookie := range resp.Cookies() {
		c.logger.Debug().Int("index", i).Str("name", cookie.Name).Str("value", cookie.Value).Msg("qBittorrent: Cookie details")
	}
	
	if len(resp.Cookies()) < 1 {
		return fmt.Errorf("no cookies in login response: %s", respBody)
	}
	
	// Parse API URL
	apiURL, err := url.Parse(c.baseURL)
	if err != nil {
		return fmt.Errorf("failed to parse base URL: %w", err)
	}
	
	// Set cookies
	c.client.Jar.SetCookies(apiURL, resp.Cookies())
	c.logger.Debug().Msg("qBittorrent: Successfully logged in")
	
	return nil
}

func (c *Client) Logout() error {
	endpoint := c.baseURL + "/auth/logout"
	request, err := http.NewRequest("POST", endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(request)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("invalid status %s", resp.Status)
	}
	return nil
}
