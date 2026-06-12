package cache

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSubsystem(t *testing.T) {
	require.Equal(t, "subsystem", GetSubsystem(SetSubsystem(context.Background(), "subsystem")))
	require.Equal(t, "none", GetSubsystem(context.Background()))
}
