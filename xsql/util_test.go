package xsql

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInsertAt(t *testing.T) {
	a := insertAt([]any{1, 2, 3, 4}, []any{5, 6}, 4)
	require.Equal(t, a, []any{1, 2, 3, 4, 5, 6})

	a = insertAt([]any{}, []any{3, 2}, 0)
	require.Equal(t, a, []any{3, 2})

	a = insertAt([]any{}, []any{}, 5)
	require.Equal(t, a, []any{})

	a = insertAt([]any{1, 2}, []any{3}, 1)
	require.Equal(t, a, []any{1, 3, 2})
}
