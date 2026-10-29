// R's keyring passphrase, read at signing time from a login-Keychain generic
// password with /usr/bin/security. The value travels only through pipes:
// security's stdout into this process, then zeroned's stdin. It is never on
// an argv (ps shows argv), never in the environment and never logged. Every
// buffer the door itself holds it in is zeroed after use; copies inside the
// Bun runtime, the pipes and the two child processes are beyond the door's
// reach. The service name comes from the local config.

export type PassphraseSource = (service: string) => Uint8Array | null;

export function keychainPassphrase(service: string): Uint8Array | null {
  const proc = Bun.spawnSync({
    cmd: ["/usr/bin/security", "find-generic-password", "-s", service, "-w"],
    stdin: "ignore",
    stdout: "pipe",
    stderr: "pipe",
    env: { HOME: process.env.HOME ?? "/nonexistent", PATH: "/usr/bin:/bin" },
  });
  if (proc.exitCode !== 0) return null;
  const out = Uint8Array.from(proc.stdout);
  proc.stdout.fill(0);
  let end = out.length;
  if (end > 0 && out[end - 1] === 0x0a) end -= 1;
  if (end === 0) return null;
  const value = out.slice(0, end);
  out.fill(0);
  return value;
}
