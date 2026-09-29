"""Task journal integration. All fixture data here is explicitly synthetic."""
import contextlib
import copy
import hashlib
import importlib.util
import io
import os
from pathlib import Path
import unittest
import json
import tempfile
import subprocess
import sys
from unittest import mock

HERE = Path(__file__).resolve().parent


class JournalIntegrationTests(unittest.TestCase):
    def test_prepare_checks_before_writing_and_preserves_exact_graph(self):
        import coordinator as c
        self.assertTrue(hasattr(c, 'prepare'), 'missing checked-artifact intake')
        # Reproducible synthetic fixture computed by the real worker, not a stub verifier.
        w = c.load_module('fixture_external_worker', HERE / 'worker.py')
        task = w.make_task('ab' * 32, 3, {
            'chain_id': 'zerone-dev-1', 'genesis_sha256': 'c' * 64,
            'recipient': 'zrn1rj7z53777w3nwgwgm2lmqlly85ntajplcda70l',
            'amount_uzrn': '10000000', 'funding_txhash': 'D' * 64})
        result = w.compute(task, 'synthetic-fixture-worker')
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            task_path, result_path = root / 'task.json', root / 'result.json'
            task_path.write_text(json.dumps(task))
            result_path.write_text(json.dumps(result))
            output = root / 'accepted'
            receipt = c.prepare(task_path, result_path, output)
            self.assertTrue(receipt['computation_verified'])
            self.assertFalse(receipt['paid'])
            checked = c.checked_workspace(output)
            self.assertEqual(checked['task'], task)
            self.assertEqual(checked['result'], result)
            self.assertEqual(checked['journal_sha256'], receipt['journal_sha256'])
            export = c.journal().load_export(output / 'journal.json')
            self.assertEqual(len(export['entries']), len(c.records_for(task, result, receipt['task_sha256'], receipt['result_sha256'])))
            with self.assertRaises((ValueError, OSError)):
                c.prepare(task_path, result_path, output)
            result['result']['root_hex'] = '0' * 64
            result_path.write_text(json.dumps(result))
            with self.assertRaises(ValueError):
                c.prepare(task_path, result_path, root / 'bad')
            self.assertFalse((root / 'bad').exists())

    def test_payment_boundary_rechecks_artifact_and_retains_one_payment_record(self):
        import coordinator as c
        self.assertTrue(hasattr(c, 'pay_workspace'), 'missing explicit checked payment boundary')
        w = c.load_module('fixture_external_worker', HERE / 'worker.py')
        task = w.make_task('ab' * 32, 3, {
            'chain_id': 'zerone-dev-1', 'genesis_sha256': 'c' * 64,
            'recipient': 'zrn1rj7z53777w3nwgwgm2lmqlly85ntajplcda70l',
            'amount_uzrn': '10000000', 'funding_txhash': 'D' * 64})
        result = w.compute(task, 'synthetic-fixture-worker')
        synthetic_payment = {'txhash': 'E' * 64, 'height': 4, 'amount_uzrn': '10000000',
            'fee_uzrn': '2000000', 'recipient': task['reward']['recipient'],
            'bank_deltas_verified': True, 'before': {'sender': 20000000, 'recipient': 0, 'supply': 100000000},
            'after': {'sender': 8000000, 'recipient': 10000000, 'supply': 100000000},
            'assurance': 'PINNED_OPERATOR_CHAIN_OBSERVATIONS_NOT_INDEPENDENT_INCLUSION_PROOF'}
        backend = mock.Mock()
        backend.pay.return_value = synthetic_payment
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'task.json').write_text(json.dumps(task))
            (root / 'result.json').write_text(json.dumps(result))
            output = root / 'accepted'
            c.prepare(root / 'task.json', root / 'result.json', output)
            with mock.patch.object(c, 'payer', return_value=backend) as factory:
                receipt = c.pay_workspace(output, root / 'synthetic-payer')
                self.assertTrue(receipt['paid'])
                original_export = (output / 'paid-journal.json').read_bytes()
                second = c.pay_workspace(output, root / 'synthetic-payer')
                self.assertEqual(receipt, second)
                self.assertEqual((output / 'paid-journal.json').read_bytes(), original_export)
                self.assertEqual(len(c.journal().load_export(output / 'paid-journal.json')['entries']), 14)
                before = factory.call_count
                unrelated = dict(c.journal().load_export(output / 'journal.json')['entries'][0]['record'])
                unrelated['id'] = 'synthetic-unrelated-record'
                c.journal().append_records(output / 'journal-store', [unrelated])
                with self.assertRaises(ValueError):
                    c.pay_workspace(output, root / 'synthetic-payer')
                self.assertEqual(factory.call_count, before, 'unexpected journal history must fail before opening a signer')
                altered = json.loads((output / 'result.json').read_text())
                altered['result']['root_hex'] = '0' * 64
                (output / 'result.json').write_bytes(c.document_bytes(altered))
                with self.assertRaises(ValueError):
                    c.pay_workspace(output, root / 'synthetic-payer')
                self.assertEqual(factory.call_count, before, 'invalid artifact must not even open a signer')

    def test_cli_payment_requires_explicit_test_only_acknowledgement(self):
        process = subprocess.run([sys.executable, '-B', str(HERE / 'coordinator.py'),
            'pay', '--workspace', '/nonexistent-fixture', '--payer-home', '/nonexistent-fixture'],
            capture_output=True, text=True, timeout=10)
        self.assertEqual(process.returncode, 2)
        self.assertIn('--confirm-test-token-payment', process.stderr)

    def test_result_has_attribution_and_scoped_graph_relations(self):
        self.assertTrue((HERE / 'coordinator.py').is_file(), 'missing worker-to-journal connection')
        spec = importlib.util.spec_from_file_location('external_coordinator', HERE / 'coordinator.py')
        assert spec is not None and spec.loader is not None
        c = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(c)
        task = {'task_id': 'a' * 64, 'leaf_count': 3, 'seed_hex': 'b' * 64,
                'reward': {'recipient': 'synthetic-test-recipient', 'amount_uzrn': '10000000'}}
        result = {'worker_id': 'synthetic-fixture-worker', 'result': {
            'root_hex': 'c' * 64, 'extended_root_hex': 'c' * 64,
            'original_mutated': False, 'extended_mutated': True,
            'original_count': 3, 'extended_count': 4}}
        records = c.records_for(task, result, 'd' * 64, 'e' * 64)
        earlier = {}
        for record in records:
            c.journal().validate_record(record, earlier)
            earlier[record['id']] = record
        findings = [r for r in records if r.get('contribution_type') == 'finding']
        self.assertEqual(len(findings), 1)
        self.assertEqual(findings[0]['attributed_to'], result['worker_id'])
        self.assertIn('not a new Bitcoin vulnerability', findings[0]['summary'])
        self.assertTrue(any(e['sha256'] == 'e' * 64 for e in findings[0]['evidence']))
        self.assertTrue(any(r.get('relation_type') == 'uses-input' and r['from_id'] == findings[0]['id'] for r in records))
        self.assertFalse(any('accepted' in r.get('contribution_type', '') for r in records))


