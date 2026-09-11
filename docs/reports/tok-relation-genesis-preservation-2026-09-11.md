# Preserve ToK relationships through knowledge genesis

Ordinary typed Fact relationships were stored in canonical forward and reverse
indexes, but knowledge genesis exported only the Fact payloads. Those payloads
normally have empty embedded relationship arrays. A protobuf-JSON export/import
could therefore preserve Facts and status history while losing their graph
relationships.

Knowledge genesis now carries `fact_relation_state`, a presence-bearing
`FactRelationGenesis` inventory. It retains every field of each current canonical
relationship: source, target, relation type, creation height, creator, inference
type, declared inference strength and method. The inventory contains one record
per ordered source/target pair, matching existing storage cardinality.

## Import and compatibility

- An absent inventory retains historical genesis behavior, including default
  doctrine relationship seeding. It does not recover ordinary relationships lost
  by an older export.
- A new export must be imported with a reader that understands
  `fact_relation_state`; an older binary is not a graph-preserving importer for
  this format.
- A present inventory replaces both canonical indexes after doctrine seeding.
  An explicitly empty inventory restores an empty graph. Modified or deliberately
  absent doctrine relationships are therefore not overwritten or resurrected.
- Embedded Fact arrays and original Claim relations retain their existing roles;
  neither is interpreted as an alternative canonical inventory.
- Duplicate pairs, missing endpoints, ambiguous slash-delimited endpoint IDs,
  unknown fields or enum values, invalid text and oversized inventories refuse
  import before genesis writes. Defined historical zero values, absent
  attribution, self-relations and cycles remain representable. No current claim
  admission or scientific-review rule is retroactively applied to these records.
- Replacement of the two relationship indexes uses a cached context. A storage
  error during replacement does not leave a partly replaced graph.

Export checks both complete relation indexes, key/payload identity, canonical
protobuf representation, matching mirrored payloads and endpoint presence.
Malformed records, missing or orphaned reverse records and mismatched metadata
refuse export instead of producing a partial graph. Ordering is deterministic.
The inventory is bounded at 100,000 stored index rows (50,000 relationships) and
64 MiB of combined key/payload bytes. This is an import/export resource ceiling,
not a lifetime admission limit or a promise of unlimited archival capacity.

## Scope

This repairs the module JSON genesis transfer path. Ordinary restart and raw
store backup use different paths. It does not backfill missing historical
relationships, retain multiple independent vouches at a single pair, change
relationship meanings or alter ToK structural-root versions.

The change adds no transaction, block handler, store namespace or migration.
Knowledge consensus version remains 9. Existing state continues to use its
current relationship indexes; a separately reviewed genesis file supplies an
explicit inventory when initializing a new state. This source release is not an
upgrade or activation of the legacy live chain.

Regression coverage exercises protobuf-JSON preservation, both indexes and graph
selection, full provenance, legacy absence versus explicit emptiness, doctrine
overrides, malformed inventories and storage failures. These local checks do not
establish scientific validity, complete module-state preservation or live-network
adoption.
