// Optional clipboard enhancement. The complete guide and commands are static HTML.
for (const button of document.querySelectorAll<HTMLButtonElement>("[data-copy-target]")) {
  const id = button.dataset.copyTarget;
  const code = id ? document.getElementById(id) : null;
  const status = id ? document.querySelector<HTMLElement>(`[data-copy-status="${CSS.escape(id)}"]`) : null;
  if (!code || !status || !navigator.clipboard?.writeText || !window.isSecureContext) continue;

  button.addEventListener("click", () => {
    button.disabled = true;
    void navigator.clipboard.writeText(code.textContent ?? "").then(() => {
      status.textContent = "Copied. Paste into your terminal when ready.";
    }).catch(() => {
      status.textContent = "Copy was unavailable. Select the command and copy it manually.";
    }).finally(() => {
      button.disabled = false;
    });
  });
  button.hidden = false;
}
