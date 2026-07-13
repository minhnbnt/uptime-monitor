package utils

import (
	"hash/fnv"
	"time"
)

func GenerateOffset(id string, interval time.Duration) time.Duration {

	hasher := fnv.New64a()
	hasher.Write([]byte(id))

	offset := hasher.Sum64() % uint64(interval)
	return time.Duration(offset)
}
