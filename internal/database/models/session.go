package models

import (
	"time"
)

// UserSession represents a user session with an associated AniList token
type UserSession struct {
	BaseModel
	SessionID  string    `gorm:"column:session_id;uniqueIndex" json:"sessionId"`
	Username   string    `gorm:"column:username" json:"username"`
	Token      string    `gorm:"column:token" json:"token"`
	Viewer     []byte    `gorm:"column:viewer" json:"viewer"`
	ExpiresAt  time.Time `gorm:"column:expires_at" json:"expiresAt"`
	LastActive time.Time `gorm:"column:last_active" json:"lastActive"`
}
