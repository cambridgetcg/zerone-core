# Launch Checklist — Superseded

The former `zerone-testnet-1` checklist mixed obsolete testnet assumptions with
later mainnet ceremony requirements. It is not launch or deployment authority.

Current posture:

- `zerone-1` is live; this source consolidation is not a validator rollout.
- `zerone-testnet-1` is a live legacy playground in observe-only mode for this
  source head.
- `zerone-2` is **NO-GO** until its fail-closed release, authority, halt, and
  signed ceremony gates all pass.

Use these current artifacts instead:

- [release posture](../README.md#release-posture)
- [validator safety](VALIDATOR-GUIDE.md)
- [`zerone-testnet-1` status](../networks/zerone-testnet-1/README.md)
- [`zerone-2` release process](../deploy/networks/zerone-2/README.md)
- [`zerone-2` dress rehearsal](../scripts/zerone-2-dress-rehearsal.sh)
- [`zerone-2` ceremony gate](../scripts/zerone-2-ceremony-test.sh)

A network-specific launch checklist must be generated from, and shipped beside,
the exact signed release manifest and ceremony packet. Generic unchecked boxes
must never substitute for those artifacts.
