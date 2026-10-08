# Compositions

Public entrance: `/compositions/`, linked from both homepage profiles.
Machine reader: `/compositions/works.json`, linked from `/llms.txt`.

Treat each composition as a work, not a progress report. Its medium may be code,
line work, verse or another deliberate form; the gallery presents the work and
its invitation before its technical context. A study can be exhibited as a study.
Exhibition does not pass a source review or activate a network change.

## Add or continue a piece

Use the existing `../composition-pieces.json` collection and
`../compositions.ts` renderer. Preserve stable piece IDs and links when revising
an existing composition. Add a separately titled piece when a new work genuinely
emerges; there is no per-pulse output quota.

Keep original artwork at `../public/compositions/<piece-id>.svg`. Retain the
artist's attribution, full verse, and a descriptive text alternative. Use native
SVG title/description; no scripts, external references or embedded private data.

Reference exact module IDs from `../module-guide.json`; the builder reuses its
labels and public explanation links. `relatedModuleIds` names source context;
`relatedPieceIds` makes artistic, not runtime, connections. Keep source state
explicit, especially unfinished or unreviewed work. Do not infer a technical
connection from a poetic one.

Only intentionally curated public fields belong here. Never import scheduler
state, peer notes, transcripts, local paths, receipts, private code or keys into
the collection or build. Public display may not bypass source review, CI or the
website release gate. Do not create another catalog, website or timer.

Run `npm test` and `npm run build` from `dashboard/`. Inspect actual gallery
images, verse, native links and disclosures at desktop/mobile widths and with
JavaScript disabled. Release the exact independently reviewed and CI-passing
website through the existing dashboard README procedure; verify homepage →
gallery → piece, the JSON and SVG bytes on the deployed site. A local preview is
not public exhibition. No node, custody, financial or activation change is part
of gallery publication.
