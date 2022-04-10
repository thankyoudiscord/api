package cache

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/thankyoudiscord/api/pkg/protos"
	protobuf "google.golang.org/protobuf/proto"
)

const BANNER_CACHE_KEY = "banner"
const BANNER_STALE_KEY = "banner_stale"
const BANNER_CACHE_TTL = time.Second * 30

type BannerCache struct {
	RedisClient *redis.Client
}

var cacheSingleton BannerCache

func (bc BannerCache) Set(bannerResp *protos.CreateBannerResponse) error {
	b, err := protobuf.Marshal(bannerResp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to serialize banner protobuf: %v\n", err)
		return err
	}

	res := bc.RedisClient.Set(
		context.Background(),
		BANNER_STALE_KEY,
		b,
		0,
	)

	if res.Err() != nil {
		fmt.Fprintf(os.Stderr, "failed to set stale banner in redis: %v\n", res.Err())
		return res.Err()
	}

	res = bc.RedisClient.SetEX(
		context.Background(),
		BANNER_CACHE_KEY,
		b,
		BANNER_CACHE_TTL,
	)

	return res.Err()
}

func (bc BannerCache) Get() (*protos.CreateBannerResponse, bool, error) {
	shouldRegen := true
	res := bc.RedisClient.Get(context.Background(), BANNER_CACHE_KEY)
	if err := res.Err(); err != nil {
		if err == redis.Nil {
			res = bc.RedisClient.Get(context.Background(), BANNER_STALE_KEY)
			if res.Err() == redis.Nil {
				return nil, shouldRegen, nil
			}
		} else {
			fmt.Fprintf(os.Stderr, "failed to get banner from redis: %v\n", err)
			return nil, shouldRegen, err
		}
	} else {
		shouldRegen = false
	}

	b, err := res.Bytes()
	if err != nil {
		return nil, shouldRegen, err
	}

	if b == nil {
		return nil, shouldRegen, nil
	}

	var msg protos.CreateBannerResponse
	err = protobuf.Unmarshal(b, &msg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to deserialize cached banner from redis: %v\n", err)
		return nil, shouldRegen, err
	}

	return &msg, shouldRegen, nil
}

var initOnce sync.Once

func InitBannerCache(r *redis.Client) {
	initOnce.Do(func() {
		cacheSingleton = BannerCache{
			RedisClient: r,
		}
	})
}

func GetBannerCache() BannerCache {
	return cacheSingleton
}
