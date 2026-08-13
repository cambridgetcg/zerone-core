# Constructive-intelligence Fold-to-Fire v0

- Status: `SEALED_STATIC_PROFILE`
- Mode: `READ_ONLY_ZERO_EFFECT`
- Economic effect: `NONE`, amount `0`

## Purpose

`dashboard/public/standards/constructive-intelligence-fold-to-fire.v0.json`
freezes one deliberately small research challenge. It joins exact finite
square-lattice self-avoiding-walk enumeration to a carefully bounded analogy:
a fold can make a catalytic geometry available, while chemistry still has its
own rate. It is a mathematical polymer toy, not an atomic protein model or a
protein-design system.

The profile has three distinct claims:

1. exact finite counts and contact polynomials through 15 steps;
2. the established open, unweighted two-dimensional closing conjecture at
   `q=1`; and
3. a bespoke contact-weighted bridge at `q>0`, proposed for exploration but
   not advertised as a new or established open problem.

Those claims do not inherit one another's evidence status. Finite computation
is not an asymptotic proof, the unweighted exponent does not transfer to
`q!=1`, and endpoint adjacency is not enzyme activity.

## Immutable upstream bindings

The profile binds both raw bytes and canonical sorted-key JSON for three
existing artifacts:

| Source | Raw SHA-256 | Canonical JSON SHA-256 |
|---|---|---|
| Math Frontier v0 | `4fdcd54c35c69c26a28c385275688351ee2a9131702e81bacf100de8d7612456` | `b6260de31969a56e601a5a81f1b4f7c1c68fcd34f9aa68b35ce2701c3f012503` |
| Life Sciences v0 | `64dc2c5b2e21dfc9697d173317254ce651dede8661993ece7b380b7e1421496e` | `a208ea9e30a16ccfbb74f3f19298a5d3f93d7f87273b0b5aa10bf72e0e708822` |
| Money–KARMA v1 | `f22e62f0706971c569bb2156400b6dbeaf72a005d822b1e40c4e2691e7a98c24` | `a41286c936d3ab83d1cbd782b119cf3b434518ba80859edfe76f0de184143b7b` |

This is an additive profile. It changes none of those documents and imports
none of their quests, evidence labels, reward templates, qualification
semantics, governance, KARMA, network, laboratory, or clinical authority.

The Fold-to-Fire profile itself has these reviewed digests:

```text
raw SHA-256:       3fb78beaec220b4f62219a120ea33f46cfbe5ca1e76286929ae7b1120ccf4033
canonical SHA-256: 545c14c655494886b502f4c81eb1b71a99caec297f063a2097cded9dad3b893b
```

## Exact model

Let an `n`-step walk be vertices

```text
omega = (v_0, v_1, ..., v_n)
```

on the square lattice, with nearest-neighbour steps and no repeated vertex.
The implementation fixes `v_0=(0,0)` and `v_1=(1,0)`. This quotients out the
four rotations; mirror images remain distinct.

A contact is a nearest-neighbour pair of visited vertices whose sequence
indices differ by at least two. Write `C(omega)` for the number of contacts.
For odd `n >= 3`, a walk is active when `v_n` is a nearest neighbour of the
origin. In the polymer mathematics this is the usual closing event: one more
edge would make a self-avoiding polygon.

Define the contact-weighted polynomials

```text
Z_n(q) = sum_omega q^{C(omega)}
A_n(q) = sum_{omega in Active_n} q^{C(omega)}
```

for positive `q`. The finite toy bridge declares

```text
J_n(q) = kappa * A_n(q) / Z_n(q).
```

This product is a declared rapid-equilibrium, unit-occupancy toy assumption,
not a derived physical flux law. A real enzyme can have slow conformational
switching, substrate-dependent occupancy, binding and release gates, and
non-exponential turnover memory that this equation does not represent.

