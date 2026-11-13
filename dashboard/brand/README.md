# Zerone: brand

This folder holds Zerone's brand kit, made with storytold's Craft apps through the artcraft-brand generator.
`brandkit.json` records this project's details, `CREDITS.md` who made what, and `check.mjs` checks that the site
serves the kit as built. The dashboard README's [Brand kit](../README.md#brand-kit) section is the fuller guide;
this page doesn't repeat it.

## The look

The site's existing look, made into a kit on 2026-10-27: the favicon's ring, cross and dot in violet `#b99bff` and
acid `#dcff78` on the `#0b0b10` tile, the dark ground `#09090c`, and the wordmark ZERONE in Inter SemiBold. An agent
(Claude Code session dd025d89) proposed these choices under Yu's word of 2026-10-27 to wire the kits into the sites,
whose scope was a kit from each site's current look with no new identity. PR #156 records them as
an agent's proposals, not Yu's decisions. Don't change the identity without Yu's word.

## What is here

| Path | What it is |
|---|---|
| `brandkit.json` | this project's details: the look and who chose it, the kit, the files the site serves, its guards, what a deploy does |
| `brand.json` | the generator's input |
| `zerone-mark.svg`, `zerone-mark-small.svg` | the mark (the favicon's ring, cross and dot) and its heavier cut for 16 to 48 px tabs |
| `kit/` | the built kit. Never edit it by hand: change `brand.json` or a mark and rebuild |
| `CREDITS.md` | who made what, with fonts and their licences |
| `check.mjs` | the check, copied from the brandkit templates; change it there, not here |

Nothing under `brand/` is served.

## Rebuild

From this folder, with the generator's own folder in place of `<artcraft-brand>`:

```sh
python3 -B <artcraft-brand>/scripts/brandkit.py brand.json --out kit --rebuild-check
```

Then copy the files `brandkit.json` lists under `serve.files` from `kit/web/` into `../public/`, byte for byte, and
run `node brand/check.mjs` and `npm test` from `dashboard/`.

## How the site uses it

Vite serves `../public/` at the site root. `public/favicon.svg` is the site's own file, not the kit's: it keeps its
signature comment from #33, so `serve.files` leaves it out. The eight page-head sources carry the kit's icon links, one
`color-scheme` `dark` and the site's single theme-color `#09090c`; only `index.html` has the share card and JSON-LD.
`tests/brand-kit.sync.test.ts` holds all of that in CI.

## Deploying

A merge deploys nothing. zerone.ai is the Cloudflare Pages project `zerone-ai`, deployed by hand from a clean detached
worktree of the merged, CI-verified commit ([Deploy](../README.md#deploy)), and a production deploy needs Yu's word.
Check new files on the deployment's `*.pages.dev` URL first, never on zerone.ai before they are deployed.
