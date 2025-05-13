package handlers

import (
	"context"
	"errors"
	"seanime/internal/database/models"
	"seanime/internal/util"
	"time"
	"net/http"

	"github.com/goccy/go-json"
	"github.com/labstack/echo/v4"
)

// HandleLogin
//
//	@summary logs in the user by saving the JWT token in the database.
//	@desc This is called when the JWT token is obtained from AniList after logging in with redirection on the client.
//	@desc It also fetches the Viewer data from AniList and saves it in the database.
//	@desc It creates a new handlers.Status and refreshes App modules.
//	@route /api/v1/auth/login [POST]
//	@returns handlers.Status
func (h *Handler) HandleLogin(c echo.Context) error {

	type body struct {
		Token string `json:"token"`
	}

	var b body

	if err := c.Bind(&b); err != nil {
		return h.RespondWithError(c, err)
	}

	// Set a new AniList client by passing to JWT token
	h.App.UpdateAnilistClientToken(b.Token)

	// Set AniList token as a secure, HTTP-only cookie (per user)
	cookie := new(http.Cookie)
	cookie.Name = "AniList-Token"
	cookie.Value = b.Token
	cookie.HttpOnly = true
	cookie.Secure = true // Set to false if not using HTTPS in dev
	cookie.Path = "/"
	cookie.Expires = time.Now().Add(7 * 24 * time.Hour)
	cookie.SameSite = http.SameSiteLaxMode
	c.SetCookie(cookie)

	// Get viewer data from AniList
	getViewer, err := h.App.AnilistClient.GetViewer(context.Background())
	if err != nil {
		h.App.Logger.Error().Msg("Could not authenticate to AniList")
		return h.RespondWithError(c, err)
	}

	if len(getViewer.Viewer.Name) == 0 {
		return h.RespondWithError(c, errors.New("could not find user"))
	}

	// Marshal viewer data
	bytes, err := json.Marshal(getViewer.Viewer)
	if err != nil {
		h.App.Logger.Err(err).Msg("scan: could not save local files")
	}

	// Save account data in database
	_, err = h.App.Database.UpsertAccount(&models.Account{
		BaseModel: models.BaseModel{
			ID:        1,
			UpdatedAt: time.Now(),
		},
		Username: getViewer.Viewer.Name,
		Token:    b.Token,
		Viewer:   bytes,
	})

	if err != nil {
		return h.RespondWithError(c, err)
	}

	h.App.Logger.Info().Msg("app: Authenticated to AniList")

	// Create a new status
	status := h.NewStatus(c)

	h.App.InitOrRefreshAnilistData()

	h.App.InitOrRefreshModules()

	go func() {
		defer util.HandlePanicThen(func() {})
		h.App.InitOrRefreshTorrentstreamSettings()
		h.App.InitOrRefreshMediastreamSettings()
		h.App.InitOrRefreshDebridSettings()
	}()

	// Return new status
	return h.RespondWithData(c, status)

}

// HandleLogout
//
//	@summary logs out the user by removing JWT token from the database.
//	@desc It removes JWT token and Viewer data from the database.
//	@desc It creates a new handlers.Status and refreshes App modules.
//	@route /api/v1/auth/logout [POST]
//	@returns handlers.Status
func (h *Handler) HandleLogout(c echo.Context) error {
	// Remove AniList token cookie for this user only
	cookie := new(http.Cookie)
	cookie.Name = "AniList-Token"
	cookie.Value = ""
	cookie.HttpOnly = true
	cookie.Secure = true // Set to false if not using HTTPS in dev
	cookie.Path = "/"
	cookie.Expires = time.Unix(0, 0) // Expire now
	cookie.SameSite = http.SameSiteLaxMode
	c.SetCookie(cookie)


	_, err := h.App.Database.UpsertAccount(&models.Account{
		BaseModel: models.BaseModel{
			ID:        1,
			UpdatedAt: time.Now(),
		},
		Username: "",
		Token:    "",
		Viewer:   nil,
	})

	if err != nil {
		return h.RespondWithError(c, err)
	}

	h.App.Logger.Info().Msg("Logged out of AniList")

	status := h.NewStatus(c)

	h.App.InitOrRefreshModules()

	h.App.InitOrRefreshAnilistData()

	return h.RespondWithData(c, status)
}
