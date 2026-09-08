#!/usr/bin/env python3
"""Local-only synthetic signed gate fixtures. No nodes or production identities.

Temporary OpenPGP RSA keys are generated in memory with test-only cryptography;
real gpgv independently verifies every signature. No gpg-agent, ambient keyring,
or key files are used. MATCH is synthetic only; fixture release documents do
NOT pass the separate release ceremony.
"""
import copy
import datetime as dt
import importlib.util
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest

sys.dont_write_bytecode = True
HERE = Path(__file__).resolve().parent
SPEC = importlib.util.spec_from_file_location("seed_gate", HERE / "verify-seed-activation.py")
v = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(v)


def hid(label):
    return v.digest(label.encode())


def stamp(value):
    return value.isoformat(timespec="milliseconds").replace("+00:00", "Z")


def raw(value):
    return v.canonical(value) + b"\n"


class Fixture:
    def __init__(self, root, keys, sign, now, runtime=None):
        self.root, self.roles, self.sign, self.now = root, keys, sign, now
        self.runtime = runtime
        self.runtime_candidate = None
        self.bundle = root / "bundle"
        self.artifacts = root / "artifacts"
        self.bundle.mkdir()
        (self.bundle / "evidence").mkdir()
        self.artifacts.mkdir()
        self.before = stamp(now - dt.timedelta(minutes=1))
        self.until = stamp(now + dt.timedelta(minutes=10))
        self.when = stamp(now)
        self.chain = "cosmos:seed-synthetic-gate"
        self.claimant = v._bech32_encode("zrn", bytes([1]) * 20)
        self.sponsor = v._bech32_encode("zrn", bytes([2]) * 20)
        self.gov = v._bech32_encode("zrn", __import__("hashlib").sha256(b"gov").digest()[:20])
        self.artifact_hashes = {}
        for key, name in v.ARTIFACTS.items():
            data = v.canonical({"schema": "synthetic-source-manifest", "files": {"test": hid("source")}}) if key == "source_digest" else b"synthetic test bytes: " + name.encode()
            (self.artifacts / name).write_bytes(data)
            self.artifact_hashes[key] = v.digest(data)
        self.artifact_hashes["verifier_sha256"] = v.file_digest(HERE / "verify-seed-activation.py")
        self.profile = {"protocol": "agent-wallet-zerone.seed-profile/0.1", "chain_reference": "seed-synthetic-gate", "chain_id": self.chain, "native_asset_id": self.chain + "/denom:uzrn", "claiming_pot_account": self.chain + ":" + v._bech32_encode("zrn", __import__("hashlib").sha256(b"claiming_pot").digest()[:20]), "genesis_hash": hid("synthetic-genesis"), "source_digest": self.artifact_hashes["source_digest"], "zerone_core_commit": "1" * 40, "cosmos_sdk_version": "v0.53.8", "runtime_sha256": self.artifact_hashes["runtime_sha256"], "helper_sha256": self.artifact_hashes["helper_sha256"], "native_denom": "uzrn", "bech32_prefix": "zrn", "seed_amount_uzrn": "222000", "claim_type_url": v.CLAIM, "claim_gas_floor": "22222", "tx_gas_cap": "11111111", "min_gas_price_uzrn": "1", "confirmation_depth": 1}
        self.profile["profile_id"] = v.semantic_id(self.profile, "profile_id")
        self.policy = {"protocol": "agent-wallet-zerone.seed-policy/0.1", "profile_id": self.profile["profile_id"], "node_trust_id": hid("synthetic-node-trust"), "claimant_account": self.chain + ":" + self.claimant, "sponsor_account": self.chain + ":" + self.sponsor, "pot_id": "bootstrap-" + self.claimant, "max_intents": 1, "seed_amount_uzrn": "222000", "max_fee_uzrn": "50000", "max_gas": "50000", "grant_spend_limit_uzrn": "200000", "grant_expires_at": self.until, "setup_fee_budget_uzrn": "200000", "not_before": self.before, "expires_at": self.until, "timeout_height": "1000", "max_observation_age_seconds": 300, "max_height_lag": 5}
        self.policy["policy_hash"] = v.semantic_id(self.policy, "policy_hash")
        if runtime:
            self.profile, self.policy = runtime["config"]["profile"], runtime["config"]["policy"]
            self.chain = self.profile["chain_id"]
            self.claimant, self.sponsor = (self.policy[k].split(":")[-1] for k in ("claimant_account", "sponsor_account"))
            self.before, self.until = self.policy["not_before"], self.policy["grant_expires_at"]
            self.when = runtime["observation"]["evidence"]["anchor"]["block_time"]
            for key, name in v.ARTIFACTS.items():
                shutil.copyfile(runtime["artifacts"][key], self.artifacts / name)
                self.artifact_hashes[key] = v.file_digest(self.artifacts / name)
        self.params = {"max_pots_active": 10, "min_claim_amount": "1000", "bootstrap_registrar": "", "bootstrap_emission_cap_uzrn": "222000", "bootstrap_daily_admission_cap": "1"}
        accepted = {"claims_live_at_genesis": False, "automatic_issuance_live_at_genesis": False, "knowledge_admission_rewards_live_at_genesis": False, "one_operator_validator_bft_f": 0}
        self.unrelated = v.digest(v.canonical(accepted))
        release = {"schema": "zerone-2-release-packet-v2", "chain_id": self.profile["chain_reference"], "signature_authority": self.authority("RELEASE-PACKET.json"), "genesis": {"sha256": self.profile["genesis_hash"][7:]}, "source": {"commit": self.profile["zerone_core_commit"]}, "components": {"zerone_2_runtime": {"binary_sha256": self.profile["runtime_sha256"][7:]}}, "accepted_policy": accepted}
        self.put_signed("RELEASE-PACKET.json", release, "release")
        opened = {"schema": "zerone-2-open-beta-decision-v1", "decision": "GO", "signature_authority": self.authority("OPEN-BETA-DECISION.json"), "release_packet": {"sha256": self.hash("RELEASE-PACKET.json")[7:], "detached_signature_sha256": self.hash("RELEASE-PACKET.json.sig")[7:]}, "history_link_transaction": {"chain_id": self.profile["chain_reference"], "expected_transaction_hash": "A" * 64}}
        self.put_signed("OPEN-BETA-DECISION.json", opened, "release")
        initiated = {"schema": "zerone-2-open-beta-initiation-evidence-v1", "signature_authority": self.authority("OPEN-BETA-INITIATION-EVIDENCE.json"), "open_beta_decision": {"sha256": self.hash("OPEN-BETA-DECISION.json")[7:], "detached_signature_sha256": self.hash("OPEN-BETA-DECISION.json.sig")[7:]}, "attestation_result": "MATCH", "deadline_satisfied": True, "history_link_transaction": {"committed_transaction_hash": "A" * 64, "expected_transaction_hash": "A" * 64, "deliver_code": 0}}
        self.put_signed("OPEN-BETA-INITIATION-EVIDENCE.json", initiated, "release")
        for name in ("RELEASE-VERIFY.stdout", "OPEN-BETA-VERIFY.stdout"):
            (self.bundle / name).write_bytes(b"authority-chain: MATCH\n")
        self.manifest = root / "beta-manifest.json"
        self.manifest.write_bytes(raw({"environment": "synthetic", "files": {name: self.hash(name) for name in sorted(v.PREREQUISITES) if (self.bundle / name).exists()}}))
        self.authority_source = root / "prior-verifier.py"
        self.authority_source.write_bytes(b"synthetic verifier bytes, never executed\n")
        receipt = {"schema": "zerone-seed-beta-verification/v1", "environment": "synthetic", "chain_id": self.chain, "genesis_hash": self.profile["genesis_hash"], "verifier_sha256": v.file_digest(self.authority_source), "bundle_manifest_sha256": v.digest(self.manifest.read_bytes()), "verified_at": stamp(now - dt.timedelta(minutes=2)), "documents": {name: self.hash(name) for name in v.PREREQUISITES if name.endswith((".json", ".json.sig")) and not name.startswith("BETA-")}, "runs": [{"stage": stage, "exit_code": 0, "stdout_sha256": self.hash(name)} for stage, name in (("dark-preinit", "RELEASE-VERIFY.stdout"), ("open-postinit", "OPEN-BETA-VERIFY.stdout"))], "dark_genesis": {"claims_live_at_genesis": False, "bootstrap_registrar": "", "pots": []}, "unrelated_policy_hash": self.unrelated}
        self.put_signed("BETA-VERIFICATION.json", receipt, "reviewer")
        self.trust = {"schema": "zerone-seed-trust/v1", "environment": "synthetic", "activation_id": "synthetic-cohort-1", "chain_id": self.chain, "genesis_hash": self.profile["genesis_hash"], "profile_id": self.profile["profile_id"], "node_trust_id": self.policy["node_trust_id"], "roles": keys, "keyring_sha256": v.digest((root.parent / "public.gpg").read_bytes()), "beta_verification_sha256": self.hash("BETA-VERIFICATION.json"), "verifier_sha256": receipt["verifier_sha256"], "bundle_manifest_sha256": receipt["bundle_manifest_sha256"]}
        (root / "trust.json").write_bytes(raw(self.trust))
        demo = {"environment": "synthetic", "result": "test-only-route-demonstration"}
        (self.bundle / "NATIVE-ROUTE-DEMONSTRATION.json").write_bytes(raw(demo))
        route = {"schema": "zerone-seed-native-route/v1", "environment": "synthetic", "profile_id": self.profile["profile_id"], "mechanism": "governance", "authority": self.gov, "execution_path": "cosmos.gov.v1.MsgSubmitProposal->zerone.claiming_pot.v1.MsgAddBootstrapEntry", "source_manifest_sha256": self.profile["source_digest"], "demonstration_sha256": self.hash("NATIVE-ROUTE-DEMONSTRATION.json"), "parameter_update_supported": False}
        (self.bundle / "NATIVE-ROUTE-EVIDENCE.json").write_bytes(raw(route))
        self.packet = {"schema": "zerone-seed-activation/v1", "environment": "synthetic", "activation_id": self.trust["activation_id"], "trust_sha256": v.digest(raw(self.trust)), "profile": self.profile, "artifacts": self.artifact_hashes, "prerequisites": {name: self.hash(name) for name in v.PREREQUISITES}, "admission": {"mechanism": "governance", "authority": self.gov, "route_evidence_sha256": self.hash("NATIVE-ROUTE-EVIDENCE.json"), "params_before": copy.deepcopy(self.params), "params_after": copy.deepcopy(self.params)}, "sponsor": {"account": self.policy["sponsor_account"], "dedicated": True, "single_signer_host_id": "synthetic-host"}, "custody": {"sponsor": "test-sponsor-binding", "admission": "test-native-route-binding", "claimant_records": "test-record-binding"}, "recipients": [self.policy], "budgets": {"issuance_commitment_uzrn": "222000", "full_grant_exposure_uzrn": "200000", "setup_fee_budget_uzrn": "200000", "total_sponsor_exposure_uzrn": "400000", "lifetime_commitment_cap_uzrn": "222000"}, "not_before": self.before, "expires_at": self.until, "admission_deadline": self.until, "stop_conditions": v.STOPS, "nonexpiring_pot_exposure_accepted": True, "unrelated_policy_hash": self.unrelated}
        if runtime:
            self.packet["sponsor"]["single_signer_host_id"] = runtime["config"]["host_id"]
        self.refresh_packet()
        self.pre = self.evidence("preactivation")
        self.post = self.evidence("postactivation")
        self.post["previous_evidence_sha256"] = v.digest(raw(self.pre))
        self.op = copy.deepcopy(self.post)
        self.op["stage"] = "operation"
        self.op["previous_evidence_sha256"] = v.digest(raw(self.post))
        operation = {"operation_id": "", "policy_hash": self.policy["policy_hash"], "plan_id": hid("plan"), "commitment_hash": hid("commitment"), "capability_record_id": hid("capability"), "intent_record_id": hid("intent"), "simulation_record_id": hid("simulation"), "host_id": "synthetic-host", "ledger_id": "synthetic-ledger", "ledger_binding_hash": hid("ledger-binding"), "profile_id": self.profile["profile_id"], "source_digest": self.profile["source_digest"], "descriptor_id": hid("descriptor"), "bundle_hash": hid("bundle"), "observation_hash": hid("observation"), "currentness_sha256": hid("currentness"), "budget_sha256": hid("budget"), "request_id": "synthetic-request", "sign_doc_bytes_hash": hid("sign-doc"), "prepared_at": self.when, "signer_key_id": hid("key"), "account_number": "10", "sequence": "0", "fee_uzrn": "50000", "gas_limit": "50000", "timeout_height": "1000", "expires_at": self.until, "ledger_snapshot_sha256": hid("ledger"), "attempt_status": "unreserved", "ledger_available": True, "max_intents_used": 0, "required_grant_exposure_uzrn": "200000", "required_setup_exposure_uzrn": "200000"}
        operation["operation_id"] = v.semantic_id(operation, "operation_id")
        self.op["operation"] = operation
        self.save_evidence()

    def authority(self, name):
        return {"algorithm": "openpgp", "authorized_signer_fingerprint": self.roles["release"], "detached_signature_filename": name + ".sig"}

    def hash(self, name):
        return v.digest((self.bundle / name).read_bytes())

    def put_signed(self, name, value, role, suffix=".sig"):
        path = self.bundle / name
        path.write_bytes(raw(value))
        self.sign(path, role, suffix)

    def refresh_packet(self):
        self.put_signed("SEED-ACTIVATION.json", self.packet, "activation")
        self.sign(self.bundle / "SEED-ACTIVATION.json", "reviewer", ".review.sig")
        custody = {"schema": "zerone-seed-custody/v1", "environment": "synthetic", "activation_sha256": self.hash("SEED-ACTIVATION.json"), "custody": self.packet["custody"], "single_signer_host_id": self.packet["sponsor"]["single_signer_host_id"], "route_evidence_sha256": self.packet["admission"]["route_evidence_sha256"], "native_route_status": "available", "valid_until": self.until}
        self.put_signed("SEED-CUSTODY.json", custody, "custodian")

    def confirmation(self, label, effect):
        height = self.runtime["observation"]["pot"]["start_block"] if self.runtime else "101"
        c = {"tx_hash": hid(label)[7:].upper(), "height": height, "block_hash": hid("block" + height), "block_time": self.when, "successor_height": str(int(height) + 1), "successor_block_hash": hid("block" + str(int(height) + 1)), "code": 0, "fee_uzrn": "50000"}
        receipt = {"schema": "zerone-seed-native-confirmation/v1", "environment": "synthetic", "profile_id": self.profile["profile_id"], "node_trust_id": self.policy["node_trust_id"], "confirmation": dict(c), "effect": effect}
        data = raw(receipt)
        c["evidence_sha256"] = v.digest(data)
        (self.bundle / "evidence" / (c["evidence_sha256"][7:] + ".json")).write_bytes(data)
        return c

    def evidence(self, stage):
        pre = stage == "preactivation"
        row = {"policy_hash": self.policy["policy_hash"], "pot": None, "allowance": None, "account_exists": not pre, "prior_claim": None, "admission_confirmation": None, "grant_confirmation": None}
        if not pre:
            row["pot"] = {"pot_id": self.policy["pot_id"], "status": "active", "total_amount_uzrn": "222000", "claimed_amount_uzrn": "0", "start_block": "101", "end_block": "102", "cliff_blocks": "0", "period_blocks": "0", "min_staking_tier": 0, "min_registration_age": "0", "whitelist": [self.claimant]}
            row["allowance"] = {"type_url": "/cosmos.feegrant.v1beta1.AllowedMsgAllowance", "inner_type_url": "/cosmos.feegrant.v1beta1.BasicAllowance", "granter": self.sponsor, "grantee": self.claimant, "allowed_messages": [v.CLAIM], "spend_limit_uzrn": "200000", "expires_at": self.until}
            if self.runtime:
                row["pot"] = self.runtime["observation"]["pot"]
                row["allowance"] = {k: value for k, value in self.runtime["observation"]["allowance"].items() if k != "status"}
            row["admission_confirmation"] = self.confirmation("admission", {"type_url": "/zerone.claiming_pot.v1.MsgAddBootstrapEntry", "authority": self.gov, "addresses": [self.claimant], "pot": row["pot"]})
            row["grant_confirmation"] = self.confirmation("grant", {"type_url": "/cosmos.feegrant.v1beta1.MsgGrantAllowance", "allowance": row["allowance"], "grantee_account_exists": True})
        result = {"schema": "zerone-seed-evidence/v1", "environment": "synthetic", "activation_sha256": self.hash("SEED-ACTIVATION.json"), "stage": stage, "previous_evidence_sha256": None, "trust_model": v.BOUNDARY, "node_trust_id": self.policy["node_trust_id"], "profile_id": self.profile["profile_id"], "chain_id": self.chain, "genesis_hash": self.profile["genesis_hash"], "observed_at": self.when, "anchor": {"height": "100" if pre else "103", "block_hash": hid("pre" if pre else "post"), "block_time": self.when}, "latest_height": "100" if pre else "103", "catching_up": False, "coherent": True, "parameters": self.params, "native_authority": {"mechanism": "governance", "authority": self.gov, "route_evidence_sha256": self.packet["admission"]["route_evidence_sha256"], "status": "available"}, "prior_commitment_uzrn": "0", "observed_total_commitment_uzrn": "0" if pre else "222000", "bootstrap_window_count": "0", "sponsor_balance_uzrn": "400000", "sponsor_other_exposure_uzrn": "0", "setup_spent_uzrn": "0" if pre else "100000", "unrelated_policy_hash": self.unrelated, "recipients": [row], "stop_reasons": [], "operation": None, "parameter_update_confirmation": None}

        if self.runtime:
            obs = self.runtime["observation"]
            if pre:
                result["anchor"]["height"] = result["latest_height"] = str(int(obs["pot"]["start_block"]) - 1)
            else:
                result.update({k: obs["evidence"][k] for k in ("anchor", "latest_height", "observed_at")})
            result["sponsor_balance_uzrn"] = obs["sponsor_balance_uzrn"]
        return result

    def synthetic_presign(self):
        # Deliberately non-Wallet unit fixture. The separate cross-layer test
        # supplies actual runtime-produced plans, records, simulation and prestate.
        op = self.op["operation"]
        row = self.op["recipients"][0]
        policy = next(p for p in self.packet["recipients"] if p["policy_hash"] == op["policy_hash"])
        obs = {"protocol": "agent-wallet-zerone.seed-observation/0.1", "status": "observed", "evidence": {"trust": "configured_full_node"} | {k: self.op[k] for k in ("node_trust_id", "profile_id", "chain_id", "genesis_hash", "anchor", "latest_height", "catching_up", "observed_at")}, "pot": row["pot"], "allowance": {"status": "found"} | row["allowance"], "prior_claim": None, "sponsor_account": policy["sponsor_account"], "sponsor_balance_uzrn": self.op["sponsor_balance_uzrn"], "min_claim_amount_uzrn": self.op["parameters"]["min_claim_amount"], "claimant": {"status": "found", "account": policy["claimant_account"], "account_number": op["account_number"], "sequence": op["sequence"]}}
        c = {k: op[k] for k in ("profile_id", "source_digest", "policy_hash", "capability_record_id", "intent_record_id", "signer_key_id", "account_number", "sequence", "gas_limit", "timeout_height", "expires_at")}
        c["fee_amount_uzrn"] = op["fee_uzrn"]
        plan = {"commitment": c, "commitment_hash": v.digest(v.canonical(c)), "observation_hash": v.digest(v.canonical(obs)), "sign_doc_bytes_hash": op["sign_doc_bytes_hash"]}
        plan["plan_id"] = v.semantic_id(plan, "plan_id")
        b = {"plan": plan, "observation": copy.deepcopy(obs), "prepared_at": op["prepared_at"], "simulation_result": {"fixture": "not-native", "evidence": copy.deepcopy(obs["evidence"])}}
        for field, record in (("descriptor_id", "descriptor"), ("capability_record_id", "capability"), ("intent_record_id", "intent"), ("simulation_record_id", "simulation")):
            b[record] = {"record_id": op[field]}
        op.update(plan_id=plan["plan_id"], commitment_hash=plan["commitment_hash"], observation_hash=plan["observation_hash"], bundle_hash=v.digest(v.canonical(b)))
        op["operation_id"] = v.semantic_id(op, "operation_id")
        return {"protocol": "zerone-seed-runtime.presign/0.1", "bundle": b, "observation": obs, "operation": op}

    def save_evidence(self):
        candidate = self.runtime_candidate or self.synthetic_presign()
        (self.bundle / "SEED-PRESIGN.json").write_bytes(v.canonical(candidate))
        self.post["previous_evidence_sha256"] = v.digest(raw(self.pre))
        self.op["previous_evidence_sha256"] = v.digest(raw(self.post))
        for stage, evidence in (("preactivation", self.pre), ("postactivation", self.post), ("operation", self.op)):
            name = "SEED-" + stage.upper() + ".json"
            self.put_signed(name, evidence, "observer")
        self.sign(self.bundle / "SEED-OPERATION.json", "custodian", ".custody.sig")

    def argv(self, stage="operation"):
        args = [stage, "--mode", "synthetic", "--bundle", str(self.bundle), "--trust", str(self.root / "trust.json"), "--trust-sha256", v.digest((self.root / "trust.json").read_bytes()), "--packet-sha256", self.hash("SEED-ACTIVATION.json"), "--evidence-sha256", self.hash("SEED-" + stage.upper() + ".json"), "--public-keyring", str(self.root.parent / "public.gpg"), "--gpgv", shutil.which("gpgv"), "--artifact-root", str(self.artifacts), "--beta-bundle-manifest", str(self.manifest), "--authority-verifier", str(self.authority_source), "--now", self.when]
        if stage == "operation":
            args += ["--operation-id", self.op["operation"]["operation_id"]]
        return args


