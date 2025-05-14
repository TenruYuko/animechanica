package handlers

import (
	"context"
	"github.com/labstack/echo/v4"
	"seanime/internal/api/anilist"
)

// GetAnilistClientForSession returns an AniList client for the current session
func (h *Handler) GetAnilistClientForSession(c echo.Context) anilist.AnilistClient {
	// Check if we already have a client in the context
	if client, ok := c.Get("anilist_client").(anilist.AnilistClient); ok && client != nil {
		return client
	}
	
	// Get the token from the context
	token := GetAnilistToken(c)
	if token == "" {
		// If no token in context, use the default client
		return h.App.AnilistClient
	}

	// Create a new client with the session token
	client := anilist.NewAnilistClient(token)
	
	// Store it in the context for future use
	c.Set("anilist_client", client)
	
	return client
}

// GetAnilistTokenFromSession retrieves the AniList token from the current session
func (h *Handler) GetAnilistTokenFromSession(c echo.Context) string {
	// Get the session ID from the context
	sessionID := GetSessionID(c)
	if sessionID == "" {
		// If no session ID, use the default token
		return h.App.GetAccountToken()
	}

	// Get the token from the database using the session ID
	return h.App.Database.GetAnilistTokenBySessionID(sessionID)
}

// GetViewerForSession retrieves the AniList viewer data for the current session
func (h *Handler) GetViewerForSession(c echo.Context) (anilist.GetViewer_Viewer, error) {
	// Get a client for this session
	client := h.GetAnilistClientForSession(c)

	// Get the viewer data
	getViewer, err := client.GetViewer(context.Background())
	if err != nil {
		return anilist.GetViewer_Viewer{}, err
	}

	// Dereference the pointer to return the actual value
	if getViewer.Viewer != nil {
		return *getViewer.Viewer, nil
	}
	
	return anilist.GetViewer_Viewer{}, nil
}

// UpdateAnilistClientTokenForSession updates the AniList client token for the current session
func (h *Handler) UpdateAnilistClientTokenForSession(c echo.Context, token string) {
	// Set the token in the context
	c.Set("anilist_token", token)
	
	// For the current request, use a client with this token
	client := anilist.NewAnilistClient(token)
	
	// Execute the request with this client
	c.Set("anilist_client", client)
}
