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

func NextExecutionTimeByPrev(prev time.Time, interval time.Duration) time.Time {

	next := prev
	now := time.Now()
	if next.Before(now) {
		missed := now.Sub(next)/interval + 1
		next = next.Add(missed * interval)
	}

	return next
}

func NextExecutionTime(id any, interval time.Duration) (time.Time, error) {

	offset, err := GenerateOffset(id, interval)
	if err != nil {
		return time.Time{}, err
	}

	prev := time.Unix(0, 0).Add(offset)
	return NextExecutionTimeByPrev(prev, interval), nil
}
