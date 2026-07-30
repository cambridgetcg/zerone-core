PRAGMA foreign_keys = ON;

CREATE TABLE pi_oauth_flows (
  state_hash TEXT PRIMARY KEY NOT NULL CHECK (length(state_hash) = 43),
  browser_hash TEXT NOT NULL UNIQUE CHECK (length(browser_hash) = 43),
  created_at INTEGER NOT NULL,
  expires_at INTEGER NOT NULL,
  consumed_at INTEGER
) STRICT;

CREATE INDEX pi_oauth_flows_expiry
  ON pi_oauth_flows (expires_at);

CREATE TABLE pi_bearer_claims (
  fingerprint TEXT PRIMARY KEY NOT NULL CHECK (length(fingerprint) = 43),
  state_hash TEXT NOT NULL UNIQUE CHECK (length(state_hash) = 43),
  browser_hash TEXT NOT NULL UNIQUE CHECK (length(browser_hash) = 43),
  claimed_at INTEGER NOT NULL
) STRICT;

CREATE TRIGGER pi_bearer_claims_consume_oauth_flow
AFTER INSERT ON pi_bearer_claims
BEGIN
  UPDATE pi_oauth_flows
  SET consumed_at = NEW.claimed_at
  WHERE state_hash = NEW.state_hash
    AND browser_hash = NEW.browser_hash
    AND consumed_at IS NULL;
END;

CREATE TABLE pi_sessions (
  token_hash TEXT PRIMARY KEY NOT NULL CHECK (length(token_hash) = 43),
  subject_hash TEXT NOT NULL CHECK (length(subject_hash) = 43),
  username TEXT NOT NULL CHECK (length(username) BETWEEN 1 AND 64),
  created_at INTEGER NOT NULL,
  expires_at INTEGER NOT NULL,
  revoked_at INTEGER
) STRICT;

CREATE INDEX pi_sessions_subject
  ON pi_sessions (subject_hash);

CREATE INDEX pi_sessions_expiry
  ON pi_sessions (expires_at);

CREATE TABLE pi_wallet_challenges (
  id_hash TEXT PRIMARY KEY NOT NULL CHECK (length(id_hash) = 43),
  session_hash TEXT NOT NULL CHECK (length(session_hash) = 43),
  subject_hash TEXT NOT NULL CHECK (length(subject_hash) = 43),
  address TEXT NOT NULL CHECK (length(address) = 42),
  account_id TEXT NOT NULL CHECK (length(account_id) = 58),
  message TEXT NOT NULL CHECK (length(message) BETWEEN 1 AND 2048),
  created_at INTEGER NOT NULL,
  expires_at INTEGER NOT NULL
) STRICT;

CREATE INDEX pi_wallet_challenges_session_created
  ON pi_wallet_challenges (session_hash, created_at);

CREATE INDEX pi_wallet_challenges_expiry
  ON pi_wallet_challenges (expires_at);

CREATE TABLE pi_wallet_challenge_uses (
  challenge_hash TEXT PRIMARY KEY NOT NULL CHECK (length(challenge_hash) = 43),
  proof_hash TEXT UNIQUE CHECK (
    proof_hash IS NULL OR length(proof_hash) = 43
  ),
  disposition TEXT NOT NULL CHECK (
    disposition IN ('bound', 'superseded', 'unlinked')
  ),
  consumed_at INTEGER NOT NULL
) STRICT;

CREATE TABLE pi_wallet_bindings (
  subject_hash TEXT PRIMARY KEY NOT NULL CHECK (length(subject_hash) = 43),
  address TEXT NOT NULL UNIQUE CHECK (length(address) = 42),
  account_id TEXT NOT NULL CHECK (length(account_id) = 58),
  challenge_hash TEXT NOT NULL UNIQUE CHECK (length(challenge_hash) = 43),
  proof_hash TEXT NOT NULL UNIQUE CHECK (length(proof_hash) = 43),
  consent_version TEXT NOT NULL CHECK (consent_version = 'pi-wallet-link-v1'),
  bound_at INTEGER NOT NULL
) STRICT;

CREATE TRIGGER pi_wallet_bindings_consume_challenge
AFTER INSERT ON pi_wallet_bindings
BEGIN
  INSERT INTO pi_wallet_challenge_uses
    (challenge_hash, proof_hash, disposition, consumed_at)
  VALUES
    (NEW.challenge_hash, NEW.proof_hash, 'bound', NEW.bound_at);

  INSERT OR IGNORE INTO pi_wallet_challenge_uses
    (challenge_hash, proof_hash, disposition, consumed_at)
  SELECT id_hash, NULL, 'superseded', NEW.bound_at
  FROM pi_wallet_challenges
  WHERE subject_hash = NEW.subject_hash
    AND id_hash <> NEW.challenge_hash;

  DELETE FROM pi_wallet_challenges
  WHERE subject_hash = NEW.subject_hash;
END;
