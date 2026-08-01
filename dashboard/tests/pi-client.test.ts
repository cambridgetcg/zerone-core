import assert from "node:assert/strict";
import test from "node:test";

import { deletePiPilotData } from "../src/pi";

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
