package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

var (
	ctx = context.Background()
	RC  = RedisClient{} // 全局 Redis 客户端实例
)

type Config struct {
	Addrs     []string
	Password  string
	DB        int
	IsCluster bool
}

type RedisClient struct {
	clusterClient *goredis.ClusterClient
	singleClient  *goredis.Client
	isCluster     bool
}

// NewRedis 创建 Redis 客户端
func NewRedis(cfg Config) (*RedisClient, error) {
	client := &RedisClient{isCluster: cfg.IsCluster}

	if cfg.IsCluster {
		client.clusterClient = goredis.NewClusterClient(&goredis.ClusterOptions{
			Addrs:    cfg.Addrs,
			Password: cfg.Password,
		})
		if err := client.clusterClient.Ping(ctx).Err(); err != nil {
			return nil, fmt.Errorf("连接 Redis Cluster 失败: %v", err)
		}
	} else {
		client.singleClient = goredis.NewClient(&goredis.Options{
			Addr:     cfg.Addrs[0],
			Password: cfg.Password,
			DB:       cfg.DB,
		})
		if err := client.singleClient.Ping(ctx).Err(); err != nil {
			return nil, fmt.Errorf("连接 Redis 单节点失败: %v", err)
		}
	}

	RC = *client
	log.Println("Redis 客户端连接成功")
	return client, nil
}

// Set 设置键值
func (r *RedisClient) Set(key string, value interface{}, expiration time.Duration) error {
	if r.isCluster {
		return r.clusterClient.Set(ctx, key, value, expiration).Err()
	}
	return r.singleClient.Set(ctx, key, value, expiration).Err()
}

// Get 获取值
func (r *RedisClient) Get(key string) (string, error) {
	if r.isCluster {
		return r.clusterClient.Get(ctx, key).Result()
	}
	return r.singleClient.Get(ctx, key).Result()
}

// Get 获取MAP值
func (r *RedisClient) GetMap(key string) (map[string]interface{}, error) {
	if r.isCluster {
		result, err := r.clusterClient.Get(ctx, key).Result()
		if err != nil {
			return nil, err
		}
		data := map[string]interface{}{}
		json.Unmarshal([]byte(result), &data)
		return data, nil
	}
	result, err := r.singleClient.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	data := map[string]interface{}{}
	json.Unmarshal([]byte(result), &data)
	return data, nil
}

// Get 获取MAP数组值
func (r *RedisClient) GetMaps(key string) ([]map[string]interface{}, error) {
	if r.isCluster {
		result, err := r.clusterClient.Get(ctx, key).Result()
		if err != nil {
			return nil, err
		}
		data := []map[string]interface{}{}
		json.Unmarshal([]byte(result), &data)
		return data, nil
	}
	result, err := r.singleClient.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	data := []map[string]interface{}{}
	json.Unmarshal([]byte(result), &data)
	return data, nil
}

// Del 删除键
func (r *RedisClient) Del(keys ...string) error {
	if r.isCluster {
		return r.clusterClient.Del(ctx, keys...).Err()
	}
	return r.singleClient.Del(ctx, keys...).Err()
}

// Exists 判断键是否存在
func (r *RedisClient) Exists(key string) (bool, error) {
	var n int64
	var err error
	if r.isCluster {
		n, err = r.clusterClient.Exists(ctx, key).Result()
	} else {
		n, err = r.singleClient.Exists(ctx, key).Result()
	}
	return n > 0, err
}

// HSet 设置哈希字段
func (r *RedisClient) HSet(key string, values ...interface{}) error {
	if r.isCluster {
		return r.clusterClient.HSet(ctx, key, values...).Err()
	}
	return r.singleClient.HSet(ctx, key, values...).Err()
}

// HGet 获取哈希字段
func (r *RedisClient) HGet(key, field string) (string, error) {
	if r.isCluster {
		return r.clusterClient.HGet(ctx, key, field).Result()
	}
	return r.singleClient.HGet(ctx, key, field).Result()
}

// HDel 删除哈希字段
func (r *RedisClient) HDel(key string, fields ...string) error {
	if r.isCluster {
		return r.clusterClient.HDel(ctx, key, fields...).Err()
	}
	return r.singleClient.HDel(ctx, key, fields...).Err()
}

// Keys 获取匹配的 key 列表（仅支持单节点）
func (r *RedisClient) Keys(pattern string) ([]string, error) {
	if r.isCluster {
		return nil, fmt.Errorf("Keys 命令不支持 Redis Cluster")
	}
	return r.singleClient.Keys(ctx, pattern).Result()
}
