import type {
  PiBinding,
  PiChallenge,
  PiD1Database,
  PiRepository,
  PiSession,
} from "./_types";

type SqlRow = Record<string, unknown>;

function stringField(row: SqlRow, name: string): string | null {
  const value = row[name];
  return typeof value === "string" ? value : null;
}

function integerField(row: SqlRow, name: string): number | null {
  const value = row[name];
  return typeof value === "number" && Number.isSafeInteger(value) ? value : null;
}

function sessionFromRow(row: SqlRow | null): PiSession | null {
  if (!row) return null;
  const tokenHash = stringField(row, "token_hash");
  const subjectHash = stringField(row, "subject_hash");
  const username = stringField(row, "username");
  const expiresAt = integerField(row, "expires_at");
  return tokenHash && subjectHash && username && expiresAt !== null
    ? { tokenHash, subjectHash, username, expiresAt }
    : null;
}

function challengeFromRow(row: SqlRow | null): PiChallenge | null {
  if (!row) return null;
  const idHash = stringField(row, "id_hash");
  const sessionHash = stringField(row, "session_hash");
  const subjectHash = stringField(row, "subject_hash");
  const address = stringField(row, "address");
  const accountId = stringField(row, "account_id");
  const message = stringField(row, "message");
  const createdAt = integerField(row, "created_at");
  const expiresAt = integerField(row, "expires_at");
  return idHash &&
    sessionHash &&
    subjectHash &&
    address &&
    accountId &&
    message &&
    createdAt !== null &&
    expiresAt !== null
    ? {
        idHash,
        sessionHash,
        subjectHash,
        address,
        accountId,
        message,
        createdAt,
        expiresAt,
      }
    : null;
}

function bindingFromRow(row: SqlRow | null): PiBinding | null {
  if (!row) return null;
  const subjectHash = stringField(row, "subject_hash");
  const address = stringField(row, "address");
  const accountId = stringField(row, "account_id");
  const consentVersion = stringField(row, "consent_version");
  const boundAt = integerField(row, "bound_at");
  return subjectHash &&
    address &&
    accountId &&
    consentVersion &&
    boundAt !== null
    ? { subjectHash, address, accountId, consentVersion, boundAt }
    : null;
}

function isConstraintFailure(error: unknown): boolean {
  if (!(error instanceof Error)) return false;
  const message = error.message.toLowerCase();
  return message.includes("constraint") || message.includes("unique");
}

export class D1PiRepository implements PiRepository {
  constructor(private readonly database: PiD1Database) {}

  async createOAuthFlow(
    stateHash: string,
    browserHash: string,
    createdAt: number,
    expiresAt: number,
  ): Promise<void> {
    await this.database
      .prepare(
        `INSERT INTO pi_oauth_flows
           (state_hash, browser_hash, created_at, expires_at)
         VALUES (?, ?, ?, ?)`,
      )
      .bind(stateHash, browserHash, createdAt, expiresAt)
      .run();
  }

  async consumeOAuthFlow(
    stateHash: string,
    browserHash: string,
    bearerFingerprint: string,
    now: number,
  ): Promise<boolean> {
    try {
      const row = await this.database
        .prepare(
          `INSERT INTO pi_bearer_claims
             (fingerprint, state_hash, browser_hash, claimed_at)
           SELECT ?, f.state_hash, f.browser_hash, ?
           FROM pi_oauth_flows AS f
           WHERE f.state_hash = ?
             AND f.browser_hash = ?
             AND f.consumed_at IS NULL
             AND f.expires_at > ?
           RETURNING fingerprint`,
        )
        .bind(bearerFingerprint, now, stateHash, browserHash, now)
        .first<SqlRow>();
      return row !== null;
    } catch (error) {
      if (isConstraintFailure(error)) return false;
      throw error;
    }
  }

  async createSession(session: PiSession, createdAt: number): Promise<void> {
    await this.database
      .prepare(
        `INSERT INTO pi_sessions
           (token_hash, subject_hash, username, created_at, expires_at)
         VALUES (?, ?, ?, ?, ?)`,
      )
      .bind(
        session.tokenHash,
        session.subjectHash,
        session.username,
        createdAt,
        session.expiresAt,
      )
      .run();
  }

  async getSession(tokenHash: string, now: number): Promise<PiSession | null> {
    const row = await this.database
      .prepare(
        `SELECT token_hash, subject_hash, username, expires_at
         FROM pi_sessions
         WHERE token_hash = ?
           AND revoked_at IS NULL
           AND expires_at > ?
         LIMIT 1`,
      )
      .bind(tokenHash, now)
      .first<SqlRow>();
    return sessionFromRow(row);
  }

  async revokeSession(tokenHash: string, now: number): Promise<void> {
    await this.database
      .prepare(
        `UPDATE pi_sessions
         SET revoked_at = COALESCE(revoked_at, ?)
         WHERE token_hash = ?`,
      )
      .bind(now, tokenHash)
      .run();
  }

