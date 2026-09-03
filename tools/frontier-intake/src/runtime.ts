export const FRONTIER_INTAKE_BUN_VERSION = "1.3.5" as const;

export function requireFrontierIntakeBunVersion(
  actual = Bun.version,
): void {
  if (actual !== FRONTIER_INTAKE_BUN_VERSION) {
    throw new Error(
      `frontier-intake requires Bun ${FRONTIER_INTAKE_BUN_VERSION}, got ${actual}`,
    );
  }
}
