import {
  beginPiSignIn,
  bindWalletProof,
  endPiSession,
  getPiSession,
  removeWalletProof,
  requestWalletChallenge,
  type PiSession,
  type PiWalletProofSignature,
} from "./pi";
import {
  initialisePiConstructiveCompass,
  type PiConstructiveCompassController,
  type PiConstructiveCompassOptions,
} from "./pi-constructive-compass";
import type { WalletState } from "./wallet";

export interface PiPilotUiOptions {
  walletProofEnabled: boolean;
  constructiveCompass: Promise<PiConstructiveCompassOptions | null> | null;
  getWallet(): WalletState | null;
  connectWallet(): Promise<WalletState | null>;
  signWalletProof(
    wallet: WalletState,
    message: string,
  ): Promise<PiWalletProofSignature>;
  notify(message: string, tone?: "success" | "error"): void;
}

const byId = <T extends HTMLElement>(id: string): T => {
  const element = document.getElementById(id);
  if (!element) throw new Error(`Missing #${id}`);
  return element as T;
};

function displayUsername(username: string): string {
  return username.startsWith("@") ? username : `@${username}`;
}

function attachDialogClose(
  dialog: HTMLDialogElement,
  closeButton: HTMLButtonElement,
  isPending: () => boolean,
): void {
  closeButton.addEventListener("click", () => {
    if (!isPending()) dialog.close();
  });
  dialog.addEventListener("click", (event) => {
    if (event.target === dialog && !isPending()) dialog.close();
  });
  dialog.addEventListener("cancel", (event) => {
    if (isPending()) event.preventDefault();
  });
}

