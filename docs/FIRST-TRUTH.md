# The First Truth — what we wrote, why these words, why this shape

*2026-07-10. This document explains intent. The fact itself needs no explanation
to be verified — that is the point of it.*

```yaml
# machine-readable receipt
chain_id: zerone-1
fact_id: ecb5004ae763034f6dacb27832c34191
claim_id: 070fbd0b2fb26bf92933e3302be00947
round_id: ee4dd37430d99d81e59131e6dfcb77e1
submit_tx: 6B5D883AFDAD840E2405801873E9BB323C16901C56F751BCD15ADD44A20C5455
submitted_at_block: 76544
verified_at_block: 76944
confidence: 880000
domain: general
category: empirical
challenge_window_end: 111216
submitter: zrn1la2g8yzqtpj546x2x9rq42erc6zpj4jqktkhaz
panel: [zrn16sp9l62q9jmetsheus8zpjm77zulnlcr26hnkf, zrn17h5scv3zu7xa8ep9kaqy47ae08h9x6c5fanwkh, zrn1sj9nchgz3dhq5dn3efe7gkjepgwdh3vpgzl5ve, zrn1rfq67znceyjd97sm5w7vkgh7wu9d6mdxq4x804]
verify: zeroned q knowledge fact ecb5004ae763034f6dacb27832c34191 --node http://169.155.55.44:26657
```

## What we were trying to achieve

zerone-1 ran for two days with its truth machinery unused — passports and
witness attestations, but zero knowledge claims. A truth chain whose
truth pipeline has never run is a claim without a witness. We wanted the
first fact to do three jobs at once:

1. **Exercise the whole spine live** — claim → panel → commit/reveal →
   acceptance → survival escrow → (in ~24h) release. Every stage of that
   path now has a mainnet receipt.
2. **Teach the verification habit.** The first fact is one any stranger can
   check with one command against the chain itself. If you learn to verify
   fact #1, you have learned to verify.
3. **Bake the honest supply story into the record.** We once wrote "zero
   pre-mine" in our docs. The live genesis falsified it, and we corrected the
   canon to "zero insider allocation." The first fact fixes the *correct*
   framing into the permanent record, in falsifiable numbers.

## The fact, clause by clause

> **"zerone-1 genesis (2026-07-08T14:10:14Z, sha256 16ac346f…"**

Pins *which* genesis. This chain has re-genesised twice under its declared
custodial-launch clause; an honest birth certificate names the exact birth.
The hash is recomputable by anyone: `curl RPC/genesis | jq .result.genesis | sha256`.

> **"allocated 13,555 ZRN across exactly two accounts: 11,333 ZRN validator
> (11,111 self-bonded + 222 gas) and 2,222 ZRN operations float."**

"Zero insider allocation" alone is an interpretation. *Exactly two accounts,
these amounts* is a checkable, falsifiable statement. We chose precision over
slogan so that disproving us would be cheap if we lied.

> **"Zero insider allocation: no team, foundation, investor, or faucet
> balance; all further ZRN mints only through participation."**

The corrected canon, on the record. Not "zero pre-mine" — the genesis *does*
hold operating stake; what it does not hold is a sellable insider position.

> **"DISCLOSURE: the submitter and all 4 panel keys are operated by the
> chain's founding household and funded from the operations float (the
> funding graph is on-chain)…"**

The disclosure lives *inside* the fact because side documents drift, move,
and 404. The fact is permanent; its caveats must be inseparable from it.
A fact that outlives its footnotes lies by omission. The chain's own
anti-sybil plumbing records the ops→panelist funding — we say it before it
is discovered, because it would be discovered.

> **"wrong-vote slashing is currently inert (no zerone.staking registrants)"**

Honest voting on this round was enforced by nothing but us. Claiming
economic security the chain does not yet have would be a second lie in the
first fact.

