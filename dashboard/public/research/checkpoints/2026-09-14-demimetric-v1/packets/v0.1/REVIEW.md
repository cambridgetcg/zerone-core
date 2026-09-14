> Packaging note from the primary assistant: the retained report below records the original three-variant run. This packet includes verifier v2, which adds the literal singleton-min inertia as a fourth 128-step variant and refuses optimized Python. Original report SHA-256: `c358608590865a5cab6d939462bb09063d9d0b8d1c416ba962f273fd90d4671c`. Source pins, page renders and the preserved failed serialization receipt mentioned below remain in private working evidence; the shareable source inventory is `SOURCES.json`.

# Internal mathematical check: witness, closure repair, and separate projection/scaling defects

This is a separately derived check by an agent working under the same operator. It is not external review, author contact, an attribution of misconduct, a novelty claim, or a formal proof-assistant certification. No repository or chain state was changed.

**Findings.** The proposed continuous scalar witness satisfies the positive generalized-demimetric condition and has a demiclosed residual, but its relaxed average is not pairwise nonexpansive. Nevertheless a direct positive-residual estimate proves a useful general demiclosedness replacement. That replacement alone does not repair the whole article: the primary PDF also confirms a missing projection premise in Lemma 3.3 and a false distance estimate in equation (13). A reconstructed version of the printed algorithm with normalized weights can converge to zero instead of its asserted projected target; the malformed literal weight statement is expressly not satisfied/verified by that reconstruction.

## 1. Exact scope of source comparison

