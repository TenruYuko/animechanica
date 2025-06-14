package handlers

import (
	"errors"
	"seanime/internal/api/anilist"
	"strconv"

	"github.com/labstack/echo/v4"
)

// HandleGetCharacterDetails
//
//	@summary returns detailed information about a character by ID.
//	@desc This endpoint fetches comprehensive character data from AniList including character details, media appearances, and voice actors.
//	@route /api/v1/character/{id} [GET]
//	@returns anilist.CharacterDetailsByID_Character
func (h *Handler) HandleGetCharacterDetails(c echo.Context) error {
	
	// Get character ID from URL parameter
	characterIdStr := c.Param("id")
	if characterIdStr == "" {
		return h.RespondWithError(c, errors.New("character ID is required"))
	}
	
	characterId, err := strconv.Atoi(characterIdStr)
	if err != nil {
		return h.RespondWithError(c, errors.New("invalid character ID"))
	}
	
	// Get AniList client
	anilistClient := h.App.AnilistClient
	if anilistClient == nil {
		return h.RespondWithError(c, errors.New("AniList client not available"))
	}
	
	// Fetch character details from AniList
	character, err := anilistClient.CharacterDetailsByID(c.Request().Context(), characterId)
	if err != nil {
		h.App.Logger.Error().Err(err).Int("characterId", characterId).Msg("Failed to fetch character details")
		return h.RespondWithError(c, err)
	}
	
	if character == nil || character.Character == nil {
		return h.RespondWithError(c, errors.New("character not found"))
	}
	
	// Return character details
	var result *anilist.CharacterDetailsByID_Character = character.Character
	return h.RespondWithData(c, result)
}

// HandleGetCharacterMedia
//
//	@summary returns media appearances for a character with pagination.
//	@desc This endpoint fetches paginated media appearances for a character from AniList.
//	@route /api/v1/character/{id}/media [POST]
//	@returns anilist.CharacterDetailsByID_Character_Media
func (h *Handler) HandleGetCharacterMedia(c echo.Context) error {
	
	type body struct {
		Page    int `json:"page"`
		PerPage int `json:"perPage"`
	}
	
	var b body
	if err := c.Bind(&b); err != nil {
		return h.RespondWithError(c, err)
	}
	
	// Set defaults
	if b.Page <= 0 {
		b.Page = 1
	}
	if b.PerPage <= 0 {
		b.PerPage = 25
	}
	if b.PerPage > 50 {
		b.PerPage = 50 // Limit to avoid excessive requests
	}
	
	// Get character ID from URL parameter
	characterIdStr := c.Param("id")
	if characterIdStr == "" {
		return h.RespondWithError(c, errors.New("character ID is required"))
	}
	
	characterId, err := strconv.Atoi(characterIdStr)
	if err != nil {
		return h.RespondWithError(c, errors.New("invalid character ID"))
	}
	
	// Get AniList client
	anilistClient := h.App.AnilistClient
	if anilistClient == nil {
		return h.RespondWithError(c, errors.New("AniList client not available"))
	}
	
	// Fetch character details including media
	character, err := anilistClient.CharacterDetailsByID(c.Request().Context(), characterId)
	if err != nil {
		h.App.Logger.Error().Err(err).Int("characterId", characterId).Msg("Failed to fetch character media")
		return h.RespondWithError(c, err)
	}
	
	if character == nil || character.Character == nil || character.Character.Media == nil {
		return h.RespondWithError(c, errors.New("character media not found"))
	}
	
	// Return character media
	var mediaResult *anilist.CharacterDetailsByID_Character_Media = character.Character.Media
	return h.RespondWithData(c, mediaResult)
}
