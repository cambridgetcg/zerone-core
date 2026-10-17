package cross_stack_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"cosmossdk.io/core/appmodule"
	"github.com/stretchr/testify/require"

	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	qualificationtypes "github.com/zerone-chain/zerone/x/qualification/types"
)

// Keep this structural check beside the runtime registration check: calling a
// keeper again can be invisible in default state even though it runs twice.
func TestHarnessAdvanceBlocksUsesApplicationBlockers(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "harness_test.go", nil, 0)
	require.NoError(t, err)

	var advance *ast.FuncDecl
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Name.Name == "AdvanceBlocks" && fn.Recv != nil {
			require.Nil(t, advance, "AdvanceBlocks must have one definition")
			advance = fn
		}
	}
	require.NotNil(t, advance, "AdvanceBlocks must remain covered by this guard")
	require.Len(t, advance.Recv.List, 1)
	require.Len(t, advance.Recv.List[0].Names, 1)
	receiver := advance.Recv.List[0].Names[0].Name

	calls := map[string]int{}
	ast.Inspect(advance.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		hook, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || (hook.Sel.Name != "BeginBlocker" && hook.Sel.Name != "EndBlocker") {
			return true
		}
		owner, ok := hook.X.(*ast.SelectorExpr)
		require.True(t, ok, "block hooks must use the application module manager")
		require.Equal(t, "App", owner.Sel.Name, "do not replay individual keeper hooks")
		base, ok := owner.X.(*ast.Ident)
		require.True(t, ok)
		require.Equal(t, receiver, base.Name)
		calls[hook.Sel.Name]++
		return true
	})
	require.Equal(t, map[string]int{"BeginBlocker": 1, "EndBlocker": 1}, calls)
}

func TestHarnessBeginBlockersRegistered(t *testing.T) {
	h := NewTestHarness(t)
	manager := h.App.ModuleManager
	for _, name := range []string{knowledgetypes.ModuleName, qualificationtypes.ModuleName} {
		count := 0
		for _, scheduled := range manager.OrderBeginBlockers {
			if scheduled == name {
				count++
			}
		}
		require.Equal(t, 1, count, "%s must be scheduled exactly once", name)
		_, ok := manager.Modules[name].(appmodule.HasBeginBlocker)
		require.True(t, ok, "%s must implement the SDK begin-block interface", name)
	}

	// No direct keeper fallback is needed: the constructed app reaches its
	// registered begin blockers successfully in the harness's next-block context.
	_, err := h.App.BeginBlocker(h.Ctx.WithBlockHeight(h.Height() + 1))
	require.NoError(t, err)
}
