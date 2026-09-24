package app_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// The public purpose map must cover the modules actually registered by the app,
// including wrapped SDK modules and the two IBC light clients. Directory counts
// would incorrectly include x/common and x/witness and miss SDK registrations.
func TestModuleGuideCoversRegisteredApplication(t *testing.T) {
	data, err := os.ReadFile("../dashboard/module-guide.json")
	require.NoError(t, err)
	var guide struct {
		Modules []struct {
			ID          string `json:"id"`
			Purpose     string `json:"purpose"`
			Owns        string `json:"owns"`
			Boundary    string `json:"boundary"`
			SourcePath  string `json:"sourcePath"`
			Connections []struct {
				Module string `json:"module"`
			} `json:"connections"`
		} `json:"modules"`
	}
	require.NoError(t, json.Unmarshal(data, &guide))
	registered := newTestApp(t).ModuleManager.Modules
	described := make(map[string]bool, len(guide.Modules))
	for _, module := range guide.Modules {
		require.Contains(t, registered, module.ID, "guide describes an unregistered module")
		require.False(t, described[module.ID], "duplicate purpose for %s", module.ID)
		described[module.ID] = true
		require.NotEmpty(t, module.Purpose, module.ID)
		require.NotEmpty(t, module.Owns, module.ID)
		require.NotEmpty(t, module.Boundary, module.ID)
		info, err := os.Stat(filepath.Join("..", module.SourcePath))
		require.NoError(t, err, module.ID)
		require.False(t, info.IsDir(), "module source should name a file: %s", module.ID)
		for _, connection := range module.Connections {
			require.Contains(t, registered, connection.Module, "unknown connection from %s", module.ID)
		}
	}
	for name := range registered {
		require.True(t, described[name], "registered module %s has no purpose in the public guide", name)
	}
}
