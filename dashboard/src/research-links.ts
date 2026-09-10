/** Only the explicit links kept in the homepage may redirect an earlier fragment. */
export function researchDestination(hash: string, root: Pick<Document, "getElementById">): string | null {
  if (!hash.startsWith("#") || hash.length > 200) return null;
  const link = root.getElementById(hash.slice(1));
  if (!link?.hasAttribute("data-research-destination")) return null;
  const destination = link.getAttribute("href");
  return destination === `/research/${hash}` ? destination : null;
}

export function routeLegacyResearchAnchor(): void {
  const destination = researchDestination(window.location.hash, document);
  if (destination) window.location.replace(destination);
}
