package dao

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/ericfialkowski/shorturl/env"
	"github.com/redis/go-redis/v9"
)

type RedisDB struct {
	client *redis.Client
}

const (
	abvKeyPrefix   = "shorturl:abv:"   // Hash: url, hits, last_access
	urlKeyPrefix   = "shorturl:url:"   // String: abbreviation
	dailyKeyPrefix = "shorturl:daily:" // Hash: date -> hit count
)

func newRedisContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), env.DurationOrDefault("redis_timeout", 10*time.Second))
}

// CreateRedisDB creates a new Redis-backed ShortUrlDao.
// The connString should be a Redis connection string, e.g.:
// "redis://user:password@localhost:6379/0" or "localhost:6379"
func CreateRedisDB(connString string) ShortUrlDao {
	ctx, cancel := newRedisContext()
	defer cancel()

	opt, err := redis.ParseURL(connString)
	if err != nil {
		// If parsing as URL fails, try as simple address
		opt = &redis.Options{
			Addr: connString,
		}
	}

	opt.PoolSize = env.IntOrDefault("redis_pool_size", 10)

	client := redis.NewClient(opt)

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatalf("Unable to connect to Redis: %v", err)
	}

	return &RedisDB{client: client}
}

func (d *RedisDB) Cleanup() {
	if err := d.client.Close(); err != nil {
		log.Printf("Error closing Redis connection: %v", err)
	}
}

func (d *RedisDB) IsLikelyOk() bool {
	ctx, cancel := newRedisContext()
	defer cancel()

	if err := d.client.Ping(ctx).Err(); err != nil {
		log.Printf("Redis ping failed: %v", err)
		return false
	}
	return true
}

func (d *RedisDB) Save(abv string, url string) error {
	ctx, cancel := newRedisContext()
	defer cancel()

	abvKey := abvKeyPrefix + abv
	urlKey := urlKeyPrefix + url

	// Check if abbreviation already exists with a different URL
	existingUrl, err := d.client.HGet(ctx, abvKey, "url").Result()
	if err == nil && existingUrl != "" && existingUrl != url {
		return fmt.Errorf("abbreviation %s already exists with different URL", abv)
	}

	// Use a transaction to ensure atomicity
	pipe := d.client.TxPipeline()
	pipe.HSet(ctx, abvKey, map[string]any{
		"url":  url,
		"hits": 0,
	})
	pipe.Set(ctx, urlKey, abv, 0)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("couldn't store (%s, %s): %v", abv, url, err)
	}

	return nil
}

func (d *RedisDB) DeleteAbv(abv string) error {
	ctx, cancel := newRedisContext()
	defer cancel()

	abvKey := abvKeyPrefix + abv

	// Get the URL first so we can delete the reverse mapping
	url, err := d.client.HGet(ctx, abvKey, "url").Result()
	if err == redis.Nil {
		return nil // Already deleted or never existed
	}
	if err != nil {
		return fmt.Errorf("couldn't get URL for abbreviation %s: %v", abv, err)
	}

	urlKey := urlKeyPrefix + url
	dailyKey := dailyKeyPrefix + abv

	// Delete all related keys
	pipe := d.client.TxPipeline()
	pipe.Del(ctx, abvKey)
	pipe.Del(ctx, urlKey)
	pipe.Del(ctx, dailyKey)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("couldn't delete abbreviation %s: %v", abv, err)
	}

	return nil
}

func (d *RedisDB) DeleteUrl(url string) error {
	ctx, cancel := newRedisContext()
	defer cancel()

	urlKey := urlKeyPrefix + url

	// Get the abbreviation first so we can delete the forward mapping
	abv, err := d.client.Get(ctx, urlKey).Result()
	if err == redis.Nil {
		return nil // Already deleted or never existed
	}
	if err != nil {
		return fmt.Errorf("couldn't get abbreviation for URL %s: %v", url, err)
	}

	abvKey := abvKeyPrefix + abv
	dailyKey := dailyKeyPrefix + abv

	// Delete all related keys
	pipe := d.client.TxPipeline()
	pipe.Del(ctx, abvKey)
	pipe.Del(ctx, urlKey)
	pipe.Del(ctx, dailyKey)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("couldn't delete URL %s: %v", url, err)
	}

	return nil
}

func (d *RedisDB) GetUrl(abv string) (string, error) {
	ctx, cancel := newRedisContext()
	defer cancel()

	abvKey := abvKeyPrefix + abv

	url, err := d.client.HGet(ctx, abvKey, "url").Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("error getting URL for %s: %v", abv, err)
	}

	// Update stats asynchronously
	go func() {
		ctx, cancel := newRedisContext()
		defer cancel()

		dailyKey := dailyKeyPrefix + abv
		date := Date()

		pipe := d.client.TxPipeline()
		pipe.HIncrBy(ctx, abvKey, "hits", 1)
		pipe.HSet(ctx, abvKey, "last_access", time.Now().Format(time.RFC3339))
		pipe.HIncrBy(ctx, dailyKey, date, 1)

		if _, err := pipe.Exec(ctx); err != nil {
			log.Printf("Error updating Redis stats: %v", err)
		}
	}()

	return url, nil
}

func (d *RedisDB) GetAbv(url string) (string, error) {
	ctx, cancel := newRedisContext()
	defer cancel()

	urlKey := urlKeyPrefix + url

	abv, err := d.client.Get(ctx, urlKey).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("error getting abbreviation for %s: %v", url, err)
	}

	return abv, nil
}

func (d *RedisDB) GetStats(abv string) (ShortUrl, error) {
	ctx, cancel := newRedisContext()
	defer cancel()

	abvKey := abvKeyPrefix + abv

	// Get all fields from the abbreviation hash
	result, err := d.client.HGetAll(ctx, abvKey).Result()
	if err != nil {
		return ShortUrl{}, fmt.Errorf("error getting stats for %s: %v", abv, err)
	}

	if len(result) == 0 {
		return ShortUrl{}, nil
	}

	var data ShortUrl
	data.Abbreviation = abv
	data.Url = result["url"]

	if hitsStr, ok := result["hits"]; ok {
		hits, _ := strconv.ParseInt(hitsStr, 10, 32)
		data.Hits = int32(hits)
	}

	if lastAccessStr, ok := result["last_access"]; ok && lastAccessStr != "" {
		if t, err := time.Parse(time.RFC3339, lastAccessStr); err == nil {
			data.LastAccess = t
		}
	}

	// Get daily hits
	dailyKey := dailyKeyPrefix + abv
	dailyHits, err := d.client.HGetAll(ctx, dailyKey).Result()
	if err != nil {
		log.Printf("Error getting daily hits for %s: %v", abv, err)
		data.DailyHits = make(map[string]int)
	} else {
		data.DailyHits = make(map[string]int)
		for date, hitsStr := range dailyHits {
			hits, _ := strconv.Atoi(hitsStr)
			data.DailyHits[date] = hits
		}
	}

	return data, nil
}
