#!/usr/bin/env python3
"""Offline tests for pre-h-census.py. Every chain response here is a UNIT MOCK."""
import base64
import copy
import json
import unittest
import urllib.parse
from unittest import mock

from test_kit import IMAGE, load

census = load("pre_h_census", "pre-h-census.py")
HEIGHT = 3_870_000
FAUCET = "zrn1xl5pnczujgqzzcaq8v53acmfyfmj394nws0rk5"


def varint(n):
    out = bytearray()
    while True:
        byte, n = n & 0x7F, n >> 7
        out.append(byte | (0x80 if n else 0))
        if not n:
            return bytes(out)


def field(number, body):
    return varint(number << 3 | 2) + varint(len(body)) + body


def pairs(items):
    return b"".join(field(1, field(1, key) + field(2, value)) for key, value in items)


def pot(pot_id, total):
    return field(1, pot_id.encode()) + field(2, b"name") + field(3, total.encode()) + field(4, b"0")


class FakeChain:
    def __init__(self):
        self.rest = {
            "/cosmos/upgrade/v1beta1/module_versions": {"module_versions": [
                {"name": k, "version": str(v)} for k, v in census.PRESTATE.items()]},
            "/cosmos/upgrade/v1beta1/applied_plan/consolidation-safety-v1": {"height": "0"},
            "/cosmos/upgrade/v1beta1/current_plan": {"plan": None},
            "/zerone/claiming_pot/v1/params": {"params": {"maxPotsActive": 10, "minClaimAmount": "1000",
                                                          "bootstrapEmissionCapUzrn": "222222000000"}},
            "/zerone/claiming_pot/v1/pots": {"pots": []},
            "/zerone/liquiditypool/v1/params": {"params": {"defaultSwapFeeBps": "3000", "maxPools": "0",
                                                           "minInitialLiquidity": "10000000000", "twapWindowBlocks": "1000",
                                                           "protocolFeeBps": "450000", "minReserve": "1",
                                                           "billingQuoteDenoms": []}},
            "/zerone/liquiditypool/v1/pools": {"pools": []},
            "/cosmos/gov/v1/proposals": {"proposals": []},
            "/zerone/auth/v1/frozen_accounts": {"accounts": []},
        }
        self.store = {("knowledge", "subspace", b"\x7f\x01"): pairs([(b"\x7f\x01upgrade_marker_agenttool-seam-v1", b"migrated")])}
        self.open_proposals = {}

    def get(self, url, headers=None):
        parts = urllib.parse.urlsplit(url)
        query = dict(urllib.parse.parse_qsl(parts.query))
        if parts.path == "/status":
            return {"result": {"node_info": {"network": "zerone-testnet-1"}}}
        if parts.path == "/abci_info":
            return {"result": {"response": {"last_block_height": str(HEIGHT), "last_block_app_hash": "AA=="}}}
        if parts.path == "/abci_query":
            store, mode = json.loads(query["path"]).split("/")[2:4]
            value = self.store.get((store, mode, bytes.fromhex(query["data"][2:])), b"")
            return {"result": {"response": {"code": 0, "height": query["height"], "value": base64.b64encode(value).decode()}}}
        assert headers == {"x-cosmos-block-height": str(HEIGHT)}, headers
        if parts.path == "/cosmos/gov/v1/proposals":
            return {"proposals": self.open_proposals.get(query["proposal_status"], [])}
        return copy.deepcopy(self.rest[parts.path])


