package handlers

import (
	"seanime/internal/api/anilist"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

// SessionMiddleware ensures each request has a valid session ID
// and sets up the appropriate AniList client for the session
func SessionMiddleware(h *Handler) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get the session
			sess, err := session.Get("session", c)
			if err != nil {
				h.App.Logger.Error().Err(err).Msg("Failed to get session")
				// Create a new session if there was an error
				sess, _ = session.Get("session", c)
			}

			// Check if the session has a session ID
			sessionID, ok := sess.Values["session_id"].(string)
			if !ok || sessionID == "" {
				// Generate a new session ID
				sessionID = uuid.New().String()
				sess.Values["session_id"] = sessionID
				sess.Save(c.Request(), c.Response())
			}

			// Store the session ID in the context for later use
			c.Set("session_id", sessionID)

			// Get the AniList token from the session
			anilistToken, ok := sess.Values["anilist_token"].(string)
			if ok && anilistToken != "" {
				// Store the token in the context
				c.Set("anilist_token", anilistToken)
			} else {
				// Try to get the token from the database using the session ID
				token := h.App.Database.GetAnilistTokenBySessionID(sessionID)
				if token != "" {
					// Store the token in the context and session
					c.Set("anilist_token", token)
					sess.Values["anilist_token"] = token
					sess.Save(c.Request(), c.Response())
				}
			}

			// Get or create a session-specific AniList client
			token := GetAnilistToken(c)
			if token != "" {
				// Create a client for this session
				client := anilist.NewAnilistClient(token)
				c.Set("anilist_client", client)
				
				// Update the last active time for this session
				go func(sid string) {
					account, err := h.App.Database.GetAccountBySessionID(sid)
					if err == nil && account != nil {
						account.LastActive = time.Now().Unix()
						h.App.Database.UpsertAccount(account)
					}
				}(sessionID)
			}

			// Continue with the request
			return next(c)
		}
	}
}

// GetSessionID retrieves the session ID from the context
func GetSessionID(c echo.Context) string {
	sessionID, ok := c.Get("session_id").(string)
	if !ok {
		return ""
	}
	return sessionID
}

// GetAnilistToken retrieves the AniList token from the context
func GetAnilistToken(c echo.Context) string {
	token, ok := c.Get("anilist_token").(string)
	if !ok {
		return ""
	}
	return token
}
