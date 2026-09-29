"""Payment evidence checks; synthetic SDK observations, no accounts or RPC."""
import copy
import unittest

import settlement as s


def fixture():
    terms = {'sender': 'test-sponsor', 'recipient': 'test-worker',
             'amount_uzrn': '10000000', 'memo': 'test-bound-work'}
    tx = {'txhash': 'A' * 64, 'height': '12', 'code': 0,
          'tx': {'body': {'messages': [{'@type': '/cosmos.bank.v1beta1.MsgSend',
                'from_address': terms['sender'], 'to_address': terms['recipient'],
                'amount': [{'denom': 'uzrn', 'amount': terms['amount_uzrn']}]}],
                'memo': terms['memo']},
                'auth_info': {'fee': {'amount': [{'denom': 'uzrn', 'amount': '2000000'}]}}}}
    before = {'sender': 20000000, 'recipient': 0, 'supply': 100000000}
    after = {'sender': 8000000, 'recipient': 10000000, 'supply': 100000000}
    return terms, tx, before, after


class PaymentEvidenceTests(unittest.TestCase):
    def test_matching_bank_payment_requires_actual_balance_changes(self):
        terms, tx, before, after = fixture()
        result = s.verify_payment(terms, tx, 'A' * 64, before, after)
        self.assertEqual(result['amount_uzrn'], '10000000')
        self.assertEqual(result['height'], 12)
        self.assertEqual(result['fee_uzrn'], '2000000')
        self.assertTrue(result['bank_deltas_verified'])

    def test_malformed_hash_even_when_both_observations_agree(self):
        terms, tx, before, after = fixture()
        tx['txhash'] = 'not-a-hash'
        with self.assertRaises(s.Error):
            s.verify_payment(terms, tx, tx['txhash'], before, after)

    def test_missing_decoded_payment_is_a_refusal(self):
        terms, tx, before, after = fixture()
        tx['tx'] = None
        with self.assertRaises(s.Error):
            s.verify_payment(terms, tx, 'A' * 64, before, after)

    def test_non_object_transaction_or_body_is_a_controlled_refusal(self):
        for target in ('transaction', 'body'):
            for malformed in (None, []):
                with self.subTest(target=target, malformed=malformed):
                    terms, tx, before, after = fixture()
                    if target == 'transaction':
                        tx = malformed
                    else:
                        tx['tx']['body'] = malformed
                    with self.assertRaisesRegex(s.Error, 'Malformed payment observation'):
                        s.verify_payment(terms, tx, 'A' * 64, before, after)

    def test_mismatch_or_pending_is_not_payment(self):
        for mutation in ('hash', 'height', 'code', 'memo', 'recipient', 'amount', 'fee', 'credit', 'debit', 'supply'):
            with self.subTest(mutation=mutation):
                terms, tx, before, after = fixture()
                if mutation == 'hash': tx['txhash'] = 'B' * 64
                if mutation == 'height': tx['height'] = '0'
                if mutation == 'code': tx['code'] = 5
                if mutation == 'memo': tx['tx']['body']['memo'] = 'other-work'
                if mutation == 'recipient': tx['tx']['body']['messages'][0]['to_address'] = 'another-worker'
                if mutation == 'amount': tx['tx']['body']['messages'][0]['amount'][0]['amount'] = '9000000'
                if mutation == 'fee': tx['tx']['auth_info']['fee']['amount'][0]['amount'] = '0'
                if mutation == 'credit': after['recipient'] = 0
                if mutation == 'debit': after['sender'] = before['sender']
                if mutation == 'supply': after['supply'] += 1
                with self.assertRaises(s.Error):
                    s.verify_payment(terms, tx, 'A' * 64, before, after)

    def test_additional_message_cannot_hide_in_payment(self):
        terms, tx, before, after = fixture()
        tx['tx']['body']['messages'].append(copy.deepcopy(tx['tx']['body']['messages'][0]))
        with self.assertRaises(s.Error):
            s.verify_payment(terms, tx, 'A' * 64, before, after)


class IntentTests(unittest.TestCase):
    def test_prior_payment_is_recovered_not_reissued(self):
        self.assertTrue(hasattr(s, 'select_attempt'), 'missing per-task payment recovery')
        context = {'task_id': 'a' * 64, 'result_sha256': 'b' * 64, 'journal_sha256': 'c' * 64}
        prior = {'action': 'external-work', 'context': context, 'status': 'committed', 'txhash': 'A' * 64}
        self.assertEqual(s.select_attempt([prior], context), prior)
        self.assertIsNone(s.select_attempt([], context))
        for change in ('result', 'journal', 'pending', 'duplicate'):
            with self.subTest(change=change):
                rows = [copy.deepcopy(prior)]
                if change == 'result': rows[0]['context']['result_sha256'] = 'd' * 64
                if change == 'journal': rows[0]['context']['journal_sha256'] = 'd' * 64
                if change == 'pending': rows[0]['status'] = 'pending'
                if change == 'duplicate': rows.append(copy.deepcopy(prior))
                with self.assertRaises(s.Error):
                    s.select_attempt(rows, context)

    def test_memo_binds_task_result_and_journal(self):
        self.assertTrue(hasattr(s, 'payment_memo'), 'missing content-bound payment memo')
        memo = s.payment_memo('a' * 64, 'b' * 64, 'c' * 64)
        self.assertEqual(memo, 'zerone-work/v1 ' + ' '.join(x * 64 for x in 'abc'))
        self.assertLessEqual(len(memo.encode()), 256)
        with self.assertRaises(s.Error):
            s.payment_memo('a' * 63, 'b' * 64, 'c' * 64)


if __name__ == '__main__':
    unittest.main()
