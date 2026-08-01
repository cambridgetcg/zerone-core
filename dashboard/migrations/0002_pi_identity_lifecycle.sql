PRAGMA foreign_keys = ON;

ALTER TABLE pi_sessions
  ADD COLUMN pepper_version INTEGER NOT NULL DEFAULT 1
  CHECK (pepper_version BETWEEN 1 AND 2147483647);

ALTER TABLE pi_sessions
  ADD COLUMN last_seen_at INTEGER NOT NULL DEFAULT 0;

UPDATE pi_sessions
SET last_seen_at = created_at
WHERE last_seen_at = 0;

CREATE INDEX pi_sessions_subject_active
  ON pi_sessions (subject_hash, revoked_at, expires_at, last_seen_at);

CREATE TABLE pi_pepper_key_versions (
  version INTEGER PRIMARY KEY NOT NULL
    CHECK (version BETWEEN 1 AND 2147483647),
  key_fingerprint TEXT NOT NULL UNIQUE
    CHECK (length(key_fingerprint) = 43),
  first_seen_at INTEGER NOT NULL
) STRICT;

CREATE TABLE pi_pepper_state (
  singleton INTEGER PRIMARY KEY NOT NULL CHECK (singleton = 1),
  active_version INTEGER NOT NULL
    CHECK (active_version BETWEEN 1 AND 2147483647),
  keyset_fingerprint TEXT NOT NULL CHECK (length(keyset_fingerprint) = 43),
  updated_at INTEGER NOT NULL
) STRICT;

CREATE TABLE pi_identity_deletion_epoch (
  singleton INTEGER PRIMARY KEY NOT NULL CHECK (singleton = 1),
  current_epoch INTEGER NOT NULL CHECK (current_epoch >= 0)
) STRICT;

INSERT INTO pi_identity_deletion_epoch (singleton, current_epoch)
VALUES (1, 0);

ALTER TABLE pi_oauth_flows
  ADD COLUMN deletion_epoch INTEGER NOT NULL DEFAULT 0
  CHECK (deletion_epoch >= 0);

CREATE TABLE pi_subject_aliases (
  pepper_version INTEGER NOT NULL
    CHECK (pepper_version BETWEEN 1 AND 2147483647),
  alias_hash TEXT NOT NULL CHECK (length(alias_hash) = 43),
  canonical_subject_hash TEXT NOT NULL
    CHECK (length(canonical_subject_hash) = 43),
  created_at INTEGER NOT NULL,
  PRIMARY KEY (pepper_version, alias_hash),
  UNIQUE (canonical_subject_hash, pepper_version)
) STRICT;

CREATE INDEX pi_subject_aliases_canonical
  ON pi_subject_aliases (canonical_subject_hash);

CREATE TABLE pi_subject_deletion_guards (
  subject_hash TEXT PRIMARY KEY NOT NULL CHECK (length(subject_hash) = 43),
  deleted_at INTEGER NOT NULL,
  expires_at INTEGER NOT NULL CHECK (expires_at > deleted_at),
  deletion_epoch INTEGER NOT NULL CHECK (deletion_epoch > 0),
  operation_hash TEXT NOT NULL CHECK (length(operation_hash) = 43)
) STRICT;

CREATE INDEX pi_subject_deletion_guards_expiry
  ON pi_subject_deletion_guards (expires_at);

CREATE INDEX pi_wallet_bindings_retention
  ON pi_wallet_bindings (bound_at, subject_hash);

CREATE TABLE pi_wallet_challenge_rate_events (
  subject_hash TEXT NOT NULL CHECK (length(subject_hash) = 43),
  challenge_hash TEXT NOT NULL UNIQUE CHECK (length(challenge_hash) = 43),
  created_at INTEGER NOT NULL,
  PRIMARY KEY (subject_hash, challenge_hash)
) STRICT;

CREATE INDEX pi_wallet_challenge_rate_events_window
  ON pi_wallet_challenge_rate_events (subject_hash, created_at);

INSERT OR IGNORE INTO pi_wallet_challenge_rate_events
  (subject_hash, challenge_hash, created_at)
SELECT subject_hash, id_hash, created_at
FROM pi_wallet_challenges;

