"""Strict evidence checks for operator-mediated development-token payments.

No RPC, signing, filesystem writes or computation on import. This is not an
on-chain task escrow, payment authorization, or production reward module.
"""
import re


class Error(ValueError):
    pass


def integer(value, label, positive=False):
    if type(value) is int:
        number = value
    elif type(value) is str and re.fullmatch(r'0|[1-9][0-9]*', value):
        number = int(value)
    else:
        raise Error(f'{label}: expected a canonical integer')
    if number < (1 if positive else 0):
        raise Error(f'{label}: outside range')
    return number


def select_attempt(attempts, context):
    """Recover one already committed attempt; ambiguous/forked state fails closed."""
    matches = [row for row in attempts if row.get('action') == 'external-work'
               and row.get('context', {}).get('task_id') == context['task_id']]
    if not matches:
        return None
    if len(matches) != 1 or matches[0].get('context') != context:
        raise Error('Conflicting payment intent for this task')
    if matches[0].get('status') != 'committed':
        raise Error('An existing task payment is unresolved or failed; reconcile that exact transaction, never create another')
    return matches[0]


def payment_memo(task_id, result_sha256, journal_sha256):
    hashes = (task_id, result_sha256, journal_sha256)
    if any(type(value) is not str or not re.fullmatch(r'[0-9a-f]{64}', value) for value in hashes):
        raise Error('Payment requires exact lowercase task, result and journal hashes')
    return 'zerone-work/v1 ' + ' '.join(hashes)


def verify_payment(terms, tx, expected_txhash, before, after):
    """Check one exact MsgSend plus H-1/H bank balances and supply.

    Caller must obtain observations from the pinned chain at those heights.
    These checks do not authenticate an independently supplied RPC response.
    """
    try:
        return _verify_payment(terms, tx, expected_txhash, before, after)
    except Error:
        raise
    except (KeyError, TypeError, ValueError) as error:
        raise Error('Malformed payment observation') from error


def _verify_payment(terms, tx, expected_txhash, before, after):
    if type(tx) is not dict:
        raise Error('Malformed payment observation: transaction must be an object')
    if type(expected_txhash) is not str or not re.fullmatch(r'[0-9A-F]{64}', expected_txhash):
        raise Error('Expected transaction hash must be 64 uppercase hexadecimal characters')
    amount = integer(terms['amount_uzrn'], 'reward', True)
    height = integer(tx.get('height'), 'committed height', True)
    if tx.get('txhash') != expected_txhash or integer(tx.get('code'), 'execution code') != 0:
        raise Error('Payment did not commit with the expected hash and execution code')
    body = tx['tx']['body']
    if type(body) is not dict:
        raise Error('Malformed payment observation: transaction body must be an object')
    expected = {'@type': '/cosmos.bank.v1beta1.MsgSend', 'from_address': terms['sender'],
                'to_address': terms['recipient'], 'amount': [{'denom': 'uzrn', 'amount': str(amount)}]}
    if body.get('messages') != [expected] or body.get('memo') != terms['memo']:
        raise Error('Signed payment differs from the bound task, artifact or journal')
    fees = tx['tx']['auth_info']['fee']['amount']
    if fees != [{'denom': 'uzrn', 'amount': '2000000'}]:
        raise Error('Unexpected development transaction fee')
    fee = 2000000
    if any(type(row[k]) is not int or row[k] < 0 for row in (before, after) for k in ('sender', 'recipient', 'supply')):
        raise Error('Malformed bank observations')
    if after['recipient'] - before['recipient'] != amount or after['sender'] - before['sender'] != -(amount + fee):
        raise Error('Actual bank changes do not match reward and fee')
    if before['supply'] != after['supply']:
        raise Error('Payment block unexpectedly changed supply')
    return {'txhash': expected_txhash, 'height': height, 'amount_uzrn': str(amount),
            'fee_uzrn': str(fee), 'recipient': terms['recipient'], 'bank_deltas_verified': True,
            'before': before, 'after': after,
            'assurance': 'PINNED_OPERATOR_CHAIN_OBSERVATIONS_NOT_INDEPENDENT_INCLUSION_PROOF'}
