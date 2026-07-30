package domain

import "time"

type ServerHttpConfig struct {
	ServerID      uint      `gorm:"primaryKey;autoIncrement:false"`
	Port          int       `gorm:"type:int;not null"`
	EndpointPath  string    `gorm:"type:varchar(500);not null;default:''"`
	ExpectedCode  int       `gorm:"type:int;not null;default:0"`
	BodyCheckExpr string    `gorm:"type:varchar(500);not null;default:''"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (ServerHttpConfig) TableName() string {
	return "server_http_configs"
}
