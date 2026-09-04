package handler

import (
	"time"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/generated/api"
)

// resolveLocation turns the optional X-Timezone header into a *time.Location.
// Missing, empty or unknown values fall back to UTC: the header is a display
// hint, never a reason to reject the request.
func resolveLocation(tz api.OptString) *time.Location {
	if !tz.IsSet() || tz.Value == "" {
		return time.UTC
	}
	if loc, err := time.LoadLocation(tz.Value); err == nil {
		return loc
	}
	return time.UTC
}
