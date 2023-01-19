package proxy

import "fmt"

type cacheEngineModel string

var memoryCacheEngine cacheEngineModel = "MEMORY"
var redisCacheEngine cacheEngineModel = "REDIS"

func getCacheEngine(s string) (cacheEngineModel, error) {
	switch s {
	case "MEMORY":
		return memoryCacheEngine, nil
	case "REDIS":
		return redisCacheEngine, nil
	default:
		return "", fmt.Errorf("Unknown cache engine type '%s'. Supported engines are: MEMORY or REDIS", s)
	}
}
