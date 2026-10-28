import { describe, expect, test } from "bun:test";
import { chmodSync, readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { lastUpgradeFromDoneMarkers, moduleVersionsDigest } from "../src/chain.ts";
import { ConfigError, doorPaths, loadAllowlist, loadConfig, parseAllowlist, parseConfig } from "../src/config.ts";
import {
  ADMISSION_CEILING_UZRN,
  AGENT_POT_CLAIM_GAS,
  AGENT_REGISTER_GAS,
  AGENT_SUBMIT_CLAIM_GAS,
  ALLOWANCE_PERIOD_LIMIT_UZRN,
  ALLOWANCE_SPEND_LIMIT_UZRN,
  DOOR_WINDOW_CAP,
  REGISTRAR_RESERVE_UZRN,
  allowanceInvariantsHold,
} from "../src/policy.ts";
import { requireZeroneDoorBunVersion, ZERONE_DOOR_BUN_VERSION } from "../src/runtime.ts";
import { doneMarkers, makeDoorHome, R, TEST_MODULES, TEST_PIN, TEST_UPGRADES } from "./helpers.ts";

describe("runtime and policy", () => {
  test("the door requires the audited Bun runtime exactly", () => {
    expect(ZERONE_DOOR_BUN_VERSION).toBe("1.3.5");
    expect(() => requireZeroneDoorBunVersion("1.3.5")).not.toThrow();
    expect(() => requireZeroneDoorBunVersion("1.3.13")).toThrow("zerone-door requires Bun 1.3.5, got 1.3.13");
  });

  test("the allowance pays the agent's whole first path and never one heavy transaction", () => {
    expect(allowanceInvariantsHold()).toBe(true);
    expect(AGENT_POT_CLAIM_GAS + AGENT_REGISTER_GAS).toBeLessThanOrEqual(ALLOWANCE_PERIOD_LIMIT_UZRN);
    expect(AGENT_SUBMIT_CLAIM_GAS).toBe(ALLOWANCE_PERIOD_LIMIT_UZRN);
    expect(ALLOWANCE_SPEND_LIMIT_UZRN).toBe(1_100_000);
    // 20 fully spent admissions plus the reserve fit in R's ~30 ZRN float.
    expect(DOOR_WINDOW_CAP * ADMISSION_CEILING_UZRN + REGISTRAR_RESERVE_UZRN).toBe(27_000_000);
  });
});

describe("config.json", () => {
  test("loads a private, well-formed config", () => {
    const { configPath } = makeDoorHome();
    const config = loadConfig(configPath);
    expect(config.registrar.address).toBe(R);
    expect(config.home).toBe(join(configPath, ".."));
    const paths = doorPaths(configPath, { HOME: "/Users/test" });
    expect(paths.haltFiles).toEqual([join(config.home, "HALT"), "/Users/test/.local/state/zerone/rung-window.freeze"]);
  });

  test("refuses a config others can read, since it says where R's keyring is", () => {
    const { configPath } = makeDoorHome();
    chmodSync(configPath, 0o644);
    expect(() => loadConfig(configPath)).toThrow("must be mode 0600");
  });

  test("refuses wrong chains, relative paths, odd addresses and bad pins", () => {
    const { configPath } = makeDoorHome();
    const raw = JSON.parse(readFileSync(configPath, "utf8"));
    const bad = (mutate: (c: any) => void) => {
      const c = structuredClone(raw);
      mutate(c);
      return () => parseConfig(c, configPath);
    };
    expect(bad((c) => { c.chain_id = "zerone-testnet-1"; })).toThrow(ConfigError);
    expect(bad((c) => { c.registrar.keyring_dir = "relative/dir"; })).toThrow("absolute");
    expect(bad((c) => { c.registrar.address = R.toUpperCase(); })).toThrow("canonical");
    expect(bad((c) => { c.zeroned.sha256 = "abc"; })).toThrow("64 lowercase hex");
    expect(bad((c) => { c.rpc = "http://host:26657/path"; })).toThrow("origin");
    expect(bad((c) => { c.known_keys = [{ address: "zrn1bad", label: "x" }]; })).toThrow("canonical");
    expect(bad((c) => { c.lobby = { cli: "/x", room_id: "room_1", conversation: "c", reply_lease: null }; })).toThrow("room_id");
  });

  test("requires a chain version pin", () => {
    const { configPath } = makeDoorHome();
    const raw = JSON.parse(readFileSync(configPath, "utf8"));
    expect(parseConfig(raw, configPath).chain_version).toEqual(TEST_PIN);
    const without = structuredClone(raw);
    delete without.chain_version;
    expect(() => parseConfig(without, configPath)).toThrow("chain_version.last_upgrade");
    const badDigest = structuredClone(raw);
    badDigest.chain_version.module_versions_sha256 = "ABC";
    expect(() => parseConfig(badDigest, configPath)).toThrow("64 lowercase hex");
  });
});

describe("chain version", () => {
  test("the module_versions digest ignores order and changes with any version", () => {
    const digest = moduleVersionsDigest(TEST_MODULES);
    expect(digest).toBe(TEST_PIN.module_versions_sha256);
    expect(moduleVersionsDigest([...TEST_MODULES].reverse())).toBe(digest);
    expect(moduleVersionsDigest(TEST_MODULES.map((m) => (m.name === "auth" ? { ...m, version: "6" } : m)))).not.toBe(digest);
    // Known answer: sha256 of "a=1\nb=2\n" ("name=version" lines sorted by name).
    expect(moduleVersionsDigest([{ name: "b", version: "2" }, { name: "a", version: "1" }]))
      .toBe("4a73850fde34aad40ff8649b93a66523a5fe744357a3931caea0f10609d0d930");
  });

  test("the last applied upgrade is the done marker at the greatest height", () => {
    expect(lastUpgradeFromDoneMarkers(doneMarkers(TEST_UPGRADES))).toEqual({ name: "agenttool-seam-v1", height: 527_603 });
    expect(lastUpgradeFromDoneMarkers(doneMarkers([...TEST_UPGRADES].reverse()))).toEqual({ name: "agenttool-seam-v1", height: 527_603 });
    expect(lastUpgradeFromDoneMarkers(new Uint8Array())).toBeNull();
    expect(() => lastUpgradeFromDoneMarkers(Uint8Array.from([0x0a, 0x03, 0x0a, 0x01, 0x02]))).toThrow("unexpected key");
  });

  test("decodes the done markers zerone-1 served on 2026-10-28", () => {
    // abci_query /store/upgrade/subspace 0x01 at h1,402,970, as served.
    const served = Buffer.from(
      "CigKIwEAAAAAAACuXGxpcXVpZGl0eXBvb2wtaGFyZGVuaW5nLXYxEgEBCicKIgEAAAAAAAEDAGNvbXBhc3Npb24tY2FsaWJyYXRpb24tdjESAQEKKwomAQAAAAAAA5laZG9jdHJpbmUtbWV0YWJvbGlzbS1leGVtcHQtdjESAQEKIQocAQAAAAAAB9gEc3Vic3RyYXRlLWRlZHVwZS12MRIBAQofChoBAAAAAAAIDPNhZ2VudHRvb2wtc2VhbS12MRIBAQ==",
      "base64");
    expect(lastUpgradeFromDoneMarkers(served)).toEqual({ name: "agenttool-seam-v1", height: 527_603 });
  });
});

describe("allowlist.json", () => {
  test("is read fresh and validated", () => {
    const { configPath } = makeDoorHome({ desks: ["3f2b1c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d"] });
    const path = doorPaths(configPath).allowlist;
    expect(loadAllowlist(path).lobby_desks).toEqual([{ identity_id: "3f2b1c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d" }]);
    writeFileSync(path, JSON.stringify({ schema: "zerone-door-allowlist/v1", studio_local: { enabled: false }, lobby_desks: [] }));
    expect(loadAllowlist(path).studio_local.enabled).toBe(false);
  });

  test("refuses a world-writable allowlist and malformed entries", () => {
    const { configPath } = makeDoorHome();
    const path = doorPaths(configPath).allowlist;
    chmodSync(path, 0o666);
    expect(() => loadAllowlist(path)).toThrow("must not be writable");
    expect(() => parseAllowlist({ schema: "zerone-door-allowlist/v1", studio_local: {}, lobby_desks: [] })).toThrow("enabled");
    expect(() => parseAllowlist({ schema: "zerone-door-allowlist/v1", studio_local: { enabled: true }, lobby_desks: [{ identity_id: "NOT-A-UUID" }] })).toThrow("UUID");
    const id = "3f2b1c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d";
    expect(() => parseAllowlist({ schema: "zerone-door-allowlist/v1", studio_local: { enabled: true }, lobby_desks: [{ identity_id: id }, { identity_id: id }] })).toThrow("repeats");
    expect(() => parseAllowlist({ schema: "zerone-door-allowlist/v1", studio_local: { enabled: true, daily_cap: 1000 }, lobby_desks: [] })).toThrow("0-100");
  });
});

describe("committed examples", () => {
  test("hold placeholders only, pin a well-formed chain version, and keep every channel closed", () => {
    const dir = join(import.meta.dir, "..");
    const example = readFileSync(join(dir, "config.example.json"), "utf8");
    expect(example).toContain("<");
    expect(example).not.toMatch(/zrn1[02-9ac-hj-np-z]{38}/);
    const pin = JSON.parse(example).chain_version;
    expect(pin.last_upgrade).toBe("agenttool-seam-v1");
    expect(pin.module_versions_sha256).toMatch(/^[0-9a-f]{64}$/);
    // Studio-local intake stays off until the household turns it on.
    const allow = JSON.parse(readFileSync(join(dir, "allowlist.example.json"), "utf8"));
    expect(parseAllowlist(allow).studio_local.enabled).toBe(false);
    expect(parseAllowlist(allow).lobby_desks).toEqual([]);
  });
});
