package ratelimit

import (
	"time"

	"github.com/grocky/ddns-service/internal/domain"
)

const (
	// MaxChangesPerHour is the maximum number of IP changes allowed per hour.
	MaxChangesPerHour = 2
)

// CheckResult contains the result of a rate limit check.
type CheckResult struct {
	Allowed    bool
	RetryAfter time.Duration
}

// Check determines if an IP change is allowed for the given mapping.
// It returns whether the change is allowed and how long until the rate limit resets.
func Check(mapping *domain.IPMapping, now time.Time) CheckResult {
	if mapping == nil {
		// New mapping - check if we're within the limit (starts at 0)
		return CheckResult{Allowed: true, RetryAfter: 0}
	}

	currentHour := now.Truncate(time.Hour)
	lastChangeHour := mapping.LastIPChangeAt.Truncate(time.Hour)

	// If the last change was in a different hour, the counter has reset
	if !lastChangeHour.Equal(currentHour) {
		return CheckResult{Allowed: true, RetryAfter: 0}
	}

	// Check if we've exceeded the limit
	if mapping.HourlyChangeCount >= MaxChangesPerHour {
		nextHour := currentHour.Add(time.Hour)
		retryAfter := nextHour.Sub(now)
		return CheckResult{Allowed: false, RetryAfter: retryAfter}
	}

	return CheckResult{Allowed: true, RetryAfter: 0}
}

// UpdateCounters updates the rate limit counters on a mapping after an IP change.
// This should be called after a successful IP change.
func UpdateCounters(mapping *domain.IPMapping, now time.Time) {
	currentHour := now.Truncate(time.Hour)
	lastChangeHour := mapping.LastIPChangeAt.Truncate(time.Hour)

	// If the last change was in a different hour, reset the counter
	if !lastChangeHour.Equal(currentHour) {
		mapping.HourlyChangeCount = 1
	} else {
		mapping.HourlyChangeCount++
	}

	mapping.LastIPChangeAt = now
}
