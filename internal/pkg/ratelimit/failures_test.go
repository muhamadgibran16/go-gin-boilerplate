package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestFailureCounter(t *testing.T) {
	ctx := context.Background()
	c := NewFailureCounter(3, time.Minute)

	for i := 1; i <= 3; i++ {
		if _, blocked, _ := c.Blocked(ctx, "bob"); blocked {
			t.Fatalf("blocked before failure %d", i)
		}
		if err := c.Fail(ctx, "bob"); err != nil {
			t.Fatal(err)
		}
	}

	retryAfter, blocked, err := c.Blocked(ctx, "bob")
	if err != nil || !blocked {
		t.Fatalf("must be blocked after 3 failures, got blocked=%v err=%v", blocked, err)
	}
	if retryAfter <= 0 || retryAfter > time.Minute {
		t.Fatalf("retryAfter = %v, want within the window", retryAfter)
	}

	if _, blocked, _ := c.Blocked(ctx, "alice"); blocked {
		t.Fatal("other keys must not be affected")
	}

	if err := c.Reset(ctx, "bob"); err != nil {
		t.Fatal(err)
	}
	if _, blocked, _ := c.Blocked(ctx, "bob"); blocked {
		t.Fatal("reset must unblock the key")
	}
}

func TestFailureCounterWindowExpires(t *testing.T) {
	ctx := context.Background()
	c := NewFailureCounter(1, 1100*time.Millisecond)

	_ = c.Fail(ctx, "bob")
	if _, blocked, _ := c.Blocked(ctx, "bob"); !blocked {
		t.Fatal("must be blocked")
	}

	time.Sleep(2100 * time.Millisecond) // reset times have a one-second resolution
	if _, blocked, _ := c.Blocked(ctx, "bob"); blocked {
		t.Fatal("must be unblocked after the window")
	}
}
