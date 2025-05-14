package db

import (
	"errors"
	"gorm.io/gorm/clause"
	"seanime/internal/database/models"
	"sync"
	"time"
)

// accountCache maps sessionID to account
var accountCache sync.Map

// UpsertAccount creates or updates an account
func (db *Database) UpsertAccount(acc *models.Account) (*models.Account, error) {
	// Set last active time
	acc.LastActive = time.Now().Unix()

	err := db.gormdb.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "session_id"}},
		UpdateAll: true,
	}).Create(acc).Error

	if err != nil {
		db.Logger.Error().Err(err).Msg("Failed to save account in the database")
		return nil, err
	}

	// Update cache
	accountCache.Store(acc.SessionID, acc)

	return acc, nil
}

// GetAccountBySessionID retrieves an account by its session ID
func (db *Database) GetAccountBySessionID(sessionID string) (*models.Account, error) {
	// Check cache first
	if acc, ok := accountCache.Load(sessionID); ok {
		account := acc.(*models.Account)
		// Update last active time
		account.LastActive = time.Now().Unix()
		return account, nil
	}

	// Not in cache, query database
	var acc models.Account
	err := db.gormdb.Where("session_id = ? AND is_active = ?", sessionID, true).First(&acc).Error
	if err != nil {
		return nil, err
	}

	if acc.Username == "" || acc.Token == "" || acc.Viewer == nil {
		return nil, errors.New("account does not exist or is incomplete")
	}

	// Update last active time
	acc.LastActive = time.Now().Unix()
	db.gormdb.Save(&acc)

	// Add to cache
	accountCache.Store(sessionID, &acc)

	return &acc, nil
}

// GetAccount returns the first active account (for backward compatibility)
func (db *Database) GetAccount() (*models.Account, error) {
	var acc models.Account
	err := db.gormdb.Where("is_active = ?", true).First(&acc).Error
	if err != nil {
		return nil, err
	}

	if acc.Username == "" || acc.Token == "" || acc.Viewer == nil {
		return nil, errors.New("account does not exist or is incomplete")
	}

	// Update last active time
	acc.LastActive = time.Now().Unix()
	db.gormdb.Save(&acc)

	// Add to cache
	accountCache.Store(acc.SessionID, &acc)

	return &acc, nil
}

// GetAnilistToken retrieves the AniList token from the account or returns an empty string
func (db *Database) GetAnilistToken() string {
	acc, err := db.GetAccount()
	if err != nil {
		return ""
	}
	return acc.Token
}

// GetAnilistTokenBySessionID retrieves the AniList token for a specific session
func (db *Database) GetAnilistTokenBySessionID(sessionID string) string {
	acc, err := db.GetAccountBySessionID(sessionID)
	if err != nil {
		return ""
	}
	return acc.Token
}

// DeactivateSession marks a session as inactive
func (db *Database) DeactivateSession(sessionID string) error {
	result := db.gormdb.Model(&models.Account{}).Where("session_id = ?", sessionID).Update("is_active", false)
	if result.Error != nil {
		return result.Error
	}

	// Remove from cache
	accountCache.Delete(sessionID)

	return nil
}

// ListActiveSessions returns all active sessions
func (db *Database) ListActiveSessions() ([]*models.Account, error) {
	var accounts []*models.Account
	err := db.gormdb.Where("is_active = ?", true).Find(&accounts).Error
	if err != nil {
		return nil, err
	}
	return accounts, nil
}
