import type {
  PiBinding,
  PiChallenge,
  PiD1Database,
  PiPepperPin,
  PiRepository,
  PiRetentionPolicy,
  PiRetentionResult,
  PiSession,
  PiSubjectAlias,
} from "./_types";
import { hashOpaque } from "./_crypto";
import {
  PI_BEARER_PENDING_TTL_MS,
  PI_CHALLENGE_RATE_WINDOW_MS,
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
  const pepperVersion = integerField(row, "pepper_version");
  const lastSeenAt = integerField(row, "last_seen_at");
  const expiresAt = integerField(row, "expires_at");
  return tokenHash &&
    subjectHash &&
    username &&
    pepperVersion !== null &&
    lastSeenAt !== null &&
    expiresAt !== null
    ? {
        tokenHash,
        subjectHash,
        username,
        pepperVersion,
        lastSeenAt,
        expiresAt,
      }
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

function returnedRows(result: { readonly results?: readonly unknown[] }): number {
  return result.results?.length ?? 0;
}

function aliasWhereClause(aliasCount: number): string {
  return Array.from(
    { length: aliasCount },
    () => "(pepper_version = ? AND alias_hash = ?)",
  ).join(" OR ");
}

function aliasValues(aliases: readonly PiSubjectAlias[]): unknown[] {
  return aliases.flatMap((alias) => [alias.pepperVersion, alias.aliasHash]);
}

export function piPepperKeysetFingerprint(
  activeVersion: number,
  pins: readonly PiPepperPin[],
): string {
  const canonicalPins = [...pins]
    .sort((left, right) => left.version - right.version)
    .map((pin) => `${pin.version}:${pin.fingerprint}`)
    .join("\u0000");
  return hashOpaque(
    `zerone-pi-pepper-keyset-v1\u0000${activeVersion}\u0000${canonicalPins}`,
  );
}

export class D1PiRepository implements PiRepository {
  constructor(private readonly database: PiD1Database) {}

  async createOAuthFlow(
    stateHash: string,
    browserHash: string,
    createdAt: number,
    expiresAt: number,
  ): Promise<void> {
    const inserted = await this.database
      .prepare(
        `INSERT INTO pi_oauth_flows
           (state_hash, browser_hash, created_at, expires_at, deletion_epoch)
         SELECT ?, ?, ?, ?, current_epoch
         FROM pi_identity_deletion_epoch
         WHERE singleton = 1
         RETURNING state_hash`,
      )
      .bind(stateHash, browserHash, createdAt, expiresAt)
      .first<SqlRow>();
    if (stringField(inserted ?? {}, "state_hash") !== stateHash) {
      throw new Error("Pi deletion epoch is unavailable");
    }
  }

  async consumeOAuthFlow(
    stateHash: string,
    browserHash: string,
    bearerReplayCommitment: string,
    activePepperVersion: number,
    now: number,
  ): Promise<number | null> {
    if (
      bearerReplayCommitment.length !== 43 ||
      !Number.isSafeInteger(activePepperVersion) ||
      activePepperVersion < 1
    ) {
      throw new Error("Bearer replay commitment is out of bounds");
    }
    const replayExists = `(
      EXISTS (
        SELECT 1
        FROM pi_bearer_replay_fingerprints
        WHERE fingerprint_scheme = 'sha256-v1'
          AND fingerprint = ?
      ) OR EXISTS (
        SELECT 1
        FROM pi_bearer_claims
        WHERE fingerprint_scheme = 'sha256-v1'
          AND fingerprint = ?
      )
    )`;
    const legacyEvidenceExists = `(
      EXISTS (
        SELECT 1
        FROM pi_bearer_legacy_state
        WHERE singleton = 1
      ) OR EXISTS (
        SELECT 1
        FROM pi_bearer_replay_fingerprints
        WHERE fingerprint_scheme = 'legacy-keyed-v1'
      ) OR EXISTS (
        SELECT 1
        FROM pi_bearer_claims
        WHERE fingerprint_scheme = 'legacy-keyed-v1'
      )
    )`;
    try {
      const results = await this.database.batch([
        this.database
          .prepare(
          `INSERT INTO pi_bearer_claims
             (fingerprint, state_hash, browser_hash, claimed_at,
              fingerprint_scheme)
           SELECT ?, f.state_hash, f.browser_hash, ?, 'sha256-v1'
           FROM pi_oauth_flows AS f
           WHERE f.state_hash = ?
             AND f.browser_hash = ?
             AND f.consumed_at IS NULL
             AND f.expires_at > ?
             AND EXISTS (
               SELECT 1
               FROM pi_pepper_state
               WHERE singleton = 1
                 AND active_version = ?
             )
             AND NOT ${legacyEvidenceExists}
             AND NOT ${replayExists}
           RETURNING fingerprint`,
        )
          .bind(
            bearerReplayCommitment,
            now,
            stateHash,
            browserHash,
            now,
            activePepperVersion,
            bearerReplayCommitment,
            bearerReplayCommitment,
          ),
      ]);
      if (returnedRows(results[0] ?? {}) !== 1) return null;
      const flow = await this.database
        .prepare(
          `SELECT f.deletion_epoch
           FROM pi_oauth_flows AS f
           INNER JOIN pi_bearer_claims AS c
             ON c.state_hash = f.state_hash
            AND c.browser_hash = f.browser_hash
           WHERE f.state_hash = ?
             AND f.browser_hash = ?
             AND c.fingerprint = ?
             AND c.fingerprint_scheme = 'sha256-v1'
           LIMIT 1`,
        )
        .bind(stateHash, browserHash, bearerReplayCommitment)
        .first<SqlRow>();
      return integerField(flow ?? {}, "deletion_epoch");
    } catch (error) {
      if (isConstraintFailure(error)) return null;
      throw error;
    }
  }

  async promoteBearerClaim(
    bearerReplayCommitment: string,
    stateHash: string,
    browserHash: string,
  ): Promise<boolean> {
    if (bearerReplayCommitment.length !== 43) {
      return false;
    }
    try {
      const results = await this.database.batch([
        this.database
          .prepare(
            `INSERT INTO pi_bearer_replay_fingerprints
               (fingerprint_scheme, fingerprint, first_claimed_at)
             SELECT 'sha256-v1', fingerprint, claimed_at
             FROM pi_bearer_claims
             WHERE fingerprint_scheme = 'sha256-v1'
               AND fingerprint = ?
               AND state_hash = ?
               AND browser_hash = ?
             RETURNING fingerprint`,
          )
          .bind(bearerReplayCommitment, stateHash, browserHash),
        this.database
          .prepare(
            `DELETE FROM pi_bearer_claims
             WHERE fingerprint_scheme = 'sha256-v1'
               AND fingerprint = ?
               AND state_hash = ?
               AND browser_hash = ?
               AND EXISTS (
                 SELECT 1
                 FROM pi_bearer_replay_fingerprints
                 WHERE fingerprint_scheme = 'sha256-v1'
                   AND fingerprint = ?
               )
             RETURNING fingerprint`,
          )
          .bind(
            bearerReplayCommitment,
            stateHash,
            browserHash,
            bearerReplayCommitment,
          ),
      ]);
      return (
        returnedRows(results[0] ?? {}) === 1 &&
        returnedRows(results[1] ?? {}) === 1
      );
    } catch (error) {
      if (isConstraintFailure(error)) return false;
      throw error;
    }
  }

  async rejectBearerClaim(
    bearerReplayCommitment: string,
    stateHash: string,
    browserHash: string,
  ): Promise<void> {
    await this.database
      .prepare(
        `DELETE FROM pi_bearer_claims
         WHERE fingerprint_scheme = 'sha256-v1'
           AND fingerprint = ?
           AND state_hash = ?
           AND browser_hash = ?
           AND NOT EXISTS (
             SELECT 1
             FROM pi_bearer_replay_fingerprints
             WHERE fingerprint_scheme = 'sha256-v1'
               AND fingerprint = ?
           )`,
      )
      .bind(
        bearerReplayCommitment,
        stateHash,
        browserHash,
        bearerReplayCommitment,
      )
      .run();
  }

  async ensurePepperConfiguration(
    activeVersion: number,
    pins: readonly PiPepperPin[],
    now: number,
  ): Promise<boolean> {
    if (
      pins.length < 1 ||
      pins.length > 8 ||
      pins[0]?.version !== activeVersion ||
      !Number.isSafeInteger(activeVersion) ||
      activeVersion < 1 ||
      activeVersion > 2_147_483_647 ||
      pins.some(
        (pin, index) =>
          !Number.isSafeInteger(pin.version) ||
          pin.version < 1 ||
          pin.version > activeVersion ||
          pin.fingerprint.length !== 43 ||
          pins.findIndex((candidate) => candidate.version === pin.version) !==
            index ||
          pins.findIndex(
            (candidate) => candidate.fingerprint === pin.fingerprint,
          ) !== index,
      )
    ) {
      return false;
    }
    const keysetFingerprint = piPepperKeysetFingerprint(activeVersion, pins);
    const pinsAreStored = pins
      .map(
        () => `EXISTS (
          SELECT 1
          FROM pi_pepper_key_versions
          WHERE version = ?
            AND key_fingerprint = ?
        )`,
      )
      .join(" AND ");
    const configuredVersionPlaceholders = pins.map(() => "?").join(", ");
    const legacyBearerEvidenceExists = `(
      EXISTS (
        SELECT 1
        FROM pi_bearer_legacy_state
        WHERE singleton = 1
      ) OR EXISTS (
        SELECT 1
        FROM pi_bearer_replay_fingerprints
        WHERE fingerprint_scheme = 'legacy-keyed-v1'
      ) OR EXISTS (
        SELECT 1
        FROM pi_bearer_claims
        WHERE fingerprint_scheme = 'legacy-keyed-v1'
      )
    )`;
    const unsupportedDurableSubject = `EXISTS (
      SELECT 1
      FROM pi_subject_aliases AS candidate
      WHERE (
        EXISTS (
          SELECT 1 FROM pi_sessions AS s
          WHERE s.subject_hash = candidate.canonical_subject_hash
        ) OR EXISTS (
          SELECT 1 FROM pi_wallet_challenges AS c
          WHERE c.subject_hash = candidate.canonical_subject_hash
        ) OR EXISTS (
          SELECT 1 FROM pi_wallet_bindings AS b
          WHERE b.subject_hash = candidate.canonical_subject_hash
        ) OR EXISTS (
          SELECT 1 FROM pi_wallet_challenge_rate_events AS r
          WHERE r.subject_hash = candidate.canonical_subject_hash
        )
      ) AND NOT EXISTS (
        SELECT 1
        FROM pi_subject_aliases AS configured
        WHERE configured.canonical_subject_hash =
                candidate.canonical_subject_hash
          AND configured.pepper_version IN (${configuredVersionPlaceholders})
      )
    )`;
    const configurationTransitionAllowed = `(
      EXISTS (
        SELECT 1
        FROM pi_pepper_state
        WHERE singleton = 1
          AND active_version = ?
          AND keyset_fingerprint = ?
      ) OR NOT EXISTS (
        SELECT 1
        FROM pi_subject_deletion_guards
        WHERE expires_at > ?
      )
    )`;
    try {
      await this.database.batch([
        ...pins.map((pin) =>
          this.database
            .prepare(
              `INSERT OR IGNORE INTO pi_pepper_key_versions
                 (version, key_fingerprint, first_seen_at)
               SELECT ?, ?, ?
               WHERE NOT EXISTS (
                 SELECT 1
                 FROM pi_pepper_state
                 WHERE singleton = 1
                   AND active_version > ?
               )
                 AND NOT ${legacyBearerEvidenceExists}
                 AND NOT ${unsupportedDurableSubject}
                 AND ${configurationTransitionAllowed}`,
            )
            .bind(
              pin.version,
              pin.fingerprint,
              now,
              activeVersion,
              ...pins.map((candidate) => candidate.version),
              activeVersion,
              keysetFingerprint,
              now,
            ),
        ),
        this.database
          .prepare(
            `INSERT INTO pi_pepper_state
               (singleton, active_version, keyset_fingerprint, updated_at)
             SELECT 1, ?, ?, ?
             WHERE ${pinsAreStored}
               AND NOT ${unsupportedDurableSubject}
               AND NOT ${legacyBearerEvidenceExists}
               AND ${configurationTransitionAllowed}
             ON CONFLICT(singleton) DO UPDATE SET
               active_version = excluded.active_version,
               keyset_fingerprint = excluded.keyset_fingerprint,
               updated_at = excluded.updated_at
             WHERE pi_pepper_state.active_version <= excluded.active_version`,
          )
          .bind(
            activeVersion,
            keysetFingerprint,
            now,
            ...pins.flatMap((pin) => [pin.version, pin.fingerprint]),
            ...pins.map((pin) => pin.version),
            activeVersion,
            keysetFingerprint,
            now,
          ),
      ]);
    } catch (error) {
      if (isConstraintFailure(error)) return false;
      throw error;
    }

    const state = await this.database
      .prepare(
        `SELECT active_version, keyset_fingerprint
         FROM pi_pepper_state
         WHERE singleton = 1
         LIMIT 1`,
      )
      .first<SqlRow>();
    if (
      integerField(state ?? {}, "active_version") !== activeVersion ||
      stringField(state ?? {}, "keyset_fingerprint") !== keysetFingerprint
    ) {
      return false;
    }
    for (const pin of pins) {
      const row = await this.database
        .prepare(
          `SELECT key_fingerprint
           FROM pi_pepper_key_versions
           WHERE version = ?
           LIMIT 1`,
        )
        .bind(pin.version)
        .first<SqlRow>();
      if (stringField(row ?? {}, "key_fingerprint") !== pin.fingerprint) {
        return false;
      }
    }
    const unsupportedSubject = await this.database
      .prepare(
        `SELECT ${unsupportedDurableSubject} AS unsupported_subject`,
      )
      .bind(...pins.map((pin) => pin.version))
      .first<SqlRow>();
    if (integerField(unsupportedSubject ?? {}, "unsupported_subject") !== 0) {
      return false;
    }
    const legacyEvidence = await this.database
      .prepare(
        `SELECT ${legacyBearerEvidenceExists} AS legacy_evidence`,
      )
      .first<SqlRow>();
    if (integerField(legacyEvidence ?? {}, "legacy_evidence") !== 0) {
      return false;
    }
    return true;
  }

  async resolveSubject(
    aliases: readonly PiSubjectAlias[],
  ): Promise<string | null> {
    if (aliases.length < 1 || aliases.length > 8) return null;
    const where = aliasWhereClause(aliases.length);
    const existing = await this.database
      .prepare(
        `SELECT DISTINCT canonical_subject_hash
         FROM pi_subject_aliases
         WHERE ${where}`,
      )
      .bind(...aliasValues(aliases))
      .all<SqlRow>();
    const canonicalSubjects = new Set(
      (existing.results ?? [])
        .map((row) => stringField(row, "canonical_subject_hash"))
        .filter((value): value is string => value !== null),
    );
    if (canonicalSubjects.size > 1) return null;
    const activeAlias = aliases[0];
    if (!activeAlias) return null;
    const canonicalSubject =
      canonicalSubjects.values().next().value ?? activeAlias.aliasHash;
    return canonicalSubject;
  }

  async replaceSubjectSessions(
    session: PiSession,
    aliases: readonly PiSubjectAlias[],
    oauthDeletionEpoch: number,
    createdAt: number,
    expectedKeysetFingerprint: string,
  ): Promise<boolean> {
    if (
      aliases.length < 1 ||
      aliases.length > 8 ||
      !Number.isSafeInteger(oauthDeletionEpoch) ||
      oauthDeletionEpoch < 0 ||
      expectedKeysetFingerprint.length !== 43
    ) {
      return false;
    }
    const expectedSubject = await this.resolveSubject(aliases);
    if (expectedSubject !== session.subjectHash) return false;

    const where = aliasWhereClause(aliases.length);
    const flattenedAliases = aliasValues(aliases);
    const guardHashes = [
      ...new Set([
        session.subjectHash,
        ...aliases.map((alias) => alias.aliasHash),
      ]),
    ];
    const guardPlaceholders = guardHashes.map(() => "?").join(", ");
    const commonCondition = `
      NOT EXISTS (
        SELECT 1
        FROM pi_subject_aliases
        WHERE (${where})
          AND canonical_subject_hash <> ?
      ) AND NOT EXISTS (
        SELECT 1
        FROM pi_subject_deletion_guards
        WHERE subject_hash IN (${guardPlaceholders})
          AND deletion_epoch > ?
          AND expires_at > ?
      ) AND EXISTS (
        SELECT 1
        FROM pi_pepper_state
        WHERE singleton = 1
          AND active_version = ?
          AND keyset_fingerprint = ?
      )`;
    const commonValues = [
      ...flattenedAliases,
      session.subjectHash,
      ...guardHashes,
      oauthDeletionEpoch,
      createdAt,
      session.pepperVersion,
      expectedKeysetFingerprint,
    ];
    try {
      const results = await this.database.batch([
        ...aliases.map((alias) =>
          this.database
            .prepare(
              `INSERT OR IGNORE INTO pi_subject_aliases
                 (pepper_version, alias_hash, canonical_subject_hash,
                  created_at)
               SELECT ?, ?, ?, ?
               WHERE ${commonCondition}`,
            )
            .bind(
              alias.pepperVersion,
              alias.aliasHash,
              session.subjectHash,
              createdAt,
              ...commonValues,
            ),
        ),
        this.database
          .prepare(
            `INSERT INTO pi_sessions
               (token_hash, subject_hash, username, created_at, expires_at,
                pepper_version, last_seen_at)
             SELECT ?, ?, ?, ?, ?, ?, ?
             WHERE ${commonCondition}
               AND (
                 SELECT COUNT(*)
                 FROM pi_subject_aliases
                 WHERE (${where})
                   AND canonical_subject_hash = ?
               ) = ?
             RETURNING token_hash`,
          )
          .bind(
            session.tokenHash,
            session.subjectHash,
            session.username,
            createdAt,
            session.expiresAt,
            session.pepperVersion,
            session.lastSeenAt,
            ...commonValues,
            ...flattenedAliases,
            session.subjectHash,
            aliases.length,
          ),
        this.database
          .prepare(
            `UPDATE pi_sessions
             SET revoked_at = COALESCE(revoked_at, ?)
             WHERE subject_hash = ?
               AND token_hash <> ?
               AND revoked_at IS NULL
               AND EXISTS (
                 SELECT 1
                 FROM pi_sessions
                 WHERE token_hash = ?
                   AND subject_hash = ?
                   AND revoked_at IS NULL
               )`,
          )
          .bind(
            createdAt,
            session.subjectHash,
            session.tokenHash,
            session.tokenHash,
            session.subjectHash,
          ),
      ]);
      return returnedRows(results[aliases.length] ?? {}) === 1;
    } catch (error) {
      if (isConstraintFailure(error)) return false;
      throw error;
    }
  }

  async getSession(
    tokenHash: string,
    now: number,
    idleSince: number,
  ): Promise<PiSession | null> {
    const row = await this.database
      .prepare(
        `UPDATE pi_sessions
         SET last_seen_at = MAX(last_seen_at, ?)
         WHERE token_hash = ?
           AND revoked_at IS NULL
           AND expires_at > ?
           AND last_seen_at > ?
         RETURNING token_hash, subject_hash, username, pepper_version,
                   last_seen_at, expires_at`,
      )
      .bind(now, tokenHash, now, idleSince)
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
    idleSince: number,
  ): Promise<boolean> {
    const row = await this.database
      .prepare(
        `INSERT INTO pi_wallet_challenges
           (id_hash, session_hash, subject_hash, address, account_id, message,
            created_at, expires_at)
         SELECT ?, ?, ?, ?, ?, ?, ?, ?
         WHERE (
           SELECT COUNT(*)
           FROM pi_wallet_challenge_rate_events
           WHERE subject_hash = ?
             AND created_at > ?
         ) < ?
           AND EXISTS (
             SELECT 1
             FROM pi_sessions AS s
             WHERE s.token_hash = ?
               AND s.subject_hash = ?
               AND s.revoked_at IS NULL
               AND s.expires_at > ?
               AND s.last_seen_at > ?
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
        challenge.subjectHash,
        recentSince,
        maximumRecent,
        challenge.sessionHash,
        challenge.subjectHash,
        challenge.createdAt,
        idleSince,
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
    idleSince: number,
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
           AND s.last_seen_at > ?
           AND NOT EXISTS (
             SELECT 1
             FROM pi_wallet_challenge_uses AS u
             WHERE u.challenge_hash = c.id_hash
           )
         LIMIT 1`,
      )
      .bind(idHash, sessionHash, subjectHash, now, now, idleSince)
      .first<SqlRow>();
    return challengeFromRow(row);
  }

  async bindChallenge(
    idHash: string,
    sessionHash: string,
    subjectHash: string,
    proofHash: string,
    consentVersion: string,
    rotatedSession: PiSession,
    idleSince: number,
    now: number,
  ): Promise<PiBinding | null> {
    if (rotatedSession.subjectHash !== subjectHash) return null;
    try {
      const results = await this.database.batch([
        this.database
          .prepare(
            `INSERT INTO pi_sessions
               (token_hash, subject_hash, username, created_at, expires_at,
                pepper_version, last_seen_at)
             SELECT ?, ?, ?, ?, ?, ?, ?
             FROM pi_wallet_challenges AS c
             INNER JOIN pi_sessions AS s
               ON s.token_hash = c.session_hash
             WHERE c.id_hash = ?
               AND c.session_hash = ?
               AND c.subject_hash = ?
               AND c.expires_at > ?
               AND s.revoked_at IS NULL
               AND s.expires_at > ?
               AND s.last_seen_at > ?
               AND EXISTS (
                 SELECT 1
                 FROM pi_pepper_state
                 WHERE singleton = 1
                   AND active_version = ?
               )
               AND NOT EXISTS (
                 SELECT 1
                 FROM pi_wallet_challenge_uses AS u
                 WHERE u.challenge_hash = c.id_hash
               )
             RETURNING token_hash`,
          )
          .bind(
            rotatedSession.tokenHash,
            rotatedSession.subjectHash,
            rotatedSession.username,
            now,
            rotatedSession.expiresAt,
            rotatedSession.pepperVersion,
            rotatedSession.lastSeenAt,
            idHash,
            sessionHash,
            subjectHash,
            now,
            now,
            idleSince,
            rotatedSession.pepperVersion,
          ),
        this.database
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
               AND s.last_seen_at > ?
               AND EXISTS (
                 SELECT 1
                 FROM pi_sessions AS fresh
                 WHERE fresh.token_hash = ?
                   AND fresh.subject_hash = ?
                   AND fresh.pepper_version = ?
                   AND fresh.revoked_at IS NULL
               )
               AND NOT EXISTS (
                 SELECT 1
                 FROM pi_wallet_challenge_uses AS u
                 WHERE u.challenge_hash = c.id_hash
               )
             RETURNING subject_hash`,
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
            idleSince,
            rotatedSession.tokenHash,
            subjectHash,
            rotatedSession.pepperVersion,
          ),
        this.database
          .prepare(
            `UPDATE pi_sessions
             SET revoked_at = COALESCE(revoked_at, ?)
             WHERE subject_hash = ?
               AND token_hash <> ?
               AND revoked_at IS NULL
               AND EXISTS (
                 SELECT 1
                 FROM pi_sessions
                 WHERE token_hash = ?
                   AND subject_hash = ?
                   AND revoked_at IS NULL
               )
               AND EXISTS (
                 SELECT 1
                 FROM pi_wallet_bindings
                 WHERE subject_hash = ?
                   AND challenge_hash = ?
                   AND proof_hash = ?
                   AND consent_version = ?
                   AND bound_at = ?
               )`,
          )
          .bind(
            now,
            subjectHash,
            rotatedSession.tokenHash,
            rotatedSession.tokenHash,
            subjectHash,
            subjectHash,
            idHash,
            proofHash,
            consentVersion,
            now,
          ),
        this.database
          .prepare(
            `DELETE FROM pi_sessions
             WHERE token_hash = ?
               AND subject_hash = ?
               AND NOT EXISTS (
                 SELECT 1
                 FROM pi_wallet_bindings
                 WHERE subject_hash = ?
                   AND challenge_hash = ?
                   AND proof_hash = ?
                   AND consent_version = ?
                   AND bound_at = ?
               )`,
          )
          .bind(
            rotatedSession.tokenHash,
            subjectHash,
            subjectHash,
            idHash,
            proofHash,
            consentVersion,
            now,
          ),
      ]);
      if (
        returnedRows(results[0] ?? {}) !== 1 ||
        returnedRows(results[1] ?? {}) !== 1
      ) {
        return null;
      }
      return this.getBinding(subjectHash);
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

  async deleteBinding(
    subjectHash: string,
    currentSessionHash: string,
    rotatedSession: PiSession,
    idleSince: number,
    now: number,
  ): Promise<boolean> {
    if (rotatedSession.subjectHash !== subjectHash) return false;
    const results = await this.database.batch([
      this.database
        .prepare(
          `INSERT INTO pi_sessions
             (token_hash, subject_hash, username, created_at, expires_at,
              pepper_version, last_seen_at)
           SELECT ?, ?, ?, ?, ?, ?, ?
           FROM pi_sessions AS current
           WHERE current.token_hash = ?
             AND current.subject_hash = ?
             AND current.revoked_at IS NULL
             AND current.expires_at > ?
             AND current.last_seen_at > ?
             AND EXISTS (
               SELECT 1
               FROM pi_wallet_bindings
               WHERE subject_hash = ?
             )
             AND EXISTS (
               SELECT 1
               FROM pi_pepper_state
               WHERE singleton = 1
                 AND active_version = ?
             )
           RETURNING token_hash`,
        )
        .bind(
          rotatedSession.tokenHash,
          rotatedSession.subjectHash,
          rotatedSession.username,
          now,
          rotatedSession.expiresAt,
          rotatedSession.pepperVersion,
          rotatedSession.lastSeenAt,
          currentSessionHash,
          subjectHash,
          now,
          idleSince,
          subjectHash,
          rotatedSession.pepperVersion,
        ),
      this.database
        .prepare(
          `DELETE FROM pi_wallet_challenges
           WHERE subject_hash = ?
             AND EXISTS (
               SELECT 1
               FROM pi_sessions
               WHERE token_hash = ?
                 AND subject_hash = ?
                 AND revoked_at IS NULL
             )
           RETURNING subject_hash`,
        )
        .bind(
          subjectHash,
          rotatedSession.tokenHash,
          subjectHash,
        ),
      this.database
        .prepare(
          `DELETE FROM pi_wallet_bindings
           WHERE subject_hash = ?
             AND EXISTS (
               SELECT 1
               FROM pi_sessions
               WHERE token_hash = ?
                 AND subject_hash = ?
                 AND revoked_at IS NULL
             )
           RETURNING subject_hash`,
        )
        .bind(
          subjectHash,
          rotatedSession.tokenHash,
          subjectHash,
        ),
      this.database
        .prepare(
          `UPDATE pi_sessions
           SET revoked_at = COALESCE(revoked_at, ?)
           WHERE subject_hash = ?
             AND token_hash <> ?
             AND revoked_at IS NULL
             AND EXISTS (
               SELECT 1
               FROM pi_sessions
               WHERE token_hash = ?
                 AND subject_hash = ?
                 AND revoked_at IS NULL
             )`,
        )
        .bind(
          now,
          subjectHash,
          rotatedSession.tokenHash,
          rotatedSession.tokenHash,
          subjectHash,
        ),
    ]);
    return (
      returnedRows(results[0] ?? {}) === 1 &&
      returnedRows(results[2] ?? {}) === 1
    );
  }

  async deleteSubject(
    subjectHash: string,
    now: number,
    guardExpiresAt: number,
    expectedActivePepperVersion: number,
    expectedKeysetFingerprint: string,
  ): Promise<boolean> {
    if (
      !Number.isSafeInteger(now) ||
      !Number.isSafeInteger(guardExpiresAt) ||
      guardExpiresAt <= now ||
      !Number.isSafeInteger(expectedActivePepperVersion) ||
      expectedActivePepperVersion < 1 ||
      expectedKeysetFingerprint.length !== 43
    ) {
      return false;
    }
    const deletionIsCurrent = `EXISTS (
      SELECT 1
      FROM pi_subject_deletion_guards
      WHERE subject_hash = ?
        AND deletion_epoch = (
          SELECT current_epoch
          FROM pi_identity_deletion_epoch
          WHERE singleton = 1
        )
    )`;
    try {
      const results = await this.database.batch([
        this.database
          .prepare(
            `UPDATE pi_identity_deletion_epoch
             SET current_epoch = current_epoch + 1
             WHERE singleton = 1
               AND EXISTS (
                 SELECT 1
                 FROM pi_pepper_state
                 WHERE singleton = 1
                   AND active_version = ?
                   AND keyset_fingerprint = ?
               )
             RETURNING current_epoch`,
          )
          .bind(expectedActivePepperVersion, expectedKeysetFingerprint),
        this.database
          .prepare(
            `INSERT INTO pi_subject_deletion_guards
               (subject_hash, deleted_at, expires_at, deletion_epoch)
             SELECT ?, ?, ?, current_epoch
             FROM pi_identity_deletion_epoch
             WHERE singleton = 1
               AND EXISTS (
               SELECT 1
               FROM pi_pepper_state
               WHERE singleton = 1
                 AND active_version = ?
                 AND keyset_fingerprint = ?
             )
             ON CONFLICT(subject_hash) DO UPDATE SET
               deleted_at = MAX(
                 pi_subject_deletion_guards.deleted_at,
                 excluded.deleted_at
               ),
               expires_at = MAX(
                 pi_subject_deletion_guards.expires_at,
                 excluded.expires_at
               ),
               deletion_epoch = excluded.deletion_epoch
             WHERE pi_subject_deletion_guards.deletion_epoch <
                   excluded.deletion_epoch
             RETURNING subject_hash`,
          )
          .bind(
            subjectHash,
            now,
            guardExpiresAt,
            expectedActivePepperVersion,
            expectedKeysetFingerprint,
          ),
        this.database
          .prepare(
            `INSERT INTO pi_subject_deletion_guards
               (subject_hash, deleted_at, expires_at, deletion_epoch)
             SELECT a.alias_hash, ?, ?, e.current_epoch
             FROM pi_subject_aliases AS a
             CROSS JOIN pi_identity_deletion_epoch AS e
             WHERE e.singleton = 1
               AND a.canonical_subject_hash = ?
               AND ${deletionIsCurrent}
             ON CONFLICT(subject_hash) DO UPDATE SET
               deleted_at = MAX(
                 pi_subject_deletion_guards.deleted_at,
                 excluded.deleted_at
               ),
               expires_at = MAX(
                 pi_subject_deletion_guards.expires_at,
                 excluded.expires_at
               ),
               deletion_epoch = excluded.deletion_epoch
             WHERE pi_subject_deletion_guards.deletion_epoch <
                   excluded.deletion_epoch`,
          )
          .bind(
            now,
            guardExpiresAt,
            subjectHash,
            subjectHash,
          ),
        this.database
          .prepare(
            `DELETE FROM pi_wallet_challenge_rate_events
             WHERE subject_hash = ?
               AND ${deletionIsCurrent}`,
          )
          .bind(
            subjectHash,
            subjectHash,
          ),
        this.database
          .prepare(
            `DELETE FROM pi_wallet_challenge_uses
             WHERE (
               challenge_hash IN (
                 SELECT id_hash
                 FROM pi_wallet_challenges
                 WHERE subject_hash = ?
               ) OR EXISTS (
                 SELECT 1
                 FROM pi_wallet_bindings AS b
                 WHERE b.subject_hash = ?
                   AND b.challenge_hash = pi_wallet_challenge_uses.challenge_hash
                   AND b.proof_hash = pi_wallet_challenge_uses.proof_hash
               )
             ) AND ${deletionIsCurrent}`,
          )
          .bind(
            subjectHash,
            subjectHash,
            subjectHash,
          ),
        this.database
          .prepare(
            `DELETE FROM pi_wallet_challenges
             WHERE subject_hash = ?
               AND ${deletionIsCurrent}`,
          )
          .bind(
            subjectHash,
            subjectHash,
          ),
        this.database
          .prepare(
            `DELETE FROM pi_wallet_bindings
             WHERE subject_hash = ?
               AND ${deletionIsCurrent}`,
          )
          .bind(
            subjectHash,
            subjectHash,
          ),
        this.database
          .prepare(
            `DELETE FROM pi_sessions
             WHERE subject_hash = ?
               AND ${deletionIsCurrent}`,
          )
          .bind(
            subjectHash,
            subjectHash,
          ),
        this.database
          .prepare(
            `DELETE FROM pi_subject_aliases
             WHERE canonical_subject_hash = ?
               AND ${deletionIsCurrent}`,
          )
          .bind(
            subjectHash,
            subjectHash,
          ),
      ]);
      return (
        returnedRows(results[0] ?? {}) === 1 &&
        returnedRows(results[1] ?? {}) === 1
      );
    } catch (error) {
      if (isConstraintFailure(error)) return false;
      throw error;
    }
  }

  async cleanupExpired(
    policy: PiRetentionPolicy,
  ): Promise<PiRetentionResult> {
    if (
      !Number.isSafeInteger(policy.now) ||
      !Number.isSafeInteger(policy.pendingBefore) ||
      !Number.isSafeInteger(policy.idleBefore) ||
      !Number.isSafeInteger(policy.retainedBefore) ||
      !Number.isSafeInteger(policy.maximumRowsPerTable) ||
      policy.idleBefore > policy.now ||
      policy.retainedBefore > policy.now ||
      policy.pendingBefore > policy.now - PI_BEARER_PENDING_TTL_MS ||
      policy.maximumRowsPerTable < 1 ||
      policy.maximumRowsPerTable > 1_000
    ) {
      throw new Error("Pi retention policy is out of bounds");
    }
    const limit = policy.maximumRowsPerTable;
    const challengeRateBefore = Math.min(
      policy.retainedBefore,
      policy.now - PI_CHALLENGE_RATE_WINDOW_MS,
    );
    const results = await this.database.batch([
      this.database
        .prepare(
          `DELETE FROM pi_bearer_claims
           WHERE rowid IN (
             SELECT c.rowid
             FROM pi_bearer_claims AS c
             WHERE c.fingerprint_scheme = 'sha256-v1'
               AND c.claimed_at <= ?
             ORDER BY c.claimed_at
             LIMIT ?
           )
           RETURNING rowid`,
        )
        .bind(policy.pendingBefore, limit),
      this.database
        .prepare(
          `DELETE FROM pi_oauth_flows
           WHERE rowid IN (
             SELECT rowid
             FROM pi_oauth_flows
             WHERE expires_at <= ?
             ORDER BY expires_at
             LIMIT ?
           )
           RETURNING rowid`,
        )
        .bind(policy.retainedBefore, limit),
      this.database
        .prepare(
          `DELETE FROM pi_wallet_challenges
           WHERE rowid IN (
             SELECT rowid
             FROM pi_wallet_challenges
             WHERE expires_at <= ?
             ORDER BY expires_at
             LIMIT ?
           )
           RETURNING rowid`,
        )
        .bind(policy.retainedBefore, limit),
      this.database
        .prepare(
          `DELETE FROM pi_wallet_challenge_rate_events
           WHERE rowid IN (
             SELECT rowid
             FROM pi_wallet_challenge_rate_events
             WHERE created_at <= ?
             ORDER BY created_at
             LIMIT ?
           )
           RETURNING rowid`,
        )
        .bind(challengeRateBefore, limit),
      this.database
        .prepare(
          `DELETE FROM pi_sessions
           WHERE rowid IN (
             SELECT rowid
             FROM pi_sessions
             WHERE expires_at <= ?
                OR (revoked_at IS NOT NULL AND revoked_at <= ?)
                OR (last_seen_at <= ? AND last_seen_at <= ?)
             ORDER BY created_at
             LIMIT ?
           )
           RETURNING rowid`,
        )
        .bind(
          policy.retainedBefore,
          policy.retainedBefore,
          policy.idleBefore,
          policy.retainedBefore,
          limit,
        ),
      this.database
        .prepare(
          `DELETE FROM pi_wallet_challenge_uses
           WHERE rowid IN (
             SELECT u.rowid
             FROM pi_wallet_challenge_uses AS u
             INNER JOIN pi_wallet_bindings AS b
               ON b.challenge_hash = u.challenge_hash
              AND b.proof_hash = u.proof_hash
             WHERE u.disposition = 'bound'
               AND b.bound_at <= ?
               AND NOT EXISTS (
                 SELECT 1 FROM pi_sessions AS s
                 WHERE s.subject_hash = b.subject_hash
                   AND s.last_seen_at > ?
               )
               AND NOT EXISTS (
                 SELECT 1 FROM pi_wallet_challenges AS c
                 WHERE c.subject_hash = b.subject_hash
                   AND (c.created_at > ? OR c.expires_at > ?)
               )
             ORDER BY b.bound_at
             LIMIT ?
           )
           RETURNING rowid`,
        )
        .bind(
          policy.retainedBefore,
          policy.retainedBefore,
          policy.retainedBefore,
          policy.now,
          limit,
        ),
      this.database
        .prepare(
          `DELETE FROM pi_wallet_bindings
           WHERE rowid IN (
             SELECT b.rowid
             FROM pi_wallet_bindings AS b
             WHERE b.bound_at <= ?
               AND NOT EXISTS (
                 SELECT 1 FROM pi_sessions AS s
                 WHERE s.subject_hash = b.subject_hash
                   AND s.last_seen_at > ?
               )
               AND NOT EXISTS (
                 SELECT 1 FROM pi_wallet_challenges AS c
                 WHERE c.subject_hash = b.subject_hash
                   AND (c.created_at > ? OR c.expires_at > ?)
               )
               AND NOT EXISTS (
                 SELECT 1 FROM pi_wallet_challenge_uses AS u
                 WHERE u.challenge_hash = b.challenge_hash
               )
             ORDER BY b.bound_at
             LIMIT ?
           )
           RETURNING rowid`,
        )
        .bind(
          policy.retainedBefore,
          policy.retainedBefore,
          policy.retainedBefore,
          policy.now,
          limit,
        ),
      this.database
        .prepare(
          `DELETE FROM pi_wallet_challenge_uses
           WHERE rowid IN (
             SELECT u.rowid
             FROM pi_wallet_challenge_uses AS u
             WHERE u.consumed_at <= ?
               AND NOT EXISTS (
                 SELECT 1
                 FROM pi_wallet_bindings AS b
                 WHERE b.challenge_hash = u.challenge_hash
                   AND b.proof_hash = u.proof_hash
               )
             ORDER BY u.consumed_at
             LIMIT ?
           )
           RETURNING rowid`,
        )
        .bind(policy.retainedBefore, limit),
      this.database
        .prepare(
          `DELETE FROM pi_subject_aliases
           WHERE rowid IN (
             SELECT a.rowid
             FROM pi_subject_aliases AS a
             WHERE a.created_at <= ?
               AND NOT EXISTS (
                 SELECT 1 FROM pi_sessions AS s
                 WHERE s.subject_hash = a.canonical_subject_hash
               )
               AND NOT EXISTS (
                 SELECT 1 FROM pi_wallet_challenges AS c
                 WHERE c.subject_hash = a.canonical_subject_hash
               )
               AND NOT EXISTS (
                 SELECT 1 FROM pi_wallet_bindings AS b
                 WHERE b.subject_hash = a.canonical_subject_hash
               )
               AND NOT EXISTS (
                 SELECT 1 FROM pi_wallet_challenge_rate_events AS r
                 WHERE r.subject_hash = a.canonical_subject_hash
               )
             ORDER BY a.created_at
             LIMIT ?
           )
           RETURNING rowid`,
        )
        .bind(policy.retainedBefore, limit),
      this.database
        .prepare(
          `DELETE FROM pi_subject_deletion_guards
           WHERE rowid IN (
             SELECT rowid
             FROM pi_subject_deletion_guards
             WHERE expires_at <= ?
             ORDER BY expires_at
             LIMIT ?
           )
           RETURNING rowid`,
        )
        .bind(policy.now, limit),
    ]);
    return {
      bearerPendingClaimsDeleted: returnedRows(results[0] ?? {}),
      oauthFlowsDeleted: returnedRows(results[1] ?? {}),
      challengesDeleted: returnedRows(results[2] ?? {}),
      challengeRateEventsDeleted: returnedRows(results[3] ?? {}),
      sessionsDeleted: returnedRows(results[4] ?? {}),
      challengeUsesDeleted:
        returnedRows(results[5] ?? {}) + returnedRows(results[7] ?? {}),
      bindingsDeleted: returnedRows(results[6] ?? {}),
      subjectAliasesDeleted: returnedRows(results[8] ?? {}),
      deletionGuardsDeleted: returnedRows(results[9] ?? {}),
    };
  }

}