class ComputeOnlyTests(unittest.TestCase):
    def setUp(self):
        import coordinator as c
        self.c = c
        self.w = c.load_module('fixture_external_worker', HERE / 'worker.py')
        self.task = self.w.make_compute_task('ab' * 32, 3)
        self.result = self.w.compute(self.task, 'unpaid-journal-fixture')
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        self.task_path, self.result_path = self.root / 'task.json', self.root / 'result.json'
        self.task_path.write_bytes(c.document_bytes(self.task))
        self.result_path.write_bytes(c.document_bytes(self.result))
        self.output = self.root / 'prepared'

    def cli(self, *args):
        return subprocess.run([sys.executable, '-B', str(HERE / 'coordinator.py'), *map(str, args)],
                              capture_output=True, text=True, timeout=10)

    @contextlib.contextmanager
    def no_financial_io(self, payer_home=None):
        load = self.c.load_module
        def safe_load(name, path):
            self.assertIn(Path(path).name, ('worker.py', 'journal.py'), 'payer import forbidden')
            return load(name, path)
        with contextlib.ExitStack() as stack:
            if payer_home is not None:
                def protect_path(function):
                    def guarded(path, *args, **kwargs):
                        if isinstance(path, (str, bytes, os.PathLike)):
                            self.assertFalse(os.fsdecode(path).startswith(str(payer_home)),
                                             'payer-home access forbidden')
                        return function(path, *args, **kwargs)
                    return guarded
                import builtins
                for module, name in ((builtins, 'open'), (io, 'open'), (os, 'open'),
                                     (os, 'stat'), (os, 'lstat'), (os, 'listdir'), (os, 'scandir')):
                    stack.enter_context(mock.patch.object(module, name,
                        side_effect=protect_path(getattr(module, name))))
            stack.enter_context(mock.patch.object(self.c, 'load_module', side_effect=safe_load))
            guards = [stack.enter_context(mock.patch.object(self.c, 'payer',
                side_effect=AssertionError('payer construction forbidden')))]
            for target in ('socket.socket', 'socket.create_connection', 'socket.getaddrinfo',
                           'subprocess.Popen', 'os.system'):
                guards.append(stack.enter_context(mock.patch(target,
                    side_effect=AssertionError('financial network/process I/O forbidden'))))
            yield
            for guard in guards:
                guard.assert_not_called()

    def refused_main(self, args):
        out, err = io.StringIO(), io.StringIO()
        home = Path(args[args.index('--payer-home') + 1])
        with self.no_financial_io(home), contextlib.redirect_stdout(out), contextlib.redirect_stderr(err):
            try:
                code = self.c.main(list(map(str, args)))
            except KeyError as error:
                self.fail('uncontrolled KeyError: ' + str(error))
        self.assertEqual((code, out.getvalue()), (1, ''))
        self.assertIn('compute-only', err.getvalue().lower())
        self.assertNotIn('Traceback', err.getvalue())

    def test_funding_rejects_compute_only_before_payer_access(self):
        for exists in (False, True):
            with self.subTest(payer_home_exists=exists):
                home = self.root / ('existing-home' if exists else 'absent-home')
                if exists:
                    home.mkdir()
                    (home / 'untouched').write_bytes(b'synthetic sentinel, not a credential')
                args = ['funding', '--task', self.task_path, '--payer-home', home]
                self.refused_main(args)
                completed = self.cli(*args)
                self.assertEqual((completed.returncode, completed.stdout), (1, ''))
                self.assertIn('compute-only', completed.stderr.lower())
                self.assertNotIn('Traceback', completed.stderr)
                if exists:
                    self.assertEqual([p.name for p in home.iterdir()], ['untouched'])
                    self.assertEqual((home / 'untouched').read_bytes(), b'synthetic sentinel, not a credential')
                else:
                    self.assertFalse(home.exists())

    def test_pay_rejects_compute_only_before_payer_or_planted_payment_evidence(self):
        self.c.prepare(self.task_path, self.result_path, self.output)
        planted = {'schema': 'zerone.external-work-payment/v1', 'task_id': self.task['task_id'],
                   'genesis_sha256': '0' * 64, 'result_sha256': '0' * 64,
                   'journal_sha256': '0' * 64, 'paid': True, 'payment': {}, 'scope': 'planted fixture'}
        home = self.root / 'payer-home'
        for exists in (False, True):
            if exists:
                home.mkdir()
                (home / 'untouched').write_bytes(b'synthetic sentinel, not a credential')
            for raw in (None, b'{', b'null\n', self.c.document_bytes(planted)):
                with self.subTest(payer_home_exists=exists, payment=raw):
                    path = self.output / 'payment.json'
                    if raw is None:
                        path.unlink(missing_ok=True)
                    else:
                        path.write_bytes(raw)
                    before = {str(p.relative_to(self.output)): p.read_bytes()
                              for p in self.output.rglob('*') if p.is_file()}
                    with self.no_financial_io(home):
                        try:
                            with self.assertRaisesRegex(ValueError, '[Cc]ompute-only'):
                                self.c.pay_workspace(self.output, home)
                        except KeyError as error:
                            self.fail('uncontrolled KeyError: ' + str(error))
                    args = ['pay', '--workspace', self.output, '--payer-home', home,
                            '--confirm-test-token-payment']
                    self.refused_main(args)
                    completed = self.cli(*args)
                    self.assertEqual((completed.returncode, completed.stdout), (1, ''))
                    self.assertIn('compute-only', completed.stderr.lower())
                    self.assertNotIn('Traceback', completed.stderr)
                    self.assertEqual({str(p.relative_to(self.output)): p.read_bytes()
                                      for p in self.output.rglob('*') if p.is_file()}, before)
                    if exists:
                        self.assertEqual([p.name for p in home.iterdir()], ['untouched'])
                        self.assertEqual((home / 'untouched').read_bytes(), b'synthetic sentinel, not a credential')
                    else:
                        self.assertFalse(home.exists())

    def test_compute_only_payment_helpers_and_inspect_refuse_retained_evidence_safely(self):
        self.c.prepare(self.task_path, self.result_path, self.output)
        checked = self.c.checked_workspace(self.output)
        planted = {'schema': 'zerone.external-work-payment/v1', 'task_id': self.task['task_id'],
                   'genesis_sha256': '0' * 64, 'result_sha256': checked['result_sha256'],
                   'journal_sha256': checked['journal_sha256'], 'paid': True, 'payment': {},
                   'scope': 'planted fixture'}
        with self.no_financial_io():
            for label, call in (
                    ('format', lambda: self.c.payment_receipt_for(checked, {})),
                    ('validate', lambda: self.c.validate_payment_receipt(planted, checked)),
                    ('validate-null', lambda: self.c.validate_payment_receipt(None, checked))):
                with self.subTest(helper=label):
                    try:
                        with self.assertRaisesRegex(ValueError, '[Cc]ompute-only'):
                            call()
                    except KeyError as error:
                        self.fail('uncontrolled KeyError: ' + str(error))
        for raw in (b'{', b'null\n', self.c.document_bytes(planted)):
            with self.subTest(retained_payment=raw):
                (self.output / 'payment.json').write_bytes(raw)
                completed = self.cli('inspect', '--workspace', self.output)
                self.assertEqual((completed.returncode, completed.stdout), (1, ''))
                self.assertIn('compute-only', completed.stderr.lower())
                self.assertNotIn('Traceback', completed.stderr)
                self.assertEqual((self.output / 'payment.json').read_bytes(), raw)
                self.assertFalse((self.output / 'paid-journal.json').exists())

    def test_compute_only_preparation_inherits_verification_and_filesystem_boundaries(self):
        with self.no_financial_io():
            self.c.prepare(self.task_path, self.result_path, self.output)
            self.c.checked_workspace(self.output)
        before = {str(p.relative_to(self.output)): p.read_bytes()
                  for p in self.output.rglob('*') if p.is_file()}
        with self.assertRaises((ValueError, OSError)):
            self.c.prepare(self.task_path, self.result_path, self.output)
        self.assertEqual({str(p.relative_to(self.output)): p.read_bytes()
                          for p in self.output.rglob('*') if p.is_file()}, before)
        (self.root / 'task-link').symlink_to(self.task_path)
        (self.root / 'result-link').symlink_to(self.result_path)
        (self.root / 'output-link').symlink_to(self.output, target_is_directory=True)
        (self.root / 'parent-link').symlink_to(self.root, target_is_directory=True)
        for task_path, result_path, output in (
                (self.root / 'task-link', self.result_path, self.root / 'rejected'),
                (self.task_path, self.root / 'result-link', self.root / 'rejected'),
                (self.task_path, self.result_path, self.root / 'output-link'),
                (self.task_path, self.result_path, self.root / 'parent-link' / 'rejected')):
            with self.subTest(task=task_path, result=result_path, output=output):
                with self.assertRaises((ValueError, OSError)):
                    self.c.prepare(task_path, result_path, output)
                self.assertFalse((self.root / 'rejected').exists())
        with self.assertRaises((ValueError, OSError)):
            self.c.checked_workspace(self.root / 'output-link')
        changed = copy.deepcopy(self.result)
        changed['result']['extended_mutated'] = False
        self.result_path.write_bytes(self.c.document_bytes(changed))
        with self.assertRaises(ValueError):
            self.c.prepare(self.task_path, self.result_path, self.root / 'rejected')
        self.assertFalse((self.root / 'rejected').exists())
        (self.output / 'result.json').write_bytes(self.c.document_bytes(changed))
        with self.assertRaises(ValueError):
            self.c.checked_workspace(self.output)

    def test_mutated_compute_tasks_fail_before_financial_io(self):
        from test_worker import synthetic_reward
        self.c.prepare(self.task_path, self.result_path, self.output)
        home = self.root / 'absent-payer-home'
        changes = ({'reward': None}, {'reward': synthetic_reward()}, {'unknown': True},
                   {'leaf_count': True}, {'schema': 'zerone.external-work-task/v1'})
        candidates = [{**self.task, **change} for change in changes]
        for task in candidates:
            task['task_id'] = self.w.digest(self.w.canonical({k: v for k, v in task.items() if k != 'task_id'}))
        candidates += [{**self.task, 'task_id': '0' * 64},
                       {**self.task, 'schema': 'zerone.external-work-task/v1', 'reward': synthetic_reward()}]
        for task in candidates:
            self.task_path.write_bytes(self.c.document_bytes(task))
            (self.output / 'task.json').write_bytes(self.c.document_bytes(task))
            for args in (['funding', '--task', self.task_path, '--payer-home', home],
                         ['pay', '--workspace', self.output, '--payer-home', home, '--confirm-test-token-payment']):
                with self.subTest(task=task, command=args[0]):
                    out, err = io.StringIO(), io.StringIO()
                    with self.no_financial_io(home), contextlib.redirect_stdout(out), contextlib.redirect_stderr(err):
                        code = self.c.main(list(map(str, args)))
                    self.assertEqual((code, out.getvalue()), (1, ''))
                    self.assertIn('external-worker refused:', err.getvalue())
                    self.assertNotIn('Traceback', err.getvalue())
                    self.assertFalse(home.exists())

    def test_prepare_and_inspect_retain_an_explicitly_unpaid_journal(self):
        made = self.cli('prepare', '--task', self.task_path, '--result', self.result_path,
                        '--output', self.output)
        self.assertEqual(made.returncode, 0, made.stderr)
        self.assertEqual(made.stderr, '')
        receipt = json.loads(made.stdout)
        self.assertIs(receipt['computation_verified'], True)
        self.assertIs(receipt['paid'], False)
        self.assertIs(receipt['payment_eligible'], False)
        self.assertIn('no funding, no nominated recipient and no payment eligibility', receipt['scope'])
        checked = self.c.checked_workspace(self.output)
        self.assertEqual(checked['task'], self.task)
        self.assertEqual(checked['result'], self.result)
        self.assertNotIn('payment_receipt', checked)
        self.assertEqual(checked['journal_sha256'], receipt['journal_sha256'])
        records = [entry['record'] for entry in checked['journal']['entries']]
        self.assertEqual(records, self.c.records_for(self.task, self.result,
            checked['task_sha256'], checked['result_sha256']))
        self.assertEqual(len(records), 12)
        experiment = next(row for row in records if row['id'].endswith(':experiment'))
        self.assertIn('no funding, no nominated recipient and no payment eligibility', experiment['summary'])
        self.assertFalse(any(row['id'].endswith((':payment', ':payment-finding')) for row in records))
        before = {str(p.relative_to(self.output)): p.read_bytes()
                  for p in self.output.rglob('*') if p.is_file()}
        inspected = self.cli('inspect', '--workspace', self.output)
        self.assertEqual(inspected.returncode, 0, inspected.stderr)
        self.assertEqual(inspected.stderr, '')
        response = json.loads(inspected.stdout)
        self.assertIs(response['paid'], False)
        self.assertIs(response['payment_eligible'], False)
        self.assertEqual(response['task_id'], self.task['task_id'])
        self.assertIn('no funding, no nominated recipient and no payment eligibility', response['scope'])
        self.assertEqual({str(p.relative_to(self.output)): p.read_bytes()
                          for p in self.output.rglob('*') if p.is_file()}, before)
        self.assertFalse((self.output / 'paid-journal.json').exists())


