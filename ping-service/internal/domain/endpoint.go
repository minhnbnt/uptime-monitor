package domain

import (
	"time"

	"gorm.io/gorm"
)

type Server struct {
	gorm.Model
	ServerID      uint          `gorm:"not null;uniqueIndex"`
	Namespace     string        `gorm:"type:varchar(253);not null"`
	Kind          string        `gorm:"type:varchar(50);not null"`
	ObjectID      string        `gorm:"type:varchar(253);not null"`
	ContainerName string        `gorm:"type:varchar(253)"`
	Interval      time.Duration `gorm:"type:bigint;not null;default:30000000000"`
	Timeout       time.Duration `gorm:"type:bigint;not null;default:10000000000"`

	// Deprecated: kept for dead code compatibility (responseChecker.go).
	ExpectedCode  int     `gorm:"-"`
	BodyCheckExpr *string `gorm:"-"`
}

func (Server) TableName() string {
	return "servers"
}

// Endpoint is a type alias for backward compatibility with dead code.
type Endpoint = Server
