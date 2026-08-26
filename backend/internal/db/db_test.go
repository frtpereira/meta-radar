package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInvalidDSN(t *testing.T) {
	pool, err := New(context.Background(), "postgres://user:pass@localhost:not-a-port/dbname")

	require.Error(t, err)
	assert.Nil(t, pool)
	assert.Contains(t, err.Error(), "creating pool")
}

func TestNewRetryLoopPathNotUnitTested(t *testing.T) {
	t.Skip("the retry loop hard-codes ten 1s sleeps and needs a real pool whose Ping fails predictably; fast unit coverage focuses on malformed DSNs instead")
}
