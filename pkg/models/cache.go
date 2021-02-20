package models

import "fmt"

// CacheEngine ...
type CacheEngine string

// MemoryCacheEngine ...
var MemoryCacheEngine CacheEngine = "MEMORY"

// RedisCacheEngine ...
var RedisCacheEngine CacheEngine = "REDIS"

// GetCacheEngine ...
func GetCacheEngine(s string) (CacheEngine, error) {
	switch s {
	case "MEMORY":
		return MemoryCacheEngine, nil
	case "REDIS":
		return RedisCacheEngine, nil
	default:
		return "", fmt.Errorf("Unknown cache engine type '%s'. Supported engines are: MEMORY or REDIS", s)
	}
}
