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
		rank int64
	}

	res := db.Raw(`
		SELECT rank
		FROM (
		  SELECT
		    referrer_id,
		    RANK() OVER (
		      ORDER BY COUNT(referrer_id) DESC
		    )
		    from signatures
		    GROUP BY referrer_id
		)
		AS ranked
		WHERE ranked.referrer_id = ?
	`, userId).Scan(&rank)

	if res.Error != nil {
		return 0, res.Error
	}

	return rank.rank, nil
}
