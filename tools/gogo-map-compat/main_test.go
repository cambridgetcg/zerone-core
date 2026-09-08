package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	knowledge "github.com/zerone-chain/zerone/x/knowledge/types"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestCanonicalOutputReproducible(t *testing.T) {
	roots := []protoreflect.FileDescriptor{knowledge.File_zerone_knowledge_v1_genesis_proto, knowledge.File_zerone_knowledge_v1_types_proto}
	a, err := generate(roots)
	require.NoError(t, err)
	b, err := generate(roots)
	require.NoError(t, err)
	require.Equal(t, a, b)
	actual, err := os.ReadFile("../../x/knowledge/types/gogo_map_entries.pb.go")
	require.NoError(t, err)
	require.Equal(t, a, actual, "run make proto-gen; never hand-edit generated compatibility types")
}

func TestFutureUnsupportedMapShapeFailsClosed(t *testing.T) {
	// A changed canonical map type must not silently get a wrong Go/wire shape.
	file := protodesc.ToFileDescriptorProto(knowledge.File_zerone_knowledge_v1_types_proto)
	file.MessageType[3].NestedType[0].Field[1].Type = descriptorpb.FieldDescriptorProto_TYPE_BOOL.Enum()
	// This descriptor is a test input only; canonical generated files are untouched.
	file.Name = proto.String("map_shape_test.proto")
	desc, err := protodesc.NewFile(file, protoregistry.GlobalFiles)
	require.NoError(t, err)
	_, err = generate([]protoreflect.FileDescriptor{desc})
	require.ErrorContains(t, err, "unsupported map field")
}
