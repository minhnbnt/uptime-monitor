package domain

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Email    string `gorm:"type:varchar(255);uniqueIndex;not null"`
	Username string `gorm:"type:varchar(100);uniqueIndex;not null"`
	Password string `gorm:"type:text;not null"`
	Name     string `gorm:"type:varchar(255);not null"`
}

func (User) TableName() string {
	return "users"
}
