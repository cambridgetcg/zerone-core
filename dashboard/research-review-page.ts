import type { DevelopmentProfile } from "./development-profile";

/** Reuse the build's verified publication; imports cannot choose a read endpoint. */
export function researchReviewPage(html: string, profile: DevelopmentProfile, predecessorDescriptor?: string): string {
  if (predecessorDescriptor !== undefined && (!profile.upgradeApplied || !/^[a-f0-9]{64}$/u.test(predecessorDescriptor))) throw new Error("Invalid predecessor descriptor publication");
  const p = profile.publication;
  const pins = p ? `<div id="research-development-reader" hidden data-descriptor-sha256="${p.descriptor_sha256}" data-genesis-sha256="${p.genesis_sha256}" data-rpc-genesis-sha256="${p.rpc_genesis_sha256}" data-runtime-source="${p.runtime_source_commit}" data-binary-sha256="${p.binary_sha256}"${predecessorDescriptor ? ` data-predecessor-descriptor-sha256="${predecessorDescriptor}"` : ""}></div>` : "";
  return html.replace("<!-- RESEARCH_DEVELOPMENT_PINS -->", pins)
    .replaceAll("https://github.com/cambridgetcg/zerone-core/tree/main/tools/research-review", `https://github.com/cambridgetcg/zerone-core/tree/${profile.source.commit}/tools/research-review`);
}
