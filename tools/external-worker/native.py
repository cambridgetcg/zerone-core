"""Loopback-only SDK-bank test-token payments using retained participant custody.

Importing this module does not load the signing client, inspect a participant
home, contact a node or create state. No escrow or cross-home replay protection.
"""


import importlib.util
import os
from pathlib import Path
import re
import subprocess


def _address(value):
    # Exactly 32 data words = 160 bits, plus six original Bech32 checksum words.
    alphabet = 'qpzry9x8gf2tvdw0s3jn54khce6mua7l'
    if type(value) is not str or not re.fullmatch('zrn1[' + alphabet + ']{38}', value):
        raise Error('Expected an exact lowercase 20-byte zrn recipient')
    words = [ord(c) >> 5 for c in 'zrn'] + [0] + [ord(c) & 31 for c in 'zrn']
    words += [alphabet.index(c) for c in value[4:]]
    checksum = 1
    for word in words:
        top = checksum >> 25
        checksum = ((checksum & 0x1ffffff) << 5) ^ word
        for bit, generator in enumerate((0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3)):
            if (top >> bit) & 1:
                checksum ^= generator
    if checksum != 1:
        raise Error('Invalid zrn recipient checksum')
    return value


def _hash(value, label, uppercase=False):
    if type(value) is not str or not re.fullmatch(r'[0-9A-F]{64}' if uppercase else r'[0-9a-f]{64}', value):
        raise Error(f'Invalid exact {label} hash')
    return value