class RetainedPaymentTests(unittest.TestCase):
    def setUp(self):
        import coordinator as c
        self.c = c
        w = c.load_module('fixture_external_worker', HERE / 'worker.py')
        self.task = w.make_task('ab' * 32, 3, {
            'chain_id': 'zerone-dev-1', 'genesis_sha256': 'c' * 64,
            'recipient': 'zrn1rj7z53777w3nwgwgm2lmqlly85ntajplcda70l',
            'amount_uzrn': '10000000', 'funding_txhash': 'D' * 64})
        self.result = w.compute(self.task, 'SYNTHETIC_RETAINED_PAYMENT_FIXTURE')
        self.payment = {'txhash': 'E' * 64, 'height': 4, 'amount_uzrn': '10000000',
            'fee_uzrn': '2000000', 'recipient': self.task['reward']['recipient'],
            'bank_deltas_verified': True,
            'before': {'sender': 20000000, 'recipient': 0, 'supply': 100000000},
            'after': {'sender': 8000000, 'recipient': 10000000, 'supply': 100000000},
            'assurance': 'PINNED_OPERATOR_CHAIN_OBSERVATIONS_NOT_INDEPENDENT_INCLUSION_PROOF'}

    def workspace(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        root = Path(temporary.name)
        (root / 'task.json').write_bytes(self.c.document_bytes(self.task))
        (root / 'result.json').write_bytes(self.c.document_bytes(self.result))
        output = root / 'accepted'
        self.c.prepare(root / 'task.json', root / 'result.json', output)
        return output

    def assert_refused_before_payer(self, output):
        backend = mock.Mock()
        backend.pay.return_value = copy.deepcopy(self.payment)
        with mock.patch.object(self.c, 'payer', return_value=backend) as factory:
            with self.assertRaises((ValueError, OSError)):
                self.c.pay_workspace(output, output.parent / 'NEVER_OPEN_A_PAYER')
            factory.assert_not_called()
            backend.pay.assert_not_called()

    def test_allowlisted_payment_ids_without_receipt_fail_before_payer(self):
        output = self.workspace()
        first = copy.deepcopy(self.c.journal().export_store(output / 'journal-store')['entries'][0]['record'])
        first['id'] = 'work:' + self.task['task_id'] + ':payment'
        second = copy.deepcopy(first)
        second['id'] = 'work:' + self.task['task_id'] + ':payment-finding'
        self.c.journal().append_records(output / 'journal-store', [first, second])
        self.assert_refused_before_payer(output)

    def receipt(self, output):
        prepared = json.loads((output / 'preparation.json').read_bytes())
        return {'schema': 'zerone.external-work-payment/v1', 'task_id': self.task['task_id'],
                'genesis_sha256': self.task['reward']['genesis_sha256'],
                'result_sha256': prepared['result_sha256'], 'journal_sha256': prepared['journal_sha256'],
                'paid': True, 'payment': copy.deepcopy(self.payment),
                'scope': 'Operator-approved test bounty. Not protocol task escrow, independent science, token value or production activation.'}

    def test_receipt_bindings_fail_before_payer(self):
        for field in ('task_id', 'genesis_sha256', 'result_sha256', 'journal_sha256'):
            with self.subTest(field=field):
                output = self.workspace()
                receipt = self.receipt(output)
                receipt[field] = 'f' * 64
                (output / 'payment.json').write_bytes(self.c.document_bytes(receipt))
                self.assert_refused_before_payer(output)

    def test_receipt_requires_exact_envelope_before_payer(self):
        for mutation in ('null', 'list', 'missing', 'extra', 'schema', 'unpaid', 'integer-paid', 'scope'):
            with self.subTest(mutation=mutation):
                output = self.workspace()
                receipt = self.receipt(output)
                if mutation == 'null': receipt = None
                elif mutation == 'list': receipt = []
                elif mutation == 'missing': del receipt['genesis_sha256']
                elif mutation == 'extra': receipt['extra'] = 'unsupported'
                elif mutation == 'schema': receipt['schema'] = 'zerone.external-work-payment/v2'
                elif mutation == 'unpaid': receipt['paid'] = False
                elif mutation == 'integer-paid': receipt['paid'] = 1
                elif mutation == 'scope': receipt['scope'] = ['not text']
                (output / 'payment.json').write_bytes(self.c.document_bytes(receipt))
                self.assert_refused_before_payer(output)

    def test_receipt_requires_canonical_json_before_payer(self):
        for mutation in ('whitespace', 'invalid-json', 'duplicate-key'):
            with self.subTest(mutation=mutation):
                output = self.workspace()
                raw = self.c.document_bytes(self.receipt(output))
                if mutation == 'whitespace': raw += b'\n'
                if mutation == 'invalid-json': raw = b'{'
                if mutation == 'duplicate-key': raw = b'{"paid":true,' + raw[1:]
                (output / 'payment.json').write_bytes(raw)
                self.assert_refused_before_payer(output)

    def test_receipt_payment_requires_exact_shapes_before_payer(self):
        for path in (('payment',), ('payment', 'before'), ('payment', 'after')):
            for mutation in ('null', 'list', 'missing', 'extra'):
                with self.subTest(path=path, mutation=mutation):
                    output = self.workspace()
                    receipt = self.receipt(output)
                    parent = receipt
                    for field in path[:-1]:
                        parent = parent[field]
                    field = path[-1]
                    if mutation == 'null': parent[field] = None
                    elif mutation == 'list': parent[field] = []
                    elif mutation == 'missing': parent[field].pop(next(iter(parent[field])))
                    elif mutation == 'extra': parent[field]['extra'] = 0
                    (output / 'payment.json').write_bytes(self.c.document_bytes(receipt))
                    self.assert_refused_before_payer(output)

    def test_receipt_payment_fields_fail_before_payer(self):
        invalid_fields = {
            'txhash': (None, 'e' * 64, 'E' * 63),
            'height': (0, -1, True, 4.0, '4'),
            'amount_uzrn': ('0', '010000000', 10000000, '9000000'),
            'fee_uzrn': ('0', '2000001', 2000000),
            'recipient': (None, self.task['reward']['recipient'] + ' '),
            'bank_deltas_verified': (False, 1),
            'assurance': (None, [], '', 'INDEPENDENT_PROOF'),
        }
        for field, variants in invalid_fields.items():
            for value in variants:
                with self.subTest(field=field, value=value):
                    output = self.workspace()
                    receipt = self.receipt(output)
                    receipt['payment'][field] = value
                    (output / 'payment.json').write_bytes(self.c.document_bytes(receipt))
                    self.assert_refused_before_payer(output)

    def test_receipt_bank_types_and_deltas_fail_before_payer(self):
        for observation in ('before', 'after'):
            for account in ('sender', 'recipient', 'supply'):
                for value in (-1, True, '0', 0.0, 123):
                    with self.subTest(observation=observation, account=account, value=value):
                        output = self.workspace()
                        receipt = self.receipt(output)
                        receipt['payment'][observation][account] = value
                        (output / 'payment.json').write_bytes(self.c.document_bytes(receipt))
                        self.assert_refused_before_payer(output)

    def retain_receipt_before_append(self, output):
        backend = mock.Mock()
        backend.pay.return_value = copy.deepcopy(self.payment)
        with mock.patch.object(self.c, 'payer', return_value=backend):
            with mock.patch.object(self.c.journal(), 'append_records', side_effect=OSError('synthetic interruption')) as append:
                with self.assertRaises(OSError):
                    self.c.pay_workspace(output, output.parent / 'SYNTHETIC_PAYER')
        return copy.deepcopy(append.call_args.args[1])

    def test_payment_additions_must_equal_complete_receipt_derived_records(self):
        for mutation in ('both-contributions', 'title', 'summary', 'attribution', 'evidence',
                         'relation-type', 'relation-from', 'relation-to', 'relation-summary', 'receipt-txhash', 'partial'):
            with self.subTest(mutation=mutation):
                output = self.workspace()
                records = self.retain_receipt_before_append(output)
                if mutation == 'both-contributions':
                    source = self.c.journal().export_store(output / 'journal-store')['entries'][0]['record']
                    records = [{**copy.deepcopy(source), 'id': record['id']} for record in records]
                elif mutation == 'title': records[0]['title'] = 'SYNTHETIC FORGED TITLE'
                elif mutation == 'summary': records[0]['summary'] = 'SYNTHETIC FORGED SUMMARY'
                elif mutation == 'attribution': records[0]['attributed_to'] = 'SYNTHETIC OTHER OPERATOR'
                elif mutation == 'evidence': records[0]['evidence'][0]['sha256'] = '0' * 64
                elif mutation == 'relation-type': records[1]['relation_type'] = 'supports'
                elif mutation == 'relation-from': records[1]['from_id'] = 'work:' + self.task['task_id'] + ':experiment'
                elif mutation == 'relation-to': records[1]['to_id'] = 'work:' + self.task['task_id'] + ':source'
                elif mutation == 'relation-summary': records[1]['summary'] = 'SYNTHETIC FORGED RELATION'
                elif mutation == 'receipt-txhash':
                    receipt = json.loads((output / 'payment.json').read_bytes())
                    receipt['payment']['txhash'] = 'F' * 64
                    (output / 'payment.json').write_bytes(self.c.document_bytes(receipt))
                elif mutation == 'partial': records = records[:1]
                self.c.journal().append_records(output / 'journal-store', records)
                self.assert_refused_before_payer(output)

    def test_paid_export_requires_receipt_and_complete_live_additions(self):
        for missing in ('receipt-and-additions', 'receipt', 'additions'):
            with self.subTest(missing=missing):
                output = self.workspace()
                records = self.retain_receipt_before_append(output)
                original = self.c.journal().export_store(output / 'journal-store')
                if missing == 'receipt':
                    exported = self.c.journal().append_records(output / 'journal-store', records)
                else:
                    # A valid export with the expected records is still invalid if
                    # its prerequisites were never appended to the live store.
                    exported = self.c.journal().make_export(
                        [row['record'] for row in original['entries']] + records,
                        collection_id=original['header']['collection_id'],
                        created_at=original['header']['created_at'],
                        recorded_at=[row['recorded_at'] for row in original['entries']]
                                    + [original['entries'][-1]['recorded_at']] * len(records))
                if missing != 'additions':
                    (output / 'payment.json').unlink()
                (output / 'paid-journal.json').write_bytes(self.c.document_bytes(exported))
                self.assert_refused_before_payer(output)

    def test_paid_export_must_validate_and_match_exact_live_history(self):
        for mutation in ('null', 'list', 'invalid-json', 'hash', 'original-only', 'different-root',
                         'different-append-time', 'whitespace'):
            with self.subTest(mutation=mutation):
                output = self.workspace()
                records = self.retain_receipt_before_append(output)
                current = self.c.journal().append_records(output / 'journal-store', records)
                exported = copy.deepcopy(current)
                if mutation == 'null': exported = None
                elif mutation == 'list': exported = []
                elif mutation == 'hash': exported['entries'][-1]['sha256'] = '0' * 64
                elif mutation == 'original-only':
                    exported = self.c.journal().load_export(output / 'journal.json')
                elif mutation == 'different-root':
                    exported = self.c.journal().make_export([row['record'] for row in current['entries']])
                elif mutation == 'different-append-time':
                    exported = self.c.journal().make_export(
                        [row['record'] for row in current['entries']],
                        collection_id=current['header']['collection_id'], created_at=current['header']['created_at'],
                        recorded_at=[row['recorded_at'] for row in current['entries'][:-2]]
                                    + ['2099-01-01T00:00:00.000000Z'] * 2)
                raw = self.c.document_bytes(exported)
                if mutation == 'invalid-json': raw = b'{'
                if mutation == 'whitespace': raw += b'\n'
                (output / 'paid-journal.json').write_bytes(raw)
                self.assert_refused_before_payer(output)

    def test_consistent_receipt_does_not_replace_native_observation(self):
        output = self.workspace()
        self.retain_receipt_before_append(output)
        receipt_bytes = (output / 'payment.json').read_bytes()
        journal_bytes = (output / 'journal-store/journal.jsonl').read_bytes()
        backend = mock.Mock()
        backend.pay.side_effect = ValueError('synthetic native recovery refusal')
        with mock.patch.object(self.c, 'payer', return_value=backend) as factory:
            with self.assertRaisesRegex(ValueError, 'synthetic native recovery refusal'):
                self.c.pay_workspace(output, output.parent / 'SYNTHETIC_PAYER')
            factory.assert_called_once()
            backend.pay.assert_called_once()
        self.assertEqual((output / 'payment.json').read_bytes(), receipt_bytes)
        self.assertEqual((output / 'journal-store/journal.jsonl').read_bytes(), journal_bytes)
        self.assertFalse((output / 'paid-journal.json').exists())

    def test_initial_and_interrupted_states_still_require_native_recovery(self):
        for state in ('initial', 'receipt', 'journal-additions', 'complete'):
            with self.subTest(state=state):
                output = self.workspace()
                backend = mock.Mock()
                backend.pay.return_value = copy.deepcopy(self.payment)
                payer_home = output.parent / 'SYNTHETIC_PAYER'
                with mock.patch.object(self.c, 'payer', return_value=backend) as factory:
                    if state == 'receipt':
                        with mock.patch.object(self.c.journal(), 'append_records', side_effect=OSError('synthetic interruption')):
                            with self.assertRaises(OSError):
                                self.c.pay_workspace(output, payer_home)
                        self.assertTrue((output / 'payment.json').is_file())
                        self.assertFalse((output / 'paid-journal.json').exists())
                    elif state == 'journal-additions':
                        write = self.c.write_or_compare

                        def interrupt_export(path, value):
                            if path.name == 'paid-journal.json':
                                raise OSError('synthetic interruption')
                            return write(path, value)

                        with mock.patch.object(self.c, 'write_or_compare', side_effect=interrupt_export):
                            with self.assertRaises(OSError):
                                self.c.pay_workspace(output, payer_home)
                        self.assertFalse((output / 'paid-journal.json').exists())
                    elif state == 'complete':
                        self.c.pay_workspace(output, payer_home)
                    factory.reset_mock()
                    backend.reset_mock()
                    checked = self.c.checked_workspace(output)
                    retained = {name: (output / name).read_bytes()
                                for name in ('payment.json', 'paid-journal.json') if (output / name).exists()}
                    journal_before = (output / 'journal-store/journal.jsonl').read_bytes()
                    receipt = self.c.pay_workspace(output, payer_home)
                    factory.assert_called_once_with(payer_home)
                    backend.pay.assert_called_once_with(self.task['task_id'], self.task['reward'],
                        checked['result_sha256'], checked['journal_sha256'],
                        expected_txhash=None if state == 'initial' else self.payment['txhash'])
                    self.assertTrue(receipt['paid'])
                    for name, raw in retained.items():
                        self.assertEqual((output / name).read_bytes(), raw)
                    current = self.c.journal().export_store(output / 'journal-store')
                    self.assertEqual(len(current['entries']), 14)
                    self.assertEqual(self.c.journal().load_export(output / 'paid-journal.json'), current)
                    if state in ('journal-additions', 'complete'):
                        self.assertEqual((output / 'journal-store/journal.jsonl').read_bytes(), journal_before)


class NativeRecoveryIntegrationTests(unittest.TestCase):
    """Real coordinator + NativePayer; only bounded synthetic observations/sends.

    All receipts, participant files and transactions are synthetic. NativeTests
    blocks sockets/subprocesses and retains real custody checks and locking.
    """
    STATES = ('receipt-only', 'receipt-with-journal-no-export', 'complete')

    def setUp(self):
        import coordinator as c
        import test_native as n
        self.c, self.n = c, n
        self.fixture = n.NativeTests()
        self.addCleanup(self.fixture.doCleanups)
        self.fixture.setUp()
        self.prepare_workspace('accepted')

    def prepare_workspace(self, name):
        c, n = self.c, self.n
        self.fixture.backend.calls.clear()
        root = Path(self.fixture.temp.name)
        worker = c.load_module('fixture_external_worker', HERE / 'worker.py')
        self.task = worker.make_task('ab' * 32, 3, self.fixture.reward())
        result = worker.compute(self.task, 'SYNTHETIC_RECOVERY_ONLY_INTEGRATION')
        (root / 'task.json').write_bytes(c.document_bytes(self.task))
        (root / 'result.json').write_bytes(c.document_bytes(result))
        self.output = root / name
        c.prepare(root / 'task.json', root / 'result.json', self.output)
        self.checked = c.checked_workspace(self.output)
        self.context = {'task_id': self.task['task_id'],
                        'result_sha256': self.checked['result_sha256'],
                        'journal_sha256': self.checked['journal_sha256']}
        self.fixture.configure_payment()
        self.native_payer = self.fixture.payer()
        self.fixture.payment_tx['tx']['body']['memo'] = self.native_payer.settlement.payment_memo(
            self.context['task_id'], self.context['result_sha256'], self.context['journal_sha256'])
        self.marker = self.fixture.home / 'claim-workflow' / ('external-work-' + self.task['task_id'] + '.json')
        self.receipt = c.payment_receipt_for(self.checked, {
            'txhash': n.PAYMENT_HASH, 'height': 12, 'amount_uzrn': self.task['reward']['amount_uzrn'],
            'fee_uzrn': '2000000', 'recipient': n.RECIPIENT, 'bank_deltas_verified': True,
            'before': {'sender': 10000000, 'recipient': 0, 'supply': 1000000000},
            'after': {'sender': 7000000, 'recipient': 1000000, 'supply': 1000000000},
            'assurance': 'PINNED_OPERATOR_CHAIN_OBSERVATIONS_NOT_INDEPENDENT_INCLUSION_PROOF'})

    def pay(self):
        def open_payer(home):
            self.assertEqual(home, self.fixture.home)
            return self.fixture.payer()
        with mock.patch.object(self.c, 'payer', side_effect=open_payer):
            return self.c.pay_workspace(self.output, self.fixture.home)

    def snapshot(self):
        return {str(path.relative_to(self.output)): path.read_bytes()
                for path in self.output.rglob('*') if path.is_file()}

    def retain_payment(self, state):
        self.assertIn(state, self.STATES)
        (self.output / 'payment.json').write_bytes(self.c.document_bytes(self.receipt))
        if state != 'receipt-only':
            exported = self.c.journal().append_records(
                self.output / 'journal-store', self.c.payment_records_for(self.receipt))
            if state == 'complete':
                (self.output / 'paid-journal.json').write_bytes(self.c.document_bytes(exported))
        self.assertEqual(self.c.checked_workspace(self.output)['payment_receipt'], self.receipt)

    def recovery_calls(self, status='committed'):
        calls: list[tuple] = [('reconcile',)]
        if status == 'pending':
            calls.append(('tx_observation', self.n.PAYMENT_HASH))
        calls.extend([
            ('query', ('tx', self.n.FUNDING_HASH), None),
            ('query', ('bank', 'balances', self.n.SENDER), 10),
            ('query', ('tx', self.n.PAYMENT_HASH), None)])
        for height in (11, 12):
            for args in (('bank', 'balances', self.n.SENDER),
                         ('bank', 'balances', self.n.RECIPIENT), ('bank', 'total')):
                calls.append(('query', args, height))
        return calls

    def sdk_timeout(self):
        return subprocess.TimeoutExpired(['SYNTHETIC PRIVATE CLI ARGUMENT'], 60,
            output=b'SYNTHETIC PRIVATE STDOUT', stderr=b'SYNTHETIC PRIVATE STDERR')

    def run_cli(self, command, *, factory=None):
        if command == 'funding':
            arguments = ['funding', '--task', str(self.output / 'task.json')]
        else:
            self.assertEqual(command, 'pay')
            arguments = ['pay', '--workspace', str(self.output), '--confirm-test-token-payment']
        arguments += ['--payer-home', str(self.fixture.home)]
        out, err = io.StringIO(), io.StringIO()
        def open_payer(home):
            self.assertEqual(home, self.fixture.home)
            return self.native_payer if factory is None else factory()
        with mock.patch.object(self.c, 'payer', side_effect=open_payer), \
                contextlib.redirect_stdout(out), contextlib.redirect_stderr(err):
            try:
                code = self.c.main(arguments)
            except subprocess.TimeoutExpired:
                self.fail('SDK subprocess.TimeoutExpired escaped coordinator.main instead of a controlled refusal')
        return code, out.getvalue(), err.getvalue()

    def assert_timeout_refusal(self, result):
        code, out, err = result
        self.assertEqual(code, 1)
        self.assertEqual(out, '')
        self.assertIn('external-worker refused:', err)
        self.assertIn('timed out', err)
        self.assertNotIn('Traceback', err)
        self.assertNotIn('SYNTHETIC PRIVATE', err)

    def timeout_query(self, arguments, *, height=None):
        workflow = self.native_payer.workflow
        real_query = type(workflow).query.__get__(workflow)
        def query(*args, height=None):
            if (args, height) == target:
                return real_query(*args, height=height)
            return self.fixture.backend.query(*args, height=height)
        target = (arguments, height)
        def cli(*args, **kwargs):
            self.fixture.backend.assert_locked()
            expected = ('query', *arguments, '--node', workflow.rpc, '--output', 'json')
            if height is not None:
                expected += ('--height', str(height))
            self.assertEqual(args, expected)
            raise self.sdk_timeout()
        workflow.query = query
        workflow.cli = mock.Mock(side_effect=cli)
        return workflow.cli

    def test_cli_constructor_subprocess_timeout_is_a_safe_refusal(self):
        backend = self.fixture.backend
        original_open = backend.open
        for command in ('funding', 'pay'):
            for stage in ('check', 'read'):
                with self.subTest(command=command, stage=stage):
                    before = self.snapshot()
                    opens, reads = backend.open_calls, backend.network_checks
                    backend.network_error = self.sdk_timeout() if stage == 'read' else None
                    def open_with_timeout(home, module, workflow_type):
                        original_check = workflow_type.check_files
                        def check_files(workflow):
                            original_check(workflow)
                            if stage == 'check':
                                raise self.sdk_timeout()
                        with mock.patch.object(workflow_type, 'check_files', check_files):
                            return original_open(home, module, workflow_type)
                    with mock.patch.object(backend, 'open', side_effect=open_with_timeout):
                        self.assert_timeout_refusal(self.run_cli(command, factory=self.fixture.payer))
                    self.assertEqual(backend.open_calls, opens + 1)
                    self.assertEqual(backend.network_checks, reads + (stage == 'read'))
                    self.assertEqual(backend.calls, [])
                    self.assertFalse(self.marker.exists())
                    self.assertEqual(self.native_payer.workflow.records('attempts'), [])
                    self.assertEqual(self.snapshot(), before)

    def test_cli_funding_subprocess_timeout_is_a_safe_refusal(self):
        cli = self.timeout_query(('tx', self.n.FUNDING_HASH))
        before = self.snapshot()
        self.assert_timeout_refusal(self.run_cli('funding'))
        cli.assert_called_once()
        self.assertEqual(self.fixture.backend.calls, [])
        self.assertFalse(self.marker.exists())
        self.assertEqual(self.native_payer.workflow.records('attempts'), [])
        self.assertEqual(self.snapshot(), before)

    def test_cli_payment_prefunding_subprocess_timeout_never_reserves_or_sends(self):
        cli = self.timeout_query(('tx', self.n.FUNDING_HASH))
        before = self.snapshot()
        self.assert_timeout_refusal(self.run_cli('pay'))
        cli.assert_called_once()
        self.assertEqual(self.fixture.backend.calls, [('reconcile',)])
        self.assertFalse(self.marker.exists())
        self.assertEqual(self.native_payer.workflow.records('attempts'), [])
        self.assertEqual(self.snapshot(), before)

    def test_cli_signing_subprocess_timeout_retains_reservation_and_blocks_repeat(self):
        workflow = self.native_payer.workflow
        before = self.snapshot()
        calls = []
        def cli(*args, **kwargs):
            self.fixture.backend.assert_locked()
            # The real send must see the durable intent even before key lookup.
            reservation = self.native_payer.core.read_json(self.marker)
            self.assertEqual(reservation['context'], self.context)
            self.assertEqual(reservation['reward'], self.task['reward'])
            self.assertEqual(reservation['funding']['tx'], self.fixture.funding_tx)
            calls.append(args[:2])
            if args[:2] == ('keys', 'show'):
                return self.n.SENDER
            if args[:2] == ('tx', 'bank'):
                self.assertIn('--generate-only', args)
                return json.dumps(self.fixture.payment_tx['tx'])
            self.assertEqual(args[:2], ('tx', 'sign'))
            self.assertEqual(self.native_payer.core.read_json(Path(args[2])), self.fixture.payment_tx['tx'])
            raise self.sdk_timeout()
        workflow.cli = cli
        with mock.patch.object(workflow, 'send', wraps=type(workflow).send.__get__(workflow)) as send:
            result = self.run_cli('pay')
            self.assert_timeout_refusal(result)
            for guidance in ('retain this home', 'operator resolution', 'exact-hash recovery', 'never a new payment'):
                self.assertIn(guidance, result[2])
            self.assertEqual(calls, [('keys', 'show'), ('tx', 'bank'), ('tx', 'sign')])
            self.assertEqual(workflow.records('attempts'), [])
            retained = {path.name: path.read_bytes() for path in workflow.directory.iterdir() if path.is_file()}
            self.assertTrue(any(name.endswith('-unsigned.json') for name in retained))
            calls.clear()
            code, out, err = self.run_cli('pay')  # Explicit second invocation, not automatic retry.
            self.assertEqual((code, out), (1, ''))
            self.assertIn('pre-attempt', err)
            self.assertIn('operator resolution', err)
            send.assert_called_once()
            self.assertEqual(calls, [])
            self.assertEqual(workflow.records('attempts'), [])
            self.assertEqual({path.name: path.read_bytes() for path in workflow.directory.iterdir() if path.is_file()}, retained)
        self.assertEqual(self.snapshot(), before)

    def test_cli_payment_readback_subprocess_timeout_retains_attempt_for_recovery(self):
        cli = self.timeout_query(('bank', 'balances', self.n.RECIPIENT), height=12)
        before = self.snapshot()
        self.assert_timeout_refusal(self.run_cli('pay'))
        cli.assert_called_once()
        self.assertEqual(len([row for row in self.fixture.backend.calls if row[0] == 'send']), 1)
        attempts = self.native_payer.workflow.records('attempts')
        self.assertEqual(len(attempts), 1)
        attempt_path, attempt = attempts[0]
        self.assertEqual((attempt['status'], attempt['txhash']), ('committed', self.n.PAYMENT_HASH))
        attempt_before, reserved = attempt_path.read_bytes(), self.marker.read_bytes()
        self.assertEqual(self.snapshot(), before)
        # A separate invocation observes the same failure without another send.
        self.fixture.backend.calls.clear()
        cli.reset_mock()
        self.assert_timeout_refusal(self.run_cli('pay'))
        cli.assert_called_once()
        self.assertFalse(any(row[0] == 'send' for row in self.fixture.backend.calls))
        self.assertEqual(attempt_path.read_bytes(), attempt_before)
        self.assertEqual(self.marker.read_bytes(), reserved)
        self.assertEqual(self.snapshot(), before)
        # Remove only the synthetic timeout; explicit recovery reuses the attempt.
        self.fixture.backend.calls.clear()
        code, out, err = self.run_cli('pay', factory=self.fixture.payer)
        self.assertEqual((code, err), (0, ''))
        self.assertEqual(json.loads(out), self.receipt)
        self.assertEqual(self.fixture.backend.calls, self.recovery_calls())
        self.assertEqual(attempt_path.read_bytes(), attempt_before)
        self.assertEqual(self.marker.read_bytes(), reserved)

    def test_cli_recovery_subprocess_timeout_preserves_receipt_and_attempt(self):
        for state in self.STATES:
            for stage in ('reconcile', 'transaction', 'historical-bank'):
                with self.subTest(state=state, stage=stage):
                    self.prepare_workspace(state + stage)
                    self.retain_payment(state)
                    attempt = self.fixture.store_attempt(context=self.context,
                        status='pending' if stage == 'reconcile' else 'committed')
                    attempt_before, before = attempt.read_bytes(), self.snapshot()
                    if stage == 'historical-bank':
                        cli = self.timeout_query(('bank', 'total'), height=11)
                    else:
                        cli = self.timeout_query(('tx', self.n.PAYMENT_HASH))
                    if stage == 'reconcile':
                        workflow = self.native_payer.workflow
                        workflow.tx_observation = type(workflow).tx_observation.__get__(workflow)
                    self.assert_timeout_refusal(self.run_cli('pay'))
                    cli.assert_called_once()
                    self.assertFalse(any(row[0] == 'send' for row in self.fixture.backend.calls))
                    self.assertFalse(self.marker.exists())
                    self.assertEqual(attempt.read_bytes(), attempt_before)
                    self.assertEqual(self.snapshot(), before)

    def test_receipt_rejects_other_committed_hash_before_native_queries(self):
        for state in self.STATES:
            with self.subTest(state=state):
                self.prepare_workspace(state)
                self.receipt['payment']['txhash'] = 'F' * 64
                self.retain_payment(state)
                attempt = self.fixture.store_attempt(context=self.context)
                attempt_before, before = attempt.read_bytes(), self.snapshot()
                with self.assertRaises(self.fixture.native.Error):
                    self.pay()
                self.assertEqual(self.fixture.backend.calls, [('reconcile',)],
                    'A different committed hash must fail at selection, not a late receipt comparison')
                self.assertFalse(self.marker.exists())
                self.assertEqual(attempt.read_bytes(), attempt_before)
                self.assertEqual(self.snapshot(), before)

    def test_matching_receipt_hash_without_attempt_cannot_authorize_send(self):
        for state in self.STATES:
            for receipt_hash in (self.n.PAYMENT_HASH, 'F' * 64):
                with self.subTest(state=state, receipt_hash=receipt_hash):
                    self.prepare_workspace(state + receipt_hash[0])
                    self.receipt['payment']['txhash'] = receipt_hash
                    self.retain_payment(state)
                    before = self.snapshot()
                    with self.assertRaisesRegex(self.fixture.native.Error, 'exact expected transaction'):
                        self.pay()
                    self.assertEqual(self.fixture.backend.calls, [('reconcile',)])
                    self.assertFalse(self.marker.exists(), 'Recovery-only must not create a fresh reservation')
                    self.assertEqual(self.fixture.backend.workflow.records('attempts'), [])
                    self.assertEqual(self.snapshot(), before)

    def test_unrelated_committed_attempt_cannot_authorize_receipt_recovery(self):
        for state in self.STATES:
            with self.subTest(state=state):
                self.prepare_workspace(state)
                self.retain_payment(state)
                attempt = self.fixture.store_attempt(context={**self.context, 'task_id': 'f' * 64})
                attempt_before, before = attempt.read_bytes(), self.snapshot()
                with self.assertRaisesRegex(self.fixture.native.Error, 'exact expected transaction'):
                    self.pay()
                self.assertEqual(self.fixture.backend.calls, [('reconcile',)])
                self.assertFalse(self.marker.exists())
                self.assertEqual(attempt.read_bytes(), attempt_before)
                self.assertEqual(self.snapshot(), before)

    def test_matching_native_attempt_recovers_exact_transaction_and_historical_bank(self):
        for state in self.STATES:
            for status in ('committed', 'pending'):
                with self.subTest(state=state, status=status):
                    self.prepare_workspace(state + status)
                    self.retain_payment(state)
                    attempt = self.fixture.store_attempt(context=self.context, status=status)
                    attempt_before, before = attempt.read_bytes(), self.snapshot()
                    # Recovery must not demand a fresh budget or trust the receipt.
                    self.fixture.backend.responses[(('bank', 'balances', self.n.SENDER), 10)]['balances'] = []
                    self.assertEqual(self.pay(), self.receipt)
                    self.assertEqual(self.fixture.backend.calls, self.recovery_calls(status))
                    self.assertFalse(self.marker.exists())
                    self.assertEqual(len(self.fixture.backend.workflow.records('attempts')), 1)
                    self.assertEqual(self.n.shared.core.read_json(attempt)['status'], 'committed')
                    if status == 'committed':
                        self.assertEqual(attempt.read_bytes(), attempt_before)
                    for name, raw in before.items():
                        if state == 'receipt-only' and name == 'journal-store/journal.jsonl':
                            continue  # Only the previously missing payment append may change.
                        self.assertEqual((self.output / name).read_bytes(), raw)
                    current = self.c.journal().export_store(self.output / 'journal-store')
                    self.assertEqual(len(current['entries']), 14)
                    self.assertEqual(self.c.journal().load_export(self.output / 'paid-journal.json'), current)

    def test_conflicting_context_refuses_even_when_receipt_hash_matches(self):
        for state in self.STATES:
            for field in ('result_sha256', 'journal_sha256'):
                with self.subTest(state=state, field=field):
                    self.prepare_workspace(state + field)
                    self.retain_payment(state)
                    attempt = self.fixture.store_attempt(context={**self.context, field: 'f' * 64})
                    attempt_before, before = attempt.read_bytes(), self.snapshot()
                    with self.assertRaisesRegex(self.fixture.native.Error, 'Conflicting'):
                        self.pay()
                    self.assertEqual(self.fixture.backend.calls, [('reconcile',)])
                    self.assertFalse(self.marker.exists())
                    self.assertEqual(attempt.read_bytes(), attempt_before)
                    self.assertEqual(self.snapshot(), before)

    def test_locally_consistent_receipt_cannot_mask_bad_native_readback(self):
        for state in self.STATES:
            for mutation in ('memo', 'payment-hash', 'bank-credit', 'supply', 'funding'):
                with self.subTest(state=state, mutation=mutation):
                    self.prepare_workspace(state + mutation)
                    self.retain_payment(state)
                    attempt = self.fixture.store_attempt(context=self.context)
                    attempt_before, before = attempt.read_bytes(), self.snapshot()
                    if mutation == 'memo':
                        self.fixture.payment_tx['tx']['body']['memo'] = 'SYNTHETIC WRONG MEMO'
                    elif mutation == 'payment-hash':
                        self.fixture.payment_tx['txhash'] = 'F' * 64
                    elif mutation == 'bank-credit':
                        self.fixture.backend.responses[(('bank', 'balances', self.n.RECIPIENT), 12)]['balances'][0]['amount'] = '1000001'
                    elif mutation == 'supply':
                        self.fixture.backend.responses[(('bank', 'total'), 12)]['supply'][0]['amount'] = '1000000001'
                    elif mutation == 'funding':
                        self.fixture.funding_tx['code'] = 5
                    with self.assertRaises(self.fixture.native.Error):
                        self.pay()
                    self.assertFalse(any(row[0] == 'send' for row in self.fixture.backend.calls))
                    self.assertIn(('query', ('tx', self.n.FUNDING_HASH), None), self.fixture.backend.calls)
                    self.assertFalse(self.marker.exists())
                    self.assertEqual(attempt.read_bytes(), attempt_before)
                    self.assertEqual(self.snapshot(), before)

    def test_initial_payment_sends_once_then_repeat_is_exact_recovery(self):
        self.assertNotIn('payment_receipt', self.c.checked_workspace(self.output))
        first = self.pay()
        self.assertEqual(first, self.receipt)
        self.assertEqual(len([row for row in self.fixture.backend.calls if row[0] == 'send']), 1)
        self.assertEqual(len(self.fixture.backend.workflow.records('attempts')), 1)
        reservation, before = self.marker.read_bytes(), self.snapshot()
        self.fixture.backend.calls.clear()
        self.fixture.backend.responses[(('bank', 'balances', self.n.SENDER), 10)]['balances'] = []
        self.assertEqual(self.pay(), first)
        self.assertEqual(self.fixture.backend.calls, self.recovery_calls())
        self.assertEqual(self.marker.read_bytes(), reservation)
        self.assertEqual(self.snapshot(), before)
        self.assertEqual(len(self.fixture.backend.workflow.records('attempts')), 1)


if __name__ == '__main__':
    unittest.main()
