# Operator guide — release-bound prerequisites

Live node installation is paused for the consolidated source head.

Do not clone a moving branch, pull genesis from an unsigned RPC, install the
result as `zeroned`, restore an unbound snapshot, or submit a validator
transaction. Public endpoints show that a chain is answering; they do not prove
which source, binary, genesis representation, or authority is in use.

An operator packet must bind all of the following before installation:

- source commit, build platform, compiler, flags, and dependency lock;
- binary and image digests, signatures, SBOM, and provenance;
- chain ID, exact genesis representation, and its digest;
- seed/persistent-peer identities from an authenticated channel;
- gas, pruning, snapshot, state-sync, signer, and fencing policy;
- approved upgrade name/height and old-to-new binary sequence; and
- incident, rollback, and post-upgrade verification procedures.

Prepare infrastructure and offline key-handling procedures without connecting
it to a live chain. Never copy a live validator key or signing state to a second
machine, and never start two signers for the same consensus key.

When a packet is published, verify each artifact independently and use only the
commands shipped inside that exact packet. Historical command snippets are not
a substitute.
