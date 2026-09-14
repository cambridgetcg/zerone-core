# A restricted repair that preserves the intended solutions

14 September 2026 · follow-up to the [coordinate check](../translation-v0.1/NOTE.md).
Internal mathematical draft by the Zerone development assistant team under
one operator. External review and accountable human adoption remain pending.

The earlier convex candidate removed coordinate bias but could introduce
spurious fixed points. Here we specify sufficient restrictions, prove that
the composite has exactly the intended solutions, and give a strong-convergence
proof for a declared iteration with small inertial errors. The restrictions
are sufficient, not claimed minimal. This is a replacement theorem for our
explicit algorithm; it does not validate the source article's literal theorem.

The useful mechanism is measurable at each stage: distance cannot increase
relative to a common solution, and the mixing steps expose residual errors
that would otherwise cancel. Even the boundary of the relaxation range can
be admitted when the inner mixing parameter stays strictly between zero and
one. Strict relaxation alone is therefore not a necessary condition here.

## 1. Precise setting and result

Let $H$ be a real Hilbert space. There are fixed finite families
$Q_j:H\to H$ and maximal monotone operators $B_k:H\rightrightarrows H$,
with $1\leq j\leq m$ and $1\leq k\leq s$. Suppose

$$\phi_j\langle x-p,x-Q_jx\rangle\geq\|x-Q_jx\|^2
\quad(x\in H,\ p\in\operatorname{Fix}Q_j),\qquad \phi_j>0.$$

Each $I-Q_j$ is demiclosed at zero: $z_i\rightharpoonup z$ and
$z_i-Q_jz_i\to0$ imply $Q_jz=z$. Assume the common solution set

$$\Theta=\bigcap_j\operatorname{Fix}Q_j\cap\bigcap_k B_k^{-1}(0)$$

is nonempty. Fix positive weights $w_j$ with $\sum_jw_j=1$, and constants

$$b>0,\quad b\phi_j\leq2\ \text{for every }j,\qquad
0<\lambda<1,\quad0<\rho<1,\quad h_k>0.$$

Let $R_k=(I+h_kB_k)^{-1}$, the everywhere-defined firm resolvent, and set

$$V_j=(1-b)I+bQ_j,\quad Z=\sum_jw_jV_j,\quad R=R_s\cdots R_1,$$
$$\beta=Rx,\quad\gamma=(1-\lambda)\beta+\lambda Z\beta,
\quad Ux=Z\gamma,\quad Sx=(1-\rho)x+\rho Ux.$$

Let $g:H\to H$ be a contraction with Lipschitz bound $0\leq\kappa<1$.
For $n\geq1$, define the complete iteration by

$$\delta_n=\frac1{n+1},\qquad \alpha_n=x_n+e_n,
\qquad\|e_n\|\leq\delta_n^2,$$
$$x_{n+1}=\delta_ng(x_n)+(1-\delta_n)S\alpha_n.$$

Initial $x_0,x_1\in H$ are arbitrary. This includes the earlier capped
inertial displacement, as well as zero inertia. The proof uses only the stated
error bound. In particular, $g$ is evaluated at $x_n$, not at $\alpha_n$.
All weights, operators and mixing parameters above remain fixed.

**Result.** The set $\Theta$ is closed and convex,
$\operatorname{Fix}U=\operatorname{Fix}S=\Theta$, and $I-S$ is demiclosed
at zero. The iterates converge in norm to the unique point

$$p=P_\Theta g(p).$$

No pairwise nonexpansiveness or continuity of $Q_j$, $Z$, $U$ or $S$ is
assumed. Firm resolvents and the metric projection have their usual
nonexpansiveness properties. The proof below does not replace the controlled
input error by an unjustified bound on $S\alpha_n-Sx_n$.

## 2. The loss estimate and the common fixed-point set

Write $r_j(t)=t-Q_jt$. Expansion and the demimetric condition give, for
$p\in\Theta$,

