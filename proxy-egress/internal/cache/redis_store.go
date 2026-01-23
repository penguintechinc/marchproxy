package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisStore struct {
	client         redis.UniversalClient
	config         RedisStoreConfig
	keyPrefix      string
	compressionEnabled bool
}

type RedisStoreConfig struct {
	Addresses        []string
	MasterName       string
	Password         string
	Database         int
	PoolSize         int
	MinIdleConns     int
	MaxRetries       int
	RetryDelay       time.Duration
	DialTimeout      time.Duration
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	KeyPrefix        string
	Compression      bool
	ClusterMode      bool
	SentinelMode     bool
	TLSEnabled       bool
}

type RedisClusterStore struct {
	*RedisStore
	clusterClient *redis.ClusterClient
}

type RedisSentinelStore struct {
	*RedisStore
	sentinelClient *redis.Client
}

func NewRedisStore(config RedisStoreConfig) (*RedisStore, error) {
	rs := &RedisStore{
		config:             config,
		keyPrefix:          config.KeyPrefix,
		compressionEnabled: config.Compression,
	}

	if config.ClusterMode {
		rs.client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:           config.Addresses,
			Password:        config.Password,
			PoolSize:        config.PoolSize,
			MinIdleConns:    config.MinIdleConns,
			MaxRetries:      config.MaxRetries,
			MinRetryBackoff: config.RetryDelay,
			DialTimeout:     config.DialTimeout,
			ReadTimeout:     config.ReadTimeout,
			WriteTimeout:    config.WriteTimeout,
		})
	} else if config.SentinelMode {
		rs.client = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:      config.MasterName,
			SentinelAddrs:   config.Addresses,
			Password:        config.Password,
			DB:              config.Database,
			PoolSize:        config.PoolSize,
			MinIdleConns:    config.MinIdleConns,
			MaxRetries:      config.MaxRetries,
			MinRetryBackoff: config.RetryDelay,
			DialTimeout:     config.DialTimeout,
			ReadTimeout:     config.ReadTimeout,
			WriteTimeout:    config.WriteTimeout,
		})
	} else {
		addr := "localhost:6379"
		if len(config.Addresses) > 0 {
			addr = config.Addresses[0]
		}

		rs.client = redis.NewClient(&redis.Options{
			Addr:            addr,
			Password:        config.Password,
			DB:              config.Database,
			PoolSize:        config.PoolSize,
			MinIdleConns:    config.MinIdleConns,
			MaxRetries:      config.MaxRetries,
			MinRetryBackoff: config.RetryDelay,
			DialTimeout:     config.DialTimeout,
			ReadTimeout:     config.ReadTimeout,
			WriteTimeout:    config.WriteTimeout,
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rs.client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return rs, nil
}

func (rs *RedisStore) Get(ctx context.Context, key string) (*CacheEntry, error) {
	redisKey := rs.buildKey(key)
	
	result, err := rs.client.Get(ctx, redisKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("redis get error: %w", err)
	}

	var entry CacheEntry
	if err := json.Unmarshal([]byte(result), &entry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache entry: %w", err)
	}

	if entry.IsExpired() {
		rs.client.Del(ctx, redisKey)
		return nil, nil
	}

	entry.Touch()
	
	entryData, _ := json.Marshal(entry)
	ttl := time.Until(entry.ExpiresAt)
	if ttl > 0 {
		rs.client.Set(ctx, redisKey, string(entryData), ttl)
	}

	return &entry, nil
}

func (rs *RedisStore) Set(ctx context.Context, key string, entry *CacheEntry, ttl time.Duration) error {
	redisKey := rs.buildKey(key)
	
	entryData, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal cache entry: %w", err)
	}

	err = rs.client.Set(ctx, redisKey, string(entryData), ttl).Err()
	if err != nil {
		return fmt.Errorf("redis set error: %w", err)
	}

	rs.updateMetadata(ctx, key, entry, ttl)
	return nil
}

func (rs *RedisStore) Delete(ctx context.Context, key string) error {
	redisKey := rs.buildKey(key)
	
	err := rs.client.Del(ctx, redisKey).Err()
	if err != nil {
		return fmt.Errorf("redis delete error: %w", err)
	}

	rs.removeMetadata(ctx, key)
	return nil
}

func (rs *RedisStore) Clear(ctx context.Context) error {
	pattern := rs.buildKey("*")
	
	keys, err := rs.client.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("redis keys error: %w", err)
	}

	if len(keys) == 0 {
		return nil
	}

	err = rs.client.Del(ctx, keys...).Err()
	if err != nil {
		return fmt.Errorf("redis delete error: %w", err)
	}

	rs.clearAllMetadata(ctx)
	return nil
}

