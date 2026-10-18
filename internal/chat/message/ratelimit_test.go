package message

import (
	"testing"
	"time"
)

func TestMessageRateLimiterAllowsBurstAndEscalatesPenalty(t *testing.T) {
	limiter := messageRateLimiter{}
	now := time.Unix(0, 0)

	for i := 0; i < messageLimit; i++ {
		if allowed, _ := limiter.Allow(now); !allowed {
			t.Fatalf("message %d was unexpectedly rejected", i+1)
		}
	}

	allowed, retryAfter := limiter.Allow(now)
	if allowed || retryAfter != messageBasePenalty {
		t.Fatalf("first excess message: allowed=%v retryAfter=%v", allowed, retryAfter)
	}

	secondAttempt := now.Add(messageBasePenalty / 2)
	allowed, retryAfter = limiter.Allow(secondAttempt)
	if allowed || retryAfter != 2*messageBasePenalty {
		t.Fatalf("second excess message: allowed=%v retryAfter=%v", allowed, retryAfter)
	}

	if allowed, _ := limiter.Allow(secondAttempt.Add(messageBasePenalty)); allowed {
		t.Fatal("message was allowed while the penalty was active")
	}
}

func TestMessageRateLimiterResetsAfterQuietWindow(t *testing.T) {
	limiter := messageRateLimiter{}
	now := time.Unix(0, 0)

	for i := 0; i < messageLimit; i++ {
		limiter.Allow(now)
	}
	limiter.Allow(now)

	quiet := now.Add(messageWindow + messageBasePenalty)
	for i := 0; i < messageLimit; i++ {
		if allowed, _ := limiter.Allow(quiet); !allowed {
			t.Fatal("message was rejected after the limiter reset")
		}
	}
	allowed, retryAfter := limiter.Allow(quiet)
	if allowed || retryAfter != messageBasePenalty {
		t.Fatalf("penalty did not reset: allowed=%v retryAfter=%v", allowed, retryAfter)
	}
}

func TestJoinRateLimiterBansOnTenthAttempt(t *testing.T) {
	limiter := joinRateLimiter{}
	now := time.Unix(0, 0)

	for i := 0; i < joinLimit-1; i++ {
		if allowed, _ := limiter.Allow(now); !allowed {
			t.Fatalf("join %d was unexpectedly rejected", i+1)
		}
	}

	allowed, retryAfter := limiter.Allow(now)
	if allowed || retryAfter != joinBanDuration {
		t.Fatalf("tenth join: allowed=%v retryAfter=%v", allowed, retryAfter)
	}

	if allowed, _ := limiter.Allow(now.Add(time.Second)); allowed {
		t.Fatal("join was allowed during the ban")
	}

	if allowed, _ := limiter.Allow(now.Add(joinBanDuration)); !allowed {
		t.Fatal("join was not allowed after the ban expired")
	}
}
