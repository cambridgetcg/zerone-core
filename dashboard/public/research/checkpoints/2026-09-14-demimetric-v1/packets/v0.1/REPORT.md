---
title: "A scoped audit of fixed-point identification and origin damping"
subtitle: "Technical review draft · case 2026-001 · version 0.1 · 14 September 2026"
lang: en-GB
---

**Status and attribution.** Prepared by the Zerone development assistant team,
including separately tasked checking agents under the same operator. No
external independent review, accountable human adoption, author response or
institutional endorsement has yet been obtained. This is a technical draft,
not a finding about anyone's intent. No novelty is claimed.

## Scope and result

We examine Lemma 3.3 and the origin-damping step in the proof of Theorem 3.4 of
Hammad, Dafaalla and Abdalla, *Creating a novel algorithm for studying the strong
convergence to a sequence with applications*, PLOS One (2025),
[DOI 10.1371/journal.pone.0319047](https://journals.plos.org/plosone/article?id=10.1371/journal.pone.0319047).
The publisher PDF, especially pp. 4–7, and its manuscript XML are the source
versions inspected. Source hashes and locators accompany this report.

The result has four distinct scopes:

| Check | Supported conclusion | Boundary |
|---|---|---|
| Generalized-demimetric relaxation | An explicit continuous scalar map disproves the claimed implication to pairwise nonexpansiveness. | This implication alone does not refute convergence. |
| Lemma 3.3's selected point | Its stated membership condition alone does not give the concluding projection inequality. | Adding the missing projection condition changes the statement. |
| Equation (13) and the recurrence in (9) | The displayed damping inequality fails; a coherent, explicitly specified interpretation of (9) converges to the wrong projected target. | The paper's inconsistent weight and algorithm notation prevents presenting this as a witness satisfying every literal printed symbol. |
| Auxiliary repair | A self-contained fixed-family lemma recovers weak-limit identification and the projection inequality. | This does not repair or validate the full algorithm. |

The separate [publisher expression of concern](https://journals.plos.org/plosone/article?id=10.1371/journal.pone.0343283)
concerns the peer-review process. It is not evidence for these mathematical
findings, and these findings do not establish misconduct.

## 1. A relaxed demimetric map need not be nonexpansive

Write the positive generalized-demimetric condition as

$$\phi\langle x-p,x-Qx\rangle\geq\|x-Qx\|^2,
\qquad p\in\operatorname{Fix}(Q),\quad\phi>0.$$

On the real line define

$$Q(x)=\begin{cases}0,&x\leq1,\\2x-2,&1<x<3/2,\\1,&x\geq3/2.\end{cases}$$

This map is continuous and its only fixed point is zero. Put $r=x-Qx$.
For $x\leq1$, $xr=r^2$. For $x>1$, $0\leq Qx\leq x$, so
$xr-r^2=Qx(x-Qx)\geq0$. Thus the condition holds everywhere with $\phi=1$.
Continuity on the real line also makes $I-Q$ demiclosed at zero.

Take two copies, weights $1/2,1/2$, and $b=1/2$. Then
$Z=V=(I+Q)/2$. Direct calculation gives

$$Z(1)=\frac12,\qquad Z(3/2)=\frac54,\qquad
|Z(3/2)-Z(1)|=\frac34>\frac12.$$

Nevertheless $|Vx|\leq|x|$ for all $x$: the map is quasi-nonexpansive.
Distances to fixed points and distances between arbitrary points are different
requirements. A normalized finite average therefore does not justify the
nonexpansiveness inference in the Lemma 3.3 proof.

## 2. Membership does not select the projection target

The lemma chooses $q\in\Theta$ but concludes an inequality of the form

$$\limsup_n\langle g(q)-q,x_n-q\rangle\leq0.$$

Set $H=\mathbb R$, all $Q_j=I$, all monotone operators $B_k=0$, and
$g\equiv0$. Every resolvent is $I$ and $\Theta=\mathbb R$. Choose $q=1$
and $x_n=a_n=b_n=0$ for every $n$. Every displayed residual vanishes,
including either the literal repeated-prefix residual or the natural
successive-prefix correction. The zero sequence also remains zero under the
algorithm's zero initialization. But

$$\langle g(1)-1,0-1\rangle=(-1)(-1)=1.$$

The additional condition $q=P_\Theta g(q)$ would exclude this choice:
here it selects zero. Theorem 3.4 does refer to a projected target, so this
particular witness does not contradict that theorem's conclusion.

## 3. Origin damping changes the target

The first inequality in (13) compares $\|\rho Z\gamma-q\|$ with
$\|Z\gamma-q\|$ using $\rho<1$. For $Z=I$, $\gamma=q=1$ and
$\rho=1/2$, it asserts $1/2\leq0$. Scaling about the origin need not reduce
distance to a different fixed point.

There is also an asymptotic witness for the explicit recurrence in (9), with
the following declared interpretation: a fixed finite family, positive weights
summing to one, $Z$ their average, and the unambiguous line
$\ell_n=\rho Z\gamma_n$ from (9). This interpretation resolves the source's
weight/index conflict; it is not a claim to satisfy the literal condition
$\sum_j\mu_j=1$, which uses the iterate symbol. The algorithm line in (2)
also contains an extra symbol absent from (9).

Set two $Q_j=I$ with weights $1/2,1/2$, two $B_k=0$ with resolvent
parameters one, $g\equiv1$, $\phi_j=1$, $b=1/4$, $\rho=1/2$,
$\lambda=1/2$, inertial cap one, and $x_0=x_1=0$, relabelling the two
starting indices as zero and one. This index shift preserves the stated
sequence-limit conditions. The constant $g$ is
contractive with, for example, Lipschitz bound $1/4$. Use

$$\delta_n=\frac1{n+1},\qquad \tau_n=\delta_n^2.$$

These satisfy $\delta_n\to0$, $\sum_n\delta_n=\infty$, and
$\tau_n/\delta_n\to0$. With $\Delta_n=x_n-x_{n-1}$, define

$$e_n=\begin{cases}\min\{\tau_n/|\Delta_n|,1\}\Delta_n,&\Delta_n\ne0,\\0,&\Delta_n=0,\end{cases}
\qquad a_n=x_n+e_n.$$

The displayed minimum in (9) has a singleton; the capped version above is an
explicitly declared interpretation for the numerical check. Taking that
singleton literally instead gives $e_n=\tau_n\operatorname{sign}(\Delta_n)$
when $\Delta_n\ne0$. Both versions have $|e_n|\leq\tau_n$, and the argument
below applies to each. Identity operators reduce (9) to

$$x_{n+1}=\delta_n+(1-\delta_n)\rho(x_n+e_n).$$

Consequently

$$|x_{n+1}|\leq\rho|x_n|+\delta_n+\rho\tau_n.$$

For completeness, any nonnegative sequence satisfying
$u_{n+1}\leq\rho u_n+\epsilon_n$, with $0<\rho<1$ and
$\epsilon_n\to0$, tends to zero: after any index $N$, iterate the inequality
to bound it by $\rho^{n-N}u_N+\sup_{k\geq N}\epsilon_k/(1-\rho)$;
then let $n$ and subsequently $N$ tend to infinity. Hence $x_n\to0$.
But $\Theta=\mathbb R$ and the projected equation $q=P_\Theta g(q)$
has the unique solution $q=1$.

The same bound survives the increasing partial-sum reading with identical
maps and weights $2^{-j}$: then $Z_n=(1-2^{-n})I$ and the additional factors
in (9) lie between zero and one. Neither variant satisfies an interpretation
that literally imposes summability of these positive iterates to one.
Clarifying the intended weight condition and algorithm is therefore an
essential question for the authors, not a typographical change we can hide.

## 4. A repaired auxiliary lemma

Here is an explicitly stated replacement for the limit-identification step.
The source prints $I-\phi_j$ where $\phi_j$ is a scalar; we expressly require
the operator residual $I-Q_j$ instead. We also specify a fixed family,
normalized weights and well-defined resolvents, and derive the prefix
residuals rather than relying on the source's self-subtracting expression.
Let $H$ be a real Hilbert space, $Q_1,\ldots,Q_m:H\to H$ satisfy the
positive generalized-demimetric condition with fixed $\phi_j>0$, and each
$I-Q_j$ be demiclosed at zero. Let $B_1,\ldots,B_s$ be maximal monotone,
$R_k=(I+\eta_k B_k)^{-1}$ with fixed $\eta_k>0$, and assume

$$\Theta=\bigcap_j\operatorname{Fix}(Q_j)\cap\bigcap_k B_k^{-1}(0)$$

is nonempty, closed and convex. Let $w_j>0$, $\sum_jw_j=1$, $c_j>0$ be
fixed, and define

$$V_j=(1-c_j)I+c_jQ_j,\qquad Z=\sum_jw_jV_j.$$

Let $g:H\to H$ and choose $q\in\Theta$ with $q=P_\Theta g(q)$.
Suppose $x_n$ is bounded,
$b_n=R_s\cdots R_1a_n$, and

$$x_n-a_n\to0,\qquad x_n-b_n\to0,\qquad b_n-Zb_n\to0.$$

Then every weak cluster point of $x_n$ belongs to $\Theta$, and

$$\limsup_n\langle g(q)-q,x_n-q\rangle\leq0.$$

**Proof.** Fix $p\in\Theta$, put $r_j(x)=x-Q_jx$ and
$d(x)=x-Zx=\sum_jw_jc_jr_j(x)$. Positivity gives

$$0\leq\sum_j\frac{w_jc_j}{\phi_j}\|r_j(b_n)\|^2
\leq\langle b_n-p,d(b_n)\rangle
\leq\|b_n-p\|\|d(b_n)\|\longrightarrow0.$$

Every coefficient is fixed and positive, so each residual tends to zero.
Along a weakly convergent subsequence of $x_n$, $b_n$ has the same weak
limit. Demiclosedness of each original residual puts that limit in every
$\operatorname{Fix}(Q_j)$. No pairwise nonexpansiveness of $Z$ is used.

For the resolvents let $y_{0,n}=a_n$ and $y_{k,n}=R_ky_{k-1,n}$.
Firm nonexpansiveness and $R_kp=p$ give the telescoping estimate

$$\sum_{k=1}^s\|y_{k-1,n}-y_{k,n}\|^2
\leq\|a_n-p\|^2-\|b_n-p\|^2\longrightarrow0.$$

The last limit follows from boundedness and $a_n-b_n\to0$.
Each prefix difference tends to zero. Since $R_k$ is nonexpansive,

$$\|a_n-R_ka_n\|\leq
2\|a_n-y_{k-1,n}\|+\|y_{k-1,n}-y_{k,n}\|\longrightarrow0.$$

Demiclosedness of $I-R_k$ therefore puts the common weak limit in
$\operatorname{Fix}(R_k)=B_k^{-1}(0)$ for every $k$.
Finally choose a subsequence attaining the scalar limsup, and a weakly
convergent subsubsequence, with limit $z\in\Theta$. The metric projection
condition gives $\langle g(q)-q,z-q\rangle\leq0$, as required.

This proof needs no upper bound on $c_j$ for this particular closure argument.
It does not assert quasi-nonexpansiveness of $V_j$ or convergence of the full
iteration for arbitrary $c_j$. Fixed positive coefficients, a common fixed
point, and demiclosedness are substantive assumptions.

The residual strategy has an antecedent in the directly cited
[Song (2018), proof of Theorem 3.4, p. 208](https://isr-publications.com/jnsa/articles-6677-iterative-methods-for-fixed-point-problems-and-generalized-split-feasibility-problems-in-banach-spaces).
Our argument is supplied in full for checking; it is not presented as a new
general principle.

## 5. Review questions and reproducibility

The included [Python verifier](verify_exact.py) uses exact rational arithmetic
for finite checks. Run `python3 -B verify_exact.py` from this packet's directory
and compare with [the stored result](exact-result.json). Python 3.10 or newer
and its standard library suffice; the program uses no network or third-party
packages. The [source inventory](SOURCES.json) gives publisher URLs, inspection
locators and the hashes of locally retained versions.
Those checks can detect errors in the examples and implementation; the
all-domain and asymptotic arguments above are mathematical proofs for internal
review, not consequences of a finite numerical test or machine-checked proofs.

An outside reviewer is asked to check the source transcription, each witness,
the declared interpretations, and the repaired lemma separately. The authors
should be asked which weights and algorithm line are intended, whether the
projection condition should be explicit in Lemma 3.3, and how the origin
damping preserves a nonzero target. No author has yet been contacted. A later
response should be retained and assessed without treating silence as assent.

Removing or changing the damping may be a useful next hypothesis, but a
complete replacement algorithm and convergence theorem have not been proved
here. Other sections, applications and the broader literature remain outside
this review. The draft does not establish novelty, fraud, a universal failure
of fixed-point methods, or the invalidity of every result in the article.
