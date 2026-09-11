import { DEVELOPMENT_GATEWAY } from "../development-profile";
import { queryClaimId, getBytes, height, renderHistory, validateHistory, verifyDescriptor } from "./development-reader";

const config = document.querySelector<HTMLElement>("#development-reader");
if (config) {
  const form = document.querySelector<HTMLFormElement>("#claim-form")!;
  const input = document.querySelector<HTMLInputElement>("#claim-id")!;
  const button = document.querySelector<HTMLButtonElement>("#claim-read")!;
  const status = document.querySelector<HTMLElement>("#history-status")!;
  const result = document.querySelector<HTMLElement>("#history-result")!;
  const share = document.querySelector<HTMLAnchorElement>("#share-link")!;
  const pins = { descriptor: config.dataset.descriptorSha256!, genesis: config.dataset.genesisSha256!, rpcGenesis: config.dataset.rpcGenesisSha256!, source: config.dataset.runtimeSource!, binary: config.dataset.binarySha256! };
  button.disabled = false;
  let pending: AbortController | undefined;
  async function read(): Promise<void> {
    pending?.abort(); const controller = new AbortController(); pending = controller;
    const timer = window.setTimeout(() => controller.abort(), 20000);
    button.disabled = true; result.replaceChildren(); share.hidden = true; status.dataset.error = "false";
    status.textContent = "Verifying the published descriptor, then reading this claim…";
    try {
      const id = queryClaimId(input.value);
      await verifyDescriptor(await getBytes(`${DEVELOPMENT_GATEWAY}/network.json`, 65536, controller.signal), pins);
      const bytes = await getBytes(`${DEVELOPMENT_GATEWAY}/claims/${encodeURIComponent(id)}`, 8 * 1024 * 1024, controller.signal);
      const response = validateHistory(JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes)), id);
      if (controller.signal.aborted) return;
      result.append(renderHistory(response));
      status.textContent = `Endpoint observation: zerone-dev-1 at block ${height(response.block_height)}, read ${new Date().toISOString()}. This snapshot does not refresh automatically. Read again for a new observation.`;
      const url = new URL("/development/", location.origin); url.searchParams.set("claim", id); share.href = url.href; share.textContent = "Link to this claim"; share.hidden = false;
      history.replaceState(null, "", url); button.textContent = "Refresh history";
    } catch (error) {
      if (pending !== controller) return;
      status.dataset.error = "true"; status.textContent = `History unavailable: ${controller.signal.aborted ? "request timed out or was cancelled" : error instanceof Error ? error.message : "unexpected read failure"}. No previous snapshot is being presented as current.`;
    } finally { window.clearTimeout(timer); if (pending === controller) button.disabled = false; }
  }
  form.addEventListener("submit", (event) => { event.preventDefault(); void read(); });
  const params = new URLSearchParams(location.search); if (params.has("claim")) { input.value = params.get("claim") ?? ""; void read(); }
}
