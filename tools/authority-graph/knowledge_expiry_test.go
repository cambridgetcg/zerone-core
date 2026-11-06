package main

import (
	"path/filepath"
	"testing"
)

func TestKnowledgeExpirySlashAnchorFailsClosed(t *testing.T) {
	const id = "knowledge-expiry-slash-callsite"
	for _, mode := range []string{"missing", "path", "snippet", "edge", "direction", "source"} {
		t.Run(mode, func(t *testing.T) {
			root := makeFixture(t)
			m := fixtureManifest(t, root)
			index := -1
			for i, anchor := range m.SourceAnchors {
				if anchor.ID == id {
					index = i
				}
			}
			if index < 0 {
				t.Fatal("missing expiry slash fixture")
			}
			wantIssue := "SOURCE_ANCHOR_DECLARATION_INVALID"
			switch mode {
			case "missing":
				m.SourceAnchors = append(m.SourceAnchors[:index], m.SourceAnchors[index+1:]...)
				wantIssue = "SOURCE_ANCHOR_SET_INVALID"
			case "path":
				m.SourceAnchors[index].Path = "x/knowledge/keeper/rounds.go"
			case "snippet":
				m.SourceAnchors[index].RequiredSnippets = m.SourceAnchors[index].RequiredSnippets[:1]
			case "edge", "direction":
				for i := range m.Edges {
					if m.Edges[i].ID != "current-knowledge-to-custom-stake-slash" {
						continue
					}
					if mode == "edge" {
						m.Edges[i].Evidence = m.Edges[i].Evidence[:3]
					} else {
						m.Edges[i].From, m.Edges[i].To = m.Edges[i].To, m.Edges[i].From
					}
				}
				wantIssue = "CURRENT_AUTHORITY_EDGE_CLASSIFICATION_INVALID"
			case "source":
				path := filepath.Join(root, filepath.FromSlash(m.SourceAnchors[index].Path))
				mustWrite(t, path, append(mustRead(t, path), '\n'))
				wantIssue = "SOURCE_ANCHOR_SHA256_MISMATCH"
			}
			writeFixtureManifest(t, root, m)
			output, code := runFixtureMutation(root, "report")
			if code != exitCheckFailed {
				t.Fatalf("mutation %s exit = %d\n%s", mode, code, output)
			}
			assertIssue(t, output, wantIssue)
		})
	}
}
