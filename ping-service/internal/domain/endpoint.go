package domain

import (
	"time"

	"gorm.io/gorm"
)

type Endpoint struct {
	gorm.Model
	ServerID     uint          `gorm:"not null;uniqueIndex"`
	URL          string        `gorm:"type:text;not null"`
	Interval     time.Duration `gorm:"type:bigint;not null;default:30000000000"`
	Timeout      time.Duration `gorm:"type:bigint;not null;default:10000000000"`
	Method       string        `gorm:"type:varchar(10);not null;default:GET"`
	ExpectedCode int           `gorm:"type:int;not null;default:200"`
	BodyCheckExpr *string      `gorm:"type:text"`
}

func (Endpoint) TableName() string {
	return "endpoints"
}
