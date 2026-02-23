package types

import _ "embed"

// genesis_axioms.json is embedded at compile time for use by prepare-genesis.
// Contains 777 seed axioms across 16 epistemic domains.
//
//go:embed genesis_axioms.json
var GenesisAxiomsJSON []byte
