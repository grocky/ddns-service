package ratelimit

import (
	"testing"
	"time"

	"github.com/grocky/ddns-service/internal/domain"
	"gotest.tools/assert"
)

func TestCheck_NilMapping(t *testing.T) {
	now := time.Now().UTC()
	result := Check(nil, now)

	assert.Assert(t, result.Allowed, "nil mapping should be allowed (new registration)")
	assert.Equal(t, time.Duration(0), result.RetryAfter)
}

func TestCheck_FirstChangeInHour(t *testing.T) {
	now := time.Now().UTC()
	mapping := &domain.IPMapping{
		LastIPChangeAt:    now.Add(-time.Minute), // Changed 1 minute ago (same hour)
		HourlyChangeCount: 1,
	}

	result := Check(mapping, now)

	assert.Assert(t, result.Allowed, "should allow second change in same hour")
	assert.Equal(t, time.Duration(0), result.RetryAfter)
}

func TestCheck_AtLimit(t *testing.T) {
	now := time.Now().UTC()
	mapping := &domain.IPMapping{
		LastIPChangeAt:    now.Add(-time.Minute),
		HourlyChangeCount: MaxChangesPerHour,
	}

	result := Check(mapping, now)

	assert.Assert(t, !result.Allowed, "should not allow change at limit")
	assert.Assert(t, result.RetryAfter > 0, "should have retry after duration")
	assert.Assert(t, result.RetryAfter <= time.Hour, "retry after should be within an hour")
}

func TestCheck_ExceedsLimit(t *testing.T) {
	now := time.Now().UTC()
	mapping := &domain.IPMapping{
		LastIPChangeAt:    now.Add(-time.Minute),
		HourlyChangeCount: MaxChangesPerHour + 1,
	}

	result := Check(mapping, now)

	assert.Assert(t, !result.Allowed, "should not allow change over limit")
}

func TestCheck_DifferentHour(t *testing.T) {
	now := time.Now().UTC()
	mapping := &domain.IPMapping{
		LastIPChangeAt:    now.Add(-2 * time.Hour), // Changed 2 hours ago
		HourlyChangeCount: MaxChangesPerHour + 5,   // Was over limit in previous hour
	}

	result := Check(mapping, now)

	assert.Assert(t, result.Allowed, "should allow change in new hour")
	assert.Equal(t, time.Duration(0), result.RetryAfter)
}

func TestCheck_RetryAfterCalculation(t *testing.T) {
	// Set time to 30 minutes into the hour
	now := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	mapping := &domain.IPMapping{
		LastIPChangeAt:    now.Add(-time.Minute),
		HourlyChangeCount: MaxChangesPerHour,
	}

	result := Check(mapping, now)

	assert.Assert(t, !result.Allowed)
	// Should be approximately 30 minutes until next hour
	assert.Assert(t, result.RetryAfter >= 29*time.Minute && result.RetryAfter <= 31*time.Minute,
		"expected ~30 minutes, got %v", result.RetryAfter)
}

func TestUpdateCounters_FirstChangeInHour(t *testing.T) {
	now := time.Now().UTC()
	mapping := &domain.IPMapping{
		LastIPChangeAt:    now.Add(-2 * time.Hour), // Last change was 2 hours ago
		HourlyChangeCount: 5,
	}

	UpdateCounters(mapping, now)

	assert.Equal(t, 1, mapping.HourlyChangeCount, "counter should reset to 1")
	assert.Equal(t, now, mapping.LastIPChangeAt)
}

func TestUpdateCounters_SubsequentChangeInHour(t *testing.T) {
	now := time.Now().UTC()
	mapping := &domain.IPMapping{
		LastIPChangeAt:    now.Add(-time.Minute), // Last change was 1 minute ago
		HourlyChangeCount: 1,
	}

	UpdateCounters(mapping, now)

	assert.Equal(t, 2, mapping.HourlyChangeCount, "counter should increment")
	assert.Equal(t, now, mapping.LastIPChangeAt)
}

func TestMaxChangesPerHour(t *testing.T) {
	assert.Equal(t, 2, MaxChangesPerHour, "max changes per hour should be 2")
}

func TestCheck_EdgeCase_ExactlyAtHourBoundary(t *testing.T) {
	// Test at exactly the start of an hour
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	mapping := &domain.IPMapping{
		LastIPChangeAt:    time.Date(2025, 1, 15, 9, 59, 59, 0, time.UTC), // Just before new hour
		HourlyChangeCount: MaxChangesPerHour,
	}

	result := Check(mapping, now)

	assert.Assert(t, result.Allowed, "should allow change at new hour boundary")
}
