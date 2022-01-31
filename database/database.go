package database

import (
	"sync"

	"gorm.io/gorm"
)

var db *gorm.DB
var initOnce sync.Once

func InitDatabase(d *gorm.DB) {
	initOnce.Do(func() {
		d.AutoMigrate(&User{}, &Signature{})
		db = d
	})
}

func GetDatabase() *gorm.DB {
	return db
}

func GetUserPosition(db *gorm.DB, userId string) (int64, error) {
	var rank struct {
		Rank int64
	}

	res := db.Raw(`
		SELECT rank
		FROM (
			SELECT
				user_id,
				ROW_NUMBER() OVER (
					ORDER BY created_at
				) AS rank
			FROM signatures
		)
		AS ranked
		WHERE user_id = ?
	`, userId).Scan(&rank)

	if res.Error != nil {
		return 0, res.Error
	}

	return rank.Rank, nil
}
