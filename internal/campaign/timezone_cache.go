package campaign

import (
	"sync"
	"time"
)

func CampaignLocation(cache *sync.Map, timezone string) *time.Location {
	if cache != nil {
		if cached, found := cache.Load(timezone); found {
			return cached.(*time.Location)
		}
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}
	if cache != nil {
		cache.Store(timezone, loc)
	}
	return loc
}