func (rs *RedisStore) Exists(ctx context.Context, key string) (bool, error) {
	redisKey := rs.buildKey(key)
	
	count, err := rs.client.Exists(ctx, redisKey).Result()
	if err != nil {
		return false, fmt.Errorf("redis exists error: %w", err)
	}

	return count > 0, nil
}

func (rs *RedisStore) Keys(ctx context.Context, pattern string) ([]string, error) {
	redisPattern := rs.buildKey(pattern)
	
	keys, err := rs.client.Keys(ctx, redisPattern).Result()
	if err != nil {
		return nil, fmt.Errorf("redis keys error: %w", err)
	}

	var result []string
	prefixLen := len(rs.buildKey(""))
	for _, key := range keys {
		if len(key) > prefixLen {
			result = append(result, key[prefixLen:])
		}
	}

	return result, nil
}

func (rs *RedisStore) Size(ctx context.Context) (int64, error) {
	pattern := rs.buildKey("*")
	
	keys, err := rs.client.Keys(ctx, pattern).Result()
	if err != nil {
		return 0, fmt.Errorf("redis keys error: %w", err)
	}

	var totalSize int64
	for _, key := range keys {
		size, err := rs.client.StrLen(ctx, key).Result()
		if err != nil {
			continue
		}
		totalSize += size
	}

	return totalSize, nil
}

func (rs *RedisStore) Stats(ctx context.Context) (StoreStats, error) {
	info, err := rs.client.Info(ctx, "memory", "keyspace").Result()
	if err != nil {
		return StoreStats{}, fmt.Errorf("redis info error: %w", err)
	}

	stats := StoreStats{}
	lines := strings.Split(info, "\r\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "used_memory:") {
			if value, err := strconv.ParseInt(strings.Split(line, ":")[1], 10, 64); err == nil {
				stats.Memory = value
			}
		} else if strings.Contains(line, "keys=") {
			parts := strings.Split(line, ",")
			for _, part := range parts {
				if strings.HasPrefix(part, "keys=") {
					if value, err := strconv.ParseInt(strings.Split(part, "=")[1], 10, 64); err == nil {
						stats.KeyCount += value
					}
				}
			}
		}
	}

	size, _ := rs.Size(ctx)
	stats.Size = size

	return stats, nil
}

func (rs *RedisStore) buildKey(key string) string {
	if rs.keyPrefix != "" {
		return rs.keyPrefix + ":" + key
	}
	return key
}

func (rs *RedisStore) updateMetadata(ctx context.Context, key string, entry *CacheEntry, ttl time.Duration) {
	metaKey := rs.buildKey("meta:" + key)
	metadata := map[string]interface{}{
		"size":         entry.Size,
		"created_at":   entry.CreatedAt.Unix(),
		"expires_at":   entry.ExpiresAt.Unix(),
		"access_count": entry.AccessCount,
		"tags":         entry.Tags,
	}

	metaData, _ := json.Marshal(metadata)
	rs.client.Set(ctx, metaKey, string(metaData), ttl+time.Minute)
}

func (rs *RedisStore) removeMetadata(ctx context.Context, key string) {
	metaKey := rs.buildKey("meta:" + key)
	rs.client.Del(ctx, metaKey)
}

func (rs *RedisStore) clearAllMetadata(ctx context.Context) {
	pattern := rs.buildKey("meta:*")
	keys, err := rs.client.Keys(ctx, pattern).Result()
	if err == nil && len(keys) > 0 {
		rs.client.Del(ctx, keys...)
	}
}

func (rs *RedisStore) DeleteByTags(ctx context.Context, tags []string) error {
	pattern := rs.buildKey("meta:*")
	metaKeys, err := rs.client.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}

	var keysToDelete []string
	for _, metaKey := range metaKeys {
		result, err := rs.client.Get(ctx, metaKey).Result()
		if err != nil {
			continue
		}

		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(result), &metadata); err != nil {
			continue
		}

		if entryTags, ok := metadata["tags"].([]interface{}); ok {
			for _, tag := range tags {
				for _, entryTag := range entryTags {
					if tagStr, ok := entryTag.(string); ok && tagStr == tag {
						originalKey := strings.TrimPrefix(metaKey, rs.buildKey("meta:"))
						keysToDelete = append(keysToDelete, rs.buildKey(originalKey))
						keysToDelete = append(keysToDelete, metaKey)
						break
					}
				}
			}
		}
	}

	if len(keysToDelete) > 0 {
		return rs.client.Del(ctx, keysToDelete...).Err()
	}

	return nil
}

