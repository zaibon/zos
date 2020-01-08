package main

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestKeyID(t *testing.T) {
	for i := 0; i < 10000; i++ {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			id := uint64(1)
			key := farmKey(id)
			result := farmID(key)
			assert.Equal(t, id, result)
		})
	}
}
