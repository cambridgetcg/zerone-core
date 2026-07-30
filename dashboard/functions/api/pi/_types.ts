export interface PiD1Result<T = Record<string, unknown>> {
  readonly results?: readonly T[];
  readonly success: boolean;
  readonly meta?: {
    readonly changes?: number;
  };
}

export interface PiD1PreparedStatement {
  bind(...values: unknown[]): PiD1PreparedStatement;
  first<T = Record<string, unknown>>(columnName?: string): Promise<T | null>;
  all<T = Record<string, unknown>>(): Promise<PiD1Result<T>>;
  run<T = Record<string, unknown>>(): Promise<PiD1Result<T>>;
}

export interface PiD1Database {
  prepare(query: string): PiD1PreparedStatement;
  batch<T = Record<string, unknown>>(
    statements: readonly PiD1PreparedStatement[],
  ): Promise<readonly PiD1Result<T>[]>;
}

export interface PiEnv {
  readonly PI_AUTH_DB?: PiD1Database;
  readonly PI_CLIENT_ID?: string;
  readonly PI_PILOT_ENABLED?: string;
  readonly PI_PUBLIC_ORIGIN?: string;
  readonly PI_SUBJECT_PEPPER?: string;
  readonly PI_WALLET_PROOF_ENABLED?: string;
}

export interface PiSession {
  readonly tokenHash: string;
  readonly subjectHash: string;
  readonly username: string;
  readonly expiresAt: number;
}

export interface PiChallenge {
  readonly idHash: string;
  readonly sessionHash: string;
  readonly subjectHash: string;
  readonly address: string;
  readonly accountId: string;
  readonly message: string;
  readonly createdAt: number;
  readonly expiresAt: number;
}

export interface PiBinding {
  readonly subjectHash: string;
  readonly address: string;
  readonly accountId: string;
  readonly consentVersion: string;
  readonly boundAt: number;
}

export interface PiSessionEnvelope {
  readonly enabled: boolean;
  readonly walletProofEnabled: boolean;
  readonly authenticated: boolean;
  readonly username?: string;
  readonly csrfToken?: string;
  readonly linked?: {
    readonly address: string;
    readonly accountId: string;
  };
}

export interface PiStdSignature {
  readonly pub_key: {
    readonly type: "tendermint/PubKeySecp256k1";
    readonly value: string;
  };
  readonly signature: string;
}

export interface PiRepository {
  createOAuthFlow(
    stateHash: string,
    browserHash: string,
    createdAt: number,
    expiresAt: number,
  ): Promise<void>;
  consumeOAuthFlow(
    stateHash: string,
    browserHash: string,
    bearerFingerprint: string,
    now: number,
  ): Promise<boolean>;
  createSession(session: PiSession, createdAt: number): Promise<void>;
  getSession(tokenHash: string, now: number): Promise<PiSession | null>;
  revokeSession(tokenHash: string, now: number): Promise<void>;
  createChallenge(
    challenge: PiChallenge,
    recentSince: number,
    maximumRecent: number,
  ): Promise<boolean>;
  getChallenge(
    idHash: string,
    sessionHash: string,
    subjectHash: string,
    now: number,
  ): Promise<PiChallenge | null>;
  bindChallenge(
    idHash: string,
    sessionHash: string,
    subjectHash: string,
    proofHash: string,
    consentVersion: string,
    now: number,
  ): Promise<PiBinding | null>;
  getBinding(subjectHash: string): Promise<PiBinding | null>;
  deleteBinding(subjectHash: string, now: number): Promise<void>;
}

export type PiFetch = (
  input: string | URL | Request,
  init?: RequestInit,
) => Promise<Response>;

export interface PiRuntime {
  readonly fetch: PiFetch;
  readonly now: () => number;
  readonly randomBytes: (length: number) => Uint8Array;
  readonly repository: PiRepository;
}

export interface PiPagesContext {
  readonly request: Request;
  readonly env: PiEnv;
}
