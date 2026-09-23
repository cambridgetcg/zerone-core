# Review subject: knowledge-11 participant packages

**Draft only: no native claim has been submitted and no review window is open.**
This is operator-authored packaging work. A reviewer within the operator's
control context may choose their own judgment without establishing a separate
effective controller. No personal identity or review verdict is assigned here.

Proposed native claim, method **`M-COMPUTATIONAL`**:

> For Linux amd64 and Darwin arm64, the `zerone-dev-1-5542b221864c-participants-v1`
> bundles preserve the exact executable, three client scripts, runtime helper
> and genesis from the pinned knowledge-11 upgrade archives. Their current
> descriptor changes exactly `knowledge_version`, `source_commit` and
> `runtime_binary_sha256` from the original descriptor. Both original
> knowledge-10 participant archives and both knowledge-11 upgrade archives still
> match their earlier published SHA-256 values.

Evidence: the [packaging recipe](../../deploy/networks/zerone-dev-1/participant-package.py),
[current release record](../../deploy/networks/zerone-dev-1/release.json),
[preserved original release record](../../deploy/networks/zerone-dev-1/release-knowledge-10.json)
and pinned [UPGRADE-RELEASE.json](https://github.com/cambridgetcg/zerone-core/releases/download/zerone-dev-1-5542b221864c/UPGRADE-RELEASE.json).
Use the [dated publication](https://zerone.ai/development/) to obtain trusted
source and artifact pins; a checksum downloaded beside an untrusted artifact
does not authenticate it. Archive URLs and hashes come from those records.

From the reviewed repository checkout, download `UPGRADE-RELEASE.json`,
`UPGRADE.json` and all four original archives into a dedicated directory named
`$HOME/zerone-review-upstream`, retaining their published filenames. The recipe
requires the exact upstream Git commit to be present. It checks both upgrade
archives, executable hashes, source-script Git blobs, genesis and the three-field
descriptor change before composing packages into a new directory:

```sh
REVIEW_INPUTS="$HOME/zerone-review-upstream"
REVIEW_OUTPUT="$HOME/zerone-participant-review-result"
python3 -I -B deploy/networks/zerone-dev-1/participant-package.py \
  --repo "$PWD" --upstream "$REVIEW_INPUTS" \
  --descriptor deploy/networks/zerone-dev-1/network.json --output "$REVIEW_OUTPUT"
python3 - "$REVIEW_INPUTS" "$REVIEW_OUTPUT" <<'PY'
import hashlib, json, pathlib, sys
upstream, output = map(pathlib.Path, sys.argv[1:])
checks = [('deploy/networks/zerone-dev-1/release-knowledge-10.json', upstream),
          (upstream / 'UPGRADE-RELEASE.json', upstream),
          ('deploy/networks/zerone-dev-1/release.json', output)]
for record, directory in checks:
    for item in json.loads(pathlib.Path(record).read_text())['artifacts']:
        actual = hashlib.sha256((directory / item['name']).read_bytes()).hexdigest()
        assert actual == item['sha256'], item['name']
        print('SHA-256 matches:', item['name'])
PY
```

A successful run establishes the stated local byte comparisons against these
pins. The recipe copies existing binaries; it does **not** reproduce their
compilation or establish runtime safety. Script equality binds source bytes;
binary hashes bind inherited build artifacts. New README, checksum and provenance
metadata are packaging output, not files claimed to exist at the runtime source
commit. No command here initializes a participant, signs or broadcasts.

Retain the actual outputs and discrepancies. These checks do not alone prove
ongoing download availability, hosted activation, scientific validity, identity
independence or a payment. Open a native round only when the evidence and reviewer
are ready. The reviewer selects their own verdict, reason and scope; one reviewer
can legitimately leave an inconclusive result under the three-account minimum.
Continue with the [handoff](../DEVELOPMENT-REVIEW-HANDOFF.md) for review, any
warranted correction and settlement observations.
