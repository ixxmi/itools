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

func (r *RedisClient) HSetWithExpire(key string, ttl time.Duration, values ...interface{}) error {
	var err error
	if r.isCluster {
		err = r.clusterClient.HSet(ctx, key, values...).Err()
		if err != nil {
			return err
		}
		// 设置过期时间
		return r.clusterClient.Expire(ctx, key, ttl).Err()
	}

	err = r.singleClient.HSet(ctx, key, values...).Err()
	if err != nil {
		return err
	}
	// 设置过期时间
	return r.singleClient.Expire(ctx, key, ttl).Err()
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

// 插入数据（左插入，最新数据在最前）
func (r *RedisClient) LPush(key string, values ...interface{}) error {
	if r.isCluster {
		return r.clusterClient.LPush(ctx, key, values...).Err()
	}
	return r.singleClient.LPush(ctx, key, values...).Err()
}

// 插入数据（右插入，最新数据在最后）
func (r *RedisClient) RPush(key string, values ...interface{}) error {
	if r.isCluster {
		return r.clusterClient.RPush(ctx, key, values...).Err()
	}
	return r.singleClient.RPush(ctx, key, values...).Err()
}

// 查询 List 长度
func (r *RedisClient) LLen(key string) (int64, error) {
	if r.isCluster {
		return r.clusterClient.LLen(ctx, key).Result()
	}
	return r.singleClient.LLen(ctx, key).Result()
}

// 获取指定区间的数据
// start=0, stop=-1 表示获取全部
func (r *RedisClient) LRange(key string, start, stop int64) ([]string, error) {
	if r.isCluster {
		return r.clusterClient.LRange(ctx, key, start, stop).Result()
	}
	return r.singleClient.LRange(ctx, key, start, stop).Result()
}

// 获取指定下标的数据
func (r *RedisClient) LIndex(key string, index int64) (string, error) {
	if r.isCluster {
		return r.clusterClient.LIndex(ctx, key, index).Result()
	}
	return r.singleClient.LIndex(ctx, key, index).Result()
}

// 弹出数据（从左边）
func (r *RedisClient) LPop(key string) (string, error) {
	if r.isCluster {
		return r.clusterClient.LPop(ctx, key).Result()
	}
	return r.singleClient.LPop(ctx, key).Result()
}

// 弹出数据（从右边）
func (r *RedisClient) RPop(key string) (string, error) {
	if r.isCluster {
		return r.clusterClient.RPop(ctx, key).Result()
	}
	return r.singleClient.RPop(ctx, key).Result()
}

// 插入并设置过期时间
func (r *RedisClient) RPushWithExpire(key string, ttl time.Duration, values ...interface{}) error {
	var err error
	if r.isCluster {
		err = r.clusterClient.RPush(ctx, key, values...).Err()
		if err != nil {
			return err
		}
		return r.clusterClient.Expire(ctx, key, ttl).Err()
	}
	err = r.singleClient.RPush(ctx, key, values...).Err()
	if err != nil {
		return err
	}
	return r.singleClient.Expire(ctx, key, ttl).Err()
}