def _load(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    if spec is None or spec.loader is None:
        raise Error('Cannot load the retained participant client')
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class Error(ValueError):
    """The narrow native settlement boundary refused an unsafe observation."""


class NativePayer:
    """Use only an explicitly supplied, already-pinned participant home.

    ``_test_backend`` is an explicit offline unit-test seam, never a runtime
    fallback. Without it, only the existing SharedWorkflow signs or broadcasts.
    """
    def __init__(self, home, *, _test_backend=None):
        self.shared = _load('external_work_shared', Path(__file__).parents[2] / 'scripts/shared-claims.py')
        self.core = self.shared.core
        try:
            self.home = self.core.local.home_path(str(home))
            self.manifest = self.core.read_json(self.home / self.shared.MARKER)
            self.descriptor = self.shared.json_value(self.shared.read_bytes(self.home / 'network.json', 1024 * 1024, True))
            self._guard()
            owner = self

            class LoopbackWorkflow(self.shared.SharedWorkflow):
                def check_files(workflow):
                    # SharedWorkflow calls this before its first network query,
                    # as well as before signing. Keep its full custody checks.
                    owner._guard()
                    if workflow.manifest != owner.manifest or workflow.descriptor != owner.descriptor:
                        raise Error('Participant binding changed after the loopback preflight')
                    super().check_files()

            self.workflow = (LoopbackWorkflow(self.home) if _test_backend is None
                             else _test_backend.open(self.home, self.shared, LoopbackWorkflow))
        except Error:
            raise
        except subprocess.TimeoutExpired as error:
            raise Error('Participant home checks timed out; retain this home for operator resolution; '
                        'no payment was attempted') from error
        except (self.shared.Error, self.core.local.LocalNodeError, OSError, ValueError, TypeError, KeyError) as error:
            raise Error('Cannot use the explicitly pinned participant home: ' + str(error)) from error
        self.settlement = _load('external_work_settlement', Path(__file__).with_name('settlement.py'))
        self.sender = self.manifest['address']

    def _reward(self, reward):
        fields = {'chain_id', 'genesis_sha256', 'recipient', 'amount_uzrn', 'funding_txhash'}
        if type(reward) is not dict or set(reward) != fields:
            raise Error('Reward requires exactly the five native payment fields')
        reward = dict(reward)
        if reward['chain_id'] != 'zerone-dev-1':
            raise Error('Reward is not for the loopback development chain')
        if _hash(reward['genesis_sha256'], 'genesis') != self.manifest['genesis_sha256']:
            raise Error('Reward genesis does not match the pinned participant manifest')
        _hash(reward['funding_txhash'], 'funding transaction', uppercase=True)
        _address(reward['recipient'])
        if reward['recipient'] == self.sender:
            raise Error('Reward recipient must differ from the payer')
        amount = reward['amount_uzrn']
        if type(amount) is not str or not re.fullmatch(r'[1-9][0-9]{0,8}', amount) or int(amount) > 100000000:
            raise Error('Reward must be canonical positive decimal, at most 100000000 uzrn')
        return reward

    def funding(self, reward):
        """Read a named funding transfer and the current reward-plus-fee budget."""
        reward = self._reward(reward)
        try:
            with self.workflow.locked():
                self._guard()
                self.workflow.check_files()
                return self._funding(reward)
        except Error:
            raise
        except subprocess.TimeoutExpired as error:
            raise Error('Native funding readback timed out; no payment was attempted') from error
        except (self.shared.Error, OSError, ValueError, TypeError, KeyError) as error:
            raise Error('Native funding readback refused: ' + str(error)) from error

    def _funding(self, reward, *, require_balance=True):
        try:
            tx = self.workflow.query('tx', reward['funding_txhash'])
            committed = self._committed(tx, reward['funding_txhash'])
            messages = tx['tx']['body']['messages']
            if type(messages) is not list or len(messages) != 1:
                raise Error('Funding requires exactly one SDK MsgSend')
            message = messages[0]
            coins = message['amount']
            if type(coins) is not list or len(coins) != 1 or set(coins[0]) != {'denom', 'amount'}:
                raise Error('Funding requires one exact uzrn amount')
            amount = coins[0]['amount']
            if type(amount) is not str or not re.fullmatch(r'[1-9][0-9]{0,77}', amount):
                raise Error('Funding amount must be a canonical positive decimal')
            sender = _address(message['from_address'])
            expected = {'@type': '/cosmos.bank.v1beta1.MsgSend', 'from_address': sender,
                        'to_address': self.sender, 'amount': [{'denom': 'uzrn', 'amount': amount}]}
            if message != expected or sender == self.sender or int(amount) < int(reward['amount_uzrn']) + 2000000:
                raise Error('Funding is not a sufficient exact transfer to this payer')
            height = self.workflow.committed_height()
            balance = self.workflow.query('bank', 'balances', self.sender, height=height)
            current = self._bank_amount(balance, 'balances')
            if require_balance and current < int(reward['amount_uzrn']) + 2000000:
                raise Error('Current payer balance is insufficient for reward plus 2000000 uzrn fee')
            return {'tx': tx, 'txhash': reward['funding_txhash'], 'height': committed,
                    'payer': self.sender, 'amount_uzrn': amount, 'balance_uzrn': str(current),
                    'balance_height': height, 'balance_sufficient': current >= int(reward['amount_uzrn']) + 2000000}
        except Error:
            raise
        except (KeyError, TypeError, ValueError, IndexError) as error:
            raise Error('Malformed funding observation') from error

    def pay(self, task_id, reward, result_sha256, journal_sha256, *, expected_txhash=None):
        """Pay or recover; an expected hash permits only exact retained recovery."""
        if expected_txhash is not None:
            _hash(expected_txhash, 'expected payment transaction', uppercase=True)
        reward = self._reward(reward)
        context = {'task_id': task_id, 'result_sha256': result_sha256, 'journal_sha256': journal_sha256}
        try:
            memo = self.settlement.payment_memo(task_id, result_sha256, journal_sha256)
            with self.workflow.locked():
                self._guard()
                self.workflow.check_files()
                pending = self.workflow.reconcile()
                attempts = [row for _, row in self.workflow.records('attempts')]
                for row in attempts:
                    if row.get('action') == 'external-work':
                        saved_context = row.get('context')
                        if (type(saved_context) is not dict or set(saved_context) != set(context)
                                or any(type(value) is not str or not re.fullmatch(r'[0-9a-f]{64}', value)
                                       for value in saved_context.values())):
                            raise Error('Conflicting or malformed retained external-work attempt context; operator resolution required')
                attempt = self.settlement.select_attempt(attempts, context)
                if expected_txhash is not None and (attempt is None or attempt.get('txhash') != expected_txhash):
                    raise Error('Recovery-only payment requires a retained committed attempt for the exact expected transaction')
                if pending and attempt is None:
                    raise Error('An earlier transaction is unresolved; use existing exact-hash recovery before any new payment')
                marker = self.workflow.directory / ('external-work-' + task_id + '.json')
                intent = {'schema': 'zerone-external-work-reservation/v1',
                          'context': context, 'reward': reward, 'payer': self.sender}
                if os.path.lexists(marker):
                    saved = self.core.read_json(marker)
                    if (type(saved) is not dict or set(saved) != set(intent) | {'funding'}
                            or any(saved[key] != value for key, value in intent.items())
                            or type(saved['funding']) is not dict):
                        raise Error('Conflicting or malformed task reservation intent')
                    if attempt is None:
                        raise Error('A pre-attempt reservation exists; operator resolution is required, never auto-pay')
                funding = self._funding(reward, require_balance=attempt is None)
                if attempt is not None:
                    return self._payment_evidence(attempt, reward, memo, funding, recovered=True)
                self.core.save_json(marker, {**intent, 'funding': funding}, fresh=True)
                attempt = self.workflow.send('external-work', self.shared.ACTOR,
                    ['bank', 'send', self.shared.ACTOR, reward['recipient'], reward['amount_uzrn'] + 'uzrn',
                     '--note', memo], context)
                return self._payment_evidence(attempt, reward, memo, funding, recovered=False)
        except Error:
            raise
        except subprocess.TimeoutExpired as error:
            raise Error('Native payment timed out; retain this home for operator resolution and '
                        'existing exact-hash recovery when an attempt exists, never a new payment') from error
        except (self.shared.Error, OSError, ValueError, TypeError, KeyError, IndexError, AttributeError) as error:
            raise Error('Native payment refused: ' + str(error) +
                        '; retain this home and use existing exact-hash recovery, never a new payment') from error

    def _payment_evidence(self, attempt, reward, memo, funding, *, recovered):
        digest = _hash(attempt['txhash'], 'payment transaction', uppercase=True)
        tx = self.workflow.query('tx', digest)
        height = self._committed(tx, digest)
        if height < 2:
            raise Error('Payment height must allow a positive H-1; SDK height zero means latest')
        terms = {'sender': self.sender, 'recipient': reward['recipient'],
                 'amount_uzrn': reward['amount_uzrn'], 'memo': memo}
        observations = []
        for at in (height - 1, height):
            observations.append({
                'sender': self._bank_amount(self.workflow.query('bank', 'balances', self.sender, height=at), 'balances'),
                'recipient': self._bank_amount(self.workflow.query('bank', 'balances', reward['recipient'], height=at), 'balances'),
                # SDK bank total has no --denom flag, even when help suggests one.
                'supply': self._bank_amount(self.workflow.query('bank', 'total', height=at), 'supply')})
        evidence = self.settlement.verify_payment(terms, tx, digest, *observations)
        return {**evidence, 'tx': tx, 'funding': funding, 'recovered': recovered,
                'chain_id': reward['chain_id'], 'genesis_sha256': reward['genesis_sha256']}

    def _bank_amount(self, observation, field):
        if type(observation) is not dict or set(observation) != {field, 'pagination'}:
            raise Error('Missing or unknown bank observation fields')
        page = observation['pagination']
        if type(page) is not dict or set(page) - {'next_key', 'total'}:
            raise Error('Missing or unknown bank pagination')
        # SDK proto JSON omits empty next_key and uncounted/zero total.
        if page.get('next_key') is not None and page.get('next_key') != '':
            raise Error('Incomplete or unknown bank pagination')
        total = self.settlement.integer(page.get('total', '0'), 'pagination total')
        coins = observation[field]
        if type(coins) is not list:
            raise Error('Malformed bank coin list')
        if total and total != len(coins):
            raise Error('Bank pagination total disagrees with the complete coin list')
        amounts = {}
        for coin in coins:
            if type(coin) is not dict or set(coin) != {'denom', 'amount'}:
                raise Error('Malformed bank coin')
            denom, amount = coin['denom'], coin['amount']
            if type(denom) is not str or not denom or denom in amounts:
                raise Error('Invalid or duplicate bank denomination')
            if type(amount) is not str or not re.fullmatch(r'0|[1-9][0-9]{0,77}', amount):
                raise Error('Bank amount is not a canonical decimal')
            amounts[denom] = int(amount)
        if field == 'supply' and amounts.get('uzrn', 0) <= 0:
            raise Error('Missing or nonpositive uzrn bank supply')
        return amounts.get('uzrn', 0)

    def _committed(self, tx, expected):
        if type(tx) is not dict or tx.get('txhash') != expected:
            raise Error('Transaction readback differs from the exact expected hash')
        height = self.settlement.integer(tx.get('height'), 'committed height', True)
        if self.settlement.integer(tx.get('code'), 'execution code') != 0:
            raise Error('Transaction execution was not successful')
        return height

    def _guard(self):
        if type(self.manifest) is not dict or type(self.descriptor) is not dict:
            raise Error('Malformed participant manifest or descriptor')
        if self.manifest.get('allow_loopback_test') is not True or self.descriptor.get('local_test') is not True:
            raise Error('Only an explicitly pinned loopback local-test participant is permitted')
        if self.manifest.get('chain_id') != 'zerone-dev-1' or self.descriptor.get('chain_id') != 'zerone-dev-1':
            raise Error('Only the loopback zerone-dev-1 logical chain is permitted')
        if _hash(self.manifest.get('genesis_sha256'), 'pinned genesis') != self.descriptor.get('genesis_sha256'):
            raise Error('Descriptor genesis differs from the pinned participant manifest')
        try:
            self.shared.validate_descriptor(self.descriptor, allow_loopback=True)
        except (self.shared.Error, ValueError, TypeError) as error:
            raise Error('Invalid pinned loopback local-test descriptor') from error
