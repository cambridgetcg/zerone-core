package types_test

import (
	"bytes"
	"testing"

	"github.com/zerone-chain/zerone/x/emergency/types"
)

func TestResumeAttemptKeyHasUnambiguousComponents(t *testing.T) {
	first := types.ResumeAttemptKey("incident-ab", "guardian-c")
	second := types.ResumeAttemptKey("incident-a", "bguardian-c")
	if bytes.Equal(first, second) {
		t.Fatal("variable-width quarantine and proposer components collided")
	}
	if !bytes.HasPrefix(first, types.ResumeAttemptKeyPrefix) {
		t.Fatal("resume attempt key lost its consensus prefix")
	}
}
