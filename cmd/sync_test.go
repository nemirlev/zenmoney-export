package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSyncCommandValidatesEntities(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		wantError bool
	}{
		{name: "all", value: "all"},
		{name: "named aliases", value: " accounts,transactions "},
		{name: "unknown", value: "payments", wantError: true},
		{name: "empty list item", value: "accounts,,tags", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewSyncCommand(&RootCommand{})
			require.NoError(t, cmd.Flags().Set("entities", tt.value))

			err := cmd.PreRunE(cmd, nil)
			if tt.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