func (rs *RedisStore) Pipeline() redis.Pipeliner {
	return rs.client.Pipeline()
}

func (rs *RedisStore) TxPipeline() redis.Pipeliner {
	return rs.client.TxPipeline()
}

func (rs *RedisStore) SetMultiple(ctx context.Context, entries map[string]*CacheEntry, ttl time.Duration) error {
	pipe := rs.client.Pipeline()

	for key, entry := range entries {
		redisKey := rs.buildKey(key)
		entryData, err := json.Marshal(entry)
		if err != nil {
			continue
		}
		pipe.Set(ctx, redisKey, string(entryData), ttl)
	}

	_, err := pipe.Exec(ctx)
	return err
}

func (rs *RedisStore) GetMultiple(ctx context.Context, keys []string) (map[string]*CacheEntry, error) {
	if len(keys) == 0 {
		return make(map[string]*CacheEntry), nil
	}

	var redisKeys []string
	for _, key := range keys {
		redisKeys = append(redisKeys, rs.buildKey(key))
	}

	results, err := rs.client.MGet(ctx, redisKeys...).Result()
	if err != nil {
		return nil, err
	}

	entries := make(map[string]*CacheEntry)
	for i, result := range results {
		if result == nil {
			continue
		}

		var entry CacheEntry
		if resultStr, ok := result.(string); ok {
			if err := json.Unmarshal([]byte(resultStr), &entry); err != nil {
				continue
			}

			if !entry.IsExpired() {
				entries[keys[i]] = &entry
			}
		}
	}

	return entries, nil
}

func (rs *RedisStore) IncrementCounter(ctx context.Context, key string, delta int64) (int64, error) {
	redisKey := rs.buildKey("counter:" + key)
	return rs.client.IncrBy(ctx, redisKey, delta).Result()
}

func (rs *RedisStore) SetExpiry(ctx context.Context, key string, ttl time.Duration) error {
	redisKey := rs.buildKey(key)
	return rs.client.Expire(ctx, redisKey, ttl).Err()
}

func (rs *RedisStore) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	redisKey := rs.buildKey(key)
	return rs.client.TTL(ctx, redisKey).Result()
}

func (rs *RedisStore) Close() error {
	return rs.client.Close()
}

func (rs *RedisStore) HealthCheck(ctx context.Context) error {
	return rs.client.Ping(ctx).Err()
}

func DefaultRedisStoreConfig() RedisStoreConfig {
	return RedisStoreConfig{
		Addresses:    []string{"localhost:6379"},
		Database:     0,
		PoolSize:     10,
		MinIdleConns: 1,
		MaxRetries:   3,
		RetryDelay:   100 * time.Millisecond,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		KeyPrefix:    "marchproxy:cache",
		Compression:  true,
		ClusterMode:  false,
		SentinelMode: false,
		TLSEnabled:   false,
	}
}

func ClusterRedisStoreConfig(addresses []string) RedisStoreConfig {
	config := DefaultRedisStoreConfig()
	config.Addresses = addresses
	config.ClusterMode = true
	config.PoolSize = 20
	return config
}

func SentinelRedisStoreConfig(sentinels []string, masterName string) RedisStoreConfig {
	config := DefaultRedisStoreConfig()
	config.Addresses = sentinels
	config.MasterName = masterName
	config.SentinelMode = true
	config.PoolSize = 15
	return config
}

type RedisDistributedLock struct {
	client redis.UniversalClient
	key    string
	value  string
	ttl    time.Duration
}

func NewRedisDistributedLock(client redis.UniversalClient, key string, ttl time.Duration) *RedisDistributedLock {
	return &RedisDistributedLock{
		client: client,
		key:    key,
		value:  fmt.Sprintf("%d", time.Now().UnixNano()),
		ttl:    ttl,
	}
}

func (rdl *RedisDistributedLock) Acquire(ctx context.Context) (bool, error) {
	result, err := rdl.client.SetNX(ctx, rdl.key, rdl.value, rdl.ttl).Result()
	return result, err
}

func (rdl *RedisDistributedLock) Release(ctx context.Context) error {
	script := `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		else
			return 0
		end
	`
	return rdl.client.Eval(ctx, script, []string{rdl.key}, rdl.value).Err()
}

func (rdl *RedisDistributedLock) Extend(ctx context.Context, ttl time.Duration) error {
	script := `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("EXPIRE", KEYS[1], ARGV[2])
		else
			return 0
		end
	`
	return rdl.client.Eval(ctx, script, []string{rdl.key}, rdl.value, int(ttl.Seconds())).Err()
}