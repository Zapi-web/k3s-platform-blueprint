package domain

type CacheStatus string

const (
	CacheUnavailable CacheStatus = "unavailable"
	CacheError       CacheStatus = "error"
	CacheHit         CacheStatus = "hit"
	CacheMiss        CacheStatus = "miss"
)
