# Restricted repair: exact finite corroboration

Fixtures declared before their first execution, 2026-09-14. The notation join
below was clarified afterward; the unchanged-fixture rerun matched the stored
results byte-for-byte. Same-operator assistant check, not external
replication, source-paper execution or a convergence proof. No dependencies or
network. The translation-v0.1 NOTE and verify_translation.py were read first;
the new checker reuses their Fraction/affine-conjugation and inertia conventions.

**Mapping.** Fixed positive weights sum to one; V_j=(1-b)I+bQ_j, Z=sum w_jV_j,
beta=R_s...R_1(a), gamma=(1-lambda)beta+lambda Zbeta, U(a)=Zgamma,
L(a)=(1-rho)a+rho U(a). The tested update is
x[n+1]=delta[n]g(x[n])+(1-delta[n])L(x[n]+e[n]), with x0=x1=0,
delta[n]=1/(n+1), tau[n]=delta[n]^2 and capped inertia |e[n]|<=tau[n].

Notation join: the checker's map `L` is the manuscript's outer averaged map
`S`. It is not the manuscript's residual-loss symbol `L`. The `joint` field is
the explicitly itemized bound below, including Jensen variance terms. Keep
this PROTOCOL.md next to the checker when copying the runnable packet.

Admission uses b>0, phi_j>0, b*phi_j<=2 and lambda,rho strictly between zero
and one. The first four fixtures have strict b*phi_j<2. Saturation is a positive
control when the mixing parameter stays interior; it is not rejected merely
because the direct demimetric decrease coefficient vanishes.

Seven fixtures, fixed before results:

1. Identity maps/resolvents, g=1, Theta=R, projected target1.
2. Affine Q maps anchored at2, slopes1/2 and -1, phi values1 and2 respectively,
   weights1/3 and2/3, b1/4. Two nonzero monotone B maps anchored at2 and fixed
   resolvent steps. g(x)=x/4+1, projected target2.
3. Prior continuous three-piece Q shifted by2, plus identity Q, weights2/3
   and1/3, phi1/1, b1/4, identity resolvents. g(x)=x/4+1, target2.
4. Q=-3I, phi4, b1/4; identity resolvents, g=1, target0. Z=0 and U=0.
5. Admitted saturation: Q=-I, phi2, b1, lambda1/2; U=0, target0.
6. Excluded combined endpoint: preceding saturated Q with lambda1; U=I
   while Theta={0}. Its failure does not establish failure of saturation alone.
7. Excluded old counterexample: Q=-3I, phi4, b3/4, lambda1/2, rho9/10; U=I,
   Theta={0}. b*phi=3 violates the restricted condition.

Unless stated otherwise lambda=1/2 and rho=3/4. We use only fixed parameters.
Coordinate shifts are exactly {-2,0,1,3/2}, with y=x-t; point maps are conjugated,
B values use input shift only, and initial/target points shift with coordinates.
The common physical grid is k/4 for k=-16,...,24 (41 points). Each fixture/shift
gets 32 updates, retaining all exact comparisons in memory and reporting digests
plus compact endpoints. There is no parameter tuning after results.

**Checks.** Exact affine/continuous-piecewise algebra enumerates Fix(U), Fix(L)
and the intersection of original fixed sets (no finite-grid extrapolation for
these sets). Each represented Q segment also gets an exact quadratic sign
certificate over its whole interval; affine resolvent slopes certify firmness.
Grid checks verify generalized-demimetric and firm-resolvent
inequalities, full joint residual decrease, its averaged-L consequence, and
the weighted inner-product bound that remains usable at saturation. Whole
trajectories are compared after pulling coordinates back. Endpoint controls
must be refused by restricted admission and show the expected spurious fixed
sets; their candidate trajectories must equal the identity-control trajectory.

The joint lower bound retained is S_R + lambda[D_Q(beta)+J(beta)] +
lambda(1-lambda)|beta-Zbeta|^2 + D_Q(gamma)+J(gamma), where
D_Q(v)=sum w_j*b*(2/phi_j-b)*|v-Q_jv|^2 and
J(v)=sum w_j*|V_jv-Zv|^2. L adds rho times this bound and
rho(1-rho)|a-Ua|^2. Nonnegativity is required only for admitted fixtures;
algebraic inequalities alone do not turn an excluded control into an admission.

The checker refuses -O/-OO and existing output directories. Failure does not
overwrite any previous result. Exact rational strings/hashes carry comparisons;
decimal endpoints are presentation only. No finite trajectory establishes its
limit, convergence rate, general fixed-point theorem or applicability to the
paper's ambiguous literal statement.
