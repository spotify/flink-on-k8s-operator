package flinkcluster

import (
	"errors"
	"hash/fnv"
	"os"
	"regexp"
	"strconv"
)

type Sharder struct {
	Shards  int // Total number of shards
	ShardId int // Shard ordinal for this instance
}

func getResource(namespace, name string) string {
	return namespace + "/" + name
}

func (s *Sharder) IsOwnedByMe(namespace, name string) bool {
	return s.GetProcessorId(namespace, name) == s.ShardId
}

func (s *Sharder) GetProcessorId(namespace, name string) int {
	resource := getResource(namespace, name)
	bigPrime := 31393
	h := fnv1aHash(resource)
	indx := (int(h) * bigPrime) % s.Shards
	return indx
}

func fnv1aHash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func NewSharderFromEnv() (*Sharder, error) {
	shards, err := strconv.Atoi(os.Getenv("SHARDS"))
	if err != nil {
		return nil, err
	}
	// Extract index from statefulset pod name
	shardName := os.Getenv("SHARD_NAME")
	if shardName == "" {
		return nil, errors.New("SHARD_NAME is not set")
	}
	r, _ := regexp.Compile("(.*-)([0-9]+)$")
	m := r.FindStringSubmatch(shardName)
	if m == nil || len(m) != 3 {
		return nil, errors.New("SHARD_NAME is not in the expected format")
	}
	shardId, err := strconv.Atoi(m[2])
	if err != nil {
		return nil, err
	}

	return NewSharder(shards, shardId), nil
}

func NewSharder(shards, shardId int) *Sharder {
	return &Sharder{
		Shards:  shards,
		ShardId: shardId,
	}
}
