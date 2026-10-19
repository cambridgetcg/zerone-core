package cross_stack_test

import (
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"os"
	"strings"
	"testing"

	"cosmossdk.io/core/appmodule"
	"github.com/stretchr/testify/require"

	knowledgetypes "github.com/zerone-chain/zerone/x/knowledge/types"
	qualificationtypes "github.com/zerone-chain/zerone/x/qualification/types"
)

// This deliberately closed template pins the whole block step, not just named
// hook calls: aliases, manager/module dispatch, helpers and extra control flow
// must not silently escape the guard. Even a legitimate helper extraction needs
// explicit review of this contract. Comments and whitespace are not pinned.
// This is a source-shape guard, not proof of which registered callbacks execute.
const harnessAdvanceBlocksContract = `func (h *TestHarness) AdvanceBlocks(n int) {
	h.T.Helper()
	for i := 0; i < n; i++ {
		h.currentHeight++
		h.Ctx = h.App.NewContext(true).
			WithBlockHeight(h.currentHeight).
			WithChainID(testChainID).
			WithBlockHeader(cmtproto.Header{
				Height: h.currentHeight,
				ChainID: testChainID,
			})
		_, err := h.App.BeginBlocker(h.Ctx)
		require.NoError(h.T, err, "BeginBlocker at height %d", h.currentHeight)
		_, err = h.App.EndBlocker(h.Ctx)
		require.NoError(h.T, err, "EndBlocker at height %d", h.currentHeight)
	}
}`

func harnessBlockStepTokens(t *testing.T, source []byte) []string {
	t.Helper()
	var scan scanner.Scanner
	scan.Init(token.NewFileSet().AddFile("", -1, len(source)), source,
		func(pos token.Position, msg string) { t.Fatalf("%s: %s", pos, msg) }, 0)
	var tokens []string
	for {
		_, kind, literal := scan.Scan()
		if kind == token.EOF {
			return tokens
		}
		if kind == token.SEMICOLON {
			literal = "" // explicit and newline-inserted semicolons are equivalent
		}
		tokens = append(tokens, kind.String()+":"+literal)
	}
}

// Keep this beside the registration check: a replay can be invisible in default
// runtime state. Pinning the full method also preserves loop bounds, ordering
// and error checks without pretending to resolve arbitrary Go call graphs.
func TestHarnessAdvanceBlocksUsesApplicationBlockers(t *testing.T) {
	source, err := os.ReadFile("harness_test.go")
	require.NoError(t, err)
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "harness_test.go", source, 0)
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
	method := source[fset.Position(advance.Pos()).Offset:fset.Position(advance.End()).Offset]
	require.Equal(t, harnessBlockStepTokens(t, []byte(harnessAdvanceBlocksContract)), harnessBlockStepTokens(t, method),
		"AdvanceBlocks changed: keep one direct App.BeginBlocker and App.EndBlocker per height; "+
			"helpers, aliases, other dispatch and control-flow refactors require explicit guard review")
}

func TestHarnessAdvanceBlocksContract(t *testing.T) {
	want := harnessBlockStepTokens(t, []byte(harnessAdvanceBlocksContract))
	t.Run("comments_and_whitespace", func(t *testing.T) {
		source := strings.ReplaceAll(harnessAdvanceBlocksContract, "	", "    ")
		source = strings.ReplaceAll(source, "h.T.Helper()", "h.T.Helper(); // mark helper\n")
		require.Equal(t, want, harnessBlockStepTokens(t, []byte(source)))
	})
	const begin = "_, err := h.App.BeginBlocker(h.Ctx)"
	for _, tc := range []struct{ name, before, after string }{
		{"keeper_replay", begin, "h.KnowledgeKeeper.BeginBlocker(h.Ctx); " + begin},
		{"manager_begin", begin, "h.App.ModuleManager.BeginBlock(h.Ctx); " + begin},
		{"manager_end", begin, "h.App.ModuleManager.EndBlock(h.Ctx); " + begin},
		{"module_begin", begin, "h.App.ModuleManager.Modules[knowledgetypes.ModuleName].(appmodule.HasBeginBlocker).BeginBlock(h.Ctx); " + begin},
		{"method_value", begin, "f := h.KnowledgeKeeper.BeginBlocker; f(h.Ctx); " + begin},
		{"replay_helper", begin, "h.replayBlockers(); " + begin},
		{"legitimate_helper", begin, "err := h.beginBlock()"},
		{"conditional_begin", begin, "var err error; if i == 0 { _, err = h.App.BeginBlocker(h.Ctx) }"},
		{"extra_iteration", "i < n", "i <= n"},
		{"discarded_error", "require.NoError(h.T, err, \"BeginBlocker at height %d\", h.currentHeight)", "_ = err"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, 1, strings.Count(harnessAdvanceBlocksContract, tc.before))
			source := strings.Replace(harnessAdvanceBlocksContract, tc.before, tc.after, 1)
			_, err := parser.ParseFile(token.NewFileSet(), "mutation.go", "package cross_stack_test\n"+source, 0)
			require.NoError(t, err, "the negative control must remain valid Go syntax")
			require.NotEqual(t, want, harnessBlockStepTokens(t, []byte(source)))
		})
	}
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

	// The application entry point returns no error in the next-block context.
	// Scheduling and interface checks above do not prove callback execution.
	_, err := h.App.BeginBlocker(h.Ctx.WithBlockHeight(h.Height() + 1))
	require.NoError(t, err)
}
