"""Pure research-record projection for the fixed Merkle reproduction.

records_for(task, result, task_sha256, result_sha256) formats already verified
inputs into the existing Research Review record shape. It does NOT validate a
task, recompute a result, authenticate hashes or grant payment authority. Call
worker.verify first; validate the records at the journal boundary before storage.
The supplied artifact digests cover canonical JSON PLUS a trailing newline, unlike
worker.verify's protocol hashes. Worker labels remain self-declared attribution.

No imports, filesystem, network, clock, environment or repository layout is
required. Inputs are not mutated; each call returns fresh records. This module
owns the research wording, not journal persistence, native claims or settlement.
"""

SOURCE_URL = 'https://raw.githubusercontent.com/bitcoin/bitcoin/v27.0/src/consensus/merkle.cpp'
SOURCE_SHA256 = '69e5bebee3d98f4ef711c1970d5814950310c4d6e5ee524625570354df0d18ad'
COMPUTE_TASK_SCHEMA = 'zerone.external-compute-task/v1'
COMPUTE_SCOPE = ('Unpaid compute-only task: no funding, no nominated recipient and no payment eligibility. '
                 'Not independent review or a derived Fact.')


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
    payment_scope = (COMPUTE_SCOPE if task.get('schema') == COMPUTE_TASK_SCHEMA
                     else f"Nominated test-payment recipient: {task['reward']['recipient']}.")
    contribution('experiment', 'experiment', 'External-worker Merkle reproduction',
        f"Task {task['task_id']}; {task['leaf_count']} nonce-derived leaves. Declared worker label is not an authenticated identity or independent controller. {payment_scope} No proof of physical compute expenditure is claimed.",
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
