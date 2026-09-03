export const BLOCK_FRESHNESS_WINDOW_MS = 75_000;

export type NetworkReadiness =
  | "checking"
  | "ready"
  | "syncing"
  | "stale"
  | "unavailable";

export type WalletReadiness =
  | { state: "disconnected" }
  | { state: "connecting" }
  | { state: "connected"; address: string; balanceZrn: string }
  | { state: "error"; message: string };

export type OnboardingState = {
  network: NetworkReadiness;
  wallet: WalletReadiness;
};

export type OnboardingPresentation = {
  network: {
    label: string;
    detail: string;
    tone: "checking" | "ready" | "error";
  };
  wallet: {
    label: string;
    detail: string;
    tone: "neutral" | "checking" | "ready" | "error";
  };
};

export const INITIAL_ONBOARDING_STATE: OnboardingState = {
  network: "checking",
  wallet: { state: "disconnected" },
};

export function assessNetworkReadiness(input: {
  blockAgeMs: number;
  catchingUp: boolean;
  chainMatches: boolean;
  regressed: boolean;
}): NetworkReadiness {
  if (
    !input.chainMatches ||
    input.regressed ||
    !Number.isFinite(input.blockAgeMs) ||
    input.blockAgeMs < 0 ||
    input.blockAgeMs > BLOCK_FRESHNESS_WINDOW_MS
  ) {
    return "stale";
  }
  return input.catchingUp ? "syncing" : "ready";
}

function shortAddress(address: string): string {
  return address.length > 17
    ? `${address.slice(0, 10)}…${address.slice(-6)}`
    : address;
}

export function onboardingPresentation(
  state: OnboardingState,
): OnboardingPresentation {
  const network: OnboardingPresentation["network"] =
    state.network === "ready"
      ? {
          label: "Ready to explore",
          detail: "zerone-1 is responding with a fresh witnessed block.",
          tone: "ready",
        }
      : state.network === "syncing"
        ? {
            label: "Node is syncing",
            detail: "Static views remain open; treat live chain data as incomplete.",
            tone: "checking",
          }
        : state.network === "stale"
          ? {
              label: "Live check needs attention",
              detail: "Static views remain open; verify the chain before relying on live data.",
              tone: "error",
            }
          : state.network === "unavailable"
            ? {
                label: "Live check unavailable",
                detail: "Static views remain open. The public mainnet read did not complete.",
                tone: "error",
              }
            : {
                label: "Checking mainnet",
                detail: "Reading chain identity and latest block.",
                tone: "checking",
              };

  const wallet: OnboardingPresentation["wallet"] =
    state.wallet.state === "connected"
      ? {
          label: "Existing account connected",
          detail: `${shortAddress(state.wallet.address)} · balance: ${state.wallet.balanceZrn} ZRN.`,
          tone: "ready",
        }
      : state.wallet.state === "connecting"
        ? {
            label: "Waiting for Keplr",
            detail: "Review the network and account request in your extension.",
            tone: "checking",
          }
        : state.wallet.state === "error"
          ? {
              label: "Connection not completed",
              detail: state.wallet.message,
              tone: "error",
            }
          : {
              label: "Optional",
              detail: "Explore first, or connect an existing account.",
              tone: "neutral",
            };

  return { network, wallet };
}

export type OnboardingController = {
  setNetwork(network: NetworkReadiness): void;
  setWallet(wallet: WalletReadiness): void;
};

export function initialiseOnboarding(root: HTMLElement): OnboardingController {
  const find = <T extends HTMLElement>(selector: string): T => {
    const element = root.querySelector<T>(selector);
    if (!element) throw new Error(`Missing onboarding element ${selector}`);
    return element;
  };

  const networkStatus = find<HTMLElement>('[data-onboarding-status="network"]');
  const walletStatus = find<HTMLElement>('[data-onboarding-status="wallet"]');
  const networkLabel = find<HTMLElement>("#onboarding-network-state");
  const networkDetail = find<HTMLElement>("#onboarding-network-detail");
  const walletLabel = find<HTMLElement>("#onboarding-wallet-state");
  const walletDetail = find<HTMLElement>("#onboarding-wallet-detail");
  let state: OnboardingState = INITIAL_ONBOARDING_STATE;

  const render = (): void => {
    const view = onboardingPresentation(state);
    networkLabel.textContent = view.network.label;
    networkDetail.textContent = view.network.detail;
    networkStatus.dataset.tone = view.network.tone;
    walletLabel.textContent = view.wallet.label;
    walletDetail.textContent = view.wallet.detail;
    walletStatus.dataset.tone = view.wallet.tone;
  };

  render();
  return {
    setNetwork(network) {
      state = { ...state, network };
      render();
    },
    setWallet(wallet) {
      state = { ...state, wallet };
      render();
    },
  };
}
