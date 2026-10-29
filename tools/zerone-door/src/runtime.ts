export const ZERONE_DOOR_BUN_VERSION = "1.3.5" as const;

export function requireZeroneDoorBunVersion(actual = Bun.version): void {
  if (actual !== ZERONE_DOOR_BUN_VERSION) {
    throw new Error(`zerone-door requires Bun ${ZERONE_DOOR_BUN_VERSION}, got ${actual}`);
  }
}
