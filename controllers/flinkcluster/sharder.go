package flinkcluster

import (
	"errors"
	"hash/fnv"
	"regexp"
	"strconv"
)

const bigPrime = 31393

var (
	podNameRegex, _ = regexp.Compile("(.*-)([0-9]+)$")
)

type Sharder struct {
	TotalShards int // Total number of shards
	Shard       int // Shard ordinal for this instance
}

func getResource(namespace, name string) string {
	return namespace + "/" + name
}

func (s *Sharder) IsOwnedByMe(namespace, name string) bool {
	return s.GetShard(namespace, name) == s.Shard
}

func (s *Sharder) GetShard(namespace, name string) int {
	resource := getResource(namespace, name)
	h := fnv1aHash(resource)
	indx := (int(h) * bigPrime) % s.TotalShards
	return indx
}

func fnv1aHash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func getShardFromPod(podName string) (int, error) {
	m := podNameRegex.FindStringSubmatch(podName)
	if m == nil || len(m) != 3 {
		return 0, errors.New("pod-name is not in the expected format")
	}
	shard, err := strconv.Atoi(m[2])
	if err != nil {
		return 0, err
	}
	return shard, nil
}

func NewSharderFromEnv(totalShardsStr string, podName string) (*Sharder, error) {
	// If totalShards or podName is not set, assume sharding is disabled.
	if totalShardsStr == "" || podName == "" {
		totalShardsStr = "1"
		podName = "flink-operator-0"
	}
	totalShards, err := strconv.Atoi(totalShardsStr)
	if err != nil {
		return nil, err
	}
	// Extract index from statefulset pod name
	shard, err := getShardFromPod(podName)
	if err != nil {
		return nil, err
	}

	return NewSharder(totalShards, shard), nil
}

func NewSharder(totalShards, shard int) *Sharder {
	return &Sharder{
		TotalShards: totalShards,
		Shard:       shard,
	}
}
