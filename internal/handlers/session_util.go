package handlers

import (
	echoSession "github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

// GetAniListTokenFromSession retrieves the AniList token from the current session.
func GetAniListTokenFromSession(c echo.Context) (string, error) {
	sess, err := echoSession.Get("session", c)
	if err != nil {
		return "", err
	}
	token, ok := sess.Values["anilist_token"].(string)
	if !ok {
		return "", nil
	}
	return token, nil
}
