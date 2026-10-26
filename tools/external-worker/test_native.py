"""Offline unit tests: all bank observations and sends are synthetic fixtures.

No live node, keyring, signing, faucet, or payment is used by this suite.
"""
from collections.abc import Callable
import copy
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import stat
import subprocess
import sys
import tempfile
import unittest
from unittest import mock


HERE = Path(__file__).parent
ROOT = HERE.parents[1]
RECIPIENT = 'zrn1rj7z53777w3nwgwgm2lmqlly85ntajplcda70l'
TASK, RESULT, JOURNAL = 'a' * 64, 'b' * 64, 'c' * 64
FUNDING_HASH, PAYMENT_HASH = 'D' * 64, 'E' * 64


def changed(value, path, replacement):
    value = copy.deepcopy(value)
    target = value
    for key in path[:-1]:
        target = target[key]
    if replacement is ...:
        del target[path[-1]]
    else:
        target[path[-1]] = replacement
    return value


def load_module(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    assert spec is not None and spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


shared = load_module('native_test_shared', ROOT / 'scripts/shared-claims.py')
addresses = load_module('native_test_addresses', ROOT / 'deploy/frozen_evidence.py')
SENDER = addresses._bech32_encode('zrn', bytes([1]) * 20)
FUNDER = addresses._bech32_encode('zrn', bytes([2]) * 20)


class BoundedTestBackend:
    """Explicit offline injection; never launches zeroned or contacts an RPC.

    Reuse real SharedWorkflow custody checks, locking, and durable records;
    replace only network observations and signing with bounded synthetic ones.
    """
    def __init__(self):
        self.open_calls = 0
        self.calls = []
        self.responses = {}
        self.before_open: Callable[[], None] | None = None
        self.network_checks = 0
        self.network_error: Exception | None = None
        self.status_height = 10
        self.app_height = 10

    def open(self, home, module, workflow_type):
        self.open_calls += 1
        self.error_type = module.Error
        if self.before_open is not None:
            self.before_open()
        def synthetic_network(_descriptor):
            self.network_checks += 1
            if self.network_error is not None:
                raise self.network_error
            return b'SYNTHETIC TEST NETWORK'
        with mock.patch.object(module, 'verify_network', side_effect=synthetic_network):
            self.workflow = workflow_type(home)
        self.workflow.check_node = self.check_node
        self.workflow.abci_info = lambda: {'response': {'last_block_height': str(self.app_height)}}
        self.workflow.query = self.query
        self.workflow.send = self.send
        self._reconcile = self.workflow.reconcile
        self.workflow.reconcile = self.reconcile
        self.workflow.tx_observation = self.tx_observation
        return self.workflow

    def assert_locked(self):
        try:
            with self.workflow.locked():
                raise AssertionError('Native operation did not hold the real workflow lock')
        except self.error_type:
            pass

    def reconcile(self):
        self.assert_locked()
        self.calls.append(('reconcile',))
        return self._reconcile()

    def tx_observation(self, digest):
        self.assert_locked()
        self.calls.append(('tx_observation', digest))
        return copy.deepcopy(self.responses.get((('tx', digest), None)))

    def check_node(self):
        self.workflow.check_files()
        return self.status_height

    def query(self, *args, height=None):
        self.assert_locked()
        self.calls.append(('query', args, height))
        if len(self.calls) > 64 or (args, height) not in self.responses:
            raise AssertionError('Unconfigured or unbounded synthetic query; no network fallback')
        return copy.deepcopy(self.responses[args, height])

    def send(self, action, actor, arguments, context):
        self.assert_locked()
        self.calls.append(('send', action, actor, arguments, context))
        marker = self.workflow.directory / ('external-work-' + context['task_id'] + '.json')
        reservation = shared.core.read_json(marker)
        if reservation['context'] != context:
            raise AssertionError('Send was not preceded by a durable exact task reservation')
        self.sent_reservation = reservation
        attempt = {**self.workflow.record_metadata(), 'id': 'd' * 32,
                   'action': action, 'actor': actor, 'address': SENDER, 'context': dict(context),
                   'status': 'committed', 'txhash': PAYMENT_HASH,
                   'tx_bytes_base64': 'PRIVATE SYNTHETIC BYTES MUST NOT LEAK',
                   'secret_test_field': 'DO NOT RETURN RAW ATTEMPTS'}
        shared.core.save_json(self.workflow.directory / 'attempts' / (attempt['id'] + '.json'), attempt, fresh=True)
        return copy.deepcopy(attempt)


class NativeTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix='native-unit-', dir=os.environ['TMPDIR'])
        self.addCleanup(self.temp.cleanup)
        self.home = Path(self.temp.name) / 'participant'
        self.home.mkdir(mode=0o700)
        for name in ('bin', 'claim-workflow'):
            (self.home / name).mkdir(mode=0o700)
        self.descriptor = {
            'schema': 'zerone-shared-development/v1', 'chain_id': 'zerone-dev-1',
            'knowledge_version': 11, 'denom': 'uzrn', 'commitment_scheme': 2,
            'review_policy_version': 1, 'account_types': ['human', 'agent'],
            'gas_limit': 2000000, 'tx_fee_uzrn': '2000000',
            'bootstrap_consensus': 'single-operator', 'reset_policy': 'new-chain-id',
            'local_test': True, 'source_commit': 'a' * 40,
            'rpc_url': 'http://127.0.0.1:12345',
            'genesis_url': 'http://127.0.0.1:12345/genesis.json',
            'faucet_url': 'http://127.0.0.1:12345/faucet',
            'genesis_sha256': hashlib.sha256(b'SYNTHETIC GENESIS').hexdigest(),
            'rpc_genesis_sha256': 'f' * 64,
        }
        self.manifest = {
            'schema': shared.SCHEMA, 'chain_id': shared.CHAIN, 'owner_uid': os.getuid(),
            'address': SENDER, 'allow_loopback_test': True,
            'genesis_sha256': self.descriptor['genesis_sha256'],
            'descriptor_sha256': shared.digest(shared.canonical(self.descriptor)),
            'binary_sha256': shared.digest(b'NOT AN EXECUTABLE; UNIT TEST ONLY'),
            'darwin_acl_helper_sha256': shared.digest(b'UNIT TEST HELPER'),
            'client_files': {name: shared.digest((ROOT / 'scripts' / name).read_bytes())
                             for name in ('shared-claims.py', 'claim-workflow.py', 'local-node.py')},
        }
        shared.write_bytes(self.home / 'network.json', shared.canonical(self.descriptor))
        shared.write_bytes(self.home / 'pinned-genesis.json', b'SYNTHETIC GENESIS')
        shared.write_bytes(self.home / 'bin/zeroned', b'NOT AN EXECUTABLE; UNIT TEST ONLY')
        shared.write_bytes(self.home / 'bin/darwin-acl-check', b'UNIT TEST HELPER')
        shared.core.save_json(self.home / shared.MARKER, self.manifest, fresh=True)
        self.native = load_module('native_under_test', HERE / 'native.py')
        self.backend = BoundedTestBackend()
        for target in ('socket.socket', 'subprocess.run'):
            blocker = mock.patch(target, side_effect=AssertionError('Offline native unit tests prohibit network and processes'))
            blocker.start()
            self.addCleanup(blocker.stop)

    def persist_controls(self):
        shared.core.save_json(self.home / shared.MARKER, self.manifest)
        shared.core.save_json(self.home / 'network.json', self.descriptor)

    def payer(self):
        return self.native.NativePayer(self.home, _test_backend=self.backend)

    def reward(self):
        return {'chain_id': shared.CHAIN, 'genesis_sha256': self.manifest['genesis_sha256'],
                'recipient': RECIPIENT, 'amount_uzrn': '1000000', 'funding_txhash': FUNDING_HASH}

    def configure_funding(self):
        self.funding_tx = {
            'txhash': FUNDING_HASH, 'height': '5', 'code': 0,
            'tx': {'body': {'messages': [{'@type': '/cosmos.bank.v1beta1.MsgSend',
                    'from_address': FUNDER, 'to_address': SENDER,
                    'amount': [{'denom': 'uzrn', 'amount': '10000000'}]}], 'memo': 'SYNTHETIC UNIT TEST FUNDING'}},
        }
        self.backend.responses[(('tx', FUNDING_HASH), None)] = self.funding_tx
        self.backend.responses[(('bank', 'balances', SENDER), 10)] = {
            'balances': [{'denom': 'uzrn', 'amount': '10000000'}],
            'pagination': {'next_key': None, 'total': '0'},
        }

    def test_funding_reads_exact_committed_send_and_current_budget(self):
        self.configure_funding()
        payer = self.payer()
        self.assertTrue(hasattr(payer, 'funding'), 'Funding readback is not implemented')
        receipt = payer.funding(self.reward())
        self.assertEqual(receipt['tx'], self.funding_tx)
        self.assertEqual(receipt['txhash'], FUNDING_HASH)
        self.assertEqual(receipt['height'], 5)
        self.assertEqual(receipt['payer'], SENDER)
        self.assertEqual(receipt['amount_uzrn'], '10000000')
        self.assertEqual(receipt['balance_uzrn'], '10000000')
        self.assertEqual(receipt['balance_height'], 10)
        self.assertTrue(receipt['balance_sufficient'])
        self.assertEqual(self.backend.calls, [
            ('query', ('tx', FUNDING_HASH), None),
            ('query', ('bank', 'balances', SENDER), 10),
        ])

    def test_funding_pins_balance_to_commit_when_status_leads(self):
        self.configure_funding()
        self.backend.status_height = 11
        payer = self.payer()
        with mock.patch.object(payer.workflow, 'check_node', wraps=payer.workflow.check_node) as check:
            receipt = payer.funding(self.reward())
        check.assert_called_once_with()
        self.assertEqual(receipt['balance_height'], 10)
        self.assertEqual(self.backend.calls, [
            ('query', ('tx', FUNDING_HASH), None),
            ('query', ('bank', 'balances', SENDER), 10),
        ])

    def test_sdk_omits_empty_pagination_fields(self):
        # Shapes observed from the actual SDK CLI; values remain synthetic.
        self.configure_funding()
        payer = self.payer()
        key = (('bank', 'balances', SENDER), 10)
        self.backend.responses[key]['pagination'] = {'total': '1'}
        self.assertTrue(payer.funding(self.reward())['balance_sufficient'])
        self.assertEqual(payer._bank_amount({'balances': [], 'pagination': {}}, 'balances'), 0)
        self.backend.responses[key]['pagination'] = {}
        self.assertTrue(payer.funding(self.reward())['balance_sufficient'])
        self.assertFalse(any(row[0] == 'send' for row in self.backend.calls))

    def test_nonzero_pagination_total_must_match_complete_coin_list(self):
        payer = self.payer()
        for field in ('balances', 'supply'):
            with self.subTest(field=field), self.assertRaises(self.native.Error):
                payer._bank_amount({field: [{'denom': 'uzrn', 'amount': '1000000'}],
                                    'pagination': {'total': '2'}}, field)

    def test_reward_is_closed_exact_and_validated_before_queries(self):
        self.configure_funding()
        payer = self.payer()
        invalid = [{**self.reward(), 'extra': True}, None, {}]
        for key in self.reward():
            invalid.append({name: value for name, value in self.reward().items() if name != key})
        for field, values in {
            'chain_id': ['zerone-1', 'zerone-2', 'zerone-dev-1 '],
            'genesis_sha256': ['0' * 64, self.manifest['genesis_sha256'].upper(), None],
            'funding_txhash': [FUNDING_HASH.lower(), 'D' * 63, 'D' * 65, None],
            'amount_uzrn': ['0', '01', '-1', '+1', '1.0', '1e6', '100000001', '1\n', 1, True, None],
            'recipient': [RECIPIENT.upper(), RECIPIENT + ' ', RECIPIENT[:-1] + 'q',
                          addresses._bech32_encode('zrn', bytes(21)),
                          addresses._bech32_encode('zrn', bytes(19)), SENDER, None],
        }.items():
            invalid.extend({**self.reward(), field: value} for value in values)
        for reward in invalid:
            with self.subTest(reward=reward):
                self.backend.calls.clear()
                with self.assertRaises(self.native.Error):
                    payer.funding(reward)
                self.assertEqual(self.backend.calls, [], 'Invalid reward reached the backend')

    def test_funding_rejects_nonexact_or_unsuccessful_transfer(self):
        self.configure_funding()
        payer = self.payer()
        message = ('tx', 'body', 'messages', 0)
        cases = [
            (('txhash',), FUNDING_HASH.lower()), (('txhash',), 'F' * 64),
            (('txhash',), ...), (('code',), 1), (('code',), ...), (('code',), False),
            (('height',), '0'), (('height',), '01'), (('height',), ...),
            (('tx',), ...), (('tx', 'body', 'messages'), []),
            (('tx', 'body', 'messages'), self.funding_tx['tx']['body']['messages'] * 2),
            (message + ('@type',), '/not.bank.MsgSend'),
            (message + ('to_address',), RECIPIENT),
            (message + ('from_address',), SENDER),
            (message + ('from_address',), ...),
            (message + ('amount',), [{'denom': 'uzrn', 'amount': '2999999'}]),
            (message + ('amount',), [{'denom': 'uzrn', 'amount': '010000000'}]),
            (message + ('amount',), [{'denom': 'other', 'amount': '10000000'}]),
            (message + ('amount',), [{'denom': 'uzrn', 'amount': 10000000}]),
            (message + ('extra',), 'unrecognized'),
        ]
        for path, replacement in cases:
            with self.subTest(path=path, replacement=replacement):
                self.backend.calls.clear()
                self.backend.responses[(('tx', FUNDING_HASH), None)] = changed(self.funding_tx, path, replacement)
                with self.assertRaises(self.native.Error):
                    payer.funding(self.reward())
                self.assertFalse(any(row[0] == 'send' for row in self.backend.calls))

    def test_funding_requires_complete_canonical_current_balance(self):
        self.configure_funding()
        payer = self.payer()
        key = (('bank', 'balances', SENDER), 10)
        original = copy.deepcopy(self.backend.responses[key])
        cases = [
            (('balances', 0, 'amount'), '2999999'),
            (('balances', 0, 'amount'), '-1'), (('balances', 0, 'amount'), '010000000'),
            (('balances', 0, 'amount'), 10000000), (('balances', 0, 'amount'), True),
            (('balances', 0, 'amount'), '9' * 79),
            (('balances', 0, 'extra'), 'unknown'),
            (('balances',), original['balances'] * 2),
            (('balances',), []), (('balances',), ...),
            (('pagination',), ...), (('pagination',), None),
            (('pagination', 'next_key'), 'MORE'), (('pagination', 'next_key'), False),
            (('pagination', 'total'), '00'), (('pagination', 'unexpected'), 1),
        ]
        for path, replacement in cases:
            with self.subTest(path=path, replacement=replacement):
                self.backend.calls.clear()
                self.backend.responses[key] = changed(original, path, replacement)
                with self.assertRaises(self.native.Error):
                    payer.funding(self.reward())
        # The exact boundary is allowed; no arbitrary minimum above reward+fee.
        self.backend.responses[key] = changed(original, ('balances', 0, 'amount'), '3000000')
        self.assertTrue(payer.funding(self.reward())['balance_sufficient'])

    def configure_payment(self):
        self.configure_funding()
        self.payment_tx = {
            'txhash': PAYMENT_HASH, 'height': '12', 'code': 0,
            'gas_used': '100000', 'gas_wanted': '2000000',
            'tx': {'body': {'messages': [{'@type': '/cosmos.bank.v1beta1.MsgSend',
                    'from_address': SENDER, 'to_address': RECIPIENT,
                    'amount': [{'denom': 'uzrn', 'amount': '1000000'}]}],
                    'memo': 'zerone-work/v1 ' + ' '.join((TASK, RESULT, JOURNAL))},
                   'auth_info': {'fee': {'amount': [{'denom': 'uzrn', 'amount': '2000000'}]}}},
        }
        self.backend.responses[(('tx', PAYMENT_HASH), None)] = self.payment_tx
        for height, sender, recipient in ((11, '10000000', '0'), (12, '7000000', '1000000')):
            for address, amount in ((SENDER, sender), (RECIPIENT, recipient)):
                self.backend.responses[(('bank', 'balances', address), height)] = {
                    'balances': [] if amount == '0' else [{'denom': 'uzrn', 'amount': amount}],
                    'pagination': {'next_key': None, 'total': '0'}}
            self.backend.responses[(('bank', 'total'), height)] = {
                'supply': [{'denom': 'uzrn', 'amount': '1000000000'}],
                'pagination': {'next_key': None, 'total': '0'}}

    def test_payment_uses_existing_send_and_verifies_public_tx_with_historical_bank(self):
        self.configure_payment()
        payer = self.payer()
        self.assertTrue(hasattr(payer, 'pay'), 'Native payment is not implemented')
        evidence = payer.pay(TASK, self.reward(), RESULT, JOURNAL)
        self.assertEqual(evidence['txhash'], PAYMENT_HASH)
        self.assertEqual(evidence['height'], 12)
        self.assertEqual(evidence['amount_uzrn'], '1000000')
        self.assertEqual(evidence['fee_uzrn'], '2000000')
        self.assertEqual(evidence['recipient'], RECIPIENT)
        self.assertTrue(evidence['bank_deltas_verified'])
        self.assertEqual(evidence['before'], {'sender': 10000000, 'recipient': 0, 'supply': 1000000000})
        self.assertEqual(evidence['after'], {'sender': 7000000, 'recipient': 1000000, 'supply': 1000000000})
        self.assertEqual(evidence['assurance'], 'PINNED_OPERATOR_CHAIN_OBSERVATIONS_NOT_INDEPENDENT_INCLUSION_PROOF')
        self.assertFalse(evidence['recovered'])
        self.assertEqual(evidence['tx'], self.payment_tx)
        self.assertEqual(evidence['funding']['tx'], self.funding_tx)
        expected_context = {'task_id': TASK, 'result_sha256': RESULT, 'journal_sha256': JOURNAL}
        send = [row for row in self.backend.calls if row[0] == 'send']
        self.assertEqual(send, [('send', 'external-work', shared.ACTOR,
            ['bank', 'send', shared.ACTOR, RECIPIENT, '1000000uzrn', '--note',
             self.payment_tx['tx']['body']['memo']], expected_context)])
        self.assertEqual(self.backend.calls[0], ('reconcile',))
        self.assertEqual(self.backend.sent_reservation['context'], expected_context)
        self.assertEqual(self.backend.sent_reservation['reward'], self.reward())
        self.assertEqual(self.backend.sent_reservation['funding']['tx'], self.funding_tx)
        marker = self.home / 'claim-workflow' / ('external-work-' + TASK + '.json')
        self.assertEqual(stat.S_IMODE(marker.stat().st_mode), 0o600)
        self.assertEqual(marker.stat().st_nlink, 1)
        for height in (11, 12):
            for args in (('bank', 'balances', SENDER), ('bank', 'balances', RECIPIENT), ('bank', 'total')):
                self.assertIn(('query', args, height), self.backend.calls)
        self.assertNotIn('PRIVATE SYNTHETIC', json.dumps(evidence))
        self.assertNotIn('secret_test_field', json.dumps(evidence))
        with self.backend.workflow.locked():
            pass  # The single lock has also been released after readback.

    def test_duplicate_recovers_without_send_even_after_budget_is_spent(self):
        self.configure_payment()
        first = self.payer().pay(TASK, self.reward(), RESULT, JOURNAL)
        marker = self.home / 'claim-workflow' / ('external-work-' + TASK + '.json')
        reserved = marker.read_bytes()
        balance = self.backend.responses[(('bank', 'balances', SENDER), 10)]
        balance['balances'] = []  # Synthetic current funds have since been spent.
        self.backend.calls.clear()
        try:
            second = self.payer().pay(TASK, self.reward(), RESULT, JOURNAL)
        except self.native.Error as error:
            self.fail('Committed recovery must not demand a fresh payment budget: ' + str(error))
        self.assertEqual(second['txhash'], first['txhash'])
        self.assertEqual(second['before'], first['before'])
        self.assertEqual(second['after'], first['after'])
        self.assertTrue(second['recovered'])
        self.assertFalse(second['funding']['balance_sufficient'])
        self.assertEqual(second['funding']['tx'], self.funding_tx)
        self.assertEqual(marker.read_bytes(), reserved)
        self.assertEqual(self.backend.calls[0], ('reconcile',))
        self.assertIn(('query', ('tx', PAYMENT_HASH), None), self.backend.calls)
        self.assertFalse(any(row[0] == 'send' for row in self.backend.calls))

    def test_pre_attempt_reservation_refuses_without_a_new_send_or_funding_query(self):
        self.configure_payment()
        payer = self.payer()
        marker = self.home / 'claim-workflow' / ('external-work-' + TASK + '.json')
        reservation = {'schema': 'zerone-external-work-reservation/v1', 'payer': SENDER,
            'context': {'task_id': TASK, 'result_sha256': RESULT, 'journal_sha256': JOURNAL},
            'reward': self.reward(), 'funding': {'tx': self.funding_tx}}
        shared.core.save_json(marker, reservation, fresh=True)
        original = marker.read_bytes()
        for expected in (None, PAYMENT_HASH):
            with self.subTest(expected=expected):
                self.backend.calls.clear()
                with self.assertRaisesRegex(self.native.Error, 'pre-attempt.*operator|Recovery-only'):
                    payer.pay(TASK, self.reward(), RESULT, JOURNAL, expected_txhash=expected)
                self.assertEqual(marker.read_bytes(), original)
                self.assertEqual(self.backend.calls, [('reconcile',)])
                self.assertEqual(self.backend.workflow.records('attempts'), [])

    def test_recovery_refuses_conflicting_reserved_intent_before_queries(self):
        self.configure_payment()
        payer = self.payer()
        payer.pay(TASK, self.reward(), RESULT, JOURNAL)
        marker = self.home / 'claim-workflow' / ('external-work-' + TASK + '.json')
        reservation = shared.core.read_json(marker)
        for path, value in [
            (('context', 'result_sha256'), 'f' * 64),
            (('context', 'journal_sha256'), 'f' * 64),
            (('context', 'task_id'), 'f' * 64),
            (('reward', 'amount_uzrn'), '2'),
            (('reward', 'funding_txhash'), 'F' * 64),
            (('payer',), RECIPIENT), (('schema',), 'unknown'),
            (('funding',), ...), (('funding',), None), (('extra',), True),
        ]:
            with self.subTest(path=path):
                shared.core.save_json(marker, changed(reservation, path, value))
                self.backend.calls.clear()
                with self.assertRaisesRegex(self.native.Error, 'reservation|intent'):
                    payer.pay(TASK, self.reward(), RESULT, JOURNAL)
                self.assertEqual(self.backend.calls, [('reconcile',)])

    def store_attempt(self, *, context=None, status='committed', action='external-work', identifier='d' * 32):
        context = context or {'task_id': TASK, 'result_sha256': RESULT, 'journal_sha256': JOURNAL}
        attempt = {**self.backend.workflow.record_metadata(), 'id': identifier,
                   'action': action, 'actor': shared.ACTOR, 'address': SENDER,
                   'context': context, 'status': status, 'txhash': PAYMENT_HASH}
        path = self.home / 'claim-workflow/attempts' / (identifier + '.json')
        shared.core.save_json(path, attempt, fresh=not path.exists())
        return path

    def test_unrelated_pending_attempt_blocks_new_reservation_and_send(self):
        self.configure_payment()
        payer = self.payer()
        del self.backend.responses[(('tx', PAYMENT_HASH), None)]
        self.store_attempt(context={'task_id': 'f' * 64}, status='pending', action='submit')
        with self.assertRaisesRegex(self.native.Error, 'exact-hash'):
            payer.pay(TASK, self.reward(), RESULT, JOURNAL)
        self.assertFalse(any(row[0] == 'send' for row in self.backend.calls))
        self.assertFalse((self.home / 'claim-workflow' / ('external-work-' + TASK + '.json')).exists())
        self.assertEqual(self.backend.calls, [('reconcile',), ('tx_observation', PAYMENT_HASH)])

    def test_descriptor_genesis_must_match_manifest_before_backend_opens(self):
        self.descriptor['genesis_sha256'] = 'e' * 64
        self.manifest['descriptor_sha256'] = shared.digest(shared.canonical(self.descriptor))
        shared.core.save_json(self.home / shared.MARKER, self.manifest)
        (self.home / 'network.json').write_bytes(shared.canonical(self.descriptor))
        with self.assertRaisesRegex(self.native.Error, 'genesis'):
            self.payer()
        self.assertEqual(self.backend.open_calls, 0)

    def test_missing_uzrn_supply_cannot_be_reported_as_verified_conservation(self):
        self.configure_payment()
        payer = self.payer()
        self.store_attempt()
        for height in (11, 12):
            self.backend.responses[(('bank', 'total'), height)]['supply'] = []
        with self.assertRaisesRegex(self.native.Error, 'supply'):
            payer.pay(TASK, self.reward(), RESULT, JOURNAL)
        self.assertFalse(any(row[0] == 'send' for row in self.backend.calls))

    def test_payment_never_treats_height_zero_as_a_historical_snapshot(self):
        self.configure_payment()
        payer = self.payer()
        self.store_attempt()
        self.backend.responses[(('tx', PAYMENT_HASH), None)]['height'] = '1'
        for new, old in ((0, 11), (1, 12)):
            for args in (('bank', 'balances', SENDER), ('bank', 'balances', RECIPIENT), ('bank', 'total')):
                self.backend.responses[(args, new)] = self.backend.responses[(args, old)]
        with self.assertRaisesRegex(self.native.Error, 'height'):
            payer.pay(TASK, self.reward(), RESULT, JOURNAL)
        self.assertFalse(any(row[0] == 'query' and row[-1] == 0 for row in self.backend.calls))

    def test_missing_or_implicit_home_fails_cleanly_without_initializing_anything(self):
        missing = Path(self.temp.name) / 'never-created'
        for home in (None, '', 'relative-participant', missing):
            with self.subTest(home=home):
                with self.assertRaises(Exception) as caught:
                    self.native.NativePayer(home, _test_backend=self.backend)
                self.assertIsInstance(caught.exception, self.native.Error)
                self.assertFalse(missing.exists())
                self.assertEqual(self.backend.open_calls, 0)
                self.assertEqual(self.backend.calls, [])

    def test_changed_custody_controls_refuse_funding_before_queries(self):
        self.configure_funding()
        payer = self.payer()
        self.manifest['allow_loopback_test'] = False
        shared.core.save_json(self.home / shared.MARKER, self.manifest)
        with self.assertRaises(Exception) as caught:
            payer.funding(self.reward())
        self.assertIsInstance(caught.exception, self.native.Error)
        self.assertEqual(self.backend.calls, [])

    def test_pending_failed_or_rejected_task_attempts_never_create_another_payment(self):
        self.configure_payment()
        payer = self.payer()
        del self.backend.responses[(('tx', PAYMENT_HASH), None)]
        for status in ('prepared', 'broadcasting', 'pending', 'failed', 'rejected', 'committed-unresolved'):
            for expected in (None, PAYMENT_HASH):
                with self.subTest(status=status, expected=expected):
                    path = self.store_attempt(status=status)
                    before = path.read_bytes()
                    self.backend.calls.clear()
                    with self.assertRaisesRegex(self.native.Error, 'exact-hash'):
                        payer.pay(TASK, self.reward(), RESULT, JOURNAL, expected_txhash=expected)
                    self.assertFalse(any(row[0] in ('query', 'send') for row in self.backend.calls))
                    self.assertFalse((self.home / 'claim-workflow' / ('external-work-' + TASK + '.json')).exists())
                    self.assertEqual(path.read_bytes(), before)

    def test_conflicting_or_duplicate_attempt_contexts_refuse_before_queries(self):
        self.configure_payment()
        payer = self.payer()
        context = {'task_id': TASK, 'result_sha256': RESULT, 'journal_sha256': JOURNAL}
        for different in ({**context, 'result_sha256': 'f' * 64},
                          {**context, 'journal_sha256': 'f' * 64},
                          {**context, 'extra': True}):
            with self.subTest(context=different):
                self.store_attempt(context=different)
                self.backend.calls.clear()
                with self.assertRaisesRegex(self.native.Error, 'Conflicting'):
                    payer.pay(TASK, self.reward(), RESULT, JOURNAL)
                self.assertEqual(self.backend.calls, [('reconcile',)])
        self.store_attempt(context=context)
        self.store_attempt(context=context, identifier='e' * 32)
        self.backend.calls.clear()
        with self.assertRaisesRegex(self.native.Error, 'Conflicting'):
            payer.pay(TASK, self.reward(), RESULT, JOURNAL)
        self.assertEqual(self.backend.calls, [('reconcile',)])

    def test_reconciliation_recovers_a_pending_attempt_that_has_actually_committed(self):
        self.configure_payment()
        payer = self.payer()
        path = self.store_attempt(status='pending')
        evidence = payer.pay(TASK, self.reward(), RESULT, JOURNAL)
        self.assertTrue(evidence['recovered'])
        self.assertEqual(shared.core.read_json(path)['status'], 'committed')
        self.assertEqual(self.backend.calls[:2], [('reconcile',), ('tx_observation', PAYMENT_HASH)])
        self.assertIn(('query', ('tx', PAYMENT_HASH), None), self.backend.calls)
        self.assertFalse(any(row[0] == 'send' for row in self.backend.calls))

    def test_readback_failure_retains_attempt_then_recovers_without_reissue(self):
        self.configure_payment()
        payer = self.payer()
        key = (('bank', 'balances', RECIPIENT), 12)
        self.backend.responses[key]['balances'][0]['amount'] = '1000001'
        with self.assertRaisesRegex(self.native.Error, 'bank changes'):
            payer.pay(TASK, self.reward(), RESULT, JOURNAL)
        self.assertEqual(len([row for row in self.backend.calls if row[0] == 'send']), 1)
        self.assertEqual(len(self.backend.workflow.records('attempts')), 1)
        self.backend.responses[key]['balances'][0]['amount'] = '1000000'
        self.backend.calls.clear()
        evidence = self.payer().pay(TASK, self.reward(), RESULT, JOURNAL)
        self.assertTrue(evidence['recovered'])
        self.assertFalse(any(row[0] == 'send' for row in self.backend.calls))

    def test_expected_txhash_is_exact_and_validated_before_workflow_operations(self):
        self.configure_payment()
        payer = self.payer()
        for value in ('', PAYMENT_HASH.lower(), 'E' * 63, 'E' * 65, 'G' * 64,
                      PAYMENT_HASH + '\n', ' ' + PAYMENT_HASH, 1, True, [], {}, b'E' * 64):
            with self.subTest(value=value):
                self.backend.calls.clear()
                with mock.patch.object(payer.workflow, 'locked', wraps=payer.workflow.locked) as locked:
                    with self.assertRaises(self.native.Error):
                        payer.pay(TASK, self.reward(), RESULT, JOURNAL, expected_txhash=value)
                    locked.assert_not_called()
                self.assertEqual(self.backend.calls, [])
                self.assertFalse((self.home / 'claim-workflow/.lock').exists())
                self.assertFalse((self.home / 'claim-workflow' / ('external-work-' + TASK + '.json')).exists())

    def test_invalid_payment_hashes_refuse_before_lock_queries_or_send(self):
        self.configure_payment()
        payer = self.payer()
        for index in range(3):
            for value in (None, 'A' * 64, 'a' * 63, 'a' * 65, 'a' * 64 + '\n', '../' + 'a' * 64):
                hashes: list[object] = [TASK, RESULT, JOURNAL]
                hashes[index] = value
                with self.subTest(index=index, value=value):
                    with self.assertRaises(self.native.Error):
                        payer.pay(hashes[0], self.reward(), hashes[1], hashes[2])
                    self.assertEqual(self.backend.calls, [])
                    self.assertFalse((self.home / 'claim-workflow/.lock').exists())

    def test_pay_rechecks_test_mode_before_reconcile_or_send(self):
        self.configure_payment()
        payer = self.payer()
        payer.manifest['allow_loopback_test'] = False
        for expected in (None, PAYMENT_HASH):
            with self.subTest(expected=expected):
                with self.assertRaisesRegex(self.native.Error, 'loopback'):
                    payer.pay(TASK, self.reward(), RESULT, JOURNAL, expected_txhash=expected)
                self.assertEqual(self.backend.calls, [])

    def test_recovery_only_rechecks_custody_before_reconcile(self):
        self.configure_payment()
        payer = self.payer()
        self.store_attempt()
        shared.core.save_json(self.home / shared.MARKER, {**self.manifest, 'address': FUNDER})
        with self.assertRaises(self.native.Error):
            payer.pay(TASK, self.reward(), RESULT, JOURNAL, expected_txhash=PAYMENT_HASH)
        self.assertEqual(self.backend.calls, [])
        self.assertFalse((self.home / 'claim-workflow' / ('external-work-' + TASK + '.json')).exists())

    def test_reward_cap_accepts_the_exact_limit(self):
        self.configure_funding()
        self.funding_tx['tx']['body']['messages'][0]['amount'][0]['amount'] = '102000000'
        self.backend.responses[(('bank', 'balances', SENDER), 10)]['balances'][0]['amount'] = '102000000'
        receipt = self.payer().funding({**self.reward(), 'amount_uzrn': '100000000'})
        self.assertTrue(receipt['balance_sufficient'])

    def test_control_rebind_between_preflight_and_workflow_open_never_queries_hosted_service(self):
        def rebind_to_hosted():
            self.manifest['allow_loopback_test'] = False
            self.descriptor.update(local_test=False, rpc_url=shared.ORIGIN,
                                   genesis_url=shared.ORIGIN + '/genesis.json',
                                   faucet_url=shared.ORIGIN + '/faucet')
            self.manifest['descriptor_sha256'] = shared.digest(shared.canonical(self.descriptor))
            shared.core.save_json(self.home / shared.MARKER, self.manifest)
            (self.home / 'network.json').write_bytes(shared.canonical(self.descriptor))
        self.backend.before_open = rebind_to_hosted
        with self.assertRaisesRegex(self.native.Error, 'changed|binding|loopback'):
            self.payer()
        self.assertEqual(self.backend.network_checks, 0)
        self.assertEqual(self.backend.calls, [])

    def test_malformed_manifest_is_a_clean_refusal(self):
        shared.core.save_json(self.home / shared.MARKER, [])
        with self.assertRaises(Exception) as caught:
            self.payer()
        self.assertIsInstance(caught.exception, self.native.Error)
        self.assertEqual(self.backend.open_calls, 0)

    def test_malformed_external_work_contexts_fail_closed_instead_of_being_ignored(self):
        self.configure_payment()
        payer = self.payer()
        path = self.store_attempt()
        original = shared.core.read_json(path)
        for context in ({}, None, [], {'task_id': TASK}, {'task_id': 'a' * 63, 'result_sha256': RESULT, 'journal_sha256': JOURNAL}):
            with self.subTest(context=context):
                shared.core.save_json(path, {**original, 'context': context})
                self.backend.calls.clear()
                with self.assertRaisesRegex(self.native.Error, 'Malformed|malformed'):
                    payer.pay(TASK, self.reward(), RESULT, JOURNAL)
                self.assertEqual(self.backend.calls, [('reconcile',)])
                self.assertFalse((self.home / 'claim-workflow' / ('external-work-' + TASK + '.json')).exists())

    def test_malformed_decoded_payment_body_is_a_clean_refusal(self):
        self.configure_payment()
        payer = self.payer()
        self.store_attempt()
        self.backend.responses[(('tx', PAYMENT_HASH), None)]['tx']['body'] = None
        with self.assertRaises(Exception) as caught:
            payer.pay(TASK, self.reward(), RESULT, JOURNAL)
        self.assertIsInstance(caught.exception, self.native.Error)
        self.assertFalse(any(row[0] == 'send' for row in self.backend.calls))

    def test_reservation_fsync_failure_never_reaches_send_and_stays_closed(self):
        self.configure_payment()
        payer = self.payer()
        with mock.patch.object(payer.core.os, 'fsync', side_effect=OSError('SYNTHETIC durability failure')):
            with self.assertRaisesRegex(self.native.Error, 'durability'):
                payer.pay(TASK, self.reward(), RESULT, JOURNAL)
        self.assertFalse(any(row[0] == 'send' for row in self.backend.calls))
        self.backend.calls.clear()
        with self.assertRaisesRegex(self.native.Error, 'pre-attempt.*operator'):
            payer.pay(TASK, self.reward(), RESULT, JOURNAL)
        self.assertEqual(self.backend.calls, [('reconcile',)])

    def test_private_reservation_links_permissions_and_corruption_are_refused(self):
        self.configure_payment()
        payer = self.payer()
        payer.pay(TASK, self.reward(), RESULT, JOURNAL)
        marker = self.home / 'claim-workflow' / ('external-work-' + TASK + '.json')
        original = marker.read_bytes()
        target = self.home / 'claim-workflow/private-fixture.json'
        shared.write_bytes(target, original)
        for kind in ('symlink', 'hardlink', 'public', 'corrupt'):
            with self.subTest(kind=kind):
                marker.unlink()
                if kind == 'symlink':
                    marker.symlink_to(target)
                elif kind == 'hardlink':
                    os.link(target, marker)
                else:
                    shared.write_bytes(marker, b'{' if kind == 'corrupt' else original,
                                       mode=0o644 if kind == 'public' else 0o600)
                self.backend.calls.clear()
                with self.assertRaises(self.native.Error):
                    payer.pay(TASK, self.reward(), RESULT, JOURNAL)
                self.assertEqual(self.backend.calls, [('reconcile',)])

    def test_recovered_decoded_transaction_must_match_exact_payment_and_fee(self):
        self.configure_payment()
        payer = self.payer()
        self.store_attempt()
        original = copy.deepcopy(self.payment_tx)
        message = ('tx', 'body', 'messages', 0)
        cases = [(('txhash',), PAYMENT_HASH.lower()), (('height',), '0'),
            (('code',), ...), (('code',), 5), (('tx',), None),
            (('tx', 'body', 'memo'), 'other task'),
            (('tx', 'body', 'messages'), original['tx']['body']['messages'] * 2),
            (message + ('from_address',), FUNDER), (message + ('to_address',), FUNDER),
            (message + ('amount', 0, 'amount'), '999999'),
            (('tx', 'auth_info', 'fee', 'amount', 0, 'amount'), '1999999')]
        for path, value in cases:
            with self.subTest(path=path):
                self.backend.calls.clear()
                self.backend.responses[(('tx', PAYMENT_HASH), None)] = changed(original, path, value)
                with self.assertRaises(self.native.Error):
                    payer.pay(TASK, self.reward(), RESULT, JOURNAL)
                self.assertFalse(any(row[0] == 'send' for row in self.backend.calls))

    def test_recovered_bank_supply_must_be_complete_and_unchanged(self):
        self.configure_payment()
        payer = self.payer()
        self.store_attempt()
        key = (('bank', 'total'), 12)
        original = copy.deepcopy(self.backend.responses[key])
        for path, value in [(('pagination',), ...), (('pagination', 'next_key'), 'MORE'),
                            (('supply', 0, 'amount'), '1000000001')]:
            with self.subTest(path=path):
                self.backend.calls.clear()
                self.backend.responses[key] = changed(original, path, value)
                with self.assertRaises(self.native.Error):
                    payer.pay(TASK, self.reward(), RESULT, JOURNAL)
                self.assertFalse(any(row[0] == 'send' for row in self.backend.calls))

    def test_only_explicit_loopback_participant_opens_backend(self):
        self.assertIn('__init__', self.native.NativePayer.__dict__, 'Native home guard is not implemented')
        cases = [
            ({'allow_loopback_test': False}, {}),
            ({'allow_loopback_test': 1}, {}),
            ({}, {'local_test': False}),
            ({}, {'local_test': 1}),
            ({'chain_id': 'zerone-1'}, {}),
            ({}, {'chain_id': 'zerone-2'}),
            ({}, {'rpc_url': 'https://zerone-dev-1.fly.dev'}),
            ({}, {'rpc_url': 'http://localhost:12345'}),
            ({}, {'rpc_url': 'http://127.0.0.1:12345@evil.invalid'}),
        ]
        original_manifest, original_descriptor = copy.deepcopy(self.manifest), copy.deepcopy(self.descriptor)
        for manifest_change, descriptor_change in cases:
            with self.subTest(manifest=manifest_change, descriptor=descriptor_change):
                self.manifest = {**original_manifest, **manifest_change}
                self.descriptor = {**original_descriptor, **descriptor_change}
                self.persist_controls()
                with self.assertRaisesRegex(self.native.Error, 'loopback|chain|local.test'):
                    self.payer()
                self.assertEqual(self.backend.open_calls, 0)
                self.assertEqual(self.backend.calls, [])
        self.manifest, self.descriptor = original_manifest, original_descriptor
        self.persist_controls()
        # Restore the descriptor's exact pinned bytes, not a reserialized equivalent.
        (self.home / 'network.json').write_bytes(shared.canonical(self.descriptor))
        self.assertIsInstance(self.payer(), self.native.NativePayer)
        self.assertEqual(self.backend.open_calls, 1)


class ImportTests(unittest.TestCase):
    def test_import_exposes_payer_without_loading_participant_client(self):
        path = HERE / 'native.py'
        self.assertTrue(path.is_file(), 'NativePayer module is not implemented')
        result = subprocess.run([sys.executable, '-B', '-c', '''
import importlib.util, pathlib, sys
path = pathlib.Path(sys.argv[1])
spec = importlib.util.spec_from_file_location('native_probe', path)
module = importlib.util.module_from_spec(spec)
def audit(event, args):
    if event.startswith(('socket.', 'subprocess.')) or event in ('os.mkdir', 'os.putenv', 'os.unsetenv'):
        raise AssertionError('Import caused an external effect: ' + event)
    if event == 'open' and isinstance(args[0], str):
        if any(name in args[0] for name in ('shared-claims', 'claim-workflow', 'local-node')):
            raise AssertionError('Import loaded participant client eagerly')
sys.addaudithook(audit)
spec.loader.exec_module(module)
assert callable(module.NativePayer)
''', str(path)], text=True, capture_output=True, timeout=10)
        self.assertEqual(result.returncode, 0, result.stderr)


if __name__ == '__main__':
    unittest.main()
