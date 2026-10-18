package message

import "time"

const (
	messageLimit       = 5
	messageWindow      = 5 * time.Second
	messageBasePenalty = 5 * time.Second
	messageMaxPenalty  = 30 * time.Second

	joinLimit       = 10
	joinWindow      = time.Minute
	joinBanDuration = 30 * time.Second
)

type messageRateLimiter struct {
	timestamps          []time.Time
	penaltyUntil        time.Time
	lastViolation       time.Time
	violationMultiplier uint
}

func (l *messageRateLimiter) Allow(now time.Time) (bool, time.Duration) {
	if now.Before(l.penaltyUntil) {
		return false, l.penalize(now)
	}

	cutoff := now.Add(-messageWindow)
	firstRecent := 0
	for firstRecent < len(l.timestamps) && !l.timestamps[firstRecent].After(cutoff) {
		firstRecent++
	}
	l.timestamps = l.timestamps[firstRecent:]

	if len(l.timestamps) >= messageLimit {
		return false, l.penalize(now)
	}

	l.timestamps = append(l.timestamps, now)
	return true, 0
}

func (l *messageRateLimiter) penalize(now time.Time) time.Duration {
	if now.Sub(l.lastViolation) > messageWindow {
		l.violationMultiplier = 0
	}
	l.violationMultiplier++
	l.lastViolation = now

	penalty := messageBasePenalty
	for i := uint(1); i < l.violationMultiplier && penalty < messageMaxPenalty; i++ {
		penalty *= 2
	}
	if penalty > messageMaxPenalty {
		penalty = messageMaxPenalty
	}
	l.penaltyUntil = now.Add(penalty)
	return penalty
}

type joinRateLimiter struct {
	timestamps  []time.Time
	bannedUntil time.Time
}

func (l *joinRateLimiter) Allow(now time.Time) (bool, time.Duration) {
	if now.Before(l.bannedUntil) {
		return false, l.bannedUntil.Sub(now)
	}

	cutoff := now.Add(-joinWindow)
	firstRecent := 0
	for firstRecent < len(l.timestamps) && !l.timestamps[firstRecent].After(cutoff) {
		firstRecent++
	}
	l.timestamps = l.timestamps[firstRecent:]

	if len(l.timestamps) >= joinLimit-1 {
		l.bannedUntil = now.Add(joinBanDuration)
		l.timestamps = nil
		return false, joinBanDuration
	}

	l.timestamps = append(l.timestamps, now)
	return true, 0
}
