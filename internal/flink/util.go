package flink

import (
	"crypto/md5"
	"encoding/hex"
)

// Deterministic job ID generation based on the input string
func GenJobId(namespace, name string) string {
	hash := md5.Sum([]byte(namespace + name))
	return hex.EncodeToString(hash[:])
}