class TestOpenPGP:
    """Tiny independent RFC4880 RSA/SHA256 fixture producer, TEST ONLY.

    No implementation from the verifier signs these bytes. cryptography supplies
    RSA key generation/signatures; GnuPG supplies the independent verification.
    """
    def __init__(self, label, epoch):
        import hashlib
        from cryptography.hazmat.primitives.asymmetric import rsa
        self.key = rsa.generate_private_key(public_exponent=65537, key_size=2048)
        self.epoch = epoch
        numbers = self.key.public_key().public_numbers()
        body = b"\x04" + epoch.to_bytes(4, "big") + b"\x01" + self.mpi(numbers.n) + self.mpi(numbers.e)
        self.key_material = b"\x99" + len(body).to_bytes(2, "big") + body
        self.fingerprint = hashlib.sha1(self.key_material).hexdigest().upper()
        uid = (label + "@synthetic.invalid").encode()
        certification = self.signature(self.key_material + b"\xb4" + len(uid).to_bytes(4, "big") + uid, 0x13)
        self.public = self.packet(6, body) + self.packet(13, uid) + certification

    @staticmethod
    def mpi(n):
        return n.bit_length().to_bytes(2, "big") + n.to_bytes((n.bit_length() + 7) // 8, "big")

    @staticmethod
    def packet(tag, body):
        return bytes([0xc0 | tag, 255]) + len(body).to_bytes(4, "big") + body

    def signature(self, message, signature_type=0):
        import hashlib
        from cryptography.hazmat.primitives import hashes
        from cryptography.hazmat.primitives.asymmetric import padding, utils
        hashed = b"\x05\x02" + self.epoch.to_bytes(4, "big") + b"\x16\x21\x04" + bytes.fromhex(self.fingerprint)
        header = bytes([4, signature_type, 1, 8]) + len(hashed).to_bytes(2, "big") + hashed
        digest = hashlib.sha256(message + header + b"\x04\xff" + len(header).to_bytes(4, "big")).digest()
        signature = self.key.sign(digest, padding.PKCS1v15(), utils.Prehashed(hashes.SHA256()))
        unhashed = b"\x09\x10" + bytes.fromhex(self.fingerprint[-16:])
        return self.packet(2, header + len(unhashed).to_bytes(2, "big") + unhashed + digest[:2] + self.mpi(int.from_bytes(signature, "big")))


class SeedGateTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        if not shutil.which("gpgv"):
            raise RuntimeError("gpgv required; signed tests must not silently skip")
        cls.tmp = tempfile.TemporaryDirectory(prefix="seed-gate-", dir="/private/tmp" if sys.platform == "darwin" else None)
        cls.addClassCleanup(cls.tmp.cleanup)
        cls.root = Path(cls.tmp.name).resolve()
        cls.now = dt.datetime.now(dt.timezone.utc).replace(microsecond=0)
        cls.keys = {role: TestOpenPGP(role, int(cls.now.timestamp()) - 120) for role in v.ROLES}
        cls.roles = {role: key.fingerprint for role, key in cls.keys.items()}
        (cls.root / "public.gpg").write_bytes(b"".join(cls.keys[role].public for role in sorted(cls.keys)))

    @classmethod
    def sign(cls, path, role, suffix=".sig"):
        Path(str(path) + suffix).write_bytes(cls.keys[role].signature(path.read_bytes()))

    @classmethod
    def tearDownClass(cls):
        cls.keys.clear()
        cls.tmp.cleanup()

    def setUp(self):
        self.case = tempfile.TemporaryDirectory(prefix="case-", dir=self.root)
        self.f = Fixture(Path(self.case.name), self.roles, self.sign, self.now)

    def tearDown(self):
        self.case.cleanup()

    def command(self, stage="operation", args=None):
        return subprocess.run([sys.executable, "-B", str(HERE / "verify-seed-activation.py"), *(args or self.f.argv(stage))], stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=45, check=False)

    def expect_refusal(self, stage="operation", args=None):
        result = self.command(stage, args)
        self.assertEqual(result.returncode, 1, (result.stdout, result.stderr))
        self.assertEqual(json.loads(result.stderr)["result"], "REFUSED")
        self.assertNotIn(b"AUTHENTICATED_INPUTS_MATCH", result.stdout)

    def test_signed_synthetic_all_three_stages(self):
        for stage in v.STAGES:
            with self.subTest(stage=stage):
                result = self.command(stage)
                self.assertEqual(result.returncode, 0, result.stderr)
                output = json.loads(result.stdout)
                self.assertEqual(output["result"], "SYNTHETIC_MATCH")
                self.assertEqual(output["effects"], "none")
                self.assertEqual(output["chain_truth"], "not_independently_verified")

    def test_signed_registrar_installation_all_stages(self):
        f = self.f
        registrar = v._bech32_encode("zrn", bytes([3]) * 20)
        route_path = f.bundle / "NATIVE-ROUTE-EVIDENCE.json"
        route = v.parse(route_path.read_bytes())
        route.update(mechanism="registrar", authority=registrar,
                     execution_path="zerone.claiming_pot.v1.MsgAddBootstrapEntry",
                     parameter_update_supported=True)
        route_path.write_bytes(raw(route))
        f.packet["admission"].update(mechanism="registrar", authority=registrar,
                                    route_evidence_sha256=f.hash("NATIVE-ROUTE-EVIDENCE.json"))
        f.packet["admission"]["params_after"]["bootstrap_registrar"] = registrar
        f.refresh_packet()
        for e in (f.pre, f.post, f.op):
            e["activation_sha256"] = f.hash("SEED-ACTIVATION.json")
            e["native_authority"] = {k: f.packet["admission"][k] for k in ("mechanism", "authority", "route_evidence_sha256")} | {"status": "available"}
        for e in (f.post, f.op):
            e["parameters"] = f.packet["admission"]["params_after"]
            e["bootstrap_window_count"] = "1"
            e["setup_spent_uzrn"] = "150000"
            e["sponsor_balance_uzrn"] = "400000"
            e["parameter_update_confirmation"] = f.confirmation("parameter-update", {"type_url": "/zerone.claiming_pot.v1.MsgUpdatePotParams", "authority": f.gov, "params": e["parameters"]})
            row = e["recipients"][0]
            row["admission_confirmation"] = f.confirmation("admission", {"type_url": "/zerone.claiming_pot.v1.MsgAddBootstrapEntry", "authority": registrar, "addresses": [f.claimant], "pot": row["pot"]})
        f.save_evidence()
        for stage in v.STAGES:
            result = self.command(stage)
            self.assertEqual(result.returncode, 0, result.stderr)
        f.pre["bootstrap_window_count"] = "1"
        f.save_evidence()
        self.expect_refusal("preactivation")

    def test_cohort_operation_observes_only_its_exact_recipient(self):
        f = self.f
        other_address = v._bech32_encode("zrn", bytes([4]) * 20)
        other = copy.deepcopy(f.policy)
        other["claimant_account"] = f.chain + ":" + other_address
        other["pot_id"] = "bootstrap-" + other_address
        other["policy_hash"] = v.semantic_id(other, "policy_hash")
        f.packet["recipients"] = sorted([f.policy, other], key=lambda p: p["claimant_account"])
        for key in f.packet["budgets"]:
            f.packet["budgets"][key] = str(int(f.packet["budgets"][key]) * 2)
        for key in ("params_before", "params_after"):
            f.packet["admission"][key]["bootstrap_emission_cap_uzrn"] = "444000"
        f.refresh_packet()
        pre_other = copy.deepcopy(f.pre["recipients"][0])
        pre_other["policy_hash"] = other["policy_hash"]
        post_other = copy.deepcopy(f.post["recipients"][0])
        post_other["policy_hash"] = other["policy_hash"]
        post_other["pot"].update(pot_id=other["pot_id"], whitelist=[other_address])
        post_other["allowance"]["grantee"] = other_address
        post_other["admission_confirmation"] = f.confirmation("other-admission", {"type_url": "/zerone.claiming_pot.v1.MsgAddBootstrapEntry", "authority": f.gov, "addresses": [other_address], "pot": post_other["pot"]})
        post_other["grant_confirmation"] = f.confirmation("other-grant", {"type_url": "/cosmos.feegrant.v1beta1.MsgGrantAllowance", "allowance": post_other["allowance"], "grantee_account_exists": True})
        ordering = {p["policy_hash"]: n for n, p in enumerate(f.packet["recipients"])}
        f.pre["recipients"] = sorted([f.pre["recipients"][0], pre_other], key=lambda r: ordering[r["policy_hash"]])
        f.post["recipients"] = sorted([f.post["recipients"][0], post_other], key=lambda r: ordering[r["policy_hash"]])
        f.op["recipients"] = [post_other]
        f.op["operation"]["policy_hash"] = other["policy_hash"]
        f.op["operation"]["operation_id"] = v.semantic_id(f.op["operation"], "operation_id")
        for e in (f.pre, f.post, f.op):
            e["activation_sha256"] = f.hash("SEED-ACTIVATION.json")
            e["parameters"] = f.packet["admission"]["params_before"]
            e["sponsor_balance_uzrn"] = "800000"
            if e is not f.pre:
                e["observed_total_commitment_uzrn"] = "444000"
                e["setup_spent_uzrn"] = "200000"
        f.save_evidence()
        result = self.command()
        self.assertEqual(result.returncode, 0, result.stderr)
        f.op["recipients"] = [f.post["recipients"][ordering[f.policy["policy_hash"]]]]
        f.save_evidence()
        self.expect_refusal()

    def test_confirmation_height_hash_joins(self):
        for kind in ("inclusion", "successor", "anchor"):
            e = copy.deepcopy(self.f.post)
            if kind == "inclusion":
                e["recipients"][0]["grant_confirmation"]["block_hash"] = hid("fork101")
            elif kind == "successor":
                e["recipients"][0]["grant_confirmation"]["successor_block_hash"] = hid("fork102")
            else:
                e["anchor"]["height"] = e["latest_height"] = "102"
            with self.subTest(kind=kind), self.assertRaises(v.Refusal):
                v.validate_evidence(e, self.f.packet, self.f.hash("SEED-ACTIVATION.json"), "postactivation", self.now, self.f.trust, self.f.pre)

    def test_latest_height_deadline_not_anchor(self):
        e = copy.deepcopy(self.f.op)
        e["anchor"]["height"], e["latest_height"] = "999", "1000"
        with self.assertRaises(v.Refusal):
            v.validate_evidence(e, self.f.packet, self.f.hash("SEED-ACTIVATION.json"), "operation", self.now, self.f.trust, self.f.post)

    def test_spent_setup_is_not_released_from_lifetime_exposure(self):
        e = copy.deepcopy(self.f.post)
        e["sponsor_balance_uzrn"] = "300000"
        with self.assertRaises(v.Refusal):
            v.validate_evidence(e, self.f.packet, self.f.hash("SEED-ACTIVATION.json"), "postactivation", self.now, self.f.trust, self.f.pre)
        e["sponsor_balance_uzrn"] = "400000"
        v.validate_evidence(e, self.f.packet, self.f.hash("SEED-ACTIVATION.json"), "postactivation", self.now, self.f.trust, self.f.pre)

    def test_independent_policy_gas_and_fee_ceilings(self):
        p = copy.deepcopy(self.f.packet)
        policy = p["recipients"][0]
        policy["max_gas"] = "100000"
        policy["policy_hash"] = v.semantic_id(policy, "policy_hash")
        v.validate_packet(p, self.f.trust, self.now)

    def test_strict_types_and_missing_confirmations(self):
        for field, value in (("min_staking_tier", False), ("status", "expired"), ("end_block", "103")):
            e = copy.deepcopy(self.f.post)
            e["recipients"][0]["pot"][field] = value
            with self.subTest(field=field), self.assertRaises(v.Refusal):
                v.validate_evidence(e, self.f.packet, self.f.hash("SEED-ACTIVATION.json"), "postactivation", self.now, self.f.trust, self.f.pre)
        e = copy.deepcopy(self.f.post)
        e["recipients"][0]["grant_confirmation"]["code"] = False
        with self.assertRaises(v.Refusal):
            v.validate_evidence(e, self.f.packet, self.f.hash("SEED-ACTIVATION.json"), "postactivation", self.now, self.f.trust, self.f.pre)
        e = copy.deepcopy(self.f.post)
        e["recipients"][0]["admission_confirmation"]["height"] = "100"
        with self.assertRaises(v.Refusal):
            v.validate_evidence(e, self.f.packet, self.f.hash("SEED-ACTIVATION.json"), "postactivation", self.now, self.f.trust, self.f.pre)

    def test_allowance_and_reservation_must_be_complete(self):
        for key, value in (("allowed_messages", [v.CLAIM, "/cosmos.bank.v1beta1.MsgSend"]), ("spend_limit_uzrn", "50000"), ("expires_at", ""), ("inner_type_url", "/cosmos.feegrant.v1beta1.PeriodicAllowance")):
            e = copy.deepcopy(self.f.post)
            e["recipients"][0]["allowance"][key] = value
            with self.subTest(key=key), self.assertRaises(v.Refusal):
                v.validate_evidence(e, self.f.packet, self.f.hash("SEED-ACTIVATION.json"), "postactivation", self.now, self.f.trust, self.f.pre)
        for key, value in (("required_grant_exposure_uzrn", "50000"), ("required_setup_exposure_uzrn", "0"), ("ledger_available", False), ("max_intents_used", 1), ("gas_limit", "22221"), ("timeout_height", "103")):
            e = copy.deepcopy(self.f.op)
            e["operation"][key] = value
            e["operation"]["operation_id"] = v.semantic_id(e["operation"], "operation_id")
            with self.subTest(key=key), self.assertRaises(v.Refusal):
                v.validate_evidence(e, self.f.packet, self.f.hash("SEED-ACTIVATION.json"), "operation", self.now, self.f.trust, self.f.post)

    def test_presign_anchors_join_stages_and_preserve_historical_preparation(self):
        base = self.f.synthetic_presign()
        def bind(candidate):
            b, op = candidate["bundle"], candidate["operation"]
            b["plan"]["observation_hash"] = v.digest(v.canonical(b["observation"]))
            b["plan"]["plan_id"] = v.semantic_id(b["plan"], "plan_id")
            op.update(plan_id=b["plan"]["plan_id"], bundle_hash=v.digest(v.canonical(b)), prepared_at=b["prepared_at"])
            op["operation_id"] = v.semantic_id(op, "operation_id")
            return dict(self.f.op, operation=op)
        # A prepared observation may age out later without invalidating its inert
        # reconstruction; the current simulation/observation still must be fresh.
        historical = copy.deepcopy(base)
        old = (self.now - dt.timedelta(seconds=180)).isoformat(timespec="milliseconds").replace("+00:00", "Z")
        b = historical["bundle"]
        b["prepared_at"] = old
        b["observation"]["evidence"].update(observed_at=old, latest_height="102", anchor={"height":"102", "block_hash":hid("historical-preparation"), "block_time":old})
        e = bind(historical)
        v.validate_presign(historical, e, self.f.packet, {}, self.now)
        for fault in ("same-height", "rollback", "stage-fork"):
            candidate = copy.deepcopy(historical)
            a = candidate["bundle"]["observation"]["evidence"]["anchor"]
            if fault == "same-height":
                a["height"] = "103"
            elif fault == "rollback":
                a["height"] = "104"
            candidate["bundle"]["observation"]["evidence"]["latest_height"] = a["height"]
            e = bind(candidate)
            blocks = {"102": (hid("different-stage-block"), old)} if fault == "stage-fork" else {}
            with self.subTest(fault=fault), self.assertRaises(v.Refusal):
                v.validate_presign(candidate, e, self.f.packet, blocks, self.now)

    def test_actual_presign_bytes_required_not_saved_result(self):
        path = self.f.bundle / "SEED-PRESIGN.json"
        original = path.read_bytes()
        path.unlink()
        self.expect_refusal()
        candidate = v.parse(original, ceremony=False)
        candidate["bundle"]["simulation_result"]["fixture"] = "substituted"
        path.write_bytes(v.canonical(candidate))
        self.expect_refusal()

    def test_outside_exposure_is_additional_to_full_cohort_lifetime_budget(self):
        e = copy.deepcopy(self.f.post)
        e["sponsor_other_exposure_uzrn"] = "1"
        with self.assertRaises(v.Refusal):
            v.validate_evidence(e, self.f.packet, self.f.hash("SEED-ACTIVATION.json"), "postactivation", self.now, self.f.trust, self.f.pre)
        e["sponsor_balance_uzrn"] = "400001"
        v.validate_evidence(e, self.f.packet, self.f.hash("SEED-ACTIVATION.json"), "postactivation", self.now, self.f.trust, self.f.pre)

    def test_no_public_phase_inference_from_example_or_missing_evidence(self):
        with self.assertRaises(v.Refusal):
            v.parse((HERE / "networks/zerone-2/SEED-ACTIVATION.example.json").read_bytes())
        (self.f.bundle / "BETA-VERIFICATION.json").unlink()
        self.expect_refusal("preactivation")

    def test_packet_tamper_even_when_caller_updates_digest(self):
        self.f.packet["activation_id"] = "tampered"
        (self.f.bundle / "SEED-ACTIVATION.json").write_bytes(raw(self.f.packet))
        self.expect_refusal("preactivation")

    def test_wrong_role_signature(self):
        self.sign(self.f.bundle / "SEED-ACTIVATION.json", "custodian")
        self.expect_refusal("preactivation")

    def test_missing_independent_review_signature(self):
        (self.f.bundle / "SEED-ACTIVATION.json.review.sig").unlink()
        self.expect_refusal("preactivation")

    def test_production_never_accepts_synthetic(self):
        args = self.f.argv("preactivation")
        args[args.index("--mode") + 1] = "production"
        del args[args.index("--now"):args.index("--now") + 2]
        self.expect_refusal(args=args)
        p, trust = copy.deepcopy(self.f.packet), copy.deepcopy(self.f.trust)
        p["environment"] = trust["environment"] = "production"
        with self.assertRaises(v.Refusal):
            v.validate_packet(p, trust, self.now)

    def test_wrong_chain_signed_observation(self):
        self.f.pre["chain_id"] = "cosmos:zerone-2"
        self.f.save_evidence()
        self.expect_refusal("preactivation")

    def test_unknown_fields_duplicates_and_numeric_forms(self):
        samples = [b'{"a":1,"a":2}\n', b'{"a":1.0}\n', b'{"a":-0}\n', b'{"a":NaN}\n', b'{"a":"\\u0000"}\n', b'{"a":"\\ud800"}\n', b'{ "a":1}\n', b'{"a":1}\n\n']
        for data in samples:
            with self.subTest(data=data), self.assertRaises(v.Refusal):
                v.parse(data)
        self.f.pre["approved"] = True
        self.f.save_evidence()
        self.expect_refusal("preactivation")

    def test_packet_and_policy_unknown_fields(self):
        for where in ("packet", "policy"):
            p = copy.deepcopy(self.f.packet)
            target = p if where == "packet" else p["recipients"][0]
            target["signAndSend"] = True
            with self.subTest(where=where), self.assertRaises(v.Refusal):
                v.validate_packet(p, self.f.trust, self.now)

    def test_full_grant_budget_not_fee_estimate(self):
        for key, value in (("full_grant_exposure_uzrn", "50000"), ("total_sponsor_exposure_uzrn", "250000"), ("issuance_commitment_uzrn", "221999"), ("setup_fee_budget_uzrn", "0")):
            p = copy.deepcopy(self.f.packet)
            p["budgets"][key] = value
            with self.subTest(key=key), self.assertRaises(v.Refusal):
                v.validate_packet(p, self.f.trust, self.now)
        self.f.pre["sponsor_balance_uzrn"] = "250000"
        self.f.save_evidence()
        self.expect_refusal("preactivation")

    def test_duplicate_recipients_and_unrelated_parameter_delta(self):
        p = copy.deepcopy(self.f.packet)
        p["recipients"] *= 2
        with self.assertRaises(v.Refusal):
            v.validate_packet(p, self.f.trust, self.now)
        p = copy.deepcopy(self.f.packet)
        p["admission"]["params_after"]["min_claim_amount"] = "1"
        with self.assertRaises(v.Refusal):
            v.validate_packet(p, self.f.trust, self.now)

    def test_policy_hash_wallet_canonical_without_lf(self):
        # Independent stdlib construction, not ceremonial LF or self-ID hashing.
        p = self.f.policy
        expected = "sha256:" + __import__("hashlib").sha256(json.dumps({k: p[k] for k in sorted(p) if k != "policy_hash"}, separators=(",", ":"), ensure_ascii=False).encode()).hexdigest()
        self.assertEqual(p["policy_hash"], expected)
        self.assertNotEqual(expected, v.digest(raw({k: x for k, x in p.items() if k != "policy_hash"})))
        changed = copy.deepcopy(self.f.packet)
        changed["recipients"][0]["max_fee_uzrn"] = "50001"
        with self.assertRaises(v.Refusal):
            v.validate_packet(changed, self.f.trust, self.now)

    def test_expiry_stale_incoherent_and_underfunded(self):
        changes = [("observed_at", stamp(self.now + dt.timedelta(seconds=1))), ("catching_up", True), ("coherent", False), ("latest_height", "1000"), ("sponsor_other_exposure_uzrn", "1"), ("prior_commitment_uzrn", "222000"), ("observed_total_commitment_uzrn", "222000")]
        for key, value in changes:
            e = copy.deepcopy(self.f.pre)
            e[key] = value
            with self.subTest(key=key), self.assertRaises(v.Refusal):
                v.validate_evidence(e, self.f.packet, self.f.hash("SEED-ACTIVATION.json"), "preactivation", self.now, self.f.trust)
        with self.assertRaises(v.Refusal):
            v.validate_packet(self.f.packet, self.f.trust, self.now + dt.timedelta(hours=1))
        e = copy.deepcopy(self.f.pre)
        e["anchor"]["block_time"] = stamp(self.now - dt.timedelta(seconds=301))
        with self.assertRaises(v.Refusal):
            v.validate_evidence(e, self.f.packet, self.f.hash("SEED-ACTIVATION.json"), "preactivation", self.now, self.f.trust)

    def test_consumed_and_unknown_attempts_never_replay(self):
        for state in ("reserved_unsigned", "signing_unknown", "submission_unknown", "signed", "included", "absent", "expired"):
            e = copy.deepcopy(self.f.op)
            e["operation"]["attempt_status"] = state
            e["operation"]["operation_id"] = v.semantic_id(e["operation"], "operation_id")
            with self.subTest(state=state), self.assertRaisesRegex(v.Refusal, "sticky"):
                v.validate_evidence(e, self.f.packet, self.f.hash("SEED-ACTIVATION.json"), "operation", self.now, self.f.trust, self.f.post)

    def test_narrowed_operation_expiry_and_same_height_fork(self):
        e = copy.deepcopy(self.f.op)
        e["operation"]["expires_at"] = stamp(self.now + dt.timedelta(seconds=30))
        e["operation"]["operation_id"] = v.semantic_id(e["operation"], "operation_id")
        v.validate_evidence(e, self.f.packet, self.f.hash("SEED-ACTIVATION.json"), "operation", self.now, self.f.trust, self.f.post)
        for expiry in (stamp(self.now), stamp(self.now + dt.timedelta(hours=1))):
            e["operation"]["expires_at"] = expiry
            e["operation"]["operation_id"] = v.semantic_id(e["operation"], "operation_id")
            with self.assertRaisesRegex(v.Refusal, "deadline"):
                v.validate_evidence(e, self.f.packet, self.f.hash("SEED-ACTIVATION.json"), "operation", self.now, self.f.trust, self.f.post)
        e = copy.deepcopy(self.f.op)
        e["anchor"]["block_hash"] = hid("another-fork")
        with self.assertRaisesRegex(v.Refusal, "same-height"):
            v.validate_evidence(e, self.f.packet, self.f.hash("SEED-ACTIVATION.json"), "operation", self.now, self.f.trust, self.f.post)

    def test_operator_account_string_is_not_native_route(self):
        self.f.pre["native_authority"]["status"] = "unavailable"
        self.f.save_evidence()
        self.expect_refusal("preactivation")

    def test_signed_confirmations_required_and_receipt_tamper(self):
        receipt_hash = self.f.post["recipients"][0]["admission_confirmation"]["evidence_sha256"]
        (self.f.bundle / "evidence" / (receipt_hash[7:] + ".json")).write_bytes(b"{}\n")
        self.expect_refusal("postactivation")

    def test_grant_revoke_does_not_cancel_pot(self):
        e = copy.deepcopy(self.f.post)
        e["recipients"][0]["allowance"] = None
        with self.assertRaises(v.Refusal):
            v.validate_evidence(e, self.f.packet, self.f.hash("SEED-ACTIVATION.json"), "postactivation", self.now, self.f.trust, self.f.pre)
        self.assertEqual(e["recipients"][0]["pot"]["total_amount_uzrn"], "222000")
        p = copy.deepcopy(self.f.packet)
        p["nonexpiring_pot_exposure_accepted"] = False
        with self.assertRaises(v.Refusal):
            v.validate_packet(p, self.f.trust, self.now)

    def test_artifact_mutation_and_missing_bundle(self):
        (self.f.artifacts / "zerone-seed-io").write_bytes(b"different helper")
        self.expect_refusal("preactivation")
        (self.f.bundle / "BETA-VERIFICATION.json").unlink()
        self.expect_refusal("preactivation")

    def test_symlink_hardlink_and_oversize_inputs(self):
        target = self.f.root / "target"
        target.write_bytes(b"x")
        link = self.f.root / "link"
        link.symlink_to(target)
        with self.assertRaises((v.Refusal, OSError)):
            v.read(link)
        link.unlink()
        os.link(target, link)
        with self.assertRaises(v.Refusal):
            v.read(target)
        with self.assertRaises(v.Refusal):
            v.parse(b" " * (v.MAX_JSON + 1))
        with self.assertRaises(v.Refusal):
            v.parse(b"[" * 34 + b"0" + b"]" * 34 + b"\n")

    def test_no_arbitrary_packet_command(self):
        self.f.packet["verify_command"] = ["touch", "/must-not-be-run"]
        self.f.refresh_packet()
        self.expect_refusal("preactivation")

    def test_exact_evidence_and_operation_pins(self):
        for flag in ("--evidence-sha256", "--operation-id", "--trust-sha256"):
            args = self.f.argv()
            args[args.index(flag) + 1] = hid("wrong")
            with self.subTest(flag=flag):
                self.expect_refusal(args=args)

    def test_help_requires_no_inputs_and_has_no_effects(self):
        result = subprocess.run([sys.executable, "-B", str(HERE / "verify-seed-activation.py"), "--help"], cwd=self.f.root, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=10)
        self.assertEqual(result.returncode, 0)
        self.assertIn(b"NOT independent chain truth", result.stdout)
        self.assertFalse((self.f.root / ".gnupg").exists())


def runtime_seam(input_path):
    """One-shot TEST ONLY driver: actual compiled host, public gate signatures,
    and disposable record provider. The native helper is explicitly simulated;
    no chain daemon, production key, persistent signing service or live funds.
    """
    data = json.loads(Path(input_path).read_bytes())
    config = data["config"]
    root = Path(data["root"])
    now = dt.datetime.now(dt.timezone.utc)
    keys = {role: TestOpenPGP(role, int(now.timestamp()) - 120) for role in v.ROLES}
    roles = {role: key.fingerprint for role, key in keys.items()}
    (root / "public.gpg").write_bytes(b"".join(keys[role].public for role in sorted(keys)))
    def sign(path, role, suffix=".sig"):
        Path(str(path) + suffix).write_bytes(keys[role].signature(path.read_bytes()))
    case_root = root / "gate"
    case_root.mkdir(mode=0o700)
    f = Fixture(case_root, roles, sign, now, runtime=data)
    executable = lambda p: {"path": str(Path(p).resolve()), "sha256": v.file_digest(Path(p).resolve())}
    config["activation_gate"] = {"mode": "synthetic", "python": executable(sys.executable), "verifier": executable(HERE / "verify-seed-activation.py"), "codec_sha256": v.file_digest(HERE / "frozen_evidence.py"), "gpgv": executable(shutil.which("gpgv")), "bundle": str(f.bundle), "trust_file": str(f.root / "trust.json"), "trust_sha256": v.digest(raw(f.trust)), "packet_sha256": f.hash("SEED-ACTIVATION.json"), "evidence_sha256": None, "public_keyring": str(root / "public.gpg"), "artifact_root": str(f.artifacts), "beta_bundle_manifest": str(f.manifest), "authority_verifier": str(f.authority_source)}
    config_path = root / "runtime-gated.json"
    def put(path, value):
        path.write_bytes(v.canonical(value))
        path.chmod(0o600)
    def invoke(command, args=(), ok=True):
        put(config_path, config)
        result = subprocess.run([data["cli"], command, "--config", str(config_path), "--config-sha256", v.digest(config_path.read_bytes()), *args], stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=40, env={"PATH": "/usr/bin:/bin"}, check=False)
        value = json.loads(result.stdout)
        if ok:
            assert result.returncode == 0, (command, value)
        else:
            assert result.returncode == 1 and value["status"] == "error", (command, value)
        return value
    invoke("init")
    original = invoke("prepare")
    prepared = root / "prepared.json"
    put(prepared, original)
    scenario = data["scenario"]
    if scenario == "prepare-fork":
        helper_path = Path(config["helper_trust_file"])
        helper = json.loads(helper_path.read_bytes())
        helper["observation"]["evidence"]["anchor"]["block_hash"] = hid("fork-after-preparation")
        put(helper_path, helper)
        config["helper_trust_sha256"] = v.digest(helper_path.read_bytes())
        invoke("pre-sign", ("--plan", str(prepared)), ok=False)
        assert invoke("status")["event_count"] == 0
        assert Path(data["log"]).read_text().splitlines().count("sign") == 0
        return {"scenario": scenario, "gate": "not-reached", "host": "refused-before-sign", "presign_event_count": 0, "real_gpgv": False, "daemon": False}
    candidate = invoke("pre-sign", ("--plan", str(prepared)))
    assert invoke("status")["event_count"] == 0
    assert candidate["operation"]["attempt_status"] == "unreserved"
    assert candidate["bundle"]["prepared_at"] == original["prepared_at"]
    assert "signing_request_hash" not in candidate["operation"]
    f.runtime_candidate = candidate
    f.op["operation"] = candidate["operation"]
    scenario = data["scenario"]
    if scenario == "signed-preparation-fork":
        b = candidate["bundle"]
        b["observation"]["evidence"]["anchor"]["block_hash"] = hid("conflicting-original-preparation")
        b["plan"]["observation_hash"] = v.digest(v.canonical(b["observation"]))
        b["plan"]["plan_id"] = v.semantic_id(b["plan"], "plan_id")
        b["simulation_result"]["plan_id"] = b["plan"]["plan_id"]
        candidate["operation"].update(plan_id=b["plan"]["plan_id"], bundle_hash=v.digest(v.canonical(b)))
        candidate["operation"]["operation_id"] = v.semantic_id(candidate["operation"], "operation_id")
    if scenario == "signed-wrong-prestate":
        candidate["operation"]["ledger_snapshot_sha256"] = hid("not-the-real-journal-head")
    elif scenario == "signed-wrong-currentness":
        candidate["operation"]["currentness_sha256"] = hid("not-the-real-currentness")
    elif scenario == "signed-wrong-budget":
        candidate["operation"]["budget_sha256"] = hid("not-the-real-budget")
    if scenario.startswith("signed-wrong-"):
        candidate["operation"]["operation_id"] = v.semantic_id(candidate["operation"], "operation_id")
    f.save_evidence()
    candidate_path = root / "candidate.json"
    put(candidate_path, candidate)
    config["activation_gate"]["evidence_sha256"] = f.hash("SEED-OPERATION.json")
    # A real verifier result precedes the host test, including intentionally
    # wrong attestations which only the host's real journal/authority can refute.
    args = f.argv()
    del args[args.index("--now"):args.index("--now") + 2]
    checked = subprocess.run([sys.executable, "-B", str(HERE / "verify-seed-activation.py"), *args], stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=40)
    if scenario == "signed-preparation-fork":
        assert checked.returncode != 0, "signed conflicting original preparation accepted by real gate"
        invoke("reserve-sign", ("--plan", str(candidate_path)), ok=False)
        assert invoke("status")["event_count"] == 0
        assert Path(data["log"]).read_text().splitlines().count("sign") == 0
        return {"scenario": scenario, "gate": "refused", "host": "refused-before-sign", "presign_event_count": 0, "real_gpgv": True, "daemon": False}
    assert checked.returncode == 0, checked.stderr
    gate_result = json.loads(checked.stdout)
    assert gate_result["operation"] == candidate["operation"] and gate_result["result"] == "SYNTHETIC_MATCH"
    if scenario == "missing-evidence":
        config["activation_gate"]["evidence_sha256"] = None
    elif scenario == "changed-request":
        changed = copy.deepcopy(candidate)
        changed["operation"]["request_id"] = "substituted-request"
        put(candidate_path, changed)
    elif scenario == "changed-bundle":
        changed = copy.deepcopy(candidate)
        changed["bundle"]["simulation_result"]["gas_used"] = "1"
        put(candidate_path, changed)
    elif scenario in ("fresh-underfunded", "fresh-timeout", "fresh-fork"):
        helper_path = Path(config["helper_trust_file"])
        helper = json.loads(helper_path.read_bytes())
        if scenario == "fresh-underfunded":
            helper["observation"]["sponsor_balance_uzrn"] = "300000"
        elif scenario == "fresh-timeout":
            helper["observation"]["evidence"]["anchor"]["height"] = str(int(config["policy"]["timeout_height"]) - 1)
            helper["observation"]["evidence"]["latest_height"] = config["policy"]["timeout_height"]
        else:
            helper["observation"]["evidence"]["anchor"]["block_hash"] = hid("different-fork")
        put(helper_path, helper)
        config["helper_trust_sha256"] = v.digest(helper_path.read_bytes())
    if scenario in ("missing-key", "crash-sign"):
        helper_path = Path(config["helper_trust_file"])
        helper = json.loads(helper_path.read_bytes())
        helper["mode"] = scenario
        put(helper_path, helper)
        config["helper_trust_sha256"] = v.digest(helper_path.read_bytes())
    rejected = scenario not in ("normal", "concurrent", "missing-key", "crash-sign")
    if scenario == "concurrent":
        put(config_path, config)
        argv = [data["cli"], "reserve-sign", "--config", str(config_path), "--config-sha256", v.digest(config_path.read_bytes()), "--plan", str(candidate_path)]
        workers = [subprocess.Popen(argv, stdout=subprocess.PIPE, stderr=subprocess.PIPE, env={"PATH": "/usr/bin:/bin"}) for _ in range(2)]
        outputs = [json.loads(worker.communicate(timeout=40)[0]) for worker in workers]
        assert sorted(worker.returncode for worker in workers) == [0, 1]
        assert sorted(value["status"] for value in outputs) == ["error", "signed"]
        result = next(value for value in outputs if value["status"] == "signed")
    else:
        result = invoke("reserve-sign", ("--plan", str(candidate_path)), ok=not rejected)
    logs = Path(data["log"]).read_text().splitlines()
    if rejected:
        assert logs.count("sign") == 0
        assert invoke("status")["event_count"] == 0
    elif scenario in ("missing-key", "crash-sign"):
        assert result["status"] == "signing_unknown" and logs.count("sign") == 1
        assert invoke("status")["operations"][0]["status"] == "signing_unknown"
        assert invoke("status")["total_reserved_sponsor_exposure_uzrn"] == "400000"
        invoke("reserve-sign", ("--plan", str(candidate_path)), ok=False)
        assert Path(data["log"]).read_text().splitlines().count("sign") == 1
        if scenario == "crash-sign":
            assert invoke("verify", ("--operation", original["plan"]["plan_id"]))["status"] == "signed"
    else:
        assert result["status"] == "signed" and logs.count("sign") == 1
        status = invoke("status")
        assert status["total_reserved_sponsor_exposure_uzrn"] == "400000"
        assert status["operations"][0]["status"] == "signed"
        invoke("reserve-sign", ("--plan", str(candidate_path)), ok=False)
        assert Path(data["log"]).read_text().splitlines().count("sign") == 1
        assert invoke("submit", ("--operation", original["plan"]["plan_id"]))["status"] == "submission_unknown"
        assert invoke("reconcile", ("--operation", original["plan"]["plan_id"]))["status"] == "included_success"
    return {"scenario": scenario, "gate": "SYNTHETIC_MATCH", "host": "refused-before-sign" if rejected else "signing_unknown" if scenario in ("missing-key", "crash-sign") else "included_success", "presign_event_count": 0, "real_gpgv": True, "daemon": False}


if __name__ == "__main__":
    if len(sys.argv) == 3 and sys.argv[1] == "--runtime-seam-input":
        print(v.canonical(runtime_seam(sys.argv[2])).decode())
    else:
        unittest.main(verbosity=2)
