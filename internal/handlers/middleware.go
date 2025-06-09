package handlers

import (
	"github.com/labstack/echo/v4"
)

// SessionMiddleware checks if the user has a valid session
// If not, it allows the request to continue but the frontend will handle the redirect
func (h *Handler) SessionMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Skip session check for certain paths
		path := c.Request().URL.Path
		
		// Skip session check for auth endpoints, status endpoint, and websocket events
		if path == "/api/v1/auth/login" || 
		   path == "/api/v1/auth/logout" || 
		   path == "/api/v1/auth/check-session" || 
		   path == "/api/v1/status" ||
		   path == "/events" {
			return next(c)
		}
		
		// Get the session cookie
		sessionCookie, err := c.Cookie("Seanime-Session-Id")
		if err != nil || sessionCookie.Value == "" {
			// No session cookie, but allow the request to continue
			// The frontend will handle the redirect based on the status response
			return next(c)
		}
		
		// Check if the session exists and is valid
		session, err := h.App.Database.GetUserSessionByID(sessionCookie.Value)
		if err != nil {
			// Session not found or expired, but allow the request to continue
			// The frontend will handle the redirect based on the status response
			return next(c)
		}
		
		// Set the AniList client token for this request
		h.App.UpdateAnilistClientToken(session.Token)
		
		// Set the username in the context for use in the request
		c.Set("Username", session.Username)
		c.Set("SessionID", session.SessionID)
		
		return next(c)
	}
}
