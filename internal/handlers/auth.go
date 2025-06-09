package handlers

import (
	"context"
	"errors"
	"net/http"
	"seanime/internal/database/models"
	"seanime/internal/util"
	"time"

	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// HandleLogin
//
//	@summary logs in the user by saving the JWT token in the database and creating a session.
//	@desc This is called when the JWT token is obtained from AniList after logging in with redirection on the client.
//	@desc It also fetches the Viewer data from AniList and creates a user session.
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

	// Set a new AniList client by passing the JWT token
	h.App.UpdateAnilistClientToken(b.Token)

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

	// For backward compatibility, also save to the global account
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

	// Create a new session ID
	sessionID := uuid.New().String()

	// Create a session that expires in one week
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	// Create a new user session
	session := &models.UserSession{
		SessionID:  sessionID,
		Username:   getViewer.Viewer.Name,
		Token:      b.Token,
		Viewer:     bytes,
		ExpiresAt:  expiresAt,
		LastActive: time.Now(),
	}

	// Save the session in the database
	_, err = h.App.Database.CreateUserSession(session)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Create a session cookie
	sessionCookie := &http.Cookie{
		Name:     "Seanime-Session-Id",
		Value:    sessionID,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   c.Request().TLS != nil,
	}

	// Set the session cookie
	c.SetCookie(sessionCookie)

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
//	@summary logs out the user by removing their session.
//	@desc It removes the user's session and clears the session cookie.
//	@desc It creates a new handlers.Status and refreshes App modules.
//	@route /api/v1/auth/logout [POST]
//	@returns handlers.Status
func (h *Handler) HandleLogout(c echo.Context) error {
	// Get the session cookie
	sessionCookie, err := c.Cookie("Seanime-Session-Id")
	if err == nil && sessionCookie.Value != "" {
		// Delete the session from the database
		err = h.App.Database.DeleteUserSession(sessionCookie.Value)
		if err != nil {
			h.App.Logger.Error().Err(err).Msg("Failed to delete user session")
		}

		// Clear the session cookie
		c.SetCookie(&http.Cookie{
			Name:     "Seanime-Session-Id",
			Value:    "",
			Path:     "/",
			Expires:  time.Now().Add(-1 * time.Hour),
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   c.Request().TLS != nil,
		})
	}

	// For backward compatibility, also clear the global account
	_, err = h.App.Database.UpsertAccount(&models.Account{
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

// HandleCheckSession
//
//	@summary checks if the user has a valid session and redirects to login if not.
//	@desc This is called to verify if the user has a valid session.
//	@desc If not, it returns a response indicating the user should be redirected to the login page.
//	@route /api/v1/auth/check-session [GET]
//	@returns {redirectToLogin: boolean}
func (h *Handler) HandleCheckSession(c echo.Context) error {
	// Get the session cookie
	sessionCookie, err := c.Cookie("Seanime-Session-Id")
	if err != nil || sessionCookie.Value == "" {
		// No session cookie, redirect to login
		return h.RespondWithData(c, map[string]interface{}{
			"redirectToLogin": true,
		})
	}

	// Check if the session exists and is valid
	session, err := h.App.Database.GetUserSessionByID(sessionCookie.Value)
	if err != nil {
		// Session not found or expired, redirect to login
		return h.RespondWithData(c, map[string]interface{}{
			"redirectToLogin": true,
		})
	}

	// Update the AniList client with the session token
	h.App.UpdateAnilistClientToken(session.Token)

	// Valid session found
	return h.RespondWithData(c, map[string]interface{}{
		"redirectToLogin": false,
		"username":        session.Username,
	})
}