$$\|V_jt-p\|^2\leq\|t-p\|^2-
b(2/\phi_j-b)\|r_j(t)\|^2.$$

The coefficients are nonnegative under the declared bound. Define

$$A(t)=\sum_jw_jb(2/\phi_j-b)\|r_j(t)\|^2,$$
$$D(t)=\sum_jw_j\|V_jt-Zt\|^2,\qquad C(t)=A(t)+D(t).$$

The weighted squared-norm identity yields
$\|Zt-p\|^2\leq\|t-p\|^2-C(t)$. Thus $Z$ and each $V_j$ are
quasi-nonexpansive: they do not increase distance to their fixed points.
For completeness, the common fixed-point identity for $Z$ follows from

$$0\leq\sum_j\frac{w_jb}{\phi_j}\|r_j(t)\|^2
\leq\langle t-p,t-Zt\rangle.\qquad\text{(1)}$$

If $Zt=t$, every original residual vanishes. The converse is immediate.
Here $p$ may be any common fixed point of the $Q_j$; it need not be a zero
of the $B_k$.

Each $\operatorname{Fix}Q_j=\operatorname{Fix}V_j$ is closed by
demiclosedness. It is convex as follows. For fixed points $u,v$, let
$z=\theta u+(1-\theta)v$, $0<\theta<1$. Weighting the two
quasi-nonexpansive inequalities for $V_jz$ gives

$$\|V_jz-z\|^2+\theta(1-\theta)\|u-v\|^2
\leq\theta(1-\theta)\|u-v\|^2,$$

so $V_jz=z$. The zero sets of the $B_k$ are the closed convex fixed sets
of their firm resolvents. Hence their finite intersection $\Theta$ is
closed and convex. The contraction $P_\Theta\circ g$ on $\Theta$ therefore
has exactly one fixed point, the target $p$.

For the resolvent prefixes set $y_0=x$, $y_k=R_ky_{k-1}$ and
$E_R(x)=\sum_k\|y_{k-1}-y_k\|^2$. Firmness gives

$$\|Rx-p\|^2\leq\|x-p\|^2-E_R(x).$$

Using the same squared-norm identity for the inner mixture and then applying
$Z$ gives the joint estimate

$$\|Ux-p\|^2\leq\|x-p\|^2-L(x),$$
$$L(x)=E_R(x)+\lambda C(\beta)
+\lambda(1-\lambda)\|\beta-Z\beta\|^2+C(\gamma)\geq0.\qquad\text{(2)}$$

Finally the outer mixture gives

$$\|Sx-p\|^2\leq\|x-p\|^2-\rho L(x)
-\rho(1-\rho)\|x-Ux\|^2.\qquad\text{(3)}$$

If $Ux=x$, (2) forces all resolvent prefix differences to vanish, so
$x=\beta$ and $x\in\bigcap_k\operatorname{Fix}R_k$. It also forces
$\beta-Z\beta=0$ because $\lambda(1-\lambda)>0$. Equation (1) now
identifies every $Q_j$ fixed point. Thus $x\in\Theta$. Conversely every
$x\in\Theta$ is fixed at every stage. Since $\rho>0$,
$\operatorname{Fix}S=\operatorname{Fix}U=\Theta$.

With $\eta=(1-\rho)/\rho>0$, (3) also implies the quantitative estimate

$$\|Sx-p\|^2\leq\|x-p\|^2-\eta\|x-Sx\|^2.\qquad\text{(4)}$$

## 3. Vanishing composite residual identifies valid solutions

Suppose $z_i\rightharpoonup z$ and $z_i-Sz_i\to0$. The sequence is
bounded. Its distance-squared loss tends to zero, since

$$\left|\|z_i-p\|^2-\|Sz_i-p\|^2\right|
\leq\|z_i-Sz_i\|(\|z_i-p\|+\|Sz_i-p\|)\longrightarrow0.$$

