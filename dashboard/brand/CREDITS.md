# Credits

Who made each asset, with what, and under which licence. Where an agent made something, the line says so, and
anything published under Yu's accounts carries that through. If an asset's maker isn't known, write "not recorded"
rather than guessing.

## Assets

| Asset | Made by | Date | Tools | Notes |
|---|---|---|---|---|
| The mark: the ring, cross and dot of `public/favicon.svg` | not recorded (first committed in 4d5ffb49, "launch live Zerone mainnet frontend", git author "YOU") | 2026-07-11 | not recorded | the served `public/favicon.svg` is still this file |
| The signature comment in `public/favicon.svg`, "Building castles in the sky — Yu and AI." | not recorded (#33, 02999d82, committed by the cambridgetcg account with co-author "zerone") | 2026-08-01 | | kept on purpose: the reason `favicon.svg` is not in `serve.files` |
| `zerone-mark.svg` | an agent (Claude Code session dd025d89), from `public/favicon.svg` at 008f8f92 | 2026-10-27 | | four edits: the kit's icon tile, `fill="none"` on the cross, shape ids, a provenance comment; the signature comment stays |
| `zerone-mark-small.svg` | an agent (Claude Code session dd025d89) | 2026-10-27 | | the same mark with heavier strokes on a 48-unit grid, for 16 to 48 px tabs |
| `brand.json` | an agent (Claude Code session dd025d89) | 2026-10-27 | | colours are the site's tokens; the share-card line is the hero line and the existing og:description |
| `kit/` (generated) | an agent (Claude Code session dd025d89), on the Studio | 2026-10-27 | artcraft-brand brandkit.py 1.8, vectorcraft-cli 0.17.0 | 278 checks, 0 failed, 4 warnings; the rebuild check reproduced all 73 files byte for byte (PR #156) |

Sources for the agent lines: commit 2e3f1011 (Co-Authored-By Claude), PR #156 ("written by an agent (Claude Code)"),
ledger `artcraft-estate-wiring` and release card R-008, which name session dd025d89.

## Fonts

| Family | File | Licence | Source | SHA-256 |
|---|---|---|---|---|
| Inter SemiBold (wordmark, lockups, share-card title; outlined) | bundled with vectorcraft-cli 0.17.0 | SIL OFL 1.1 | [rsms/inter](https://github.com/rsms/inter) | not recorded |
| Inter Light (share-card subtitle and footer) | `Inter-Light.otf`, installed on the Studio (that the build used this file is an inference: `kit.json` records family and style only) | SIL OFL 1.1, from the file's name table | [rsms/inter](https://github.com/rsms/inter), Version 4.001;git-9221beed3; the download is not recorded | `0817fd071377740a8e1cba673baf2c726ef5e5c4e6e0962c6e5bdc096329586d` |
| JetBrains Mono Regular (colour-sheet labels; outlined) | bundled with vectorcraft-cli 0.17.0 | SIL OFL 1.1 | [JetBrains/JetBrainsMono](https://github.com/JetBrains/JetBrainsMono) | not recorded |

The site's hero is set in Georgia, a system font; the kit doesn't use it.

## Images and other material

| Item | Maker | Licence | Source |
|---|---|---|---|
| none: the kit and share card hold no photographs or outside artwork | | | |
