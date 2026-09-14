package scheduler

import (
	"math/rand"
	"testing"
	"time"
)

func TestJitterRange(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 200; i++ {
		d := Jitter(rng)
		if d < 5*time.Second || d > 20*time.Second {
			t.Fatalf("jitter out of range: %v", d)
		}
	}
}