CREATE TRIGGER pi_wallet_challenges_record_rate_event
AFTER INSERT ON pi_wallet_challenges
BEGIN
  INSERT INTO pi_wallet_challenge_rate_events
    (subject_hash, challenge_hash, created_at)
  VALUES (NEW.subject_hash, NEW.id_hash, NEW.created_at);
END;

CREATE TRIGGER pi_sessions_enforce_pepper_floor
BEFORE INSERT ON pi_sessions
WHEN EXISTS (
  SELECT 1
  FROM pi_pepper_state
  WHERE singleton = 1
    AND active_version <> NEW.pepper_version
)
BEGIN
  SELECT RAISE(ABORT, 'pi session pepper floor mismatch');
END;

CREATE TRIGGER pi_sessions_backfill_legacy_last_seen
AFTER INSERT ON pi_sessions
WHEN NEW.last_seen_at = 0
BEGIN
  UPDATE pi_sessions
  SET last_seen_at = NEW.created_at
  WHERE token_hash = NEW.token_hash;
END;

CREATE TRIGGER pi_sessions_alias_legacy_v1
AFTER INSERT ON pi_sessions
WHEN NEW.pepper_version = 1
BEGIN
  INSERT OR IGNORE INTO pi_subject_aliases
    (pepper_version, alias_hash, canonical_subject_hash, created_at)
  VALUES (1, NEW.subject_hash, NEW.subject_hash, NEW.created_at);
END;

INSERT OR IGNORE INTO pi_subject_aliases
  (pepper_version, alias_hash, canonical_subject_hash, created_at)
SELECT 1, subject_hash, subject_hash, MIN(created_at)
FROM pi_sessions
GROUP BY subject_hash;

INSERT OR IGNORE INTO pi_subject_aliases
  (pepper_version, alias_hash, canonical_subject_hash, created_at)
SELECT 1, subject_hash, subject_hash, MIN(created_at)
FROM pi_wallet_challenges
GROUP BY subject_hash;

INSERT OR IGNORE INTO pi_subject_aliases
  (pepper_version, alias_hash, canonical_subject_hash, created_at)
SELECT 1, subject_hash, subject_hash, MIN(bound_at)
FROM pi_wallet_bindings
GROUP BY subject_hash;

CREATE TABLE pi_bearer_replay_fingerprints (
  fingerprint_scheme TEXT NOT NULL
    CHECK (fingerprint_scheme IN ('legacy-keyed-v1', 'sha256-v1')),
  fingerprint TEXT NOT NULL CHECK (length(fingerprint) = 43),
  first_claimed_at INTEGER NOT NULL,
  PRIMARY KEY (fingerprint_scheme, fingerprint)
) STRICT;

CREATE INDEX pi_bearer_replay_fingerprints_retention
  ON pi_bearer_replay_fingerprints
    (fingerprint_scheme, first_claimed_at);

CREATE INDEX pi_wallet_challenge_uses_retention
  ON pi_wallet_challenge_uses (consumed_at);

ALTER TABLE pi_bearer_claims
  ADD COLUMN fingerprint_scheme TEXT NOT NULL DEFAULT 'legacy-keyed-v1'
  CHECK (fingerprint_scheme IN ('legacy-keyed-v1', 'sha256-v1'));

CREATE TABLE pi_bearer_legacy_state (
  singleton INTEGER PRIMARY KEY NOT NULL CHECK (singleton = 1),
  detected_at INTEGER NOT NULL
) STRICT;

INSERT INTO pi_bearer_legacy_state (singleton, detected_at)
SELECT 1, MIN(claimed_at)
FROM pi_bearer_claims
HAVING COUNT(*) > 0;

CREATE INDEX pi_bearer_claims_retention
  ON pi_bearer_claims
    (fingerprint_scheme, claimed_at);

CREATE TRIGGER pi_bearer_claims_reject_new_legacy
BEFORE INSERT ON pi_bearer_claims
WHEN NEW.fingerprint_scheme = 'legacy-keyed-v1'
BEGIN
  SELECT RAISE(ABORT, 'legacy Pi bearer claim scheme is closed');
END;
