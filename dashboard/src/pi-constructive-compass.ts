export const PI_CONSTRUCTIVE_COMPASS_PATHS = [
  {
    id: "follow-question",
    title: "Follow the question",
    invitation: "Start with proof, uncertainty, and claims that can be checked.",
    capabilityIds: [
      "math-proofcraft@1",
      "math-probability-information-complexity@1",
      "assurance-formal-verification@1",
    ],
  },
  {
    id: "guard-edges",
    title: "Guard the edges",
    invitation: "Look for failure modes while keeping repair and disclosure humane.",
    capabilityIds: [
      "security-threat-models-games@1",
      "assurance-differential-fuzzing@1",
      "assurance-coordinated-disclosure@1",
    ],
  },
  {
    id: "make-exact",
    title: "Make it exact",
    invitation: "Trace bytes, rejection rules, and behavior across implementations.",
    capabilityIds: [
      "systems-exact-bytes-state-machines@1",
      "assurance-vectors-negative-tests@1",
      "assurance-conformance-interoperability@1",
    ],
  },
] as const;

export type PiConstructiveCompassPathId =
  (typeof PI_CONSTRUCTIVE_COMPASS_PATHS)[number]["id"];

export interface ConstructiveCompassCapability {
  id: string;
  title: string;
}

export interface ResolvedPiConstructiveCompassPath {
  id: PiConstructiveCompassPathId;
  title: string;
  invitation: string;
  capabilities: readonly ConstructiveCompassCapability[];
}

export interface PiConstructiveCompassOptions {
  resolveCapability(id: string): ConstructiveCompassCapability | null;
  openCapability(id: string): boolean | Promise<boolean>;
}

export interface PiConstructiveCompassController {
  reset(): void;
  setAuthenticated(authenticated: boolean): void;
}

export class PiConstructiveCompassError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "PiConstructiveCompassError";
  }
}

function byId<T extends HTMLElement>(id: string): T {
  const node = document.getElementById(id);
  if (!node) throw new PiConstructiveCompassError(`Missing #${id}`);
  return node as T;
}

function element<K extends keyof HTMLElementTagNameMap>(
  tag: K,
  className?: string,
  text?: string,
): HTMLElementTagNameMap[K] {
  const node = document.createElement(tag);
  if (className) node.className = className;
  if (text !== undefined) node.textContent = text;
  return node;
}

export function resolvePiConstructiveCompassPaths(
  resolveCapability: (
    id: string,
  ) => ConstructiveCompassCapability | null,
): readonly ResolvedPiConstructiveCompassPath[] {
  const pathIds = new Set<string>();
  const capabilityIds = new Set<string>();

  const resolved = PI_CONSTRUCTIVE_COMPASS_PATHS.map((path) => {
    if (pathIds.has(path.id)) {
      throw new PiConstructiveCompassError(
        `Duplicate constructive compass path: ${path.id}`,
      );
    }
    pathIds.add(path.id);
    if (path.capabilityIds.length !== 3) {
      throw new PiConstructiveCompassError(
        `Constructive compass path ${path.id} must contain three capabilities`,
      );
    }

    const capabilities = path.capabilityIds.map((capabilityId) => {
      if (capabilityIds.has(capabilityId)) {
        throw new PiConstructiveCompassError(
          `Duplicate constructive compass capability: ${capabilityId}`,
        );
      }
      capabilityIds.add(capabilityId);
      const capability = resolveCapability(capabilityId);
      if (
        capability === null ||
        capability.id !== capabilityId ||
        capability.title.trim() === "" ||
        capability.title !== capability.title.trim()
      ) {
        throw new PiConstructiveCompassError(
          `Canonical constructive capability unavailable: ${capabilityId}`,
        );
      }
      return Object.freeze({
        id: capability.id,
        title: capability.title,
      });
    });

    return Object.freeze({
      id: path.id,
      title: path.title,
      invitation: path.invitation,
      capabilities: Object.freeze(capabilities),
    });
  });

  return Object.freeze(resolved);
}

