package handlers

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"github.com/labstack/echo/v4"
)

// HandleAniListOAuthCallback handles the OAuth callback and exchanges code for token
func (h *Handler) HandleAniListOAuthCallback(c echo.Context) error {
	code := c.QueryParam("code")
	if code == "" {
		return c.String(http.StatusBadRequest, "Missing code parameter")
	}

	clientID := os.Getenv("ANILIST_CLIENT_ID")
	clientSecret := os.Getenv("ANILIST_CLIENT_SECRET")
	redirectURI := os.Getenv("ANILIST_REDIRECT_URI")
	if clientID == "" { clientID = "26797" }
	if clientSecret == "" { clientSecret = "eOXJYYPnOLQhwTudR3wrakUfMZgRi6U5dyHMcyYw" }
	if redirectURI == "" { redirectURI = "http://localhost:43211/auth/callback" }

	// Prepare request body
	body := map[string]string{
		"grant_type": "authorization_code",
		"client_id": clientID,
		"client_secret": clientSecret,
		"redirect_uri": redirectURI,
		"code": code,
	}
	jsonBody, _ := json.Marshal(body)

	// Exchange code for token
	resp, err := http.Post("https://anilist.co/api/v2/oauth/token", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to exchange code: "+err.Error())
	}
	defer resp.Body.Close()
	respBody, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return c.String(resp.StatusCode, string(respBody))
	}

	// Parse access token from AniList response
	type AniListTokenResponse struct {
		AccessToken string `json:"access_token"`
	}
	var tokenResp AniListTokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil || tokenResp.AccessToken == "" {
		return c.String(http.StatusInternalServerError, "Failed to parse AniList access token: "+string(respBody))
	}

	// Redirect to frontend callback with token
	return c.Redirect(http.StatusFound, "/auth/callback?token="+tokenResp.AccessToken)
}