`A_n(q)/Z_n(q)` is only a geometric availability fraction under this lattice
weight. `kappa` is a separately declared chemical rate once the chosen
geometry is available. The equation does not derive `kappa`, binding,
transition-state stabilization, an equilibrium constant, or an experimental
turnover rate.

For the first hand-checkable case,

```text
Z_3(q) = 7 + 2q
A_3(q) = 2q
J_3(q)/kappa = 2q/(7+2q).
```

## Exact finite evidence

`fold-to-fire-exact-dfs/v0` enumerates every walk under the fixed-first-step
convention. Coefficients below are stored from contact degree zero upward.

| n | Z_n coefficients | A_n coefficients | Z_n(1) | A_n(1) |
|---:|---|---|---:|---:|
| 3 | 7, 2 | 0, 2 | 9 | 2 |
| 5 | 41, 22, 8 | 0, 0, 6 | 71 | 6 |
| 7 | 235, 184, 86, 38 | 0, 4, 0, 24 | 543 | 28 |
| 9 | 1331, 1344, 850, 346, 196 | 0, 10, 40, 0, 90 | 4067 | 140 |
| 11 | 7485, 9244, 6900, 3888, 1606, 888, 62 | 0, 54, 120, 240, 0, 306, 24 | 30073 | 744 |
| 13 | 41867, 60884, 52934, 33472, 19076, 7444, 3978, 720 | 0, 252, 672, 770, 1092, 112, 966, 252 | 220375 | 4116 |
| 15 | 233157, 389792, 383628, 276892, 169214, 91128, 37466, 17324, 5410, 138 | 0, 1232, 3264, 4496, 3904, 4672, 1408, 2976, 1504, 48 | 1604149 | 23504 |

These integers are exact for the declared finite convention. They are still
computational evidence, not proof of any limit, critical exponent, protein
folding rule, or catalytic mechanism. The validator recomputes each
coefficient sum and reduced closing fraction; the separate enumerator tests
cross-check the smaller cases with a differently scored reference kernel.

## The open problem

At `q=1`, all `n`-step walks have equal weight, so

```text
A_n(1)/Z_n(1)
```

is the square-lattice closing probability. The established conjecture is

```text
A_n(1)/Z_n(1) = n^{-59/32+o(1)}
```

along odd `n` as `n` tends to infinity. The profile marks this exactly as
`ESTABLISHED_OPEN_CONJECTURE` and `LITERATURE_CONJECTURE_NOT_PROVED`.
Duminil-Copin, Glazman, Hammond, and Manolescu state this predicted exponent
and prove weaker endpoint-delocalization bounds. Hammond gives the exact
closing identity in terms of polygon and walk counts and stronger later
upper bounds. Guttmann and Jensen describe the continuing open status of
rigorous two-dimensional critical exponents.

A practical challenge ladder is:

1. reproduce `n=3` by hand;
2. independently reproduce all exact rows through `n=15`;
3. find sharper finite identities, injections, or inequalities for the
   closing fraction;
4. relate a proved new finite bound to existing polygon-joining methods; and
5. only then attempt progress toward the asymptotic exponent.

Plotting seven points, fitting a slope, increasing the enumeration cutoff, or
matching the predicted number numerically does not prove `59/32`.

## The weighted bridge

The `q>0` formulation is labeled `BESPOKE_RESEARCH_BRIDGE`. Contact-weighted
interacting self-avoiding walks are established in the literature; the
profile claims no novelty for contact weighting. What is bespoke is the
specific pedagogical separation of `A_n/Z_n` from `kappa` as Fold-to-Fire.

Before publishing a claimed weighted theorem or open problem, perform a
novelty audit. No `q!=1` exponent is asserted here. In particular, the
`59/32` prediction must not be copied from the uniform model into an
interacting or collapsed regime.

## How folding relates to catalysis

For an ordinary enzyme, folding is usually not itself the catalyst. The
conformational ensemble can make catalytic arrangements accessible: residues
that are distant along the chain can approach, charges and solvent can be
organized, and substrates can be positioned. The chemical landscape then
has its own activation barriers. Roca and colleagues explicitly analyze the
relationship between folding and chemical landscapes; that motivates the
factorization, but does not validate this lattice toy as a quantitative
enzyme model.