export function initialisePiConstructiveCompass(
  options: PiConstructiveCompassOptions,
): PiConstructiveCompassController {
  const paths = resolvePiConstructiveCompassPaths(options.resolveCapability);
  const root = byId<HTMLElement>("pi-constructive-compass");
  const form = byId<HTMLFormElement>("pi-compass-form");
  const consent = byId<HTMLInputElement>("pi-compass-consent");
  const result = byId<HTMLElement>("pi-compass-result");
  const resultTitle = byId<HTMLElement>("pi-compass-result-title");
  const resultCopy = byId<HTMLParagraphElement>("pi-compass-result-copy");
  const trail = byId<HTMLOListElement>("pi-compass-trail");
  const status = byId<HTMLParagraphElement>("pi-compass-status");
  const resetButton = byId<HTMLButtonElement>("pi-compass-reset");
  const firstChoice = form.querySelector<HTMLInputElement>(
    'input[name="pi-compass-path"]',
  );
  if (!firstChoice) {
    throw new PiConstructiveCompassError(
      "The constructive compass has no path choices",
    );
  }

  let openPending = false;

  const setTrailButtonsDisabled = (disabled: boolean): void => {
    trail
      .querySelectorAll<HTMLButtonElement>("button")
      .forEach((button) => {
        button.disabled = disabled;
      });
  };

  const reset = (): void => {
    openPending = false;
    form.reset();
    trail.replaceChildren();
    resultTitle.textContent = "Your temporary trail";
    resultCopy.textContent = "";
    status.textContent = "";
    result.hidden = true;
  };

  const openCapability = async (
    capability: ConstructiveCompassCapability,
  ): Promise<void> => {
    if (openPending || root.hidden) return;
    openPending = true;
    setTrailButtonsDisabled(true);
    status.textContent = `Opening ${capability.title} in the public explorer…`;
    try {
      const opened = await options.openCapability(capability.id);
      status.textContent = opened
        ? `${capability.title} is open in the public explorer.`
        : "That public capability is unavailable in this static snapshot.";
    } catch {
      status.textContent =
        "The public explorer could not open that capability. This control did not send or save your temporary choice.";
    } finally {
      openPending = false;
      setTrailButtonsDisabled(false);
    }
  };

  const renderPath = (path: ResolvedPiConstructiveCompassPath): void => {
    resultTitle.textContent = path.title;
    resultCopy.textContent =
      `${path.invitation} This is a direction for this visit, not a judgment about you.`;
    trail.replaceChildren();
    path.capabilities.forEach((capability, index) => {
      const item = element("li");
      const button = element("button", "pi-compass-capability");
      button.type = "button";
      button.append(
        element("span", "pi-compass-step", `${index + 1}`.padStart(2, "0")),
        element("strong", undefined, capability.title),
        element("small", undefined, "Open public capability"),
      );
      button.setAttribute(
        "aria-label",
        `Open ${capability.title} in the public constructive-intelligence explorer`,
      );
      button.addEventListener("click", () => {
        void openCapability(capability);
      });
      item.append(button);
      trail.append(item);
    });
    status.textContent =
      "Temporary trail ready. This control did not send the choice to Pi or Zerone servers.";
    result.hidden = false;
    result.focus({ preventScroll: true });
  };

  form.addEventListener("submit", (event) => {
    event.preventDefault();
    if (!consent.checked) return;
    const data = new FormData(form);
    const selectedId = data.get("pi-compass-path");
    if (typeof selectedId !== "string") return;
    const path = paths.find((candidate) => candidate.id === selectedId);
    if (!path) return;
    renderPath(path);
  });

  resetButton.addEventListener("click", () => {
    reset();
    firstChoice.focus();
  });

  reset();
  root.hidden = true;
  return Object.freeze({
    reset,
    setAuthenticated(authenticated: boolean): void {
      if (!authenticated) reset();
      root.hidden = !authenticated;
    },
  });
}
