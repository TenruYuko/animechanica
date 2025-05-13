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
	var req struct {
		Code string `json:"code"`
	}
	if err := c.Bind(&req); err != nil || req.Code == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing or invalid code in request body"})
	}

	clientID := os.Getenv("ANILIST_CLIENT_ID")
	clientSecret := os.Getenv("ANILIST_CLIENT_SECRET")
	redirectURI := os.Getenv("ANILIST_REDIRECT_URI")
	if clientID == "" { clientID = "26797" }
	if clientSecret == "" { clientSecret = "eOXJYYPnOLQhwTudR3wrakUfMZgRi6U5dyHMcyYw" }
	if redirectURI == "" { redirectURI = "http://localhost:43211/auth/callback" }

	body := map[string]string{
		"grant_type": "authorization_code",
		"client_id": clientID,
		"client_secret": clientSecret,
		"redirect_uri": redirectURI,
		"code": req.Code,
	}
	jsonBody, _ := json.Marshal(body)

	resp, err := http.Post("https://anilist.co/api/v2/oauth/token", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to exchange code", "details": err.Error()})
	}
	defer resp.Body.Close()
	respBody, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return c.JSON(resp.StatusCode, map[string]string{"error": "AniList token exchange failed", "details": string(respBody)})
	}

	return c.JSON(http.StatusOK, json.RawMessage(respBody))
}

