package utils

import (
	"fmt"
	"hash/fnv"
	"time"
)

func Hash(value any) (uint64, error) {

	hasher := fnv.New64a()
	_, err := fmt.Fprint(hasher, value)
	if err != nil {
		return 0, err
	}

	return hasher.Sum64(), nil
}

func GenerateOffset(id any, interval time.Duration) (time.Duration, error) {

	hash, err := Hash(id)
	if err != nil {
		return 0, err
	}

	return time.Duration(hash % uint64(interval)), nil
}

func NextExecutionTime(id any, interval time.Duration) (time.Time, error) {

	offset, err := GenerateOffset(id, interval)
	if err != nil {
		return time.Time{}, err
	}

	nowMs := time.Now().UnixMilli()
	offsetMs := offset.Milliseconds()
	intervalMs := interval.Milliseconds()

	periods := (nowMs - offsetMs + intervalMs - 1) / intervalMs
	return time.UnixMilli(offsetMs + periods*intervalMs), nil
}
