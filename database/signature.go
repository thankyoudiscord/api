package database

import (
	"gorm.io/gorm"
)

type Signature struct {
	gorm.Model `json:"-"`

	UserID     string  `json:"user_id" gorm:"uniqueIndex;not null"`
	ReferrerID *string `json:"referrer_id"`
}