class CensusTests(unittest.TestCase):
    def run_census(self, chain, expect=None, allow=None):
        with mock.patch.object(census.Census, "_get", side_effect=lambda self, url, headers=None: chain.get(url, headers),
                               autospec=True):
            result, _ = census.run("http://rpc.invalid", "http://rest.invalid", "zerone-testnet-1", expect, allow)
        return result, {c["id"]: c for c in result["checks"]}

    def assertFails(self, chain, check, text, **kwargs):
        result, checks = self.run_census(chain, **kwargs)
        self.assertFalse(result["passed"])
        self.assertEqual(result["verdict"], "FAIL")
        self.assertFalse(checks[check]["passed"], checks[check])
        self.assertIn(text, checks[check]["detail"])

    def test_live_shaped_testnet_passes(self):
        result, checks = self.run_census(FakeChain())
        self.assertTrue(result["passed"], result)
        self.assertEqual(checks["claiming-pot-units"]["detail"]["headroom_units"], 1_001_000)
        self.assertEqual(checks["liquiditypool-v4-v5"]["detail"]["params_after_migration"]["max_pools"], 16)
        self.assertEqual(result["height"], HEIGHT)

    def test_H1_F4_pot_units_over_the_cap_fail(self):
        chain = FakeChain()
        bootstrap = [("bootstrap-zrn1a", "222000"), ("bootstrap-zrn1b", "222000"), ("bootstrap-zrn1c", "222000")]
        chain.store[("claiming_pot", "subspace", b"\x02")] = pairs(
            [(b"\x02" + i.encode(), pot(i, t)) for i, t in bootstrap + [("pot-1", "222221500000")]])
        chain.store[("claiming_pot", "key", b"\x06")] = (3).to_bytes(8, "big")
        chain.rest["/zerone/claiming_pot/v1/pots"] = {"pots": [{}] * 4}
        self.assertFails(chain, "claiming-pot-units", "above bootstrap_emission_cap_uzrn")

    def test_minted_entries_count_against_the_cap(self):
        chain = FakeChain()
        chain.store[("claiming_pot", "key", b"\x06")] = (1_001_001).to_bytes(8, "big")
        self.assertFails(chain, "claiming-pot-units", "above bootstrap_emission_cap_uzrn")
        chain.store[("claiming_pot", "key", b"\x06")] = (1_001_000).to_bytes(8, "big")
        self.assertTrue(self.run_census(chain)[0]["passed"])

    def test_any_general_pot_needs_review(self):
        chain = FakeChain()
        chain.store[("claiming_pot", "subspace", b"\x02")] = pairs([(b"\x02pot-2", pot("pot-2", "1000"))])
        chain.rest["/zerone/claiming_pot/v1/pots"] = {"pots": [{}]}
        self.assertFails(chain, "claiming-pot-units", "review each before H")

    def test_malformed_pot_amount_fails(self):
        chain = FakeChain()
        for total in ("0", "-5", "1e6", "", "1_000"):
            chain.store[("claiming_pot", "subspace", b"\x02")] = pairs([(b"\x02bootstrap-x", pot("bootstrap-x", total))])
            chain.rest["/zerone/claiming_pot/v1/pots"] = {"pots": [{}]}
            with self.subTest(total=total):
                self.assertFails(chain, "claiming-pot-units", "positive integer")

    def test_verifier_liquiditypool_settings_fail(self):
        for change, text in ((("twapWindowBlocks", "10001"), "twap_window_blocks"),
                             (("billingQuoteDenoms", [f"ibc/{i:03d}" for i in range(33)]), "cannot exceed 32"),
                             (("billingQuoteDenoms", ["uzrn"]), "uzrn itself"),
                             (("billingQuoteDenoms", ["ibc/abc", "ibc/abc"]), "duplicate"),
                             (("billingQuoteDenoms", ["9bad"]), "invalid denom"),
                             (("defaultSwapFeeBps", "100001"), "default_swap_fee_bps"),
                             (("minInitialLiquidity", "010"), "canonical"),
                             (("minReserve", "-1"), "non-negative"),
                             (("poolCreators", [FAUCET[:-1] + "x"]), "pool creator")):
            chain = FakeChain()
            chain.rest["/zerone/liquiditypool/v1/params"]["params"][change[0]] = change[1]
            with self.subTest(change=change):
                self.assertFails(chain, "liquiditypool-v4-v5", text)
        chain = FakeChain()
        chain.rest["/zerone/liquiditypool/v1/params"]["params"].update(twapWindowBlocks="10000", maxPools="999",
                                                                        poolCreators=[FAUCET])
        self.assertTrue(self.run_census(chain)[0]["passed"])

    def test_pools_and_orphan_records_fail_closed(self):
        chain = FakeChain()
        chain.rest["/zerone/liquiditypool/v1/pools"] = {"pools": [
            {"poolId": "pool-1", "reserveA": "5", "reserveB": "0", "lpTokenSupply": "5"}]}
        chain.store[("liquiditypool", "subspace", b"\x01")] = pairs([(b"\x01pool-1", b"x")])
        self.assertFails(chain, "liquiditypool-v4-v5", "inconsistent reserve/supply zero state")
        chain.rest["/zerone/liquiditypool/v1/pools"]["pools"][0]["reserveB"] = "5"
        self.assertFails(chain, "liquiditypool-v4-v5", "pools exist")
        for prefix in (b"\x02", b"\x05", b"\x11", b"\x12"):
            chain = FakeChain()
            chain.store[("liquiditypool", "subspace", prefix)] = pairs([(prefix + b"pool-9", b"x")])
            with self.subTest(prefix=prefix):
                self.assertFails(chain, "liquiditypool-v4-v5", "with no pools")
        for counter, text in ((b"\x00", "invalid liquiditypool next pool ID"), ((10_002).to_bytes(8, "big"), "exceeds")):
            chain = FakeChain()
            chain.store[("liquiditypool", "key", b"\x04")] = counter
            with self.subTest(counter=counter):
                self.assertFails(chain, "liquiditypool-v4-v5", text)

    def test_lineage_preconditions(self):
        chain = FakeChain()
        chain.rest["/cosmos/upgrade/v1beta1/module_versions"]["module_versions"][0]["version"] = "9"
        self.assertFails(chain, "version-map", "exact prestate")
        chain = FakeChain()
        chain.store[("knowledge", "subspace", b"\x7f\x01")] = pairs(
            [(b"\x7f\x01upgrade_marker_consolidation-safety-v1", b"migrated")])
        self.assertFails(chain, "markers", "already present")
        chain = FakeChain()
        chain.rest["/cosmos/upgrade/v1beta1/applied_plan/consolidation-safety-v1"] = {"height": "12"}
        self.assertFails(chain, "markers", "done height")

    def test_plan_expectations(self):
        info = census.proposal_tool.canonical_info(IMAGE)
        good = {"name": "consolidation-safety-v1", "height": "3871000", "info": info,
                "time": "0001-01-01T00:00:00Z", "upgraded_client_state": None}
        chain = FakeChain()
        chain.rest["/cosmos/upgrade/v1beta1/current_plan"] = {"plan": good}
        self.assertFails(chain, "plan", "already committed")
        result, _ = self.run_census(chain, expect=(IMAGE, 3_871_000))
        self.assertTrue(result["passed"], result)
        for change in ({"info": ""}, {"height": "3871001"}):
            chain.rest["/cosmos/upgrade/v1beta1/current_plan"] = {"plan": dict(good, **change)}
            with self.subTest(change=change):
                self.assertFails(chain, "plan", "differs", expect=(IMAGE, 3_871_000))
        chain.rest["/cosmos/upgrade/v1beta1/current_plan"] = {"plan": dict(good, height=str(HEIGHT))}
        self.assertFails(chain, "plan", "not before H", expect=(IMAGE, HEIGHT))

    def test_other_open_proposals_are_a_stop_condition(self):
        chain = FakeChain()
        chain.open_proposals["PROPOSAL_STATUS_VOTING_PERIOD"] = [{"id": "5", "title": "H1"}, {"id": "6", "title": "pot"}]
        self.assertFails(chain, "governance-quiet", "other open proposals", allow="5")
        chain.open_proposals["PROPOSAL_STATUS_VOTING_PERIOD"] = [{"id": "5", "title": "H1"}]
        self.assertTrue(self.run_census(chain, allow="5")[0]["passed"])

    def test_bech32_and_protobuf_helpers(self):
        self.assertTrue(census.bech32_account(FAUCET))
        self.assertTrue(census.bech32_account("zrn10d07y265gmmuvt4z0w9aw880jnsr700j47tt89"))
        for bad in (FAUCET.upper(), FAUCET[:-1] + "q", "cosmos1xl5pnczujgqzzcaq8v53acmfyfmj394nws0rk5", "zrn1"):
            self.assertFalse(census.bech32_account(bad), bad)
        self.assertEqual(census.pot_record(pot("pot-7", "123")), {"id": "pot-7", "total_amount": "123"})
        self.assertEqual(census.kv_pairs(pairs([(b"k", b"v")])), [(b"k", b"v")])
        self.assertEqual(census.pot_commitment_units("222001"), 2)
        self.assertEqual(census.general_pot_sequence("pot-0"), None)
        self.assertEqual(census.general_pot_sequence("pot-+1"), None)


if __name__ == "__main__":
    unittest.main(verbosity=2)
