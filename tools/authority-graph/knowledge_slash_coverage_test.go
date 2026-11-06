package main

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestKnowledgeAdditionalSlashAnchorsFailClosed(t *testing.T) {
	for _, id := range []string{"knowledge-vindication-slash-callsite", "knowledge-missed-verification-slash-callsite"} {
		for _, mode := range []string{"missing", "path", "snippet", "edge", "direction", "source"} {
			t.Run(id+"/"+mode, func(t *testing.T) {
				root := makeFixture(t)
				m := fixtureManifest(t, root)
				index := -1
				for i, a := range m.SourceAnchors {
					if a.ID == id {
						index = i
					}
				}
				if index < 0 {
					t.Fatal("missing source anchor")
				}
				issue := "SOURCE_ANCHOR_DECLARATION_INVALID"
				switch mode {
				case "missing":
					m.SourceAnchors = append(m.SourceAnchors[:index], m.SourceAnchors[index+1:]...)
					issue = "SOURCE_ANCHOR_SET_INVALID"
				case "path":
					m.SourceAnchors[index].Path = "x/knowledge/keeper/rounds.go"
				case "snippet":
					m.SourceAnchors[index].RequiredSnippets = m.SourceAnchors[index].RequiredSnippets[:1]
				case "edge", "direction":
					for i := range m.Edges {
						if m.Edges[i].ID == "current-knowledge-to-custom-stake-slash" {
							if mode == "edge" {
								m.Edges[i].Evidence = removeString(m.Edges[i].Evidence, id)
							} else {
								m.Edges[i].From, m.Edges[i].To = m.Edges[i].To, m.Edges[i].From
							}
						}
					}
					issue = "CURRENT_AUTHORITY_EDGE_CLASSIFICATION_INVALID"
				case "source":
					path := filepath.Join(root, filepath.FromSlash(m.SourceAnchors[index].Path))
					mustWrite(t, path, append(mustRead(t, path), '\n'))
					issue = "SOURCE_ANCHOR_SHA256_MISMATCH"
				}
				writeFixtureManifest(t, root, m)
				out, code := runFixtureMutation(root, "report")
				if code != exitCheckFailed {
					t.Fatalf("mutation exit %d: %s", code, out)
				}
				assertIssue(t, out, issue)
			})
		}
	}
}

// Exhaustive for direct calls to the declared slash interface in production
// knowledge keeper Go files. This includes the currently uncalled helper; it
// does not claim runtime reachability, indirect-call discovery or live execution.
func TestEveryDirectKnowledgeSlashCallHasSourceEvidence(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test source unavailable")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
	var m manifest
	if err := json.Unmarshal(mustRead(t, filepath.Join(root, manifestPath)), &m); err != nil {
		t.Fatal(err)
	}
	anchors := map[string]sourceAnchor{}
	for _, a := range m.SourceAnchors {
		anchors[a.Path] = a
	}
	var edgeEvidence []string
	for _, e := range m.Edges {
		if e.ID == "current-knowledge-to-custom-stake-slash" {
			edgeEvidence = e.Evidence
		}
	}
	files, err := filepath.Glob(filepath.Join(root, "x/knowledge/keeper/*.go"))
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		set := token.NewFileSet()
		node, err := parser.ParseFile(set, path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(node, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || (sel.Sel.Name != "SlashValidator" && sel.Sel.Name != "SlashValidatorToModule") {
				return true
			}
			count++
			relative, err := filepath.Rel(root, path)
			if err != nil {
				t.Fatal(err)
			}
			a, present := anchors[filepath.ToSlash(relative)]
			if !present {
				t.Errorf("unanchored slash at %s", set.Position(call.Pos()))
				return true
			}
			var rendered bytes.Buffer
			if err := printer.Fprint(&rendered, set, call); err != nil {
				t.Fatal(err)
			}
			covered := false
			for _, snippet := range a.RequiredSnippets {
				if strings.Contains(snippet, rendered.String()) {
					covered = true
				}
			}
			if !covered {
				t.Errorf("unbound slash snippet at %s", set.Position(call.Pos()))
			}
			bound := false
			for _, id := range edgeEvidence {
				if id == a.ID {
					bound = true
				}
			}
			if !bound {
				t.Errorf("slash anchor %s not bound to knowledge->custom-staking edge", a.ID)
			}
			return true
		})
	}
	if count != 5 {
		t.Fatalf("direct slash call census changed: got %d, expected 5; review coverage and semantics", count)
	}
}
