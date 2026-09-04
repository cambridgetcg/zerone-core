# Mainnet preparation — 2026-09-04

This is a dated engineering observation, not a signed release or launch decision.
Source inspection began at GitHub main
`056f0c180d2e9f5a42a2a387a353baa960f1de8c`. Live observations below were made
between 19:18 and 19:20 UTC using public reads only.

## Release sequencing repaired

The previous sequence required notice publication before CUTOVER was signed,
while assigning publication authority to CUTOVER itself. The new
[`notice-prepublish` gate](../../deploy/networks/zerone-2/AUTHORITY-BUNDLE.md)
verifies a separate main-key PRE-NOTICE decision after completed private soak
and halt rehearsal. It binds the exact notice, HTTPS destination, proposed
F/A/H, and publication deadline without requiring future publication or
CUTOVER artifacts.

CUTOVER then requires that decision's exact payload/signature pair, the same
plan and notice, and canonical v2 publication evidence. The captured response
body must byte-match the notice. CUTOVER initiation, archive transition, and
FINAL retain the exact publication-evidence hash. Later historical verification
does not renew expired publication authority. Captured delivery and time remain
reviewed operator attestations, not cryptographic proof of Internet publication.

The [runbook](../../deploy/networks/zerone-2/CUTOVER.md) and
[decision checklist](../../deploy/networks/zerone-2/GO-NO-GO.md) describe the
complete sequence. The new gate has no publication client and no deployment,
broadcast, halt, or DNS action.

## Live network observation

| Observation | Result | Read surface |
| --- | --- | --- |
| Chain progress | `1247591 -> 1247592 -> 1247593`, approximately 30.007 seconds per block, `catching_up=false` | `/status` and `/block` on direct RPC, `rpc.zerone.ai`, and the dashboard proxy |
| Runtime identity | `zerone-1`, CometBFT `0.38.19`, application version `dev` | `/status`, `/abci_info` |
| Consensus concentration | One validator, all `11111` voting power, zero peers | `/validators`, `/net_info` |
| Consensus gas | `max_gas=-1` | `/consensus_params` |
| Supply | `13667372296 uzrn` across direct LCD, API gateway, and dashboard | `/cosmos/bank/v1beta1/supply` |
| Origin exposure | Direct legacy RPC port 26657 and LCD port 1317 remain reachable | Existing official origin |
| Legacy REST syncing | HTTP 501, `Not Implemented` | `api.zerone.ai/cosmos/base/tendermint/v1beta1/syncing` |
| Dashboard syncing | HTTP 200, `syncing=false`, through its RPC compatibility adapter | `zerone.ai/api/rest/cosmos/base/tendermint/v1beta1/syncing` |
| Pi | `enabled=false`, `walletProofEnabled=false`, `authenticated=false`; challenge and authorize GETs unavailable | `zerone.ai/api/pi/me` |

At height 1247593, the three RPC surfaces agreed on block hash
`58B3BBE2390432A9B0442EF0D5A61B391B08BFB1A1BF0E3784C4E41E812DE700`.
The website's HTML, JavaScript, CSS, and authority JSON still matched the
September 3 onboarding deployment `a86755e4` at source `113f9a26`.

These observations establish continued legacy-chain service, not that the new
source is deployed. The new semantic query gateway requires typed REST syncing
responses and cannot simply be pointed at this legacy LCD. Archive mode also
requires the later exact frozen A/A evidence.

## Remaining preparation priorities

1. Establish the protected production-signing environment and reviewer policy.
   The authenticated GitHub request for `zerone-production-signing` returned
   HTTP 404 during this preparation pass. Verify the environment-level policy
   and all three approved image variables through the API before dispatch;
   workflow variable lookup alone does not establish their scope.
2. Produce the actual signed release inputs on the required release hosts:
   exact source/tag, independent main/transition authority pins, new role
   identities, reproducible binaries and three immutable images, reviewed
   SBOM/provenance/vulnerability decisions, Sigstore bundles and trusted root,
   and the tested monitoring evidence. The committed examples are templates.
3. Execute the private rehearsal and decision sequence with those exact inputs.
   Real stopped-state census, signer replacement, halt and archive evidence,
   and public activation remain subsequent operational work under the signed
   phase decisions.
4. Resolve remaining public-surface exposure within the appropriate release:
   direct legacy origins remain open, and the dashboard proxy still admits
   `broadcast_tx_sync`. Removing that relay also requires updating its
   existing-account send flow. This pass sent no broadcast probes and changed
   no endpoint.

One-validator availability and custody remain material launch limits. A passing
local rehearsal or source CI run does not establish independent production
validators, safe production signer custody, or public launch readiness.
