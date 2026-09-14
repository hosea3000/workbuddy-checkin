package scheduler

import (
	"math/rand"
	"time"
)

// Jitter 返回 5~20 秒随机间隔（防风控）。
func Jitter(rng *rand.Rand) time.Duration {
	if rng == nil {
		return 5 * time.Second
	}
	return time.Duration(5+rng.Intn(16)) * time.Second
}
