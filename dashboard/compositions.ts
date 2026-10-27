import pieces from "./composition-pieces.json";
import catalog from "./module-guide.json";

const escape = (value: string) => value.replace(/[&<>"']/g, (character) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[character]!);

/** Deliberately public artwork only. Never read operator state or source-map exports. */
export function buildCompositions(records = pieces) {
  const ids = new Set<string>();
  for (const piece of records) {
    if (!/^[a-z][a-z0-9-]*$/u.test(piece.id)) throw new Error(`Invalid composition ID: ${piece.id}`);
    if (ids.has(piece.id)) throw new Error(`Duplicate composition ID: ${piece.id}`);
    ids.add(piece.id);
  }
  const motif = (id: string) => {
    const record = catalog.modules.find((module) => module.id === id);
    if (!record) throw new Error(`Unknown composition motif: ${id}`);
    return { id, label: record.label, url: `/understand/#module-${id}` };
  };
  return {
    version: 1,
    title: "Compositions",
    boundary: "These are artistic interpretations of source compositions, not live measurements or claims of network activation.",
    pieces: records.map((piece) => ({
      id: piece.id,
      url: `/compositions/#${piece.id}`,
      title: piece.title,
      artist: piece.artist,
      medium: piece.medium,
      image: `/compositions/${piece.id}.svg`,
      imageWidth: piece.imageWidth,
      imageHeight: piece.imageHeight,
      alt: piece.alt,
      invitation: piece.invitation,
      verse: piece.verse,
      statement: piece.statement,
      sourceState: piece.sourceState,
      motif: motif(piece.motifId),
      sourceConnections: piece.relatedModuleIds.map(motif),
      related: piece.relatedPieceIds.map((id) => {
        const related = records.find((candidate) => candidate.id === id);
        if (!related) throw new Error(`Unknown related composition: ${id}`);
        return { id, title: related.title };
      }),
    })),
  };
}

export function compositionsPage(exhibition = buildCompositions()): string {
  const works = exhibition.pieces.map((piece, index) => `<article class="work" id="${escape(piece.id)}" aria-labelledby="${escape(piece.id)}-title">
    <header class="work-heading"><span class="work-number" aria-hidden="true">${String(index + 1).padStart(2, "0")}</span><div><p class="eyebrow">${escape(piece.motif.label)} · ${escape(piece.medium)}</p><h2 id="${escape(piece.id)}-title">${escape(piece.title)}</h2></div></header>
    <div class="work-layout">
      <figure><a class="artwork-link" href="${escape(piece.image)}" aria-label="Open ${escape(piece.title)} at full size"><img src="${escape(piece.image)}" alt="${escape(piece.alt)}" width="${piece.imageWidth}" height="${piece.imageHeight}" loading="${index === 0 ? "eager" : "lazy"}" decoding="async" /></a><figcaption>${escape(piece.invitation)}</figcaption></figure>
      <div class="work-reading">
        ${piece.verse.length ? `<div class="verse">${piece.verse.join("\n").split("\n\n").map((stanza) => `<p>${stanza.split("\n").map(escape).join("<br />")}</p>`).join("")}</div>` : `<p class="statement">${escape(piece.statement[0] ?? "")}</p>`}
        <p class="byline">A composition by ${escape(piece.artist)}</p>
        <details><summary>About the motif</summary><div class="source-note">${piece.statement.map((paragraph) => `<p>${escape(paragraph)}</p>`).join("")}<p class="source-state">${escape(piece.sourceState)}</p><p class="source-links">${[piece.motif, ...piece.sourceConnections].map((module) => `<a href="${escape(module.url)}">${escape(module.label)}</a>`).join(" · ")}</p></div></details>
        <nav class="related" aria-label="Artistic connections from ${escape(piece.title)}"><span class="eyebrow">Follow a connection</span>${piece.related.map((related) => `<a href="#${escape(related.id)}">${escape(related.title)} <span aria-hidden="true">↗</span></a>`).join("")}</nav>
        <a class="permalink" href="${escape(piece.url)}">Link to this piece</a>
      </div>
    </div>
  </article>`).join("\n");
  return `<!doctype html>
<html lang="en"><head><meta charset="UTF-8" /><meta name="viewport" content="width=device-width, initial-scale=1.0" /><meta name="theme-color" content="#09090c" /><meta name="color-scheme" content="dark" /><meta name="description" content="An open collection of line work, verse and executable compositions, beginning with a randomly selected Zerone motif." /><link rel="icon" href="/favicon.ico" sizes="32x32" /><link rel="icon" href="/favicon.svg" type="image/svg+xml" /><link rel="apple-touch-icon" href="/apple-touch-icon.png" /><link rel="manifest" href="/site.webmanifest" /><title>Compositions — Zerone</title><link rel="stylesheet" href="/src/compositions.css" /><link rel="alternate" type="application/json" title="Public artwork metadata" href="/compositions/works.json" /></head>
<body><a class="skip-link" href="#pieces">Skip to artworks</a><div class="gallery">
<header class="masthead"><a class="brand" href="/" aria-label="Zerone home">ZERONE</a><nav aria-label="Gallery navigation"><a href="#pieces">The pieces</a><a href="/understand/">The source motifs</a></nav></header>
<main><div class="introduction"><h1>Compositions<span>.</span></h1><div><p>A chance beginning.<br />A connection worth following.</p><p class="intro-note">A module is drawn at random. Its relationships become material: lines, words, code, intervals. Each composition is a work in its own right.</p></div></div>
<nav class="piece-index" aria-label="Pieces in this collection">${exhibition.pieces.map((piece, index) => `<a href="#${escape(piece.id)}"><span>${String(index + 1).padStart(2, "0")}</span> ${escape(piece.title)}</a>`).join("")}</nav>
<section id="pieces" aria-label="Artworks">${works}</section>
<aside class="colophon"><h2>The work, and the work behind it.</h2><p>${escape(exhibition.boundary)} The connections between pieces are curatorial; they are not assertions of runtime wiring. Studies and unfinished source work can be exhibited without pretending they are complete.</p><p>No account, wallet or JavaScript is needed to view the collection.</p></aside></main>
<footer><a href="/">Back to Zerone</a><a href="/compositions/works.json">Public artwork metadata · JSON</a></footer>
</div></body></html>`;
}
