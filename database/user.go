package database

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model `json:"-"`

	UserID        string `json:"id" gorm:"uniqueIndex;not null"`
	Username      string `json:"username" gorm:"not null"`
	Discriminator string `json:"discriminator" gorm:"not null"`
	AvatarHash    string `json:"avatar" gorm:"not null"`

	Signature Signature `json:"-" gorm:"foreignKey:UserID;references:UserID"`
}