Equation (3) forces $E_R(z_i)\to0$ and
$\beta_i-Z\beta_i\to0$. In particular, every prefix $y_{k,i}$ and
$\beta_i$ has the same weak limit $z$ as $z_i$. Equation (1) and boundedness
give $\beta_i-Q_j\beta_i\to0$ for every $j$. The assumed demiclosedness
puts $z$ in every $\operatorname{Fix}Q_j$.

Also $y_{k-1,i}-R_ky_{k-1,i}\to0$. Demiclosedness of the nonexpansive
resolvent residual gives $R_kz=z$ for every $k$. This standard closure fact
can be seen directly: if $u_i\rightharpoonup u$, $u_i-Tu_i\to0$ and $T$
is nonexpansive, expand $\|Tu_i-Tu\|^2\leq\|u_i-u\|^2$ after substituting
$Tu_i=u_i+o(1)$. Boundedness and weak convergence yield
$\|u-Tu\|^2\leq0$.

Therefore $z\in\Theta=\operatorname{Fix}S$, proving demiclosedness of
$I-S$. Notice that this argument still works if some or all coefficients
in $A$ vanish. The positive inner mixing term and (1) supply the needed
individual residual control at the endpoint $b\phi_j=2$.

## 4. Strong convergence of the declared iteration

Put $c=1-\kappa>0$, $h=g(p)-p$, $a_n=\|x_n-p\|^2$ and
$q_n=\|\alpha_n-S\alpha_n\|$. Quasi-nonexpansiveness gives

$$\|x_{n+1}-p\|\leq(1-c\delta_n)\|x_n-p\|
+\delta_n\|h\|+(1-\delta_n)\delta_n^2.$$

Thus $\|x_n-p\|\leq M$ for
$M=\max\{\|x_1-p\|,(\|h\|+1)/c\}$, by induction. This also bounds
$\alpha_n$, $S\alpha_n$ and $g(x_n)$ relative to $p$.

**Descent with vanishing forcing.** Apply convexity of squared norm to the
update and then (4). Since
$\|\alpha_n-p\|^2\leq a_n+2M\delta_n^2+\delta_n^4$, there is a fixed
finite $K_1$ such that

$$(1-\delta_n)\eta q_n^2\leq a_n-a_{n+1}+K_1\delta_n.\qquad\text{(5)}$$

For example, $K_1=G^2+2M+1$ works for
$G=\sup_n\|g(x_n)-p\|$. No limit for $a_n$ or $q_n$ has been assumed.

**Projection-sensitive recurrence.** Write

$$x_{n+1}-p=v_n+\delta_nh,\qquad
v_n=\delta_n(g(x_n)-g(p))+(1-\delta_n)(S\alpha_n-p).$$

Then $\|v_n\|\leq(1-c\delta_n)\sqrt{a_n}+\delta_n^2$.
The identity
$\|v_n+\delta_nh\|^2\leq\|v_n\|^2+
2\delta_n\langle h,x_{n+1}-p\rangle$ gives

$$a_{n+1}\leq(1-c\delta_n)a_n+c\delta_n t_n,\qquad\text{(6)}$$
$$t_n=\frac2c\langle h,x_{n+1}-p\rangle+
\frac{K_2}{c}\delta_n,\qquad K_2=2M+1.$$

Here $(1-c\delta_n)^2\leq1-c\delta_n$ and the remaining square terms
are bounded by $K_2\delta_n^2$.

**Where the projection sign becomes available.** Along any indices for which
$q_n\to0$, the update shows

$$x_{n+1}-\alpha_n=
\delta_n(g(x_n)-S\alpha_n)+(S\alpha_n-\alpha_n)\longrightarrow0.$$

Take a subsequence attaining the limsup of
$\langle h,x_{n+1}-p\rangle$, and a further weakly convergent subsequence
of the bounded $\alpha_n$. Demiclosedness of $I-S$ puts its limit $z$ in
$\Theta$; the same limit holds for $x_{n+1}$. The characterization of
$p=P_\Theta g(p)$ gives $\langle h,z-p\rangle\leq0$. Consequently
$\limsup t_n\leq0$ along those indices.

