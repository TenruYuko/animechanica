package handlers

import (
	"github.com/labstack/echo/v4"
)

// HandleTestSession is a test endpoint to verify that the session management is working
// It returns the username of the current session if one exists
func (h *Handler) HandleTestSession(c echo.Context) error {
	// Get the session cookie
	sessionCookie, err := c.Cookie("Seanime-Session-Id")
	if err != nil || sessionCookie.Value == "" {
		return h.RespondWithData(c, map[string]interface{}{
			"authenticated": false,
			"message":      "No session found",
		})
	}

	// Check if the session exists and is valid
	session, err := h.App.Database.GetUserSessionByID(sessionCookie.Value)
	if err != nil {
		return h.RespondWithData(c, map[string]interface{}{
			"authenticated": false,
			"message":      "Session not found or expired",
		})
	}

	return h.RespondWithData(c, map[string]interface{}{
		"authenticated": true,
		"username":     session.Username,
		"sessionId":    session.SessionID,
		"expiresAt":    session.ExpiresAt,
		"lastActive":   session.LastActive,
	})
}
