package vulnerable_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"go-dvwa/vulnerable"
)

func TestSystem(t *testing.T) {
	output, err := vulnerable.System(context.Background(), "echo ok")
	require.NoError(t, err)
	require.Equal(t, "ok\n", string(output))
}
