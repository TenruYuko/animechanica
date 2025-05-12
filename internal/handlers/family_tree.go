package handlers

import (
	"strconv"
	"seanime/internal/library/anime"

	"github.com/labstack/echo/v4"
)

// HandleGetFamilyTree returns the family tree for a given AniList ID
func (h *Handler) HandleGetFamilyTree(c echo.Context) error {
	idStr := c.Param("id")
	anilistID, err := strconv.Atoi(idStr)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	canonical, chronological, alternatives, charactersFrom, err := anime.FindFamilyTree(anilistID)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, map[string]interface{}{
		"canonical": canonical,
		"chronological": chronological,
		"alternatives": alternatives,
		"charactersFrom": charactersFrom,
	})
}