There is a different meaning of *folding catalyst*: another molecule can
accelerate a client's route to a folded state. Peptidyl-prolyl isomerases
accelerate otherwise slow proline isomerization steps. Protein disulfide
isomerase can accelerate disulfide formation or rearrangement in trapped
intermediates. ATP-consuming chaperone systems can be driven nonequilibrium
machines, so they must not be compressed into an equilibrium-catalyst story.

For a minimal equilibrium first-passage illustration,

```text
U --u--> M --a--> F
U <--v-- M <--b-- F
```

the mean time to first reach `F` from `U` is

```text
T_U = (a + u + v)/(a*u).
```

Multiplying both `M -> F` and `F -> M` rates by the same positive factor `f`
preserves their equilibrium ratio `a/b`, while the first-passage time becomes

```text
T_U(f) = 1/u + (u+v)/(f*a*u).
```

This is a conceptual kinetic calculation, not a claim that all foldases act
by this three-state equilibrium mechanism. The ATP-chaperone source is
included precisely to preserve the nonequilibrium counterexample.

## Non-implication and safety walls

The machine profile freezes seven walls:

- finite enumeration is not asymptotic proof;
- a `q=1` exponent is not a `q!=1` or protein-folding exponent;
- a square-lattice polymer is not an atomic protein;
- endpoint adjacency is not an active site, binding event, or catalytic rate;
- a geometric availability fraction is not chemistry;
- client folding is not evidence of PPIase, PDI, or ATP-chaperone activity;
  and
- mathematics or biophysics creates no KARMA, money, reward, person worth,
  qualification, governance, or authority.

The profile contains no amino-acid or nucleotide sequence, wet-lab protocol,
medical or clinical guidance, protein prediction, catalyst design, network
request, event, receipt, score, or person evaluation. Every release switch is
false. Rest, refusal, silence, correction, and stopping remain valid outcomes;
there is no task completion event or negative recognition here.

## Offline validation

The strict validator rejects malformed, duplicate-key, excessively nested,
oversized, missing-field, reordered-field, and unknown-field JSON. It pins the
complete reviewed profile semantically and by raw digest. Each upstream
binding must be an existing regular non-symlink file inside the repository;
both its raw and canonical digests must match. All source URLs are exact
strings and are never fetched.

Run:

```sh
cd dashboard
node scripts/validate-constructive-intelligence-fold-to-fire.mjs \
  public/standards/constructive-intelligence-fold-to-fire.v0.json
node --test scripts/constructive-intelligence-fold-to-fire.test.mjs
```

Passing proves only exact static source shape, bounded arithmetic consistency,
and digest identity. It does not prove the conjecture, authenticate a research
contribution, establish scientific truth, or activate any external system.

## Exact source locators

The validator requires these exact URLs and performs no request:

- [Duminil-Copin et al., closing probability](https://doi.org/10.1214/14-AOP993)
- [Hammond, counting, joining and closing](https://arxiv.org/abs/1504.05286)
- [Jensen, square-lattice enumeration](https://doi.org/10.1088/0305-4470/37/21/002)
- [Guttmann and Jensen, existence of critical exponents](https://doi.org/10.1088/1751-8121/ac943a)
- [Bennett-Wood et al., interacting SAW enumeration](https://doi.org/10.1088/0305-4470/31/20/010)
- [Roca et al., folding and chemical landscapes](https://doi.org/10.1073/pnas.0803405105)
- [Lang, Schmid, and Fischer, prolyl isomerase](https://doi.org/10.1038/329268a0)
- [Weissman and Kim, protein disulfide isomerase](https://doi.org/10.1038/365185a0)
- [Goloubinoff et al., ATP-driven chaperones](https://doi.org/10.1038/s41589-018-0013-8)
