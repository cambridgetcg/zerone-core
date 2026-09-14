import { EXPORT_LIMIT, RELATION_TYPES, validateExport, compareJournals, selectSnapshot, orderedContributions, impact, type Journal, type JournalComparison, type Entry, type Snapshot, type Contribution, type Relation, type Concern, type Assessment } from "./research-review-journal";

const $ = <T extends Element = HTMLElement>(id: string): T => document.getElementById(id) as unknown as T;
const make = <K extends keyof HTMLElementTagNameMap>(tag: K, content?: string, className?: string): HTMLElementTagNameMap[K] => {
  const element = document.createElement(tag); if (content !== undefined) element.textContent = content; if (className) element.className = className; return element;
};
const button = (content: string, key: string, action: () => void, className = ""): HTMLButtonElement => {
  const b = make("button", content, className); b.type = "button"; b.dataset.focusKey = key; b.addEventListener("click", action); return b;
};
let journal: Journal | undefined, snapshot: Snapshot | undefined, selected = "", inspected = "", through = 0, loadNumber = 0;
let showImpact = false;
let primaryFile: { bytes: Uint8Array; hash: string; name: string } | undefined, comparisonNumber = 0;
const enabledRelations = new Set<string>(RELATION_TYPES);
const file = $<HTMLInputElement>("journal-file"), example = $<HTMLButtonElement>("load-example"), cutoff = $<HTMLInputElement>("cutoff"), clock = $<HTMLSelectElement>("clock");
const nodeSelect = $<HTMLSelectElement>("node-select"), impactButton = $<HTMLButtonElement>("impact");
const comparisonFile = $<HTMLInputElement>("comparison-file");
const graph = $<SVGSVGElement>("graph");
const notice = (content: string): HTMLParagraphElement => make("p", content, "small");
const get = (id: string): Entry | undefined => snapshot?.entries.find(e => e.record.id === id);
const title = (id: string): string => get(id)?.record.title ?? id;