There are two exhaustive cases.

1. If $a_n$ is eventually nonincreasing, it has a nonnegative limit, so
   $a_n-a_{n+1}\to0$. Equation (5) gives $q_n\to0$, hence
   $\limsup t_n\leq0$. Given $\epsilon>0$, eventually (6) is bounded
   by $(1-c\delta_n)a_n+c\delta_n\epsilon$. Iteration and
   $\sum_n c\delta_n=\infty$ imply $\limsup a_n\leq\epsilon$.
   Letting $\epsilon\downarrow0$ proves $a_n\to0$.
2. Otherwise there are infinitely many indices $k$ with $a_k<a_{k+1}$.
   Along these indices (5) gives $q_k\to0$, hence $\limsup t_k\leq0$.
   Equation (6) and the increase imply $0\leq a_k\leq t_k$, so
   $a_k\to0$ and then $a_{k+1}\to0$ by (5). For each sufficiently
   large $n$, take the last increase index $k(n)\leq n$. Then
   $k(n)\to\infty$ and $a_n\leq a_{k(n)+1}$: all later steps up to
   $n$ decrease or are equal. Hence $a_n\to0$ in this case too.

Thus $x_n\to p$ in norm. This proves the complete declared restricted
iteration. The argument never infers an infinite limit from the finite runs.

## 5. Boundary controls and relation to earlier work

The previous $Q=-3I$, $\phi=4$, $b=3/4$ example violates the new bound:
$b\phi=3>2$. It remains a counterexample to the unrestricted candidate.
Using $b=1/4$ for the same map gives $Z=0$ and avoids that cancellation.

The relaxation endpoint by itself is admissible. For $Q=-I$, $\phi=2$,
$b=1$ and $\lambda=1/2$, we have $Z=-I$, $\gamma=0$ and $U=0$ when
$R=I$. In contrast, also taking the excluded value $\lambda=1$ makes
$U=Z^2=I$ while $\Theta=\{0\}$. With $g=1$ the resulting identity-type
iteration tends to one. This shows why a bound and a mixing condition must
be considered together. It does not show that every excluded endpoint fails.

The coefficients of the repaired point-valued combinations sum to one, so
the coordinate-covariance argument from the earlier note still applies.
When testing translated trajectories, translate $g$, the operator domains,
point-valued maps, initial points and solution set consistently.

The convergence mechanism has established antecedents. In particular,
[Wongchan and Saejung (2011), Theorem 2.3](https://www.maths.tcd.ie/EMIS/journals/HOA/AAA/Volume2011/385843.pdf)
studies viscosity iteration for strongly quasinonexpansive mappings with
demiclosed residuals. [Aoyama and Kohsaka (2014), Corollary 3.5](https://link.springer.com/article/10.1186/1687-1812-2014-17)
provides a corresponding fixed-map result within a broader sequence framework.
Our explicit proof above handles the declared input perturbation and verifies
the composite's assumptions. We do not claim a novel general convergence
principle or infer coverage of the perturbed iteration merely from a citation.
The earlier [Song predecessor check](../../v0.1/PREDECESSOR-CHECK.md) remains
relevant to original-map residual identification.

## 6. Reproduction and scope

Run `python3 -I -B verify_restricted.py --output-dir ../restricted-rerun`
from this directory, choosing a destination that does not exist. The checker
uses standard-library rational arithmetic. It tests explicit affine and
piecewise examples, the joint loss estimate, translated trajectories and
excluded-parameter controls. `results.json` stores its finite output.
`verify_packet.py` checks the inventory and hashes in `MANIFEST.json`.

These computations corroborate examples and implementation; the Hilbert-space
and limiting statements depend on the mathematical proof and its assumptions.
Internal reviewers share one operator. This is neither external peer review
nor a machine-checked proof. No claim of novelty, misconduct, author assent,
on-chain settlement or general validation of the original article is made.
The original review, failed unrestricted candidate and its counterexample
remain unchanged in their earlier packets and in the local history.
