package cache

import (
	s "github.com/bariiss/echo-ip/structs"
	"sync"
	"time"
)

// Entry structure with response and expiration time
type Entry struct {
	Response   *s.GeoInfo
	Expiration time.Time
}

// Cache structure with mutex for concurrent access
var Cache = struct {
	sync.RWMutex
	Data map[string]Entry
}{Data: make(map[string]Entry)}
