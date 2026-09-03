import assert from "node:assert/strict";
import test from "node:test";

import {
  beginPiSignIn,
  deletePiPilotData,
} from "../src/pi";

const ORIGIN = "https://zerone.ai";
const STATE = "S".repeat(43);

function authorizationUrl(overrides: Record<string, string> = {}): string {
  const target = new URL("https://accounts.pinet.com/oauth/authorize");
  const parameters = {
    client_id: "pi-client-id",
    redirect_uri: `${ORIGIN}/pi/callback/`,
    response_type: "token",
    scope: "username",
    state: STATE,
    ...overrides,
  };
  Object.entries(parameters).forEach(([key, value]) => {
    target.searchParams.set(key, value);
  });
  return target.toString();
}

test("Pi sign-in starts with an empty same-origin POST and navigates only after validation", async (context) => {
  const originalFetch = globalThis.fetch;
  const originalWindow = Object.getOwnPropertyDescriptor(globalThis, "window");
  context.after(() => {
    globalThis.fetch = originalFetch;
    if (originalWindow) {
      Object.defineProperty(globalThis, "window", originalWindow);
    } else {
      Reflect.deleteProperty(globalThis, "window");
    }
  });

  let navigatedTo = "";
  Object.defineProperty(globalThis, "window", {
    configurable: true,
    value: {
      location: {
        origin: ORIGIN,
        assign(value: string | URL) {
          navigatedTo = String(value);
        },
      },
    },
    writable: true,
  });
  let inputSeen: string | URL | Request | undefined;
  let initSeen: RequestInit | undefined;
  globalThis.fetch = async (input, init) => {
    inputSeen = input;
    initSeen = init;
    return Response.json({ authorizationUrl: authorizationUrl() });
  };

  await beginPiSignIn();

  assert.equal(String(inputSeen), `${ORIGIN}/api/pi/authorize`);
  assert.equal(initSeen?.method, "POST");
  assert.equal(initSeen?.credentials, "same-origin");
  assert.equal(initSeen?.cache, "no-store");
  assert.equal(initSeen?.redirect, "error");
  assert.equal(initSeen?.referrerPolicy, "no-referrer");
  assert.deepEqual(JSON.parse(String(initSeen?.body)), {});
  assert.equal(new Headers(initSeen?.headers).get("Content-Type"), "application/json");
  assert.equal(navigatedTo, authorizationUrl());
});

test("Pi sign-in refuses authorization URLs outside the exact provider contract", async (context) => {
  const originalFetch = globalThis.fetch;
  const originalWindow = Object.getOwnPropertyDescriptor(globalThis, "window");
  context.after(() => {
    globalThis.fetch = originalFetch;
    if (originalWindow) {
      Object.defineProperty(globalThis, "window", originalWindow);
    } else {
      Reflect.deleteProperty(globalThis, "window");
    }
  });

  let navigationCount = 0;
  Object.defineProperty(globalThis, "window", {
    configurable: true,
    value: {
      location: {
        origin: ORIGIN,
        assign() {
          navigationCount += 1;
        },
      },
    },
    writable: true,
  });

  const unsafeUrls = [
    authorizationUrl().replace("https://accounts.pinet.com", "https://attacker.invalid"),
    authorizationUrl({ redirect_uri: "https://attacker.invalid/callback" }),
    authorizationUrl({ scope: "username payments" }),
    authorizationUrl({ state: "short" }),
  ];
  for (const authorizationUrlValue of unsafeUrls) {
    globalThis.fetch = async () =>
      Response.json({ authorizationUrl: authorizationUrlValue });
    await assert.rejects(() => beginPiSignIn(), /authorization response was invalid/u);
  }
  assert.equal(navigationCount, 0);
});

test("pilot-data deletion uses the exact same-origin DELETE contract", async (context) => {
  const originalFetch = globalThis.fetch;
  const originalWindow = Object.getOwnPropertyDescriptor(globalThis, "window");
  context.after(() => {
    globalThis.fetch = originalFetch;
    if (originalWindow) {
      Object.defineProperty(globalThis, "window", originalWindow);
    } else {
      Reflect.deleteProperty(globalThis, "window");
    }
  });

  Object.defineProperty(globalThis, "window", {
    configurable: true,
    value: { location: { origin: "https://zerone.ai" } },
    writable: true,
  });
  let inputSeen: string | URL | Request | undefined;
  let initSeen: RequestInit | undefined;
  globalThis.fetch = async (input, init) => {
    inputSeen = input;
    initSeen = init;
    return Response.json({
      enabled: true,
      walletProofEnabled: true,
      authenticated: false,
    });
  };

  const csrfToken = "C".repeat(43);
  const session = await deletePiPilotData(csrfToken);

  assert.deepEqual(session, {
    enabled: true,
    walletProofEnabled: true,
    authenticated: false,
  });
  assert.equal(String(inputSeen), "https://zerone.ai/api/pi/data");
  assert.equal(initSeen?.method, "DELETE");
  assert.equal(initSeen?.credentials, "same-origin");
  assert.equal(initSeen?.cache, "no-store");
  assert.equal(initSeen?.redirect, "error");
  assert.equal(initSeen?.referrerPolicy, "no-referrer");
  assert.deepEqual(JSON.parse(String(initSeen?.body)), {
    confirmation: "delete-pi-pilot-data-v1",
  });
  const headers = new Headers(initSeen?.headers);
  assert.equal(headers.get("Content-Type"), "application/json");
  assert.equal(headers.get("X-Zerone-CSRF"), csrfToken);
});

test("pilot-data deletion rejects a response that leaves a session active", async (context) => {
  const originalFetch = globalThis.fetch;
  const originalWindow = Object.getOwnPropertyDescriptor(globalThis, "window");
  context.after(() => {
    globalThis.fetch = originalFetch;
    if (originalWindow) {
      Object.defineProperty(globalThis, "window", originalWindow);
    } else {
      Reflect.deleteProperty(globalThis, "window");
    }
  });
  Object.defineProperty(globalThis, "window", {
    configurable: true,
    value: { location: { origin: "https://zerone.ai" } },
    writable: true,
  });
  globalThis.fetch = async () =>
    Response.json({
      enabled: true,
      walletProofEnabled: true,
      authenticated: true,
      username: "pioneer",
      csrfToken: "C".repeat(43),
    });

  await assert.rejects(
    () => deletePiPilotData("C".repeat(43)),
    /data was not deleted/u,
  );
});
