#!/usr/bin/env python3
"""Connect checked worker artifacts to the existing journal and test-token payer.

Preparation is offline. Payment is explicit, operator-mediated and loopback-test
only. Neither a record nor a transfer is scientific acceptance or task escrow.
"""
import argparse
import functools
import hashlib
import importlib.util
import os
import re
import sys
from pathlib import Path

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[1]
SOURCE_URL = 'https://raw.githubusercontent.com/bitcoin/bitcoin/v27.0/src/consensus/merkle.cpp'
SOURCE_SHA256 = '69e5bebee3d98f4ef711c1970d5814950310c4d6e5ee524625570354df0d18ad'
PAYMENT_FIELDS = ('txhash', 'height', 'amount_uzrn', 'fee_uzrn', 'recipient',
                  'bank_deltas_verified', 'before', 'after', 'assurance')


@functools.lru_cache(maxsize=None)
def load_module(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    if spec is None or spec.loader is None:
        raise ValueError('Cannot load pinned repository tool')
    value = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(value)
    return value


def journal():
    return load_module('external_work_journal', ROOT / 'tools/research-review/journal.py')


def records_for(task, result, task_sha256, result_sha256):
    """Format already checked inputs; this helper grants no payment authority."""
    prefix = 'work:' + task['task_id'] + ':'
    records = []
    coordinator = 'Zerone task coordinator (operator context; not independent scientific review)'
    evidence = lambda label, digest, url=None: {'label': label, 'url': url, 'sha256': digest}
    task_ref = evidence('Canonical task document', task_sha256)
    result_ref = evidence('Canonical worker result document', result_sha256)

    def contribution(name, kind, title, summary, attribution=coordinator, references=()):
        records.append({'id': prefix + name, 'kind': 'contribution', 'title': title,
                        'summary': summary, 'attributed_to': attribution, 'occurred_on': None,
                        'evidence': list(references), 'contribution_type': kind, 'doi': None})

    def relation(name, source, target, kind, summary):
        records.append({'id': prefix + name, 'kind': 'relation', 'title': summary,
                        'summary': summary, 'attributed_to': coordinator, 'occurred_on': None,
                        'evidence': [], 'from_id': prefix + source, 'to_id': prefix + target,
                        'relation_type': kind})

    contribution('source', 'source', 'Bitcoin Core v27.0 Merkle construction',
        'Version-pinned implementation and commentary document duplicate-last root ambiguity and mutation detection. Source authors are not participants in this task.',
        'Bitcoin Core developers (source attribution, not endorsement)',
        [evidence('Version-pinned consensus/merkle.cpp', SOURCE_SHA256, SOURCE_URL)])
    contribution('question', 'question', 'Does a Merkle root alone fix the length of this leaf list?',
        'Bounded computational reproduction: compare an odd-length deterministic leaf list with that list plus an explicit final-leaf duplicate. No live-network security assertion.', references=[task_ref])
    contribution('method', 'method', 'Double-SHA256 tree with duplicate-last padding',
        'Leaf i is SHA256(seed bytes || unsigned 4-byte big-endian i), indexed from zero. Leaves and roots use raw digest order, not displayed Bitcoin txid order. Pair hashes use double SHA256; mutation checks compare real adjacent pairs before odd padding. A separate reduction implementation recomputes both roots and flags.', references=[task_ref])
    contribution('experiment', 'experiment', 'External-worker Merkle reproduction',
        f"Task {task['task_id']}; {task['leaf_count']} nonce-derived leaves. Declared worker label is not an authenticated identity or independent controller. Nominated test-payment recipient: {task['reward']['recipient']}. No proof of physical compute expenditure is claimed.",
        result['worker_id'], [task_ref, result_ref])
    observed = result['result']
    contribution('finding', 'finding', 'Scoped duplicate-tail root equivalence',
        f"For this task's {observed['original_count']}-leaf input and {observed['extended_count']}-leaf extension, both roots are {observed['root_hex']}. Original mutation flag: {observed['original_mutated']}; extended flag: {observed['extended_mutated']}. This is a reproduction of a known construction-level property, not a new Bitcoin vulnerability, a double-SHA256 collision or proof that an invalid block is accepted. Candidate research finding; retains its exact method and scope.",
        result['worker_id'], [result_ref])
    contribution('review', 'review', 'Coordinator recomputation of worker artifact',
        'Both reductions, counts and mutation flags were recomputed from the task. This checks the supplied artifact, not independent worker control, scientific novelty, a current Bitcoin implementation, task funding, or an actual payment. A passing computation remains distinct from native review and a derived Fact.', references=[task_ref, result_ref])
    relation('question-source', 'question', 'source', 'cites', 'Question refers to the version-pinned source')
    relation('method-source', 'method', 'source', 'cites', 'Method refers to the specified source construction')
    relation('experiment-question', 'experiment', 'question', 'uses-input', 'Experiment addresses the scoped question')
    relation('experiment-method', 'experiment', 'method', 'uses-input', 'Experiment uses the declared computational method')
    relation('finding-experiment', 'finding', 'experiment', 'uses-input', 'Finding retains its experiment as an input')
    relation('review-finding', 'review', 'finding', 'uses-input', 'Coordinator review checks this finding only')
    return records


def document_bytes(value):
    return journal().canonical(value) + b'\n'


def sha256(raw):
    return hashlib.sha256(raw).hexdigest()


def read_bytes(path, limit):
    """Reuse the journal's bounded no-symlink directory/file traversal."""
    j = journal()
    path = Path(path)
    parent = j._open_dir(path.parent)
    try:
        descriptor = j._open_file(path.name, parent)
        try:
            return j._read_fd(descriptor, limit)
        finally:
            os.close(descriptor)
    finally:
        os.close(parent)


def prepare(task_path, result_path, output):
    j = journal()
    worker = load_module('external_work_computation', HERE / 'worker.py')
    task = j.read_json_file(task_path, 32768)
    result = j.read_json_file(result_path, 32768)
    verification = worker.verify(task, result)  # Must pass BEFORE any output exists.
    task_raw, result_raw = document_bytes(task), document_bytes(result)
    task_sha, result_sha = sha256(task_raw), sha256(result_raw)
    records = records_for(task, result, task_sha, result_sha)
    earlier = {}
    for record in records:
        j.validate_record(record, earlier)
        earlier[record['id']] = record
    output = Path(output)
    parent = j._open_dir(output.parent)
    try:
        os.mkdir(output.name, mode=0o700, dir_fd=parent)
        os.fsync(parent)
    finally:
        os.close(parent)
    # A partial preparation remains inspectable and cannot be overwritten.
    j._write_output(output / 'task.json', task)
    j._write_output(output / 'result.json', result)
    j._write_output(output / 'verification.json', verification)
    store = output / 'journal-store'
    j.create_store(store)
    j.append_records(store, records)
    export = j.export_store(store)
    j._write_output(output / 'journal.json', export)
    receipt = {'schema': 'zerone.external-work-preparation/v1', 'task_id': task['task_id'],
               'task_sha256': task_sha, 'result_sha256': result_sha,
               'journal_sha256': sha256(document_bytes(export)), 'journal_entries': len(export['entries']),
               'finding_id': 'work:' + task['task_id'] + ':finding',
               'computation_verified': True, 'paid': False,
               'scope': 'Scoped computational reproduction; operator-mediated test bounty, not independent review or a derived Fact.'}
    j._write_output(output / 'preparation.json', receipt)
    return receipt


def payment_receipt_for(checked, payment):
    task = checked['task']
    return {'schema': 'zerone.external-work-payment/v1', 'task_id': task['task_id'],
            'genesis_sha256': task['reward']['genesis_sha256'],
            'result_sha256': checked['result_sha256'], 'journal_sha256': checked['journal_sha256'],
            'paid': True, 'payment': payment,
            'scope': 'Operator-approved test bounty. Not protocol task escrow, independent science, token value or production activation.'}


def validate_payment_receipt(receipt, checked):
    """Check retained local consistency only; native recovery must still read the chain."""
    journal()._keys(receipt, {'schema', 'task_id', 'genesis_sha256', 'result_sha256',
                             'journal_sha256', 'paid', 'payment', 'scope'})
    if receipt['paid'] is not True or receipt != payment_receipt_for(checked, receipt['payment']):
        raise ValueError('Payment receipt differs from the bound task, result or pre-payment journal')
    payment = receipt['payment']
    journal()._keys(payment, set(PAYMENT_FIELDS))
    reward = checked['task']['reward']
    if (type(payment['txhash']) is not str or not re.fullmatch(r'[0-9A-F]{64}', payment['txhash'])
            or type(payment['height']) is not int or payment['height'] <= 0):
        raise ValueError('Payment receipt requires a committed transaction hash and positive integer height')
    if (payment['recipient'] != reward['recipient'] or payment['amount_uzrn'] != reward['amount_uzrn']
            or payment['fee_uzrn'] != '2000000' or payment['bank_deltas_verified'] is not True
            or payment['assurance'] != 'PINNED_OPERATOR_CHAIN_OBSERVATIONS_NOT_INDEPENDENT_INCLUSION_PROOF'):
        raise ValueError('Payment receipt differs from the expected test payment fields')
    before, after = payment['before'], payment['after']
    for row in (before, after):
        journal()._keys(row, {'sender', 'recipient', 'supply'})
        if any(type(value) is not int or value < 0 for value in row.values()):
            raise ValueError('Payment receipt requires nonnegative integer bank observations')
    amount, fee = int(payment['amount_uzrn']), int(payment['fee_uzrn'])
    if (after['recipient'] - before['recipient'] != amount
            or after['sender'] - before['sender'] != -(amount + fee)
            or before['supply'] != after['supply']):
        raise ValueError('Payment receipt bank changes differ from reward, fee or unchanged supply')


def payment_records_for(receipt):
    """Format the same complete records for retained-state checks and append."""
    prefix = 'work:' + receipt['task_id'] + ':'
    stable = receipt['payment']
    common = {'attributed_to': 'Zerone task coordinator (operator observations)', 'occurred_on': None}
    payment_record = {'id': prefix + 'payment', 'kind': 'contribution', 'contribution_type': 'finding', 'doi': None,
        'title': 'Observed development-token bounty payment',
        'summary': f"Committed SDK bank transfer {stable['txhash']} at height {stable['height']} paid {stable['amount_uzrn']} uzrn to {stable['recipient']}. H-1/H bank changes and supply were checked. These are operator-chain observations, not independent inclusion proofs or scientific acceptance.",
        'evidence': [{'label': 'Bound payment receipt', 'url': None, 'sha256': sha256(document_bytes(receipt))}], **common}
    relation = {'id': prefix + 'payment-finding', 'kind': 'relation', 'relation_type': 'uses-input',
        'from_id': payment_record['id'], 'to_id': prefix + 'finding', 'title': 'Payment binds this checked finding',
        'summary': 'Compensation records this task and result; it does not establish scientific truth.', 'evidence': [], **common}
    return [payment_record, relation]


def checked_workspace(output):
    j = journal()
    output = Path(output)
    values, hashes = {}, {}
    for name, limit in (('task', 32768), ('result', 32768), ('journal', j.MAX_BYTES)):
        raw = read_bytes(output / (name + '.json'), limit)
        value = j.parse_json(raw, limit)
        if raw != document_bytes(value):
            raise ValueError('Prepared artifacts must retain their canonical bytes')
        values[name] = value
        hashes[name + '_sha256'] = sha256(raw)
    load_module('external_work_computation', HERE / 'worker.py').verify(values['task'], values['result'])
    j.validate_export(values['journal'])
    expected = records_for(values['task'], values['result'], hashes['task_sha256'], hashes['result_sha256'])
    if [entry['record'] for entry in values['journal']['entries']] != expected:
        raise ValueError('Journal does not match the verified task and artifact')
    current = j.export_store(output / 'journal-store')
    original = values['journal']
    length = len(original['entries'])
    if current['header'] != original['header'] or current['entries'][:length] != original['entries']:
        raise ValueError('Retained task journal history differs from the bound snapshot')
    additions = [entry['record'] for entry in current['entries'][length:]]
    checked = {**values, **hashes}
    try:
        raw = read_bytes(output / 'payment.json', 32768)
    except FileNotFoundError:
        if additions:
            raise ValueError('Payment journal additions require a retained payment receipt')
    else:
        receipt = j.parse_json(raw, 32768)
        if raw != document_bytes(receipt):
            raise ValueError('Payment receipt must retain its canonical bytes')
        validate_payment_receipt(receipt, checked)
        if additions and additions != payment_records_for(receipt):
            raise ValueError('Payment journal additions differ from the retained receipt')
        checked['payment_receipt'] = receipt
    try:
        raw = read_bytes(output / 'paid-journal.json', j.MAX_BYTES)
    except FileNotFoundError:
        pass  # Initial, receipt-only and receipt-plus-append states can resume.
    else:
        j.validate_export(j.parse_json(raw, j.MAX_BYTES))
        # Nonempty additions already require a valid receipt and its exact records.
        if not additions or raw != document_bytes(current):
            raise ValueError('Paid journal export requires the exact retained payment journal history')
    return checked


def payer(home):
    return load_module('external_work_native', HERE / 'native.py').NativePayer(home)


def write_or_compare(path, value):
    j = journal()
    path = Path(path)
    if path.exists():
        if read_bytes(path, j.MAX_BYTES) != document_bytes(value):
            raise ValueError('Existing evidence differs; nothing was overwritten')
    else:
        j._write_output(path, value)


def pay_workspace(output, payer_home):
    checked = checked_workspace(output)  # Recheck before opening the signer.
    j = journal()
    output = Path(output)
    task = checked['task']
    retained = checked.get('payment_receipt')
    evidence = payer(payer_home).pay(task['task_id'], task['reward'],
        checked['result_sha256'], checked['journal_sha256'],
        expected_txhash=None if retained is None else retained['payment']['txhash'])
    stable = {key: evidence[key] for key in PAYMENT_FIELDS}
    if stable['bank_deltas_verified'] is not True or stable['recipient'] != task['reward']['recipient'] or stable['amount_uzrn'] != task['reward']['amount_uzrn']:
        raise ValueError('Native payment evidence differs from the task')
    receipt = payment_receipt_for(checked, stable)
    write_or_compare(output / 'payment.json', receipt)
    records = payment_records_for(receipt)
    store = output / 'journal-store'
    current = j.export_store(store)
    original = checked['journal']
    length = len(original['entries'])
    if current['header'] != original['header'] or current['entries'][:length] != original['entries']:
        raise ValueError('Original task journal history was changed; payment evidence is retained')
    additions = [entry['record'] for entry in current['entries'][length:]]
    if additions == []:
        j.append_records(store, records)
        current = j.export_store(store)
    elif additions != records:
        raise ValueError('Unexpected task journal additions; no history was overwritten')
    write_or_compare(output / 'paid-journal.json', current)
    return receipt


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest='command', required=True)
    prepare_command = commands.add_parser('prepare', help='Verify computation and create a fresh local task journal; no signing')
    for name in ('task', 'result', 'output'):
        prepare_command.add_argument('--' + name, required=True, type=Path)
    inspect_command = commands.add_parser('inspect', help='Recompute and validate a prepared task workspace; no signing')
    inspect_command.add_argument('--workspace', required=True, type=Path)
    funding_command = commands.add_parser('funding', help='Read sponsor funding on the exact loopback-test chain; no signing')
    funding_command.add_argument('--task', required=True, type=Path)
    funding_command.add_argument('--payer-home', required=True, type=Path)
    payment_command = commands.add_parser('pay', help='Explicit operator-approved SDK transfer of valueless test tokens')
    payment_command.add_argument('--workspace', required=True, type=Path)
    payment_command.add_argument('--payer-home', required=True, type=Path)
    payment_command.add_argument('--confirm-test-token-payment', required=True, action='store_true')
    args = parser.parse_args(argv)
    try:
        if args.command == 'prepare':
            response = prepare(args.task, args.result, args.output)
        elif args.command == 'inspect':
            checked = checked_workspace(args.workspace)
            response = {key: checked[key] for key in ('task_sha256', 'result_sha256', 'journal_sha256')}
            response.update(task_id=checked['task']['task_id'], computation_verified=True,
                            journal_entries=len(checked['journal']['entries']),
                            scope='Offline computation and journal integrity only; no payment observation')
        elif args.command == 'funding':
            task = journal().read_json_file(args.task, 32768)
            load_module('external_work_computation', HERE / 'worker.py').validate_task(task)
            response = payer(args.payer_home).funding(task['reward'])
        else:
            response = pay_workspace(args.workspace, args.payer_home)
        print(journal().canonical(response).decode())
        return 0
    except (ValueError, OSError, RuntimeError) as error:
        print('external-worker refused: ' + str(error), file=sys.stderr)
        return 1


if __name__ == '__main__':
    raise SystemExit(main())
