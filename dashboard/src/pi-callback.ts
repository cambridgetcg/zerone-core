declare global {
  interface Window {
    __takeZeronePiCallbackFragment?: () => string;
  }
}

// This entry intentionally has no static imports. The parser-blocking bootstrap
// has already erased the URL and exposes only a one-shot closure, never the
// bearer value itself, on window.
const takeCapturedFragment = window.__takeZeronePiCallbackFragment;
let capturedFragment =
  typeof takeCapturedFragment === "function"
    ? takeCapturedFragment()
    : "";
delete window.__takeZeronePiCallbackFragment;
if (window.location.hash !== "") {
  window.history.replaceState(
    window.history.state,
    "",
    `${window.location.pathname}${window.location.search}`,
  );
}

const callbackTitle = document.getElementById("callback-title");
const callbackStatus = document.getElementById("callback-status");
const callbackReturn = document.getElementById("callback-return");

function setCallbackCopy(title: string, status: string, showReturn: boolean): void {
  if (callbackTitle) callbackTitle.textContent = title;
  if (callbackStatus) callbackStatus.textContent = status;
  if (callbackReturn instanceof HTMLAnchorElement) {
    callbackReturn.hidden = !showReturn;
  }
}

setCallbackCopy(
  "Finishing sign-in…",
  "Confirming the Pi account with Zerone. No wallet or on-chain action is being requested.",
  true,
);

async function completePiCallback(): Promise<void> {
  try {
    if (
      window.location.pathname !== "/pi/callback/" ||
      window.location.search !== ""
    ) {
      throw new Error("The Pi callback location is invalid.");
    }
    const { parsePiCallbackFragment } = await import("./pi-fragment");
    const parsed = parsePiCallbackFragment(capturedFragment);
    capturedFragment = "";

    if (parsed.kind === "error") {
      setCallbackCopy(
        "Sign-in was not completed.",
        "Pi did not approve this sign-in. No Zerone account, wallet proof, contribution record, payment, qualification, or reward was created.",
        true,
      );
      return;
    }

    const { finishPiSignIn } = await import("./pi");
    await finishPiSignIn(parsed.accessToken, parsed.state);
    setCallbackCopy(
      "Pi account authenticated.",
      "This confirms a Pi account login only. It is not KYC, unique-human proof, a Zerone identity, or wallet control.",
      false,
    );
    window.location.replace("/#contribute");
  } catch {
    capturedFragment = "";
    setCallbackCopy(
      "Sign-in could not be completed.",
      "No Zerone account, wallet proof, contribution record, payment, qualification, or reward was created. You can return to the ordinary dashboard.",
      true,
    );
  }
}

void completePiCallback();