The primary source is Hammad, Dafaalla and Abdalla, [PLOS One, DOI 10.1371/journal.pone.0319047](https://journals.plos.org/plosone/article?id=10.1371/journal.pone.0319047). I inspected retained original HTML/equation images, parsed the primary manuscript XML, and independently rendered/viewed the printable PDF pages 5–7. SOURCE-PINS.json binds the exact files; primary-page-05/06/07.png are the inspected renderings.

Definition 2.3(b), equation e505, gives the generalized condition used below. Section 3 requires positive parameters. Lemma 3.3's proof then infers nonexpansiveness of the weighted relaxed map from demimetricity; Lemma 2.7(3) supplies only quasi-nonexpansiveness under its own parameter condition. These properties are different: the former compares arbitrary pairs, the latter compares a point with a fixed point. [Primary article, Definitions 2.3 and Lemmas 2.7, 3.3](https://journals.plos.org/plosone/article?id=10.1371/journal.pone.0319047#article1.body1.sec3.p14).

Do not silently normalize the source. Image/XML e086 literally names `I−φ_j` where φ_j is introduced as a scalar parameter. Reading this as `I−Q_j` is an explicit notation repair. The family is described as finite, yet the family limit and iteration index share `u`; e138 is finite while other weight expressions are inconsistent. Theorem 3.4 prints an infinite sum of the iterate symbol μ_j equal to one. Algorithm equation (2) also contains an extra β factor inside the ℓ expression, whereas the source's own rewrite (9) says ℓ_n=ρZγ_n. Results about a coherent normalized finite average or equation (9) must name that interpretation.

## 2. Independent verification of the original witness

Let H=R and

```
Q(x) = 0          for x≤1,
       2x−2       for 1<x<3/2,
       1          for x≥3/2.
```

The branch values meet at both endpoints. Solving Q(x)=x gives candidates 0, 2, 1; only 0 belongs to its corresponding branch. Hence Fix(Q)={0}. For r=x−Q(x), the slack in the φ=1 generalized condition is

```
xr−r² = 0                   for x≤1,
        (2−x)(2x−2)         for 1<x<3/2,
        x−1                for x≥3/2.
```

Every slack is nonnegative. This is a global factor/sign proof, not evidence from a grid. Since r is continuous and weak convergence equals strong convergence in R, I−Q is demiclosed at zero.

Choose two identical maps, weights 1/2, b=1/2 and λ₁=3/4. Then Z=(I+Q)/2. Z(1)=1/2 and Z(3/2)=5/4: the output separation is 3/4, the input separation 1/2. More generally the output separation is (1+b)/2 for every b>0. Thus positive relaxation does not give the disputed all-pairs property. Z remains quasi-nonexpansive relative to its sole fixed point: for x≥0, 0≤Q(x)≤x; for x<0, Z(x)=x/2.

## 3. Sufficient Hilbert-space repair, without nonexpansiveness

**Precise proposition.** Let C be a subset of a real Hilbert space H. Let Q_j:C→H, j=1,…,m, with finite m and nonempty common fixed-point set F. Suppose fixed constants φ_j>0 satisfy, for every x∈C and p∈F,

\[
\phi_j\langle x-p,x-Q_jx\rangle\geq\|x-Q_jx\|^2.
\]

Suppose each I−Q_j is demiclosed at zero on C. This means that whenever x_n∈C, x_n⇀x∈C and x_n−Q_jx_n→0 in norm, then Q_jx=x. For fixed weights w_j>0 with sum one and fixed b_j>0, put

\[
V_j=I-b_j(I-Q_j),\qquad Z=\sum_{j=1}^m w_jV_j.
\]

Then Fix(Z)=F and I−Z is demiclosed at zero on C. No upper bound on b_j, and no pairwise Lipschitz assumption on Q_j or Z, is needed for this proposition. If an application requires Z to map C into itself, that is an additional domain requirement; the proof only evaluates Z:C→H. A closed convex C ensures weak limits of sequences in C stay in C.

**Proof.** Fix p∈F and write r_j=x−Q_jx, d=x−Zx. Normalization gives d=Σ_j w_jb_jr_j. Positivity therefore gives

\[
0\leq\sum_{j=1}^m\frac{w_jb_j}{\phi_j}\|r_j\|^2
\leq\langle x-p,d\rangle
\leq\|x-p\|\,\|d\|. \tag{A}
\]

If d=0, every r_j=0, proving Fix(Z)⊆F; the reverse inclusion follows directly from normalized weights. If x_n⇀x∈C and d_n→0, the weakly convergent sequence is bounded. For each fixed j,

\[
\|r_{j,n}\|^2\leq
\frac{\phi_j}{w_jb_j}\|x_n-p\|\,\|d_n\|\longrightarrow0.
\]

Demiclosedness of each original residual gives x∈F, hence Zx=x. This proves the proposition. Notice that no cancellation estimate based on pairwise nonexpansiveness was used.

**How much of Lemma 3.3 this repairs.** A bounded weakly convergent subsequence of β_n, residual β_n−Zβ_n→0, and μ_n−β_n→0 give a common Q_j-fixed weak limit through (A). They do not by themselves establish membership in the monotone-operator zero sets, or the final projection inequality. The first residual line in the printed lemma subtracts Φ^{k−1}α_n from itself; using consecutive Φ^{k−1}, Φ^k instead is another explicit correction, although the identity-operator witness below satisfies either version. Full resolvent hypotheses must also be stated for a complete repaired lemma.

## 4. Coefficients and genuine boundaries

For each j the elementary expansion gives

\[
\|V_jx-p\|^2\leq\|x-p\|^2
-b_j(2/\phi_j-b_j)\|x-Q_jx\|^2.
\]

The coefficient is equivalently `(2/(b_j φ_j)−1)||x−V_jx||²`. Strict decrease in this estimate requires `0<b_j<2/φ_j`; equality at the upper boundary loses strictness. Merely assuming b_j<1 is insufficient for arbitrary positive φ_j. These restrictions belong to this squared-distance argument, not proposition (A). For Q=0, φ=1, b=2 gives V=−I, an isometry with no strict decrease; b=3 gives V=−2I, which expands from the fixed point while its residual remains demiclosed.

Other adversarial cases explain the proposition's hypotheses:

- A zero weight or b=0 can hide a map entirely. Even b_n=1/n>0 with Q=0, x_n=1 gives averaged residual→0 but original residual=1. Variable coefficients need uniform lower bounds on the relevant w_j,n b_j,n/φ_j, not pointwise positivity alone.
- Signed weights 2,−1 with Q₁=0, Q₂=−I (φ₁=1, φ₂=2), b=1, cancel residuals and produce Z=I while the common fixed set is {0}.
- Without a common fixed point, constant maps −1 and +1 average to the zero map, whose fixed point belongs to neither common intersection.
- Allowing φ<0 permits Q₂=2I with φ₂=−1; averaging it with Q₁=0, φ₁=1 yields I and loses the intersection identity.
- Dropping normalization: weight 2 with constant Q=1 and b=1 gives Z=2, so its fixed point differs from Q's fixed point.
- Dropping demiclosedness: set r(x)=a(x)x, Q=x−r, with a(x)=min(1,|x−1|) away from 1 and a(1)=1. Fix(Q)={0} and xr≥r² globally, but x_n=1+1/n has r(x_n)=(n+1)/n²→0 while r(1)=1. A fixed positive relaxation retains this closure failure.

Some examples break the claimed *intersection/residual inference* without breaking demiclosedness of the averaged identity map itself. They must not be advertised as failures of every conclusion simultaneously. No infinite-family extension is asserted here.

## 5. Separate missing-projection witness

The primary PDF p5 states only μ*∈Θ for Lemma 3.3. Set Q_j=I, all B_k=0 with identity resolvents, normalized finite weights, g≡0, μ*=1 and μ_n=α_n=β_n=0. Then Θ=R, all displayed residuals vanish, and

\[
\langle g(\mu^*)-\mu^*,\mu_n-\mu^*\rangle=(-1)(-1)=1.
\]

Even if one additionally requires these sequences to follow the algorithm, zero initialization with g=0 stays zero. The asserted nonpositive limsup therefore fails under the coherent operator/weight notation, independently of the earlier nonexpansiveness issue. The missing condition is μ*=P_Θg(μ*), which this witness intentionally does not satisfy. The later Theorem 3.4 does include the projected target; this witness is not a counterexample to that theorem.

A sufficient repaired last step is: Θ nonempty closed convex, μ*=P_Θg(μ*), μ_n bounded and every weak cluster point in Θ. Choose a subsequence attaining the scalar limsup and then a weakly convergent subsequence. Its limit z∈Θ gives the desired inequality from the metric projection characterization ⟨g(μ*)−μ*,z−μ*⟩≤0. The projection condition cannot be inferred from μ*∈Θ alone.

## 6. Separate scaling defect and interpreted-algorithm stress test

PDF p7 equation (13) claims `||ρZγ−μ*||≤||Zγ−μ*||` because ρ<1. At Z=I, γ=μ*=1, ρ=1/2 this is `1/2≤0`, a false scalar inequality. Scaling toward zero is not scaling toward an arbitrary nonzero fixed point.

For an expressly reconstructed version of equation (9), take H=R, Q_j=I, B_k=0, g≡1, φ_j=1, b=1/4, ρ=1/2, λ=1/2 and a fixed normalized two-map average. Thus Z=I, Θ=R and the asserted projected target is 1. Use δ_n=1/(n+1), τ_n=δ_n², μ₀=μ₁=0 and the requested capped inertial displacement

\[
e_n=\operatorname{sign}(\mu_n-\mu_{n-1})
\min(|\mu_n-\mu_{n-1}|,\tau_n).
\]

Then |e_n|≤τ_n and

\[
\mu_{n+1}=\delta_n+(1-\delta_n)\rho(\mu_n+e_n),\qquad
|\mu_{n+1}|\leq\rho|\mu_n|+\delta_n+\rho\tau_n. \tag{B}
\]

Since ρ<1 and the forcing on the right tends to zero, (B) implies μ_n→0. Explicitly, for any ε>0 choose N so the forcing is at most (1−ρ)ε/2 after N; iterating gives `|μ_{N+k}|≤ρ^k|μ_N|+ε/2`, and then let k→∞. This is the limit argument; 128 exact steps are only a numerical/algebraic cross-check. The same bound holds if the printed singleton-min inertia instead gives e_n=τ_n sign(Δ_n) whenever Δ_n≠0. Thus adding or removing a finite positive cap does not remove this defect.

**Interpretation boundaries:** finite normalized weights and the source's coherent rewritten equation (9) are expressly used; the printed infinite sum of iterates μ_j equal to one is not asserted/satisfied by this construction. The capped inertia is also explicit. Therefore this is a counterexample to the stated *reconstructed algorithm/target*, plus an unqualified counterexample to equation (13)'s scalar inequality. It is not a proof that every malformed literal hypothesis has been instantiated.

The verifier additionally checks the growing partial-average reading with positive weights 2^{−j}, so Z_n=(1−2^{−n})I. Its extra factor `s_n(1−λ+λs_n)` lies in (0,1), and (B) still applies. This does not make finite partial sums normalized or resolve the source's indexing. The undamped ρ=1 run is a control **outside** the paper's parameter interval. Its approach toward 1 is not a proposed fully verified repair of the original algorithm.

## 7. Reproduction and limits

Run `python3 -I -B verify_exact.py --output NEW_RESULT.json`. Output is exclusive: an existing result is not overwritten. `exact-result.json` passed the piecewise polynomial certificate, fixed-point/continuity identities, separation witness, coefficient identities/boundaries, adversarial cases, projection/scaling arithmetic and three 128-step recurrences. Arithmetic uses only Python's exact Fraction type. For large optional partial-average rationals, the receipt records a rigorously checked enclosing rational interval plus a SHA256 of the normalized signed-hex numerator/denominator; calculations are not rounded internally.

The first run hit Python's decimal-serialization length limit while printing a large optional rational. Its source hash/error is preserved in first-attempt-serialization-failure.json. The correction bounds presentation only; it does not alter arithmetic or an asserted mathematical condition.

The hand proof above establishes the sufficient finite-family closure result. The script independently checks exact witness algebra and finite recurrence bounds; it does not mechanize Hilbert-space weak compactness, the infinite-time limit, the full article, or any claim about the editorial notice. No experimental ML replication, external referee adoption, or accepted scientific status is inferred.
