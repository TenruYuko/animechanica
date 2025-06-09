package db

import (
	"errors"
	"gorm.io/gorm/clause"
	"seanime/internal/database/models"
	"time"
)

// CreateUserSession creates a new user session with the given session ID and AniList token information
func (db *Database) CreateUserSession(session *models.UserSession) (*models.UserSession, error) {
	err := db.gormdb.Create(session).Error
	if err != nil {
		db.Logger.Error().Err(err).Msg("Failed to create user session in the database")
		return nil, err
	}

	return session, nil
}

// GetUserSessionByID retrieves a user session by its session ID
func (db *Database) GetUserSessionByID(sessionID string) (*models.UserSession, error) {
	var session models.UserSession
	err := db.gormdb.Where("session_id = ?", sessionID).First(&session).Error
	if err != nil {
		return nil, err
	}

	// Check if the session has expired
	if time.Now().After(session.ExpiresAt) {
		// Delete the expired session
		db.DeleteUserSession(sessionID)
		return nil, errors.New("session expired")
	}

	// Update the last active timestamp
	session.LastActive = time.Now()
	db.gormdb.Save(&session)

	return &session, nil
}

// UpdateUserSession updates an existing user session
func (db *Database) UpdateUserSession(session *models.UserSession) (*models.UserSession, error) {
	err := db.gormdb.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "session_id"}},
		UpdateAll: true,
	}).Create(session).Error

	if err != nil {
		db.Logger.Error().Err(err).Msg("Failed to update user session in the database")
		return nil, err
	}

	return session, nil
}

// DeleteUserSession deletes a user session by its session ID
func (db *Database) DeleteUserSession(sessionID string) error {
	err := db.gormdb.Where("session_id = ?", sessionID).Delete(&models.UserSession{}).Error
	if err != nil {
		db.Logger.Error().Err(err).Msg("Failed to delete user session from the database")
		return err
	}

	return nil
}

// CleanupExpiredSessions removes all expired sessions from the database
func (db *Database) CleanupExpiredSessions() error {
	err := db.gormdb.Where("expires_at < ?", time.Now()).Delete(&models.UserSession{}).Error
	if err != nil {
		db.Logger.Error().Err(err).Msg("Failed to clean up expired sessions")
		return err
	}

	return nil
}