  async createChallenge(
    challenge: PiChallenge,
    recentSince: number,
    maximumRecent: number,
  ): Promise<boolean> {
    const row = await this.database
      .prepare(
        `INSERT INTO pi_wallet_challenges
           (id_hash, session_hash, subject_hash, address, account_id, message,
            created_at, expires_at)
         SELECT ?, ?, ?, ?, ?, ?, ?, ?
         WHERE (
           SELECT COUNT(*)
           FROM pi_wallet_challenges
           WHERE session_hash = ?
             AND created_at >= ?
         ) < ?
           AND EXISTS (
             SELECT 1
             FROM pi_sessions AS s
             WHERE s.token_hash = ?
               AND s.subject_hash = ?
               AND s.revoked_at IS NULL
               AND s.expires_at > ?
           )
           AND NOT EXISTS (
             SELECT 1
             FROM pi_wallet_bindings
             WHERE subject_hash = ?
           )
         RETURNING id_hash`,
      )
      .bind(
        challenge.idHash,
        challenge.sessionHash,
        challenge.subjectHash,
        challenge.address,
        challenge.accountId,
        challenge.message,
        challenge.createdAt,
        challenge.expiresAt,
        challenge.sessionHash,
        recentSince,
        maximumRecent,
        challenge.sessionHash,
        challenge.subjectHash,
        challenge.createdAt,
        challenge.subjectHash,
      )
      .first<SqlRow>();
    return row !== null;
  }

  async getChallenge(
    idHash: string,
    sessionHash: string,
    subjectHash: string,
    now: number,
  ): Promise<PiChallenge | null> {
    const row = await this.database
      .prepare(
        `SELECT c.id_hash, c.session_hash, c.subject_hash, c.address,
                c.account_id, c.message, c.created_at, c.expires_at
         FROM pi_wallet_challenges AS c
         INNER JOIN pi_sessions AS s
           ON s.token_hash = c.session_hash
         WHERE c.id_hash = ?
           AND c.session_hash = ?
           AND c.subject_hash = ?
           AND c.expires_at > ?
           AND s.revoked_at IS NULL
           AND s.expires_at > ?
           AND NOT EXISTS (
             SELECT 1
             FROM pi_wallet_challenge_uses AS u
             WHERE u.challenge_hash = c.id_hash
           )
         LIMIT 1`,
      )
      .bind(idHash, sessionHash, subjectHash, now, now)
      .first<SqlRow>();
    return challengeFromRow(row);
  }

  async bindChallenge(
    idHash: string,
    sessionHash: string,
    subjectHash: string,
    proofHash: string,
    consentVersion: string,
    now: number,
  ): Promise<PiBinding | null> {
    try {
      const row = await this.database
        .prepare(
          `INSERT INTO pi_wallet_bindings
             (subject_hash, address, account_id, challenge_hash, proof_hash,
              consent_version, bound_at)
           SELECT c.subject_hash, c.address, c.account_id, c.id_hash, ?, ?, ?
           FROM pi_wallet_challenges AS c
           INNER JOIN pi_sessions AS s
             ON s.token_hash = c.session_hash
           WHERE c.id_hash = ?
             AND c.session_hash = ?
             AND c.subject_hash = ?
             AND c.expires_at > ?
             AND s.revoked_at IS NULL
             AND s.expires_at > ?
             AND NOT EXISTS (
               SELECT 1
               FROM pi_wallet_challenge_uses AS u
               WHERE u.challenge_hash = c.id_hash
             )
           RETURNING subject_hash, address, account_id, consent_version,
                     bound_at`,
        )
        .bind(
          proofHash,
          consentVersion,
          now,
          idHash,
          sessionHash,
          subjectHash,
          now,
          now,
        )
        .first<SqlRow>();
      return bindingFromRow(row);
    } catch (error) {
      if (isConstraintFailure(error)) return null;
      throw error;
    }
  }

  async getBinding(subjectHash: string): Promise<PiBinding | null> {
    const row = await this.database
      .prepare(
        `SELECT subject_hash, address, account_id, bound_at
                , consent_version
         FROM pi_wallet_bindings
         WHERE subject_hash = ?
         LIMIT 1`,
      )
      .bind(subjectHash)
      .first<SqlRow>();
    return bindingFromRow(row);
  }

  async deleteBinding(subjectHash: string, now: number): Promise<void> {
    await this.database.batch([
      this.database
        .prepare(
          `INSERT OR IGNORE INTO pi_wallet_challenge_uses
             (challenge_hash, proof_hash, disposition, consumed_at)
           SELECT id_hash, NULL, 'unlinked', ?
           FROM pi_wallet_challenges
           WHERE subject_hash = ?`,
        )
        .bind(now, subjectHash),
      this.database
        .prepare(
          `DELETE FROM pi_wallet_challenges
           WHERE subject_hash = ?`,
        )
        .bind(subjectHash),
      this.database
        .prepare(
          `DELETE FROM pi_wallet_bindings
           WHERE subject_hash = ?`,
        )
        .bind(subjectHash),
    ]);
  }

  async deleteSubject(subjectHash: string): Promise<void> {
    await this.database.batch([
      this.database
        .prepare(
          `DELETE FROM pi_wallet_challenge_uses
           WHERE challenge_hash IN (
             SELECT id_hash
             FROM pi_wallet_challenges
             WHERE subject_hash = ?
             UNION
             SELECT challenge_hash
             FROM pi_wallet_bindings
             WHERE subject_hash = ?
           )`,
        )
        .bind(subjectHash, subjectHash),
      this.database
        .prepare(
          `DELETE FROM pi_wallet_challenges
           WHERE subject_hash = ?`,
        )
        .bind(subjectHash),
      this.database
        .prepare(
          `DELETE FROM pi_wallet_bindings
           WHERE subject_hash = ?`,
        )
        .bind(subjectHash),
      this.database
        .prepare(
          `DELETE FROM pi_sessions
           WHERE subject_hash = ?`,
        )
        .bind(subjectHash),
    ]);
  }
}
