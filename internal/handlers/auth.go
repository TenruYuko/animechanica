package handlers

import (
	"context"
	"errors"
	"seanime/internal/database/models"
	"seanime/internal/util"
	"time"

	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
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
	sess, _ := session.Get("session", c)
	type body struct {
		Token string `json:"token"`
	}
	var b body
	if err := c.Bind(&b); err != nil {
		return h.RespondWithError(c, err)
	}

	// Generate a unique session ID if not already present
	sessionID, ok := sess.Values["session_id"].(string)
	if !ok || sessionID == "" {
		sessionID = uuid.New().String()
		sess.Values["session_id"] = sessionID
	}

	// Store the AniList token in the session
	sess.Values["anilist_token"] = b.Token
	sess.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   60 * 60 * 24 * 7, // 1 week
		HttpOnly: true,
	}
	sess.Save(c.Request(), c.Response())

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

	// Save account data in database with the session ID
	_, err = h.App.Database.UpsertAccount(&models.Account{
		BaseModel: models.BaseModel{
			UpdatedAt: time.Now(),
		},
		Username:  getViewer.Viewer.Name,
		Token:     b.Token,
		Viewer:    bytes,
		SessionID: sessionID,
		IsActive:  true,
	})

	if err != nil {
		return h.RespondWithError(c, err)
	}

	h.App.Logger.Info().Str("username", getViewer.Viewer.Name).Str("sessionID", sessionID).Msg("app: Authenticated to AniList")

	// Create a new status
	status := h.NewStatus(c)

	// Initialize or refresh AniList data for this session
	h.App.InitOrRefreshAnilistData()

	// Initialize or refresh modules
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
	sess, _ := session.Get("session", c)

	// Get the session ID
	sessionID, ok := sess.Values["session_id"].(string)
	if ok && sessionID != "" {
		// Deactivate the session in the database
		err := h.App.Database.DeactivateSession(sessionID)
		if err != nil {
			h.App.Logger.Error().Err(err).Str("sessionID", sessionID).Msg("Failed to deactivate session")
		}

		h.App.Logger.Info().Str("sessionID", sessionID).Msg("Logged out of AniList")
	}

	// Invalidate the browser session
	sess.Options.MaxAge = -1
	sess.Save(c.Request(), c.Response())

	// Create a new status
	status := h.NewStatus(c)

	return h.RespondWithData(c, status)
}

// HandleListSessions
//
//	@summary lists all active sessions for the administrator.
//	@desc This endpoint is only accessible to administrators.
//	@route /api/v1/auth/sessions [GET]
//	@returns []models.Account
func (h *Handler) HandleListSessions(c echo.Context) error {
	// In a real implementation, you would check if the user is an administrator
	// For now, we'll just return all active sessions
	sessions, err := h.App.Database.ListActiveSessions()
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Remove sensitive information
	for _, s := range sessions {
		s.Token = "[redacted]"
		s.Viewer = nil
	}

	return h.RespondWithData(c, sessions)
}