export async function initialisePiPilot(
  options: PiPilotUiOptions,
): Promise<void> {
  let session: PiSession;
  try {
    session = await getPiSession();
  } catch {
    return;
  }
  if (!session.enabled) return;

  const navigationLink = byId<HTMLAnchorElement>("pi-nav");
  const section = byId<HTMLElement>("contribute");
  const signedOut = byId<HTMLDivElement>("pi-signed-out");
  const signedIn = byId<HTMLDivElement>("pi-signed-in");
  const username = byId<HTMLSpanElement>("pi-username");
  const proofSection = byId<HTMLDivElement>("pi-wallet-proof");
  const proofStatus = byId<HTMLParagraphElement>("pi-wallet-proof-status");
  const linkedAddressRow = byId<HTMLDivElement>("pi-linked-address-row");
  const linkedAddress = byId<HTMLElement>("pi-linked-address");
  const proofReviewButton = byId<HTMLButtonElement>("pi-proof-review");
  const proofRemoveButton = byId<HTMLButtonElement>("pi-proof-remove");
  const logoutButton = byId<HTMLButtonElement>("pi-logout");

  const consentDialog = byId<HTMLDialogElement>("pi-consent-dialog");
  const consentForm = byId<HTMLFormElement>("pi-consent-form");
  const consentCheck = byId<HTMLInputElement>("pi-consent-check");
  const consentClose = byId<HTMLButtonElement>("pi-consent-close");
  const consentSubmit = byId<HTMLButtonElement>("pi-consent-submit");

  const proofDialog = byId<HTMLDialogElement>("pi-proof-dialog");
  const proofForm = byId<HTMLFormElement>("pi-proof-form");
  const proofCheck = byId<HTMLInputElement>("pi-proof-check");
  const proofClose = byId<HTMLButtonElement>("pi-proof-close");
  const proofSubmit = byId<HTMLButtonElement>("pi-proof-submit");

  let authPending = false;
  let proofPending = false;
  let sessionPending = false;
  let constructiveCompass: PiConstructiveCompassController | null = null;

  const render = (): void => {
    signedOut.hidden = session.authenticated;
    signedIn.hidden = !session.authenticated;
    constructiveCompass?.setAuthenticated(session.authenticated);
    if (!session.authenticated) {
      username.textContent = "Pi account";
      linkedAddress.textContent = "Not linked";
      linkedAddress.removeAttribute("title");
      linkedAddressRow.hidden = true;
      proofStatus.textContent = "";
      proofSection.hidden = true;
      proofReviewButton.hidden = true;
      proofRemoveButton.hidden = true;
      return;
    }

    username.textContent = displayUsername(session.username ?? "Pi account");
    const linked = session.linked;
    const proofAvailable =
      options.walletProofEnabled && session.walletProofEnabled;

    proofSection.hidden = !proofAvailable && linked === undefined;
    proofReviewButton.hidden = !proofAvailable || linked !== undefined;
    proofRemoveButton.hidden = linked === undefined;
    linkedAddressRow.hidden = linked === undefined;

    if (linked) {
      linkedAddress.textContent = linked.address;
      linkedAddress.title = linked.accountId;
      proofStatus.textContent =
        "A text-only signature demonstrated control of this Zerone address. This remains separate from Pi Sign-in and creates no identity, qualification, reward, or transaction.";
    } else if (proofAvailable) {
      linkedAddress.textContent = "Not linked";
      linkedAddress.removeAttribute("title");
      proofStatus.textContent =
        "Optional: prove control of one zrn1… address with a text-only Keplr signature. This is not required for the dashboard or public contribution paths.";
    }
  };

  navigationLink.hidden = false;
  section.hidden = false;
  render();

  const pendingCompass = options.constructiveCompass;
  if (pendingCompass) {
    void pendingCompass.then((compassOptions) => {
      if (!compassOptions || !session.authenticated) return;
      try {
        constructiveCompass = initialisePiConstructiveCompass(compassOptions);
        constructiveCompass.setAuthenticated(true);
      } catch {
        options.notify(
          "The optional Constructive Compass is unavailable. The public skill tree remains available.",
          "error",
        );
      }
    });
  }

  attachDialogClose(consentDialog, consentClose, () => authPending);
  attachDialogClose(proofDialog, proofClose, () => proofPending);

  byId("pi-consent-open").addEventListener("click", () => {
    consentCheck.checked = false;
    consentDialog.showModal();
    window.setTimeout(() => consentCheck.focus(), 0);
  });

  consentForm.addEventListener("submit", (event) => {
    event.preventDefault();
    if (authPending || !consentCheck.checked) return;
    authPending = true;
    consentClose.disabled = true;
    consentSubmit.disabled = true;
    consentSubmit.textContent = "Continuing to Pi…";
    try {
      beginPiSignIn();
    } catch {
      authPending = false;
      consentClose.disabled = false;
      consentSubmit.disabled = false;
      consentSubmit.textContent = "Continue to Pi";
      options.notify(
        "Pi Sign-in could not start. The dashboard and Keplr remain available.",
        "error",
      );
    }
  });

  logoutButton.addEventListener("click", async () => {
    if (
      sessionPending ||
      !session.authenticated ||
      !session.csrfToken
    ) {
      return;
    }
    sessionPending = true;
    logoutButton.disabled = true;
    logoutButton.textContent = "Ending session…";
    try {
      session = await endPiSession(session.csrfToken);
      render();
      options.notify(
        "Pi session ended. Your Pi account and Zerone wallet were not changed.",
      );
    } catch {
      options.notify(
        "The Pi session could not be ended right now. No wallet or on-chain action occurred.",
        "error",
      );
    } finally {
      sessionPending = false;
      logoutButton.disabled = false;
      logoutButton.textContent = "End Pi session";
    }
  });

  proofReviewButton.addEventListener("click", () => {
    if (
      !session.authenticated ||
      !options.walletProofEnabled ||
      !session.walletProofEnabled
    ) {
      return;
    }
    proofCheck.checked = false;
    proofDialog.showModal();
    window.setTimeout(() => proofCheck.focus(), 0);
  });

  proofForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    if (
      proofPending ||
      !proofCheck.checked ||
      !session.authenticated ||
      !session.csrfToken ||
      !options.walletProofEnabled ||
      !session.walletProofEnabled
    ) {
      return;
    }

    proofPending = true;
    proofClose.disabled = true;
    proofSubmit.disabled = true;
    proofSubmit.textContent = "Preparing proof…";
    try {
      let wallet = options.getWallet();
      if (!wallet) wallet = await options.connectWallet();
      if (!wallet) {
        throw new Error("A Zerone wallet is required for this optional proof.");
      }

      const challenge = await requestWalletChallenge(
        wallet.address,
        session.csrfToken,
      );
      if (Date.parse(challenge.expiresAt) <= Date.now()) {
        throw new Error("The wallet proof challenge expired before signing.");
      }
      proofSubmit.textContent = "Waiting for Keplr…";
      const signature = await options.signWalletProof(wallet, challenge.message);
      proofSubmit.textContent = "Checking proof…";
      const updatedSession = await bindWalletProof(
        challenge.challengeId,
        signature,
        session.csrfToken,
      );
      if (updatedSession.linked?.address !== wallet.address) {
        throw new Error(
          "The stored wallet proof did not match the connected address.",
        );
      }
      session = updatedSession;
      render();
      proofDialog.close();
      options.notify(
        "Zerone address control demonstrated. No transaction was broadcast.",
      );
    } catch {
      options.notify(
        "The optional address proof was not completed. No transaction, payment, qualification, or reward was created.",
        "error",
      );
    } finally {
      proofPending = false;
      proofClose.disabled = false;
      proofSubmit.disabled = false;
      proofSubmit.textContent = "Sign text proof in Keplr";
    }
  });

  proofRemoveButton.addEventListener("click", async () => {
    if (
      proofPending ||
      !session.authenticated ||
      !session.csrfToken ||
      !session.linked
    ) {
      return;
    }

    proofPending = true;
    proofRemoveButton.disabled = true;
    proofRemoveButton.textContent = "Removing…";
    try {
      session = await removeWalletProof(session.csrfToken);
      render();
      options.notify(
        "Stored Zerone address proof removed. No wallet or on-chain state changed.",
      );
    } catch {
      options.notify(
        "The stored address proof could not be removed right now.",
        "error",
      );
    } finally {
      proofPending = false;
      proofRemoveButton.disabled = false;
      proofRemoveButton.textContent = "Remove address proof";
    }
  });
}
