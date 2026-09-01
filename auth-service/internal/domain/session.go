package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const SessionTableName = "sessions"

type Session struct {
	gorm.Model
	UserID    uint      `gorm:"not null;index"`
	JTI       uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	Counter   int64     `gorm:"not null;default:0"`
	Scopes    string    `gorm:"type:text;not null"`
	ExpiresAt time.Time `gorm:"not null;index"`
}

func (Session) TableName() string {
	return SessionTableName
}

func (s Session) ScopeList() []string {
	if s.Scopes == "" {
		return nil
	}
	return strings.Fields(s.Scopes)
}
