package flinkcluster

import (
	"os"
	"testing"

	"gotest.tools/v3/assert"
)

func TestSharder(t *testing.T) {
	os.Setenv("SHARDS", "2")
	os.Setenv("SHARD_NAME", "flink-operator-1")
	sharder, err := NewSharderFromEnv()
	if err != nil {
		t.Error(err)
	}
	os.Setenv("SHARD_NAME", "flink-operator-0")
	sharder2, err := NewSharderFromEnv()
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, sharder.ShardId, 1)
	assert.Equal(t, sharder.Shards, 2)
	assert.Equal(t, sharder.IsOwnedByMe("test-namespace", "fxoxo1"), true)
	assert.Equal(t, sharder2.ShardId, 0)
	assert.Equal(t, sharder2.IsOwnedByMe("test-namespace", "fxoxo2"), true)
}
