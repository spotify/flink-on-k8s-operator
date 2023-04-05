package flinkcluster

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestSharder(t *testing.T) {
	sharder, err := NewSharderFromEnv("2", "flink-operator-1")
	if err != nil {
		t.Error(err)
	}
	sharder2, err := NewSharderFromEnv("2", "flink-operator-0")
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, sharder.Shard, 1)
	assert.Equal(t, sharder.TotalShards, 2)
	assert.Equal(t, sharder.IsOwnedByMe("test-namespace", "fxoxo1"), true)
	assert.Equal(t, sharder2.Shard, 0)
	assert.Equal(t, sharder2.IsOwnedByMe("test-namespace", "fxoxo2"), true)
}
