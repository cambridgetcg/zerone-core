"""Modularity contracts; all tasks and reward terms are synthetic, never paid."""
import json
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest

HERE = Path(__file__).resolve().parent

# Captured from coordinator.records_for at df848ce79c0b9ff02279a3ca2b3e46b216ae4574.
# Canonical JSON has no newline; artifact hashes passed to the formatter do.
RECORD_HASHES = {
    'paid': '5a01c3cf5d862a8930e4dc505210b3b922b5de8a555abdf34467aa632367d79b',
    'unpaid': '9c7f7c6f5922834d81b0ac4aa31dfdb861ff1fe61bd407466965ba1891776cda',
}


class StandaloneContributionTests(unittest.TestCase):
    def test_coordinator_uses_same_projection_and_preserves_preparation_bytes(self):
        from unittest import mock
        import coordinator as c
        import worker as w

        projection = c.load_module('external_work_contribution', HERE / 'contribution.py')
        reward = {'chain_id': 'zerone-dev-1', 'genesis_sha256': 'c' * 64,
                  'recipient': 'zrn1rj7z53777w3nwgwgm2lmqlly85ntajplcda70l',
                  'amount_uzrn': '10000000', 'funding_txhash': 'D' * 64}
        # Exact deterministic prepared files captured before extraction at df848ce7.
        expected = {
            'paid': {
                'journal-store/.lock': 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855',
                'journal-store/journal.jsonl': '58ffd37caf9c7461367e242baf17060e727dad192d6429430e83eac8f36e6995',
                'journal.json': 'b0d5b5f5c0600780f40f31b9b4436b66dc120ad52ce8641a44ba12631b000d05',
                'preparation.json': '04900a207003650773e291f296d25f44540d7d04c16e340b365dbd9289722b43',
                'result.json': 'b2350a1ec90bd52236deba0c4a567b777abaf9c401375b0c07bd4837dea96495',
                'task.json': '3f5796663d6e6362abcd37ae7291b819c529b51dd6b11ee2174acb93db7582df',
                'verification.json': '928c37c40eff011813ac2d4da7e523adeb928d4cbaee73b464dd1b8237f78c36',
            },
            'unpaid': {
                'journal-store/.lock': 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855',
                'journal-store/journal.jsonl': '9935b4b0b310fc6fc8345351b9ac0bf42b7444ac4f387d8403181ad7c4ccfc7a',
                'journal.json': '152779ada563f1de93657a6929e5079c513591ca1161577f273473084c60eeed',
                'preparation.json': 'e80a6b04473d1299dca16b52393df444c1764f73883ad8034cca66c6379deb08',
                'result.json': 'f4587d36fbb2587673a0943762650856a1f0815670129bca0bb51a8057c82217',
                'task.json': '848e578e60f22f835d8665226b6d15d3d65f71fcccb1934a70cb70495997021a',
                'verification.json': '14d2a632a63a9cdad827c586611eb6815b128f60145459ae2e88fad8af29337a',
            },
        }
        for name, task in [('unpaid', w.make_compute_task('ab' * 32, 3)),
                           ('paid', w.make_task('ab' * 32, 3, reward))]:
            with self.subTest(schema=task['schema']), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                result = w.compute(task, 'SYNTHETIC_MODULARITY_FIXTURE')
                (root / 'task.json').write_bytes(w.canonical(task) + b'\n')
                (root / 'result.json').write_bytes(w.canonical(result) + b'\n')
                j = c.journal()
                with mock.patch.object(projection, 'records_for', wraps=projection.records_for) as project, \
                     mock.patch.object(j, '_now', return_value='2026-01-01T00:00:00.000000Z'), \
                     mock.patch.object(j.uuid, 'uuid4', return_value=j.uuid.UUID('11111111-1111-4111-8111-111111111111')):
                    c.prepare(root / 'task.json', root / 'result.json', root / 'prepared')
                    checked = c.checked_workspace(root / 'prepared')
                self.assertEqual(project.call_count, 2, 'prepare and inspect must reuse the standalone projection')
                records = [entry['record'] for entry in checked['journal']['entries']]
                self.assertEqual(w.digest(w.canonical(records)), RECORD_HASHES[name])
                files = {str(path.relative_to(root / 'prepared')): w.digest(path.read_bytes())
                         for path in (root / 'prepared').rglob('*') if path.is_file()}
                self.assertEqual(files, expected[name])

    def test_projection_runs_with_only_worker_and_contribution(self):
        self.assertTrue((HERE / 'contribution.py').is_file(),
                        'research projection needs a standalone owner, not the coordinator')
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for name in ('worker.py', 'contribution.py'):
                shutil.copyfile(HERE / name, root / name)
            before = {path.name: path.read_bytes() for path in root.iterdir()}
            # Fresh interpreter, no repository import path or journal/payer files.
            # Audit tripwires are installed before either module is imported.
            program = r'''
import importlib.util
import json
import os
import sys

def no_effects(event, args):
    if event.startswith(('socket.', 'subprocess.', 'os.spawn', 'os.exec')):
        raise AssertionError('unexpected process or network effect: ' + event)
    if event in ('os.mkdir', 'os.remove', 'os.rename', 'os.rmdir', 'os.chmod', 'os.link', 'os.symlink'):
        raise AssertionError('unexpected filesystem mutation: ' + event)
    if event == 'open':
        mode, flags = args[1], args[2]
        if (mode and any(char in mode for char in 'wax+')) or flags & (os.O_WRONLY | os.O_RDWR | os.O_CREAT | os.O_TRUNC | os.O_APPEND):
            raise AssertionError('unexpected writable file open')

sys.addaudithook(no_effects)
def load(name):
    spec = importlib.util.spec_from_file_location(name, name + '.py')
    value = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(value)
    return value

w, projection = load('worker'), load('contribution')
reward = {'chain_id': 'zerone-dev-1', 'genesis_sha256': 'c' * 64,
          'recipient': 'zrn1rj7z53777w3nwgwgm2lmqlly85ntajplcda70l',
          'amount_uzrn': '10000000', 'funding_txhash': 'D' * 64}
outputs = {}
for name, task in [('unpaid', w.make_compute_task('ab' * 32, 3)),
                   ('paid', w.make_task('ab' * 32, 3, reward))]:
    result = w.compute(task, 'SYNTHETIC_MODULARITY_FIXTURE')
    w.verify(task, result)
    before = w.canonical([task, result])
    task_hash = w.digest(w.canonical(task) + b'\n')
    result_hash = w.digest(w.canonical(result) + b'\n')
    records = projection.records_for(task, result, task_hash, result_hash)
    if before != w.canonical([task, result]):
        raise AssertionError('projection mutated its inputs')
    outputs[name] = w.digest(w.canonical(records))
    records[0]['evidence'][0]['label'] = 'changed output only'
    if w.digest(w.canonical(projection.records_for(task, result, task_hash, result_hash))) != outputs[name]:
        raise AssertionError('projection retains shared mutable output')
print(json.dumps(outputs, sort_keys=True))
'''
            process = subprocess.run([sys.executable, '-I', '-B', '-c', program], cwd=root,
                                     capture_output=True, text=True, timeout=15)
            self.assertEqual(process.returncode, 0, process.stderr)
            self.assertEqual(process.stderr, '')
            self.assertEqual(json.loads(process.stdout), RECORD_HASHES)
            self.assertEqual({path.name: path.read_bytes() for path in root.iterdir()}, before)


if __name__ == '__main__':
    unittest.main()
