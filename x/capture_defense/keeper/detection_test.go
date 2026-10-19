package keeper

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math/big"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zerone-chain/zerone/x/capture_defense/types"
)

func TestTimingCorrelationExact(t *testing.T) {
	const max = ^uint64(0)
	for _, tc := range []struct {
		name   string
		blocks []uint64
		want   uint64
	}{
		{"no samples", nil, 0},
		{"one sample", []uint64{100}, 0},
		{"identical", []uint64{100, 100}, types.BPSScale},
		{"below threshold", []uint64{0, 3}, types.BPSScale},
		{"exact threshold", []uint64{0, 4}, 0},
		{"above threshold", []uint64{0, 5}, 0},
		{"fractional mean below", []uint64{0, 0, 4}, types.BPSScale},
		{"fractional mean above", []uint64{0, 0, 5}, 0},
		{"population not sample variance", []uint64{0, 2, 4}, types.BPSScale},
		// Exact variance is 324/81 = 4. The old float accumulation can
		// classify this differently with fused and unfused multiply-add.
		{"reported architecture flip", []uint64{2718285, 2718286, 2718292, 2718287, 2718288, 2718288, 2718287, 2718285, 2718288}, 0},
		{"beyond float precision below", []uint64{1 << 54, 1<<54 + 3}, types.BPSScale},
		{"beyond float precision equal", []uint64{1 << 54, 1<<54 + 4}, 0},
		{"beyond float precision above", []uint64{1 << 54, 1<<54 + 5}, 0},
		{"overflowing sum identical", []uint64{max, max}, types.BPSScale},
		{"overflowing sum below", []uint64{max - 3, max}, types.BPSScale},
		{"overflowing sum equal", []uint64{max - 4, max}, 0},
		{"overflowing sum above", []uint64{max - 5, max}, 0},
		{"full uint64 range", []uint64{0, max}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			history := []*types.VerificationHistoryEntry{{SubmitBlocks: tc.blocks}}
			if got := timingCorrelation(history); got != tc.want {
				t.Fatalf("timingCorrelation(%v) = %d, want %d", tc.blocks, got, tc.want)
			}
		})
	}
}

