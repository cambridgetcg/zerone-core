package cross_stack_test

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	authtypes "github.com/zerone-chain/zerone/x/auth/types"
	liquiditypooltypes "github.com/zerone-chain/zerone/x/liquiditypool/types"
	substratebridgetypes "github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

type typescriptSDKWireFixture struct {
	Schema  string                    `json:"schema"`
	Vectors []typescriptSDKWireVector `json:"vectors"`
}

type typescriptSDKWireVector struct {
	Name    string          `json:"name"`
	TypeURL string          `json:"typeUrl"`
	Value   json.RawMessage `json:"value"`
	Hex     string          `json:"hex"`
}

func TestTypeScriptSDKWireVectors(t *testing.T) {
	fixture := loadTypeScriptSDKWireFixture(t)
	require.Equal(t, "zerone-typescript-sdk-wire-vectors/v1", fixture.Schema)
	require.NotEmpty(t, fixture.Vectors)

	seenNames := make(map[string]struct{}, len(fixture.Vectors))
	for _, vector := range fixture.Vectors {
		vector := vector
		t.Run(vector.Name, func(t *testing.T) {
			_, duplicate := seenNames[vector.Name]
			require.False(t, duplicate, "duplicate wire-vector name")
			seenNames[vector.Name] = struct{}{}

			message := newTypeScriptSDKWireMessage(t, vector.TypeURL)
			require.NoError(t, (protojson.UnmarshalOptions{
				DiscardUnknown: false,
			}).Unmarshal(vector.Value, message))

			goWire, err := (proto.MarshalOptions{Deterministic: true}).Marshal(message)
			require.NoError(t, err)
			require.Equal(t, vector.Hex, hex.EncodeToString(goWire),
				"Go protobuf encoding drifted from the shared TypeScript SDK golden")

			golden, err := hex.DecodeString(vector.Hex)
			require.NoError(t, err)
			decoded := newTypeScriptSDKWireMessage(t, vector.TypeURL)
			require.NoError(t, proto.Unmarshal(golden, decoded))
			require.True(t, proto.Equal(message, decoded),
				"Go protobuf decoder did not reproduce the shared fixture value")
		})
	}
}

func loadTypeScriptSDKWireFixture(t *testing.T) typescriptSDKWireFixture {
	t.Helper()

	_, sourceFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	fixturePath := filepath.Join(
		filepath.Dir(sourceFile),
		"..",
		"..",
		"sdk",
		"typescript",
		"testdata",
		"wire-vectors.json",
	)
	contents, err := os.ReadFile(fixturePath)
	require.NoError(t, err)

	var fixture typescriptSDKWireFixture
	require.NoError(t, json.Unmarshal(contents, &fixture))
	return fixture
}

func newTypeScriptSDKWireMessage(t *testing.T, typeURL string) proto.Message {
	t.Helper()

	switch typeURL {
	case "/zerone.auth.v1.MsgRegisterAccount":
		return &authtypes.MsgRegisterAccount{}
	case "/zerone.auth.v1.MsgRotateKey":
		return &authtypes.MsgRotateKey{}
	case "/zerone.liquiditypool.v1.MsgCreatePool":
		return &liquiditypooltypes.MsgCreatePool{}
	case "/zerone.substrate_bridge.v1.MsgSubmitExternalAttestation":
		return &substratebridgetypes.MsgSubmitExternalAttestation{}
	default:
		t.Fatalf("unsupported TypeScript SDK wire-vector type URL %q", typeURL)
		return nil
	}
}
