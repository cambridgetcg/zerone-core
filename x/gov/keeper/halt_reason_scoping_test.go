package keeper_test

import (
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// These are exact function-body bytes from 77521a0d, independently compared
// before this test was added. Renaming the legacy functions changes no body.
func TestHaltReasonLegacyBodiesMatchPreRepairSource(t *testing.T) {
	for _, tc := range []struct{ path, name, digest string }{
		{"../../emergency/keeper/gov_adapters_legacy.go", "CountHaltsForReason", "ae83b2419a510a5ea56a17fe2ad99cbb7c082ad54d1b2a92d4d7afaea2f1c537"},
		{"phase_rollback_legacy.go", "validatePhaseRollbackLegacy", "38d0d6d4f8c8523142bbaf9c7e0f4213a84f7b780a1c633d98106a27c9bb3f88"},
		{"phase_transition_legacy.go", "beginBlockPhaseTransitionLegacy", "2415d731cb98bc14dda905b2a4259666260985c6f1b96544d3ecd4e66f4c044e"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source, err := os.ReadFile(tc.path)
			require.NoError(t, err)
			set := token.NewFileSet()
			file, err := parser.ParseFile(set, tc.path, source, 0)
			require.NoError(t, err)
			found := false
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Name.Name != tc.name {
					continue
				}
				found = true
				body := source[set.Position(fn.Body.Pos()).Offset:set.Position(fn.Body.End()).Offset]
				require.Equal(t, tc.digest, fmt.Sprintf("%x", sha256.Sum256(body)))
			}
			require.True(t, found)
		})
	}
}