func TestTimingCorrelationEligibleRoundDenominator(t *testing.T) {
	for _, tc := range []struct {
		name    string
		history []*types.VerificationHistoryEntry
		want    uint64
	}{
		{"no history", nil, 0},
		{"no eligible rounds", []*types.VerificationHistoryEntry{{}, {SubmitBlocks: []uint64{1}}}, 0},
		{"eligible rounds only", []*types.VerificationHistoryEntry{
			{},
			{SubmitBlocks: []uint64{1}},
			{SubmitBlocks: []uint64{0, 1}},
			{SubmitBlocks: []uint64{0, 4}},
			{SubmitBlocks: []uint64{0, 5}},
		}, types.BPSScale / 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := timingCorrelation(tc.history); got != tc.want {
				t.Fatalf("timingCorrelation = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestTimingCorrelationLargeRoundBoundary(t *testing.T) {
	blocks := make([]uint64, 4096)
	for i := range blocks {
		blocks[i] = 100
		if i%2 == 1 {
			blocks[i] = 104
		}
	}
	history := []*types.VerificationHistoryEntry{{SubmitBlocks: blocks}}
	if got := timingCorrelation(history); got != 0 {
		t.Fatalf("variance exactly 4: got %d", got)
	}
	blocks[0] = 101
	if got := timingCorrelation(history); got != types.BPSScale {
		t.Fatalf("variance just below 4: got %d", got)
	}
	blocks[0] = 99
	if got := timingCorrelation(history); got != 0 {
		t.Fatalf("variance just above 4: got %d", got)
	}
}

// Use a different identity from the implementation: population variance is
// sum_{i<j}(x_i-x_j)^2 / n^2. Keep this quadratic oracle confined to short tests.
func pairwiseTimingCorrelation(blocks []uint64) uint64 {
	if len(blocks) < 2 {
		return 0
	}
	var numerator, a, b, diff big.Int
	for i, x := range blocks {
		for _, y := range blocks[i+1:] {
			a.SetUint64(x)
			b.SetUint64(y)
			diff.Sub(&a, &b)
			diff.Mul(&diff, &diff)
			numerator.Add(&numerator, &diff)
		}
	}
	n := uint64(len(blocks)) // test inputs are bounded to 20 samples
	if numerator.Cmp(new(big.Int).SetUint64(4*n*n)) < 0 {
		return types.BPSScale
	}
	return 0
}

func TestTimingCorrelationAgainstPairwiseOracle(t *testing.T) {
	rng := rand.New(rand.NewSource(20261019))
	for trial := 0; trial < 1000; trial++ {
		blocks := make([]uint64, 2+rng.Intn(19))
		for i := range blocks {
			blocks[i] = rng.Uint64()
		}
		check := func() {
			t.Helper()
			before := append([]uint64(nil), blocks...)
			want := pairwiseTimingCorrelation(blocks)
			got := timingCorrelation([]*types.VerificationHistoryEntry{{SubmitBlocks: blocks}})
			if got != want {
				t.Fatalf("trial %d, blocks %v: got %d, want %d", trial, blocks, got, want)
			}
			for i := range blocks {
				if blocks[i] != before[i] {
					t.Fatal("timing calculation changed its input")
				}
			}
		}
		check()
		// Exercise tight, threshold-adjacent distributions at low heights,
		// beyond float's integer precision, and near MaxUint64, in both orders.
		for i := range blocks {
			blocks[i] = rng.Uint64() % 9
		}
		lowBlocks := append([]uint64(nil), blocks...)
		for _, offset := range []uint64{0, 1 << 54, ^uint64(0) - 8} {
			for i := range blocks {
				blocks[i] = lowBlocks[i] + offset
			}
			check()
			for i, j := 0, len(blocks)-1; i < j; i, j = i+1, j-1 {
				blocks[i], blocks[j] = blocks[j], blocks[i]
			}
			check()
		}
	}
}

// The capture-defense keeper and its state types must not reintroduce native
// floating-point arithmetic. This is a source guard, not a transitive audit of
// SDK dependencies: reject float/complex types, literals and common float APIs,
// including aliases of math and math/big. Comments and strings are not code.
func floatSyntax(fset *token.FileSet, file *ast.File) []string {
	var violations []string
	bigNames := map[string]bool{}
	for _, imp := range file.Imports {
		if imp.Path.Value == `"math"` || imp.Path.Value == `"math/cmplx"` {
			violations = append(violations, fmt.Sprintf("%s: floating-point package %s", fset.Position(imp.Pos()), imp.Path.Value))
		}
		if imp.Path.Value == `"math/big"` {
			name := "big"
			if imp.Name != nil {
				name = imp.Name.Name
			}
			bigNames[name] = true
		}
	}
	ast.Inspect(file, func(node ast.Node) bool {
		bad := false
		switch n := node.(type) {
		case *ast.Ident:
			switch n.Name {
			case "float32", "float64", "complex64", "complex128", "ParseFloat", "ParseComplex", "Float32", "Float64", "NewFloat":
				bad = true
			case "Float":
				bad = bigNames["."]
			}
		case *ast.BasicLit:
			bad = n.Kind == token.FLOAT || n.Kind == token.IMAG
		case *ast.SelectorExpr:
			if pkg, ok := n.X.(*ast.Ident); ok && bigNames[pkg.Name] && n.Sel.Name == "Float" {
				bad = true
			}
		}
		if bad {
			violations = append(violations, fmt.Sprintf("%s: floating-point syntax", fset.Position(node.Pos())))
		}
		return true
	})
	return violations
}

func TestCaptureDefenseStatePathHasNoFloat(t *testing.T) {
	for _, dir := range []string{".", "../types"} {
		fset := token.NewFileSet()
		packages, err := parser.ParseDir(fset, dir, func(info os.FileInfo) bool {
			return !strings.HasSuffix(info.Name(), "_test.go")
		}, 0)
		if err != nil {
			t.Fatal(err)
		}
		if len(packages) != 1 {
			t.Fatalf("%s: expected one production package, got %d", dir, len(packages))
		}
		for _, pkg := range packages {
			for name, file := range pkg.Files {
				for _, violation := range floatSyntax(fset, file) {
					t.Errorf("%s: %s", filepath.Clean(name), violation)
				}
			}
		}
	}
}

func TestCaptureDefenseFloatGuardRejectsControls(t *testing.T) {
	for _, source := range []string{
		"package fixture; var x float64",
		"package fixture; type score = float32",
		"package fixture; var x = 0.5",
		"package fixture; var x = 2i",
		"package fixture; import m \"math\"; var x = m.Sqrt(4)",
		"package fixture; import b \"math/big\"; var x b.Float",
		"package fixture; import . \"math/big\"; var x Float",
		"package fixture; import \"strconv\"; var x, _ = strconv.ParseFloat(\"1\", 64)",
	} {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "control.go", source, 0)
		if err != nil {
			t.Fatal(err)
		}
		if len(floatSyntax(fset, file)) == 0 {
			t.Fatalf("guard accepted %q", source)
		}
	}
}
