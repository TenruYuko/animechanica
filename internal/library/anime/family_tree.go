package anime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type OfflineDBEntry struct {
	Sources      []string            `json:"sources"`
	Title        string              `json:"title"`
	Type         string              `json:"type"`
	Relations    []OfflineDBRelation `json:"relations"`
	AnilistID    *int                `json:"anilist_id,omitempty"`
	MALID        *int                `json:"mal_id,omitempty"`
}

type OfflineDBRelation struct {
	RelationType string `json:"relationType"`
	ID           int    `json:"id"`
}

type AnimeOfflineDatabase struct {
	Data struct {
		Anime []OfflineDBEntry `json:"anime"`
	} `json:"data"`
}

var offlineDB *AnimeOfflineDatabase

func LoadAnimeOfflineDatabase(jsonPath string) error {
	f, err := os.Open(filepath.Clean(jsonPath))
	if err != nil {
		return fmt.Errorf("failed to open anime-offline-database.json: %w", err)
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	var db AnimeOfflineDatabase
	if err := dec.Decode(&db); err != nil {
		return fmt.Errorf("failed to decode anime-offline-database.json: %w", err)
	}
	offlineDB = &db
	return nil
}

// FindFamilyTree returns the canonical, chronological, alternative, and character-used-from relations for the given AniList ID
func FindFamilyTree(anilistID int) (canonical []OfflineDBEntry, chronological []OfflineDBEntry, alternatives []OfflineDBEntry, charactersFrom []OfflineDBEntry, err error) {
	if offlineDB == nil {
		err = fmt.Errorf("offline database not loaded")
		return
	}
	var thisEntry *OfflineDBEntry
	for i, entry := range offlineDB.Data.Anime {
		if entry.AnilistID != nil && *entry.AnilistID == anilistID {
			thisEntry = &offlineDB.Data.Anime[i]
			break
		}
	}
	if thisEntry == nil {
		err = fmt.Errorf("anime not found in offline database: %d", anilistID)
		return
	}
	for _, rel := range thisEntry.Relations {
		for i, e := range offlineDB.Data.Anime {
			if e.AnilistID != nil && *e.AnilistID == rel.ID {
				switch rel.RelationType {
				case "Prequel", "Sequel":
					canonical = append(canonical, offlineDB.Data.Anime[i])
				case "Alternative version", "Alternative setting":
					alternatives = append(alternatives, offlineDB.Data.Anime[i])
				case "Character":
					charactersFrom = append(charactersFrom, offlineDB.Data.Anime[i])
				case "Side story", "Summary", "Other":
					chronological = append(chronological, offlineDB.Data.Anime[i])
				}
			}
		}
	}
	return
}