function choose(id: string): void {
  const entry = get(id); if (!entry) return;
  inspected = id;
  if (entry.record.kind === "contribution") selected = id;
  else if (entry.record.kind === "concern") selected = entry.record.target_id;
  else if (entry.record.kind === "relation") selected = entry.record.from_id;
  else {
    const target = get(entry.record.target_id)?.record;
    if (target?.kind === "concern") selected = target.target_id;
    else if (target?.kind === "relation") selected = target.from_id;
  }
  render();
}
function recordButton(entry: Entry, context: string): HTMLButtonElement {
  const r = entry.record, b = button("", `${context}-${r.id}`, () => choose(r.id), "record-button");
  b.dataset.testid = `${context}-${r.id}`; b.setAttribute("aria-current", String(inspected === r.id));
  b.append(make("span", r.kind === "contribution" ? r.contribution_type : r.kind, "kind"), make("span", r.title));
  b.append(make("span", `#${entry.sequence} · ${r.attributed_to}`, "small")); return b;
}
function reference(id: string, context: string): HTMLButtonElement { return button(`${title(id)} · ${id}`, `ref-${context}-${id}`, () => choose(id)); }
function assessment(entry: Entry, context: string): HTMLElement {
  const r = entry.record as Assessment, container = make("article", undefined, "assessment");
  container.dataset.testid = `${context}-${r.id}`;
  container.append(make("p", `${r.disposition} · ${r.attributed_to}`), make("p", r.summary, "small"), notice(`#${entry.sequence} · added ${entry.recorded_at}`), button("Inspect assessment and evidence", `${context}-inspect-${r.id}`, () => choose(r.id)));
  return container;
}
function allAssessments(target: string, context: string): HTMLElement {
  const container = make("div"); const entries = snapshot!.assessments.filter(e => (e.record as Assessment).target_id === target);
  if (!entries.length) container.append(notice("No assessments of this record in this snapshot. This does not establish agreement or absence of concerns elsewhere."));
  else for (const e of entries) container.append(assessment(e, context)); return container;
}
function renderDetails(): void {
  const container = $("details"); container.replaceChildren(); const e = get(inspected);
  if (!e) { container.append(notice("No record selected in this snapshot.")); return; }
  const r = e.record; container.append(make("h3", r.title)); const meta = make("dl", undefined, "detail-meta");
  const field = (label: string, value: string | HTMLElement): void => { const dd = make("dd"); if (typeof value === "string") dd.textContent = value; else dd.append(value); meta.append(make("dt", label), dd); };
  field("Record", `${r.id} · ${r.kind}`); field("Attributed to", `${r.attributed_to} (declared label)`);
  field("Reported occurrence", r.occurred_on ?? "Unknown; no date inferred"); field("Added to collection", `#${e.sequence} · ${e.recorded_at}`);
  if (r.kind === "contribution") { field("Contribution type", r.contribution_type); field("DOI", r.doi ?? "Not recorded"); }
  else if (r.kind === "relation") { field("Recorded relation", r.relation_type); field("From", reference(r.from_id, "detail-from")); field("To", reference(r.to_id, "detail-to")); }
  else if (r.kind === "concern") { field("Concern about", reference(r.target_id, "detail-target")); field("Category", r.category); field("Provider", r.provider); field("Notice type", r.notice_type ?? "Not recorded"); }
  else { field("Assesses", reference(r.target_id, "assessment-target")); field("Disposition", r.disposition); }
  container.append(meta, make("p", r.summary, "detail-summary"));
  if (r.supersedes) { container.append(make("h4", "Explicit earlier version"), reference(r.supersedes, "supersedes"), notice("This later record retains the earlier version. Assessments of the earlier record do not automatically apply to this one.")); }
  const versions = snapshot!.entries.filter(item => item.record.supersedes === r.id);
  if (versions.length) { container.append(make("h4", "Later versions present in this snapshot")); for (const v of versions) container.append(reference(v.record.id, "versions")); }
  container.append(make("h4", "Evidence as recorded"));
  if (!r.evidence.length) container.append(notice("No evidence reference recorded. The journal hash is not evidence for the scientific content."));
  else {
    const list = make("ul", undefined, "evidence-list");
    for (const evidence of r.evidence) {
      const item = make("li", evidence.label);
      if (evidence.url) { const a = make("a", evidence.url); a.href = evidence.url; a.target = "_blank"; a.rel = "noopener noreferrer"; item.append(make("br"), a); }
      if (evidence.sha256) item.append(make("code", `SHA256 ${evidence.sha256}`)); list.append(item);
    }
    container.append(list, notice("References are not fetched or independently checked by this page."));
  }
  if (r.kind === "concern" || r.kind === "relation") container.append(make("h4", "Every assessment of this record in the snapshot"), allAssessments(r.id, "detail-assessment"));
  if (r.kind === "contribution") {
    container.append(make("h4", "Concerns about this contribution"));
    const concerns = snapshot!.concerns.filter(item => (item.record as Concern).target_id === r.id);
    if (!concerns.length) container.append(notice("No concern about this contribution is recorded in this snapshot. This is not a clean bill of health."));
    else for (const concern of concerns) container.append(recordButton(concern, "detail-concern"), allAssessments(concern.record.id, "contribution-assessment"));
  }
  const hashes = make("details"); hashes.append(make("summary", "Internal hash linkage"), make("p", `Entry SHA256 ${e.sha256}`, "mono"), make("p", `Previous SHA256 ${e.previous_sha256}`, "mono"), notice("These hashes establish consistency within this export only. No signature or independent time anchor is verified here.")); container.append(hashes);
}
function svg<K extends keyof SVGElementTagNameMap>(tag: K, attrs: Record<string, string>): SVGElementTagNameMap[K] {
  const e = document.createElementNS("http://www.w3.org/2000/svg", tag); for (const [key, value] of Object.entries(attrs)) e.setAttribute(key, value); return e;
}
function renderGraph(affected: Set<string>, paths: ReturnType<typeof impact>["steps"]): void {
  const edges = snapshot!.relations.filter(e => { const r = e.record as Relation; return enabledRelations.has(r.relation_type) && (r.from_id === selected || r.to_id === selected); });
  const neighbours = [...new Set(edges.flatMap(e => { const r = e.record as Relation; return [r.from_id, r.to_id]; }).filter(id => id !== selected))];
  const height = Math.max(250, Math.ceil(neighbours.length / 2) * 115 + 55); graph.setAttribute("viewBox", `0 0 840 ${height}`); graph.setAttribute("height", String(height)); graph.replaceChildren();
  const defs = svg("defs", {}), marker = svg("marker", { id: "arrow", markerWidth: "8", markerHeight: "8", refX: "7", refY: "3", orient: "auto", markerUnits: "strokeWidth" });
  marker.append(svg("path", { d: "M0,0 L0,6 L7,3 z", fill: "#b1abb9" })); defs.append(marker); graph.append(defs);
  const positions = new Map<string, { x: number; y: number }>(); if (selected) positions.set(selected, { x: 420, y: 95 });
  neighbours.forEach((id, i) => positions.set(id, { x: i % 2 === 0 ? 120 : 720, y: 95 + Math.floor(i / 2) * 115 }));
  const activeEdges = new Set(paths.map(p => p.relation.record.id));
  for (const e of edges) {
    const r = e.record as Relation, a = positions.get(r.from_id)!, b = positions.get(r.to_id)!;
    const dx = b.x - a.x, dy = b.y - a.y, factor = Math.min(92 / Math.max(1, Math.abs(dx)), 34 / Math.max(1, Math.abs(dy)));
    graph.append(svg("line", { x1: String(a.x + dx * factor), y1: String(a.y + dy * factor), x2: String(b.x - dx * factor), y2: String(b.y - dy * factor), class: `graph-edge${activeEdges.has(r.id) ? " highlight" : ""}`, "marker-end": "url(#arrow)" }));
  }
  for (const [id, p] of positions) {
    const entry = get(id)!; const r = entry.record as Contribution;
    const g = svg("g", { class: `graph-node${id === selected ? " selected" : ""}${affected.has(id) ? " affected" : ""}`, transform: `translate(${p.x},${p.y})`, role: "button", tabindex: "0", "aria-label": `${r.title}; ${r.contribution_type}; inspect contribution`, "aria-pressed": String(id === selected), "data-testid": `graph-node-${id}`, "data-focus-key": `graph-node-${id}` });
    g.append(svg("rect", { x: "-92", y: "-34", width: "184", height: "68", rx: "5" }));
    const kind = svg("text", { x: "-80", y: "-15", class: "node-kind" }); kind.textContent = r.contribution_type; g.append(kind);
    const words = Array.from(r.title), first = words.slice(0, 25).join(""), second = words.slice(25, 49).join("") + (words.length > 49 ? "…" : "");
    [first, second].filter(Boolean).forEach((line, i) => { const t = svg("text", { x: "-80", y: String(4 + i * 16) }); t.textContent = line; g.append(t); });
    const t = svg("title", {}); t.textContent = r.title; g.append(t);
    g.addEventListener("click", () => choose(id)); g.addEventListener("keydown", event => { if (event.key === "Enter" || event.key === " ") { event.preventDefault(); choose(id); } }); graph.append(g);
  }
  const list = $("edge-list"); list.replaceChildren();
  if (!edges.length) list.append(notice("No direct relations of the selected types in this snapshot."));
  for (const entry of edges) {
    const r = entry.record as Relation, item = make("div", undefined, "edge-item");
    const b = button(`${title(r.from_id)} → ${r.relation_type} → ${title(r.to_id)} · asserted by ${r.attributed_to}`, `edge-${r.id}`, () => { inspected = r.id; render(); }, `edge-button${inspected === r.id ? " selected" : ""}`);
    b.dataset.testid = `edge-${r.id}`; item.append(b); list.append(item);
  }
}
function renderTimeline(affected: Set<string>): void {
  const container = $("timeline"); container.replaceChildren(); const order = clock.value === "recorded" ? "recorded" : "occurred";
  const ordered = orderedContributions(snapshot!, order);
  $("clock-explanation").textContent = order === "recorded" ? "Date added to this collection, after applying the entry cutoff." : "Reconstruction by reported occurrence, using only records already in this snapshot. Unknown dates have a separate bucket.";
  const group = (entries: Entry[]): void => {
    const list = make("ol", undefined, "timeline-group");
    for (const e of entries) {
      const item = make("li", undefined, "timeline-item"), b = recordButton(e, "timeline");
      if (affected.has(e.record.id)) b.classList.add("affected"); b.append(make("span", order === "recorded" ? e.recorded_at : e.record.occurred_on ?? "Unknown occurrence date", "small")); item.append(b); list.append(item);
    }
    container.append(list);
  };
  group(ordered.dated);
  if (ordered.unknown.length) { container.append(make("h3", "Unknown occurrence date", "unknown-heading")); group(ordered.unknown); }
  if (!snapshot!.contributions.length) container.append(notice("No contributions in this snapshot."));
}
function render(): void {
  if (!journal) return;
  const active = document.activeElement as HTMLElement | SVGElement | null;
  const focusKey = active?.getAttribute("data-focus-key"), focusId = active?.id;
  const timelineScroll = document.querySelector(".timeline-panel")?.scrollTop ?? 0;
  snapshot = selectSnapshot(journal, through);
  if (!snapshot.contributions.some(e => e.record.id === selected)) selected = snapshot.contributions.at(-1)?.record.id ?? "";
  if (!snapshot.entries.some(e => e.record.id === inspected)) inspected = selected;
  cutoff.max = String(journal.entries.length); cutoff.value = String(through); $("cutoff-value").textContent = `${through} / ${journal.entries.length}`;
  $("snapshot-mode").textContent = snapshot.historical ? "Historical snapshot · later entries excluded" : "Full imported collection · no wider completeness claim";
  $("snapshot-time").textContent = `Through ${snapshot.entries.at(-1)?.recorded_at ?? journal.header.created_at}`;
  $("snapshot-counts").textContent = `${snapshot.contributions.length} contributions · ${snapshot.relations.length} relations · ${snapshot.concerns.length} concerns · ${snapshot.assessments.length} assessments in this snapshot`;
  nodeSelect.replaceChildren(); for (const e of snapshot.contributions) { const option = make("option", `${e.record.title} · ${e.record.id}`); option.value = e.record.id; nodeSelect.append(option); } nodeSelect.value = selected;
  nodeSelect.disabled = !selected; impactButton.disabled = !selected;
  const result = showImpact && selected ? impact(snapshot, selected) : { nodes: new Set<string>(), steps: [] };
  impactButton.setAttribute("aria-pressed", String(showImpact)); $("impact-description").hidden = !showImpact;
  $("impact-description").textContent = `Possible reassessment from “${title(selected)}”: ${Math.max(0, result.nodes.size - 1)} other contributions. Follows every supports / reverse-requires / reverse-uses-input relation in this snapshot, regardless of display filters. Shared input is not logical support. No automatic falsity or transfer of assessments.`;
  const paths = $("impact-paths"); paths.replaceChildren(); paths.hidden = !showImpact;
  if (showImpact) {
    if (!result.steps.length) paths.append(notice("No downstream path of these types is recorded in this snapshot."));
    else { const list = make("ul", undefined, "small"); for (const step of result.steps) { const item = make("li"); item.append(button(`${step.from} → ${step.to} · ${step.mode} · ${step.relation.record.attributed_to}`, `impact-${step.relation.record.id}`, () => { inspected = step.relation.record.id; render(); })); list.append(item); } paths.append(list); }
  }
  renderTimeline(result.nodes); renderGraph(result.nodes, result.steps); renderDetails();
  const concerns = $("concerns"); concerns.replaceChildren();
  if (!snapshot.concerns.length) concerns.append(notice("No concerns recorded in this snapshot. Absence here is not evidence of reliability."));
  for (const e of snapshot.concerns) { const item = make("div", undefined, "ledger-record"); item.append(recordButton(e, "concern"), notice(`About ${title((e.record as Concern).target_id)}`), allAssessments(e.record.id, "concern-assessment")); concerns.append(item); }
  const assessments = $("assessments"); assessments.replaceChildren();
  if (!snapshot.assessments.length) assessments.append(notice("No assessments recorded in this snapshot."));
  for (const e of snapshot.assessments) { const item = make("div", undefined, "ledger-record"); item.append(notice(`Assesses ${title((e.record as Assessment).target_id)}`), assessment(e, "all-assessment")); assessments.append(item); }
  const panel = document.querySelector(".timeline-panel"); if (panel) panel.scrollTop = timelineScroll;
  if (active && !active.isConnected) {
    const replacement = focusKey ? Array.from(document.querySelectorAll<HTMLElement | SVGElement>("[data-focus-key]")).find(e => e.getAttribute("data-focus-key") === focusKey) : focusId ? document.getElementById(focusId) : undefined;
    (replacement ?? (selected ? nodeSelect : cutoff)).focus({ preventScroll: true });
  }
  revealSelection();
}
function revealSelection(): void {
  const node = Array.from(graph.querySelectorAll<SVGGElement>(".graph-node")).find(e => e.dataset.testid === `graph-node-${selected}`); if (!node) return;
  const box = $("graph-scroll"), viewport = box.getBoundingClientRect(), rect = node.getBoundingClientRect();
  if (rect.left < viewport.left + 12) box.scrollLeft -= viewport.left + 12 - rect.left;
  else if (rect.right > viewport.right - 12) box.scrollLeft += rect.right - viewport.right + 12;
  if (rect.top < viewport.top + 12) box.scrollTop -= viewport.top + 12 - rect.top;
  else if (rect.bottom > viewport.bottom - 12) box.scrollTop += rect.bottom - viewport.bottom + 12;
}
async function fileHash(bytes: Uint8Array): Promise<string> {
  const hash = await crypto.subtle.digest("SHA-256", new Uint8Array(bytes));
  return [...new Uint8Array(hash)].map(value => value.toString(16).padStart(2, "0")).join("");
}
function clearComparison(resetInput = true): void {
  comparisonNumber++;
  const result = $("comparison-result"); result.hidden = true; result.replaceChildren(); delete result.dataset.relationship;
  $("comparison-error").hidden = true; $("comparison-error").textContent = "";
  comparisonFile.disabled = !journal || !primaryFile;
  if (resetInput) comparisonFile.value = "";
  $("comparison-status").textContent = comparisonFile.disabled ? "Open a primary collection first." : "Choose a second local export to compare both complete histories.";
}
function renderComparison(result: JournalComparison, leftFile: NonNullable<typeof primaryFile>, rightFile: NonNullable<typeof primaryFile>): void {
  const container = $("comparison-result"); container.replaceChildren(); container.dataset.relationship = result.relationship;
  const labels: Record<JournalComparison["relationship"], string> = {
    "same-history": "Same journal history", "left-prefix": "A is a strict prefix of B", "right-prefix": "B is a strict prefix of A",
    diverged: "Different continuations of the same root", "different-root": "Different roots: collection headers differ",
  };
  container.append(make("h3", labels[result.relationship]));
  const equalBytes = leftFile.bytes.length === rightFile.bytes.length && leftFile.bytes.every((value, i) => value === rightFile.bytes[i]);
  const equality = make("p", equalBytes ? "File bytes: identical." : "File bytes: different. Equal history, if reported, does not mean the files match the same byte commitment.");
  equality.id = "comparison-byte-equality"; container.append(equality);
  const files = make("div", undefined, "comparison-files");
  for (const [label, info, summary] of [["A · primary file", leftFile, result.left], ["B · compared file", rightFile, result.right]] as const) {
    const card = make("article"), meta = make("dl"); card.append(make("h4", label));
    for (const [key, value] of [["File", info.name], ["File SHA256", info.hash], ["Collection", summary.collection_id], ["Complete entry count", String(summary.entry_count)], ["History head SHA256", summary.head_sha256]] as const) {
      meta.append(make("dt", key), make("dd", value, key.includes("SHA256") ? "mono" : ""));
    }
    card.append(meta); files.append(card);
  }
  container.append(files);
  const common = make("p"); common.id = "comparison-common-prefix";
  if (result.common_prefix) {
    const n = result.common_prefix.entry_count;
    common.textContent = `Shared prefix: ${n} entries. Entries after that prefix: A ${result.left.entry_count - n}; B ${result.right.entry_count - n}.`;
    container.append(common, make("p", `Shared-prefix head SHA256 ${result.common_prefix.head_sha256}${n === 0 ? " (header hash; no shared entries)" : ""}`, "mono"));
  } else { common.textContent = "No common prefix is reported across different collection headers, even if their collection IDs match."; container.append(common); }
  container.append(notice("These results compare the complete files, regardless of the timeline cutoff. Neither continuation is selected as canonical, authentic or scientifically correct."));
  container.hidden = false;
}
async function compareFile(chosen: File): Promise<void> {
  clearComparison(false);
  const number = comparisonNumber, primaryNumber = loadNumber, left = journal, leftFile = primaryFile;
  if (!left || !leftFile) return;
  $("comparison-status").textContent = "Validating the complete second export and comparing both histories…";
  try {
    if (chosen.size > EXPORT_LIMIT || chosen.size === 0) throw new Error("Export must be nonempty and at most 8 MiB");
    const bytes = new Uint8Array(await chosen.arrayBuffer());
    if (number !== comparisonNumber || primaryNumber !== loadNumber || journal !== left) return;
    const [right, hash] = await Promise.all([validateExport(bytes), fileHash(bytes)]);
    const result = await compareJournals(left, right);
    if (number !== comparisonNumber || primaryNumber !== loadNumber || journal !== left) return;
    renderComparison(result, leftFile, { bytes, hash, name: chosen.name });
    $("comparison-status").textContent = "Both complete exports validated. The primary file remains in the timeline below.";
  } catch (error) {
    if (number !== comparisonNumber || primaryNumber !== loadNumber || journal !== left) return;
    $("comparison-status").textContent = "No comparison is displayed. The validated primary collection remains open.";
    $("comparison-error").hidden = false;
    $("comparison-error").textContent = `Comparison refused: ${error instanceof Error ? error.message : "invalid export"}.`;
  }
}
async function load(read: () => Promise<Uint8Array>, name: string, fictional: boolean): Promise<void> {
  const number = ++loadNumber; journal = undefined; snapshot = undefined; primaryFile = undefined; clearComparison();
  $("primary-file-hash").hidden = true; $("primary-file-hash").textContent = "";
  $("workspace").hidden = true; $("load-error").hidden = true;
  $("load-status").textContent = "Checking the complete export: schema, bounds, references and every hash…";
  try {
    const bytes = await read(); if (number !== loadNumber) return;
    const [checked, hash] = await Promise.all([validateExport(bytes), fileHash(bytes)]); if (number !== loadNumber) return;
    primaryFile = { bytes, hash, name };
    journal = checked; through = journal.entries.length; selected = journal.entries.find(e => e.record.kind === "contribution")?.record.id ?? ""; inspected = selected; showImpact = false;
    $("collection-label").textContent = fictional ? "FICTIONAL EXAMPLE · illustrative records only" : `LOCAL IMPORT · ${name} · contents not independently authenticated`;
    $("collection-identity").textContent = `Collection ${journal.header.collection_id}`;
    $("load-status").textContent = `Internal consistency checked for all ${journal.entries.length} entries. No identity, external evidence or independent timestamp verified.`;
    $("primary-file-hash").textContent = `Primary file SHA256 ${hash}`; $("primary-file-hash").hidden = false; clearComparison();
    $("workspace").hidden = false; render();
  } catch (error) {
    if (number !== loadNumber) return;
    $("load-status").textContent = "No collection is displayed."; $("load-error").hidden = false;
    $("load-error").textContent = `Import refused: ${error instanceof Error ? error.message : "invalid export"}. Fix the export or choose another file; no partial snapshot is shown.`;
    for (const id of ["timeline", "details", "concerns", "assessments", "edge-list", "impact-paths"]) $(id).replaceChildren(); graph.replaceChildren();
  }
}
file.disabled = false; example.disabled = false; $("load-status").textContent = "Choose a local export or explore the fictional example. Nothing has been loaded.";
file.addEventListener("change", () => {
  const chosen = file.files?.[0];
  void load(async () => { if (!chosen) throw new Error("No export selected"); if (chosen.size > EXPORT_LIMIT || chosen.size === 0) throw new Error("Export must be nonempty and at most 8 MiB"); return new Uint8Array(await chosen.arrayBuffer()); }, chosen?.name ?? "", false);
});
comparisonFile.addEventListener("change", () => { const chosen = comparisonFile.files?.[0]; if (chosen) void compareFile(chosen); else clearComparison(false); });
example.addEventListener("click", () => { void load(async () => {
  const response = await fetch("/research/review/fictional-journal.v1.json", { method: "GET", credentials: "omit", redirect: "error" });
  if (!response.ok || !response.body) throw new Error("Fictional example is unavailable");
  const reader = response.body.getReader(), chunks: Uint8Array[] = []; let size = 0;
  try { while (true) { const part = await reader.read(); if (part.done) break; size += part.value.length; if (size > EXPORT_LIMIT) throw new Error("Example exceeds 8 MiB"); chunks.push(part.value); } } finally { await reader.cancel(); }
  const bytes = new Uint8Array(size); let offset = 0; for (const chunk of chunks) { bytes.set(chunk, offset); offset += chunk.length; } return bytes;
}, "fictional example", true); });
cutoff.addEventListener("input", () => { through = Number(cutoff.value); render(); });
$("latest").addEventListener("click", () => { if (!journal) return; through = journal.entries.length; render(); });
clock.addEventListener("change", render); nodeSelect.addEventListener("change", () => choose(nodeSelect.value));
impactButton.addEventListener("click", () => { showImpact = !showImpact; render(); });
for (const type of RELATION_TYPES) {
  const label = make("label"), input = make("input"); input.type = "checkbox"; input.checked = true; input.value = type; input.id = `filter-${type}`;
  input.addEventListener("change", () => { if (input.checked) enabledRelations.add(type); else enabledRelations.delete(type); render(); }); label.append(input, document.createTextNode(type)); $("relation-filters").append(label);
}
window.addEventListener("resize", revealSelection);