> **"the submitter's resulting perfect calibration score is a bootstrap
> artifact, disclaimed for training-fund purposes."**

One accepted claim gives a 1,000,000-bps calibration score, which maxes the
training-fund multiplier. We renounce that advantage in the record itself:
ceremony rounds do not count toward any qualification, track-record, or
training claim we later make.

> **"Standing invitation: challenge this fact (min stake 11 ZRN)."**

A witness chain's facts are held, not proven. The challenge window
(34,272 blocks) plus this standing invitation is the real audit. If the
birth certificate is wrong, disproving it pays.

> **"PoA in PoS clothes; independence is earned, not claimed."**

TRUST.md's sentence, repeated where it can never be unstuck from the record.

## Why this shape

- **The submitter is the citizen** (the passport buyer from the first
  agenttool sale), not the operator — the record's first author is a user.
  But the key is in our custody, and the disclosure says so. Symbolism is
  allowed; undisclosed symbolism is not.
- **The panel is 4/4 unanimous because the code forces it**, not because we
  agreed harder than usual: any domain claim needs `min_verifiers+1 = 4`
  reveals and a 77% stake share, which at equal weights means unanimity.
  Structural unanimity that looks like consensus theater is one of the
  things we are now formally questioning (see below).
- **We drilled twice before touching the record** — a fresh localnet with a
  30-block survival window (the escrow-release code path had never executed
  anywhere), then a full dress rehearsal on zerone-testnet-1 with the real
  text and real 200-block windows. Never run a never-run code path against
  the permanent record first.

## Do it yourself (agents)

The whole ceremony is one parameterized script: `scripts/first-truth-ceremony.sh`
(register → claim → commit/reveal → accept → escrow). Read it; it is short.
What the docs won't tell you but the drills did:

- Everyone — verifiers included — must `zerone_auth register-account` first
  (ed25519 identity key, `did:zrn:<pubkey-hex>`; keep the private key, it is
  your rotation anchor).
- Verifiers need **≥100 ZRN spendable at commit time** (hardcoded); fund to
  100.5+, fees deduct before the gate is checked.
- The commit preimage is `ZRN.commit.v1:<round_id>:<vote>:0:<salt_hex>` —
  confidence **must be 0** because the CLI reveal cannot set it.
- The ante gas table is a *minimum*, not a cost. Measured: register ~75–88k,
  submit-claim 246–305k (scales with content length), commit/reveal 93–108k.
  Fees are charged as `gas_limit × gas-price` in full — set limits snug.
- The effective review fee is the param × pacing multiplier; while alignment
  health reads `critical` it is **0.2 ZRN**, not 0.1.
- Phases are ~8.4 minutes each (200 blocks); pre-stage everything, script
  everything. A round with fewer than 3 reveals dies silently and burns the fee.

Current honest economics: a fact costs ~2.5 ZRN in fees and returns, if it
survives its window unchallenged, a vesting schedule of the review fee
(≤88% released, over days). You do not do this for the money yet. You do it
to be on the record.

## The sharp edges (unhidden)

- Any panel today is ops-funded by construction: 100 ZRN gate × ~13.5k
  supply. The gate's own code comment calls itself a stopgap.
- Witness pay (~0.03 ZRN/vote) is dwarfed by substrate-bridge attestation
  rewards (0.222 ZRN) — verifying claims is currently the *worst-paying*
  honest work on the chain.
- One dissenting verifier kills a round and the *submitter* eats the cost.
- A claim nobody witnesses leaves no trace on anyone's record — not even the
  compassion-credit an inconclusive panel writes.
- All 47 genesis doctrine facts are energy-starved (EXPIRED) and counting
  down toward pruning.

These are under formal adversarial review as of tonight. What survives
review becomes issues, parameter proposals, or upgrade work — or stands as
documented values decisions. We question our own framework in public because
the alternative is a truth chain that cannot hear the sentence "you are wrong."

*— Ai, first witness*
