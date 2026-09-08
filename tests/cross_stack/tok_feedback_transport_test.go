package cross_stack_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	cmted25519 "github.com/cometbft/cometbft/crypto/ed25519"
	cmttypes "github.com/cometbft/cometbft/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdkserver "github.com/cosmos/cosmos-sdk/server"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	gogoproto "github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"

	zeroneapp "github.com/zerone-chain/zerone/app"
	zeroneauth "github.com/zerone-chain/zerone/x/auth/types"
	knowledgekeeper "github.com/zerone-chain/zerone/x/knowledge/keeper"
	knowledge "github.com/zerone-chain/zerone/x/knowledge/types"
)

const feedbackLocalChain = "tok-feedback-local-only"
const feedbackLocalGas uint64 = 400_000
const feedbackLocalEpoch uint64 = 20

// Public deterministic local-only identities, never read from disk or used on a
// production chain. Identity Ed25519 and transaction secp256k1 keys are distinct.
func feedbackLocalAddress(actor byte) sdk.AccAddress {
	key := &secp256k1.PrivKey{Key: bytes.Repeat([]byte{actor}, 32)}
	return sdk.AccAddress(key.PubKey().Address())
}

type feedbackABCIServer struct{ abci.Application }

func (s feedbackABCIServer) Echo(_ context.Context, req *abci.RequestEcho) (*abci.ResponseEcho, error) {
	return &abci.ResponseEcho{Message: req.Message}, nil
}
func (s feedbackABCIServer) Flush(context.Context, *abci.RequestFlush) (*abci.ResponseFlush, error) {
	return &abci.ResponseFlush{}, nil
}

type feedbackTransport struct {
	t              *testing.T
	app            *zeroneapp.ZeroneApp
	client         abci.ABCIClient
	height         int64
	gas            uint64
	epoch          uint64
	home, dbDir    string
	closeTransport func()
}

func newFeedbackTransport(t *testing.T, enabled bool, consumers []string, quota uint64, configure ...func(*knowledge.GenesisState)) *feedbackTransport {
	return newFeedbackTransportAt(t, 1, enabled, consumers, quota, configure...)
}

func newFeedbackTransportAt(t *testing.T, initialHeight int64, enabled bool, consumers []string, quota uint64, configure ...func(*knowledge.GenesisState)) *feedbackTransport {
	t.Helper()
	home, dbDir := t.TempDir(), t.TempDir()
	db, err := dbm.NewDB("feedback", dbm.GoLevelDBBackend, dbDir)
	require.NoError(t, err)
	app := zeroneapp.NewZeroneApp(log.NewNopLogger(), db, nil, true,
		simtestutil.NewAppOptionsWithFlagHome(home), baseapp.SetChainID(feedbackLocalChain))
	gen := app.DefaultGenesis()
	accounts := []authtypes.GenesisAccount{}
	balances := []banktypes.Balance{}
	identity := zeroneauth.DefaultGenesis()
	for _, actor := range []byte{7, 8, 9, 10, 11, 12, 13} {
		key := &secp256k1.PrivKey{Key: bytes.Repeat([]byte{actor}, 32)}
		addr := feedbackLocalAddress(actor)
		accounts = append(accounts, authtypes.NewBaseAccount(addr, key.PubKey(), 0, 0))
		balances = append(balances, banktypes.Balance{Address: addr.String(), Coins: sdk.NewCoins(sdk.NewInt64Coin("uzrn", 1_000_000_000))})
		if actor == 9 {
			continue
		} // Funded and cohort-admitted, but NOT registered.
		pub := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{actor}, 32)).Public().(ed25519.PublicKey)
		pubHex := hex.EncodeToString(pub)
		hash, err := zeroneauth.OperationalKeyHash(pub)
		require.NoError(t, err)
		a := &zeroneauth.Account{Address: addr.String(), Did: "did:zrn:" + pubHex,
			PublicKey: pubHex, OperationalPublicKey: pubHex, OperationalKeyHash: hash, OperationalKeyVersion: 1,
			AccountType: "human", CreatedAtBlock: 1, LastActiveBlock: 1,
			Flags: &zeroneauth.AccountFlags{Frozen: actor == 10, CanSubmitClaims: true, CanChallenge: true},
		}
		identity.Accounts = append(identity.Accounts, a)
		identity.DidMappings = append(identity.DidMappings, &zeroneauth.DIDMapping{Did: a.Did, Bech32: a.Address, PubKey: pubHex})
	}
	require.NoError(t, identity.Validate())
	gen[zeroneauth.ModuleName] = app.AppCodec().MustMarshalJSON(identity)
	// The sole local SDK validator delegates from actor 7, permitting signed
	// governance lifecycle tests without impersonating the module authority.
	val := cmttypes.NewValidator(cmted25519.GenPrivKey().PubKey(), 1)
	gen, err = genesisStateWithValSetHelper(app, gen, cmttypes.NewValidatorSet([]*cmttypes.Validator{val}), accounts, balances...)
	require.NoError(t, err)
	kg := knowledge.DefaultGenesis()
	kg.Params.FactUseEnabled, kg.Params.FactUseConsumers = enabled, consumers
	kg.Params.FitnessWeightQueryBps, kg.Params.FitnessWeightSatisfactionBps, kg.Params.MetabolismEnergyPerQuery = 0, 0, 0
	kg.Params.FitnessEpochBlocks = feedbackLocalEpoch
	kg.Params.FactUseMaxPerConsumerEpoch, kg.Params.FactUseMaxPerEpoch = quota, quota
	for _, id := range []string{"local-fact", "local-second", "local-third"} {
		kg.Facts = append(kg.Facts, &knowledge.Fact{Id: id, Content: "Explicit local genesis fixture", Domain: "general",
			Status: knowledge.FactStatus_FACT_STATUS_VERIFIED, Confidence: 800_000, Energy: 100, EnergyCap: 1000})
	}
	for _, configureGenesis := range configure {
		configureGenesis(kg)
	}
	require.NoError(t, kg.Validate())
	gen[knowledge.ModuleName] = app.AppCodec().MustMarshalJSON(kg)
	var gg govv1.GenesisState
	app.AppCodec().MustUnmarshalJSON(gen["gov"], &gg)
	voting, expedited := 3*time.Second, time.Second
	gg.Params.VotingPeriod, gg.Params.ExpeditedVotingPeriod = &voting, &expedited
	gg.Params.MinDeposit = sdk.NewCoins(sdk.NewInt64Coin("uzrn", 1))
	gg.Params.ExpeditedMinDeposit = sdk.NewCoins(sdk.NewInt64Coin("uzrn", 2))
	require.NoError(t, gg.Params.ValidateBasic())
	gen["gov"] = app.AppCodec().MustMarshalJSON(&gg)

	h := &feedbackTransport{t: t, app: app, height: initialHeight - 1, gas: feedbackLocalGas, epoch: kg.Params.FitnessEpochBlocks, home: home, dbDir: dbDir}
	h.connect()
	t.Cleanup(func() { h.closeTransport(); require.NoError(t, h.app.Close()) })
	encoded, err := json.Marshal(gen)
	require.NoError(t, err)
	_, err = h.client.InitChain(context.Background(), &abci.RequestInitChain{ChainId: feedbackLocalChain, InitialHeight: initialHeight, Time: h.blockTime(initialHeight - 1), AppStateBytes: encoded, ConsensusParams: simtestutil.DefaultConsensusParams})
	require.NoError(t, err)
	h.block(nil)
	return h
}

func (h *feedbackTransport) connect() {
	h.t.Helper()
	// Real serialized gRPC ABCI over an in-memory socket, never network ingress.
	listener := bufconn.Listen(8 << 20)
	server := grpc.NewServer()
	abci.RegisterABCIServer(server, feedbackABCIServer{sdkserver.NewCometABCIWrapper(h.app)})
	go func() { _ = server.Serve(listener) }()
	conn, err := grpc.NewClient("passthrough:///local-feedback-abci", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(h.t, err)
	h.closeTransport = func() { _ = conn.Close(); server.Stop(); _ = listener.Close() }
	h.client = abci.NewABCIClient(conn)
}

func (h *feedbackTransport) coldLoadLocalFixture() {
	h.t.Helper()
	before := h.app.LastCommitID()
	h.closeTransport()
	require.NoError(h.t, h.app.Close())
	db, err := dbm.NewDB("feedback", dbm.GoLevelDBBackend, h.dbDir)
	require.NoError(h.t, err)
	// Store/ABCI restart acceptance only. This fresh native local genesis has
	// NO historical upgrade proofs and must still fail production startup.
	// Load the store explicitly; never manufacture lineage to make it pass.
	h.app = zeroneapp.NewZeroneApp(log.NewNopLogger(), db, nil, false,
		simtestutil.NewAppOptionsWithFlagHome(h.home), baseapp.SetChainID(feedbackLocalChain))
	require.NoError(h.t, h.app.LoadLatestVersion())
	require.Equal(h.t, before, h.app.LastCommitID())
	require.ErrorContains(h.t, h.app.ValidateToKFeedbackStartupCoordination(), "lacks committed ToK feedback completion")
	h.connect()
}

func (h *feedbackTransport) blockTime(height int64) time.Time {
	return time.Unix(1_800_000_000+height, 0).UTC()
}
func (h *feedbackTransport) committed() sdk.Context {
	h.t.Helper()
	ctx, err := h.app.CreateQueryContext(h.height, false)
	require.NoError(h.t, err)
	return ctx
}
func (h *feedbackTransport) block(txs [][]byte) *abci.ResponseFinalizeBlock {
	h.t.Helper()
	h.height++
	response, err := h.client.FinalizeBlock(context.Background(), &abci.RequestFinalizeBlock{Height: h.height, Time: h.blockTime(h.height), Txs: txs})
	require.NoError(h.t, err)
	_, err = h.client.Commit(context.Background(), &abci.RequestCommit{})
	require.NoError(h.t, err)
	return response
}
func (h *feedbackTransport) advance(height int64) {
	for h.height < height {
		h.block(nil)
	}
}

func (h *feedbackTransport) sign(actor byte, operation, fact string, extra map[string]any) []byte {
	h.t.Helper()
	account := h.app.AccountKeeper.GetAccount(h.committed(), feedbackLocalAddress(actor))
	require.NotNil(h.t, account)
	input := map[string]any{"chainId": feedbackLocalChain, "actor": actor, "operation": operation, "factId": fact,
		"accountNumber": fmt.Sprint(account.GetAccountNumber()), "sequence": account.GetSequence(), "gas": h.gas}
	for k, v := range extra {
		input[k] = v
	}
	encoded, err := json.Marshal(input)
	require.NoError(h.t, err)
	_, file, _, ok := runtime.Caller(0)
	require.True(h.t, ok)
	cmd := exec.Command("node", filepath.Join(filepath.Dir(file), "tok_feedback_sign.mjs"))
	cmd.Stdin = bytes.NewReader(encoded)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	require.NoError(h.t, err, "build sdk/typescript before this test; signer error: %s", stderr.String())
	var signed struct {
		Address string `json:"address"`
		Tx      string `json:"tx"`
		SHA256  string `json:"sha256"`
	}
	require.NoError(h.t, json.Unmarshal(output, &signed))
	require.Equal(h.t, feedbackLocalAddress(actor).String(), signed.Address)
	tx, err := base64.StdEncoding.DecodeString(signed.Tx)
	require.NoError(h.t, err)
	digest := sha256.Sum256(tx)
	require.Equal(h.t, hex.EncodeToString(digest[:]), signed.SHA256)
	return tx
}

func (h *feedbackTransport) submit(actor byte, tx []byte, checkError, messageError string) *abci.ExecTxResult {
	h.t.Helper()
	ctx := h.committed()
	addr := feedbackLocalAddress(actor)
	seq := h.app.AccountKeeper.GetAccount(ctx, addr).GetSequence()
	balance := h.app.BankKeeper.GetBalance(ctx, addr, "uzrn").Amount
	check, err := h.client.CheckTx(context.Background(), &abci.RequestCheckTx{Tx: tx, Type: abci.CheckTxType_New})
	require.NoError(h.t, err)
	if checkError == "" {
		require.Zero(h.t, check.Code, check.Log)
	} else {
		require.NotZero(h.t, check.Code)
		require.Contains(h.t, check.Log, checkError)
	}
	// CheckTx is not commitment: its ante cache must not alter committed funds
	// or the nonce returned by a freshly loaded IAVL query context.
	require.Equal(h.t, seq, h.app.AccountKeeper.GetAccount(h.committed(), addr).GetSequence())
	require.True(h.t, balance.Equal(h.app.BankKeeper.GetBalance(h.committed(), addr, "uzrn").Amount))
	// A proposer can include a mempool-rejected tx: FinalizeBlock must enforce
	// the same authentication boundary independently.
	result := h.block([][]byte{tx}).TxResults[0]
	if messageError == "" {
		require.Zero(h.t, result.Code, result.Log)
	} else {
		require.NotZero(h.t, result.Code)
		require.Contains(h.t, result.Log, messageError)
	}
	ctx = h.committed()
	after := h.app.BankKeeper.GetBalance(ctx, addr, "uzrn").Amount
	fee := balance.Sub(after)
	if checkError == "" {
		require.Equal(h.t, seq+1, h.app.AccountKeeper.GetAccount(ctx, addr).GetSequence(), "valid ante sequence survives failed module execution")
		// Governance submissions separately lock a 1-uzrn deposit; ordinary
		// feedback must charge exactly the declared paid gas limit.
		decoded, err := h.app.TxConfig().TxDecoder()(tx)
		require.NoError(h.t, err)
		expectedDebit := h.gas
		stakeDebit := func(s string) uint64 { n, ok := sdkmath.NewIntFromString(s); require.True(h.t, ok); return n.Uint64() }
		switch msg := decoded.GetMsgs()[0].(type) {
		case *govv1.MsgSubmitProposal:
			expectedDebit++
		case *knowledge.MsgSubmitClaim:
			if messageError == "" {
				expectedDebit += stakeDebit(msg.Stake)
			}
		case *knowledge.MsgChallengeFact:
			if messageError == "" {
				expectedDebit += stakeDebit(msg.Stake)
			}
		case *knowledge.MsgChallengeProvisionalFact:
			if messageError == "" {
				expectedDebit += stakeDebit(msg.Stake)
			}
		}
		require.Equal(h.t, sdkmath.NewIntFromUint64(expectedDebit).String(), fee.String())
	} else {
		require.Equal(h.t, seq, h.app.AccountKeeper.GetAccount(ctx, addr).GetSequence())
		require.True(h.t, fee.IsZero(), fee.String())
	}
	digest := sha256.Sum256(tx)
	h.t.Logf("local tx=%X height=%d CheckTx(code=%d wanted=%d used=%d) Finalize(code=%d wanted=%d used=%d) signer_balance_delta=%suzrn declared_fee=%duzrn", digest, h.height, check.Code, check.GasWanted, check.GasUsed, result.Code, result.GasWanted, result.GasUsed, fee, h.gas)
	return result
}

func (h *feedbackTransport) query(method string, req, resp proto.Message) {
	h.t.Helper()
	data, err := proto.Marshal(req)
	require.NoError(h.t, err)
	result, err := h.client.Query(context.Background(), &abci.RequestQuery{Path: "/zerone.knowledge.v1.Query/" + method, Data: data, Height: h.height})
	require.NoError(h.t, err)
	require.Zero(h.t, result.Code, result.Log)
	require.Equal(h.t, h.height, result.Height)
	require.NoError(h.t, proto.Unmarshal(result.Value, resp))
}
func (h *feedbackTransport) receipt(actor byte, fact string) *knowledge.QueryFactUseReceiptResponse {
	r := &knowledge.QueryFactUseReceiptResponse{}
	h.query("FactUseReceipt", &knowledge.QueryFactUseReceiptRequest{Consumer: feedbackLocalAddress(actor).String(), FactId: fact}, r)
	require.Equal(h.t, uint64(h.height), r.SnapshotBlockHeight)
	require.Equal(h.t, uint64(h.height)/h.epoch, r.Epoch)
	return r
}
func (h *feedbackTransport) fact(id string) *knowledge.Fact {
	// Inspect canonical committed counters separately from the public Fact RPC.
	// The latter has its own enabled, end-to-end nonmutation regression below.
	f, found := h.app.KnowledgeKeeper.GetFact(h.committed(), id)
	require.True(h.t, found)
	return f
}
func (h *feedbackTransport) knowledgeState() map[string]string {
	h.t.Helper()
	store := h.committed().KVStore(h.app.GetStoreKeyForTests(knowledge.StoreKey))
	it := store.Iterator(nil, nil)
	defer it.Close()
	out := map[string]string{}
	for ; it.Valid(); it.Next() {
		out[string(it.Key())] = string(it.Value())
	}
	return out
}

func TestToKFeedbackSignedTransport(t *testing.T) {
	cohort := []string{feedbackLocalAddress(7).String(), feedbackLocalAddress(9).String(), feedbackLocalAddress(10).String()}
	h := newFeedbackTransport(t, true, cohort, 2)
	supply := h.app.BankKeeper.GetSupply(h.committed(), "uzrn")
	h.submit(7, h.sign(7, "report", "local-fact", nil), "", "")
	r := h.receipt(7, "local-fact")
	require.True(t, r.Found)
	require.Equal(t, uint64(2), r.Receipt.UseHeight)
	require.Equal(t, uint64(40), r.Receipt.ExpiryHeight)
	require.Equal(t, knowledge.FactUseRating_FACT_USE_RATING_UNRATED, r.Receipt.Rating)
	f := h.fact("local-fact")
	require.Equal(t, uint64(1), f.QueryCount)
	require.Equal(t, uint64(1), f.QueryCountEpoch)

	ctx := h.committed() // a new committed IAVL query context, not a MsgServer cache
	persisted, found, err := h.app.KnowledgeKeeper.GetFactUseReceipt(ctx, 0, feedbackLocalAddress(7).String(), "local-fact")
	require.NoError(t, err)
	require.True(t, found)
	require.True(t, proto.Equal(r.Receipt, persisted))

	h.submit(7, h.sign(7, "report", "local-fact", nil), "", "already reported")
	require.Equal(t, uint64(1), h.fact("local-fact").QueryCount)
	h.submit(7, h.sign(7, "rate", "local-fact", nil), "", "")
	require.Equal(t, knowledge.FactUseRating_FACT_USE_RATING_USEFUL, h.receipt(7, "local-fact").Receipt.Rating)
	h.submit(7, h.sign(7, "rate", "local-fact", nil), "", "already rated")
	require.Equal(t, uint64(1), h.fact("local-fact").SatisfactionUp)

	h.submit(8, h.sign(8, "report", "local-fact", map[string]any{"consumer": feedbackLocalAddress(7).String()}), "pubKey", "pubKey")
	h.submit(8, h.sign(8, "report", "local-fact", nil), "", "not in the fact-use cohort")
	h.submit(9, h.sign(9, "report", "local-fact", nil), "not registered", "not registered")
	h.submit(10, h.sign(10, "report", "local-fact", nil), "frozen", "frozen")
	h.submit(7, h.sign(7, "report", "local-second", nil), "", "")
	h.submit(7, h.sign(7, "report", "local-third", nil), "", "quota reached")
	require.False(t, h.receipt(7, "local-third").Found)
	require.Zero(t, h.fact("local-third").QueryCount)

	h.advance(19) // H-1: old receipt can still be read, but not rated at H.
	require.True(t, h.receipt(7, "local-second").Found)
	h.submit(7, h.sign(7, "rate", "local-second", nil), "", "no current-epoch") // H
	require.Equal(t, int64(20), h.height)
	require.False(t, h.receipt(7, "local-second").Found)
	h.submit(7, h.sign(7, "report", "local-second", nil), "", "") // H+1
	require.Equal(t, uint64(21), h.receipt(7, "local-second").Receipt.UseHeight)
	beforeRating := h.fact("local-second")
	h.submit(7, h.sign(7, "rate", "local-second", map[string]any{"useful": false}), "", "")
	require.Equal(t, uint64(1), h.fact("local-second").SatisfactionDown)
	require.Equal(t, uint64(2), h.fact("local-second").QueryCount)
	// The normal epoch metabolism may hibernate a fixture at H; rating does
	// not change that standing or grant energy after the epoch transition.
	require.Equal(t, beforeRating.Status, h.fact("local-second").Status)
	require.Equal(t, beforeRating.Energy, h.fact("local-second").Energy)
	require.Equal(t, supply, h.app.BankKeeper.GetSupply(h.committed(), "uzrn"), "feedback must not mint")

	// A real signed governance proposal/vote disables feedback; no module
	// authority key or direct UpdateParams call is used.
	params, err := h.app.KnowledgeKeeper.GetParams(h.committed())
	require.NoError(t, err)
	params.FactUseEnabled = false
	h.governParams(params, govv1.StatusPassed)
	h.submit(7, h.sign(7, "report", "local-third", nil), "", "disabled")
	h.advance(62) // receipts from epochs 0 and 1 are now expired and pruned
	retained, err := h.app.KnowledgeKeeper.HasRetainedFactUseReceipts(h.committed())
	require.NoError(t, err)
	require.False(t, retained)
	state, err := h.app.KnowledgeKeeper.GetFactUsePruningState(h.committed())
	require.NoError(t, err)
	require.True(t, state.EverReported)
	for _, economic := range []string{"query", "satisfaction", "energy"} {
		p := proto.Clone(params).(*knowledge.Params)
		switch economic {
		case "query":
			p.FitnessWeightQueryBps = 1
		case "satisfaction":
			p.FitnessWeightSatisfactionBps = 1
		case "energy":
			p.MetabolismEnergyPerQuery = 1
		}
		h.governParams(p, govv1.StatusFailed)
		actual, err := h.app.KnowledgeKeeper.GetParams(h.committed())
		require.NoError(t, err)
		require.True(t, proto.Equal(params, actual), economic)
	}
}

func TestToKFeedbackSignedColdStoreReload(t *testing.T) {
	h := newFeedbackTransport(t, true, []string{feedbackLocalAddress(7).String()}, 2)
	h.submit(7, h.sign(7, "report", "local-fact", nil), "", "")
	before := h.knowledgeState()
	supply := h.app.BankKeeper.GetSupply(h.committed(), "uzrn")
	h.coldLoadLocalFixture()
	require.Equal(t, before, h.knowledgeState())
	require.True(t, h.receipt(7, "local-fact").Found)
	h.submit(7, h.sign(7, "rate", "local-fact", nil), "", "")
	require.Equal(t, knowledge.FactUseRating_FACT_USE_RATING_USEFUL, h.receipt(7, "local-fact").Receipt.Rating)
	before = h.knowledgeState()
	h.coldLoadLocalFixture()
	require.Equal(t, before, h.knowledgeState())
	h.submit(7, h.sign(7, "rate", "local-fact", nil), "", "already rated")
	require.Equal(t, uint64(1), h.fact("local-fact").SatisfactionUp)
	require.Equal(t, supply, h.app.BankKeeper.GetSupply(h.committed(), "uzrn"))
	exported := h.app.KnowledgeKeeper.ExportGenesis(h.committed())
	require.NoError(t, exported.Validate())
	require.True(t, exported.FactUsePruning.EverReported)
	require.Len(t, exported.FactUseReceipts, 1)
	require.True(t, proto.Equal(exported.FactUseReceipts[0], h.receipt(7, "local-fact").Receipt))
}

func TestToKFeedbackSignedRetainedCapacityGas(t *testing.T) {
	h := newFeedbackTransportAt(t, 21, true, []string{feedbackLocalAddress(7).String()}, 100, func(g *knowledge.GenesisState) {
		// Synthetic bounded historical LOAD fixture, not 1,999 claimed signed
		// transactions. Both epochs precede the local genesis height; no future
		// receipt or fabricated migration lineage is imported.
		g.Params.FactUseMaxPerEpoch = 1000
		g.FactUsePruning = &knowledge.FactUsePruningState{EverReported: true}
		for i := 0; i < 100; i++ {
			g.Facts = append(g.Facts, &knowledge.Fact{Id: fmt.Sprintf("load-%03d", i), Content: "Local load fixture", Domain: "general", Status: knowledge.FactStatus_FACT_STATUS_VERIFIED})
		}
		for epoch := uint64(0); epoch < 2; epoch++ {
			for i := 0; i < 1000; i++ {
				if epoch == 1 && i == 999 {
					continue
				}
				useHeight := epoch * feedbackLocalEpoch
				if useHeight == 0 {
					useHeight = 1
				}
				g.FactUseReceipts = append(g.FactUseReceipts, &knowledge.FactUseReceipt{
					Version: 1, Epoch: epoch, Consumer: feedbackLocalAddress(byte(11 + i/100)).String(),
					FactId: fmt.Sprintf("load-%03d", i%100), UseHeight: useHeight,
					ExpiryHeight: (epoch + 2) * feedbackLocalEpoch, Rating: knowledge.FactUseRating_FACT_USE_RATING_UNRATED,
				})
			}
		}
	})
	h.gas = 2_000_000 // measured paid local ceiling, not a recommended production fee
	supply := h.app.BankKeeper.GetSupply(h.committed(), "uzrn")
	before := h.fact("local-fact")
	h.submit(7, h.sign(7, "report", "local-fact", nil), "", "")
	receipts, err := h.app.KnowledgeKeeper.ExportFactUseReceipts(h.committed())
	require.NoError(t, err)
	require.Len(t, receipts, knowledge.MaxFactUseRetainedReceipts)
	h.submit(7, h.sign(7, "rate", "local-fact", nil), "", "")
	h.submit(7, h.sign(7, "report", "local-second", nil), "", "retained capacity reached")
	require.False(t, h.receipt(7, "local-second").Found)
	require.Equal(t, before.Energy, h.fact("local-fact").Energy)
	require.Equal(t, before.Status, h.fact("local-fact").Status)
	require.Equal(t, supply, h.app.BankKeeper.GetSupply(h.committed(), "uzrn"))
}

func TestToKFeedbackSignedAtomicBatchAndBadSignature(t *testing.T) {
	h := newFeedbackTransport(t, true, []string{feedbackLocalAddress(7).String()}, 2)
	initial := proto.Clone(h.fact("local-fact")).(*knowledge.Fact)
	// First message succeeds, second fails semantic deduplication; the entire
	// module write-set rolls back but the single transaction's ante fee/nonce do not.
	h.submit(7, h.sign(7, "report-twice", "local-fact", nil), "", "already reported")
	require.False(t, h.receipt(7, "local-fact").Found)
	require.True(t, proto.Equal(initial, h.fact("local-fact")))
	state, err := h.app.KnowledgeKeeper.GetFactUsePruningState(h.committed())
	require.NoError(t, err)
	require.False(t, state.EverReported)

	var raw txtypes.TxRaw
	require.NoError(t, gogoproto.Unmarshal(h.sign(7, "report", "local-fact", nil), &raw))
	raw.Signatures[0][0] ^= 1
	corrupt, err := gogoproto.Marshal(&raw)
	require.NoError(t, err)
	h.submit(7, corrupt, "signature verification failed", "signature verification failed")
	require.False(t, h.receipt(7, "local-fact").Found)
	// Correctly re-signed at the unchanged nonce succeeds, proving rejection did
	// not poison the dedup/counter/latch namespaces.
	h.submit(7, h.sign(7, "report", "local-fact", nil), "", "")
	require.True(t, h.receipt(7, "local-fact").Found)
	require.Equal(t, uint64(1), h.fact("local-fact").QueryCount)
}

func TestToKFeedbackSDKMapWireParity(t *testing.T) {
	// Same golden asserted by sdk/typescript/tests/fact-use.test.ts. The
	// synthetic entry uses wire2 even though its value uses varint/wire0.
	value := &knowledge.MsgUpdateParams{Params: &knowledge.Params{MethodologyNormalizationBps: map[string]uint64{"x": 1}}}
	wire, err := proto.Marshal(value)
	require.NoError(t, err)
	require.Equal(t, "1208e208050a01781001", hex.EncodeToString(wire))
}

func TestToKFeedbackSignedDefaultParams(t *testing.T) {
	h := newFeedbackTransport(t, true, []string{feedbackLocalAddress(7).String()}, 2)
	params, err := h.app.KnowledgeKeeper.GetParams(h.committed())
	require.NoError(t, err)
	require.NotEmpty(t, params.MethodologyNormalizationBps)
	params.FactUseEnabled = false
	h.governParams(params, govv1.StatusPassed)
}

func TestToKFeedbackSignedReadNonMutation(t *testing.T) {
	h := newFeedbackTransport(t, true, []string{feedbackLocalAddress(7).String()}, 2)
	h.submit(7, h.sign(7, "report", "local-fact", nil), "", "")
	// Query through ABCI with obsolete tracking fields. This remains an enabled
	// acceptance test even if a separate read-lane dependency breaks the RPC.
	before, hash := h.knowledgeState(), bytes.Clone(h.app.LastCommitID().Hash)
	for i := 0; i < 3; i++ {
		response := &knowledge.QueryFactResponse{}
		h.query("Fact", &knowledge.QueryFactRequest{Id: "local-fact", TrackQuery: true, Querier: feedbackLocalAddress(8).String()}, response)
		require.Equal(t, uint64(1), response.Fact.QueryCount)
		h.receipt(7, "local-fact")
	}
	require.Equal(t, before, h.knowledgeState())
	require.Equal(t, hash, h.app.LastCommitID().Hash)
}

func TestToKFeedbackSignedClosedAdmission(t *testing.T) {
	for _, tc := range []struct {
		name    string
		enabled bool
		cohort  []string
		err     string
	}{
		{"disabled", false, []string{feedbackLocalAddress(7).String()}, "disabled"},
		{"empty-cohort", true, nil, "not in the fact-use cohort"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newFeedbackTransport(t, tc.enabled, tc.cohort, 2)
			h.submit(7, h.sign(7, "report", "local-fact", nil), "", tc.err)
			require.False(t, h.receipt(7, "local-fact").Found)
			require.Zero(t, h.fact("local-fact").QueryCount)
			state, err := h.app.KnowledgeKeeper.GetFactUsePruningState(h.committed())
			require.NoError(t, err)
			require.False(t, state.EverReported)
		})
	}
}

func TestToKFeedbackSignedCanonicalValidation(t *testing.T) {
	h := newFeedbackTransport(t, true, []string{feedbackLocalAddress(7).String()}, 2)
	for _, tc := range []struct {
		op, fact string
		extra    map[string]any
		err      string
	}{
		{"report", "bad/id", nil, "fact_id"},
		{"report", "local-fact", map[string]any{"consumer": strings.ToUpper(feedbackLocalAddress(7).String())}, "canonical"},
		{"rate", "local-fact", map[string]any{"memo": strings.Repeat("é", 129)}, "256 bytes"},
	} {
		h.submit(7, h.sign(7, tc.op, tc.fact, tc.extra), tc.err, tc.err)
	}
	require.False(t, h.receipt(7, "local-fact").Found)
}

// Four scripted distinct addresses are a local test panel, NOT evidence of
// independently operated validators or truth probability. Keep the normal base
// quorum (3) plus the non-empty-domain surcharge (1); only local clocks shorten.
func newCorrectionTransport(t *testing.T, configure ...func(*knowledge.GenesisState)) *feedbackTransport {
	h := newFeedbackTransport(t, true, []string{feedbackLocalAddress(7).String()}, 10, func(g *knowledge.GenesisState) {
		g.Facts = nil // every scenario/evidence fact must be admitted by signed review
		g.Params.CommitPhaseBlocks, g.Params.RevealPhaseBlocks, g.Params.AggregationPhaseBlocks = 6, 6, 2
		g.Params.ClaimCooldownBlocks = 0
		g.Params.FitnessEpochBlocks = 1000 // isolate review from epoch metabolism
		g.Domains = append(g.Domains, &knowledge.Domain{Name: "local_feedback", Status: knowledge.DomainStatus_DOMAIN_STATUS_ACTIVE})
		for _, adjust := range configure {
			adjust(g)
		}
	})
	h.gas = 2_000_000 // explicit paid local ceiling, not a production recommendation
	return h
}

func feedbackEventID(t *testing.T, result *abci.ExecTxResult, key string) string {
	t.Helper()
	for _, event := range result.Events {
		for _, attr := range event.Attributes {
			if attr.Key == key && attr.Value != "" {
				return attr.Value
			}
		}
	}
	t.Fatalf("missing committed event attribute %q", key)
	return ""
}

func (h *feedbackTransport) claim(id string) *knowledge.Claim {
	h.t.Helper()
	result := &knowledge.QueryClaimResponse{}
	h.query("Claim", &knowledge.QueryClaimRequest{Id: id}, result)
	require.NotNil(h.t, result.Claim)
	return result.Claim
}

func (h *feedbackTransport) round(id string) *knowledge.VerificationRound {
	h.t.Helper()
	result := &knowledge.QueryVerificationRoundResponse{}
	h.query("VerificationRound", &knowledge.QueryVerificationRoundRequest{Id: id}, result)
	require.NotNil(h.t, result.Round)
	return result.Round
}

func (h *feedbackTransport) submitReviewedClaim(content string, kind knowledge.ClaimType, relations []map[string]any, votes []string, verdict knowledge.Verdict) (*knowledge.Claim, *knowledge.Fact) {
	h.t.Helper()
	fee := h.app.KnowledgeKeeper.GetEffectiveMinReviewFee(h.committed())
	result := h.submit(7, h.sign(7, "claim", "", map[string]any{"content": content, "claimType": kind, "stake": fee, "relations": relations}), "", "")
	id := feedbackEventID(h.t, result, "claim_id")
	claim := h.claim(id)
	h.reviewRound(claim.VerificationRoundId, votes, verdict)
	claim = h.claim(id)
	if verdict != knowledge.Verdict_VERDICT_ACCEPT {
		require.Equal(h.t, knowledge.ClaimStatus_CLAIM_STATUS_REJECTED, claim.Status)
		return claim, nil
	}
	require.Equal(h.t, knowledge.ClaimStatus_CLAIM_STATUS_ACCEPTED, claim.Status)
	round := h.round(claim.VerificationRoundId)
	resultFact := &knowledge.QueryFactResponse{}
	h.query("Fact", &knowledge.QueryFactRequest{Id: knowledgekeeper.GenerateFactID(id, round.VerdictBlock)}, resultFact)
	require.NotNil(h.t, resultFact.Fact)
	require.Equal(h.t, id, resultFact.Fact.ClaimId)
	return claim, resultFact.Fact
}

func (h *feedbackTransport) reviewRound(id string, votes []string, verdict knowledge.Verdict) {
	h.t.Helper()
	actors := []byte{8, 11, 12, 13}
	require.Len(h.t, votes, len(actors))
	initial := h.round(id)
	require.Equal(h.t, knowledge.VerificationPhase_VERIFICATION_PHASE_COMMIT, initial.Phase)
	for i, actor := range actors {
		salt := []byte(fmt.Sprintf("public-local-salt-%s-%d", id, actor))
		digest := knowledge.ComputeCommitmentHash(id, votes[i], 0, salt)
		h.submit(actor, h.sign(actor, "commit", "", map[string]any{"roundId": id, "commitHash": base64.StdEncoding.EncodeToString(digest)}), "", "")
	}
	require.Len(h.t, h.round(id).Commits, len(actors))
	h.advance(int64(initial.CommitDeadline) + 1)
	require.Equal(h.t, knowledge.VerificationPhase_VERIFICATION_PHASE_REVEAL, h.round(id).Phase)
	for i, actor := range actors {
		salt := []byte(fmt.Sprintf("public-local-salt-%s-%d", id, actor))
		h.submit(actor, h.sign(actor, "reveal", "", map[string]any{"roundId": id, "vote": votes[i], "confidence": "0", "salt": base64.StdEncoding.EncodeToString(salt)}), "", "")
	}
	// Only real BeginBlock aggregation may close a round. No CompleteRound call,
	// fabricated aggregate, SetClaim, SetVerificationRound or accepted SetFact.
	h.block(nil)
	round := h.round(id)
	require.Equal(h.t, knowledge.VerificationPhase_VERIFICATION_PHASE_COMPLETE, round.Phase)
	require.Equal(h.t, verdict, round.Verdict)
	require.Equal(h.t, uint64(h.height), round.VerdictBlock)
	require.Len(h.t, round.Commits, 4)
	require.Len(h.t, round.Reveals, 4)
	for i, actor := range actors {
		require.Equal(h.t, feedbackLocalAddress(actor).String(), round.Reveals[i].Verifier)
		require.Equal(h.t, votes[i], round.Reveals[i].Vote)
		require.NotZero(h.t, round.Reveals[i].RevealedAtBlock)
	}
}

var feedbackAcceptVotes = []string{"accept", "accept", "accept", "accept"}

func (h *feedbackTransport) startChallengeWithEvidence(target, evidence *knowledge.Fact) *knowledge.Claim {
	h.t.Helper()
	operation := "challenge"
	if target.ClaimType == knowledge.ClaimType_CLAIM_TYPE_CONJECTURE {
		operation = "challenge-provisional"
	}
	reason := "Local scripted counterexample; evidence is not a provenance citation."
	result := h.submit(7, h.sign(7, operation, target.Id, map[string]any{"stake": "100000000", "reason": reason, "evidenceIds": []string{evidence.Id}}), "", "")
	// The two wire responses name the ROUND differently; resolve its persisted
	// claim rather than conflating a challenge_id with a Claim identifier.
	key := "round_id"
	if operation == "challenge-provisional" {
		key = "challenge_id"
	}
	roundID := feedbackEventID(h.t, result, key)
	id := h.round(roundID).ClaimId
	claim := h.claim(id)
	require.Equal(h.t, []string{evidence.Id}, claim.ChallengeEvidenceIds)
	require.Equal(h.t, reason, claim.ArgumentText)
	require.Empty(h.t, claim.References)
	require.Empty(h.t, claim.Relations)
	return claim
}

func (h *feedbackTransport) challengeWithEvidence(target, evidence *knowledge.Fact, votes []string, verdict knowledge.Verdict) *knowledge.Claim {
	h.t.Helper()
	initial := h.startChallengeWithEvidence(target, evidence)
	h.reviewRound(initial.VerificationRoundId, votes, verdict)
	claim := h.claim(initial.Id)
	require.Equal(h.t, []string{evidence.Id}, claim.ChallengeEvidenceIds)
	require.Equal(h.t, initial.ArgumentText, claim.ArgumentText)
	expected := map[knowledge.Verdict]knowledge.ClaimStatus{
		knowledge.Verdict_VERDICT_ACCEPT:       knowledge.ClaimStatus_CLAIM_STATUS_ACCEPTED,
		knowledge.Verdict_VERDICT_REJECT:       knowledge.ClaimStatus_CLAIM_STATUS_REJECTED,
		knowledge.Verdict_VERDICT_INCONCLUSIVE: knowledge.ClaimStatus_CLAIM_STATUS_INSUFFICIENT,
		knowledge.Verdict_VERDICT_MALFORMED:    knowledge.ClaimStatus_CLAIM_STATUS_MALFORMED,
	}
	require.Equal(h.t, expected[verdict], claim.Status)
	return claim
}

func TestToKFeedbackSignedChallengeTerminalOutcomes(t *testing.T) {
	for _, conjecture := range []bool{false, true} {
		for _, tc := range []struct {
			name    string
			votes   []string
			verdict knowledge.Verdict
		}{
			{"accept", feedbackAcceptVotes, knowledge.Verdict_VERDICT_ACCEPT},
			{"reject", []string{"reject", "reject", "reject", "reject"}, knowledge.Verdict_VERDICT_REJECT},
			{"inconclusive", []string{"accept", "accept", "reject", "reject"}, knowledge.Verdict_VERDICT_INCONCLUSIVE},
			{"malformed", []string{"malformed", "malformed", "malformed", "malformed"}, knowledge.Verdict_VERDICT_MALFORMED},
		} {
			t.Run(fmt.Sprintf("conjecture=%t/%s", conjecture, tc.name), func(t *testing.T) {
				h := newCorrectionTransport(t)
				_, evidence := h.submitReviewedClaim("The local test instrument recorded a counterexample.", knowledge.ClaimType_CLAIM_TYPE_ASSERTION, nil, feedbackAcceptVotes, knowledge.Verdict_VERDICT_ACCEPT)
				kind := knowledge.ClaimType_CLAIM_TYPE_ASSERTION
				if conjecture {
					kind = knowledge.ClaimType_CLAIM_TYPE_CONJECTURE
				}
				_, target := h.submitReviewedClaim("Every specimen in the local fixture has property A.", kind, nil, feedbackAcceptVotes, knowledge.Verdict_VERDICT_ACCEPT)
				before := proto.Clone(h.fact(target.Id)).(*knowledge.Fact)
				claim := h.challengeWithEvidence(target, evidence, tc.votes, tc.verdict)
				after := h.fact(target.Id)
				switch {
				case tc.verdict == knowledge.Verdict_VERDICT_ACCEPT:
					require.Equal(t, knowledge.FactStatus_FACT_STATUS_DISPROVEN, after.Status)
				case conjecture:
					require.Equal(t, knowledge.FactStatus_FACT_STATUS_PROVISIONAL, after.Status)
					require.Zero(t, after.Confidence)
				default:
					require.Equal(t, knowledge.FactStatus_FACT_STATUS_ACTIVE, after.Status)
				}
				history := h.app.KnowledgeKeeper.GetStatusHistory(h.committed(), target.Id)
				require.NotEmpty(t, history)
				if tc.verdict == knowledge.Verdict_VERDICT_INCONCLUSIVE || tc.verdict == knowledge.Verdict_VERDICT_MALFORMED {
					require.Equal(t, before.Confidence, after.Confidence)
					require.Equal(t, before.Energy, after.Energy)
					require.Equal(t, before.CorroborationCount, after.CorroborationCount)
					require.Equal(t, before.LastCorroboratedBlock, after.LastCorroboratedBlock)
					require.Equal(t, claim.Id, history[len(history)-1].CauseId)
					require.Contains(t, history[len(history)-1].CauseEventType, "completed_"+tc.name)
				}
				h.assertHistoryDurable(claim, target.Id)
			})
		}
	}
}

func TestToKFeedbackSignedStarvedChallenge(t *testing.T) {
	for _, conjecture := range []bool{false, true} {
		t.Run(fmt.Sprint(conjecture), func(t *testing.T) {
			// A valid zero-duration aggregation window makes EXPIRED reachable
			// through consecutive real blocks. With the usual positive window,
			// the same no-reveal round completes INCONCLUSIVE at RevealDeadline.
			h := newCorrectionTransport(t, func(g *knowledge.GenesisState) { g.Params.AggregationPhaseBlocks = 0 })
			_, evidence := h.submitReviewedClaim("The local starvation fixture contains an accepted evidence record.", knowledge.ClaimType_CLAIM_TYPE_ASSERTION, nil, feedbackAcceptVotes, knowledge.Verdict_VERDICT_ACCEPT)
			kind := knowledge.ClaimType_CLAIM_TYPE_ASSERTION
			if conjecture {
				kind = knowledge.ClaimType_CLAIM_TYPE_CONJECTURE
			}
			_, target := h.submitReviewedClaim("The local target receives a challenge without enough reviewers.", kind, nil, feedbackAcceptVotes, knowledge.Verdict_VERDICT_ACCEPT)
			before := proto.Clone(h.fact(target.Id)).(*knowledge.Fact)
			claim := h.startChallengeWithEvidence(target, evidence)
			round := h.round(claim.VerificationRoundId)
			// A real signed commitment with no reveal: absence is not a vote.
			digest := knowledge.ComputeCommitmentHash(round.Id, "accept", 0, []byte("public-starved-salt"))
			h.submit(8, h.sign(8, "commit", "", map[string]any{"roundId": round.Id, "commitHash": base64.StdEncoding.EncodeToString(digest)}), "", "")
			h.advance(int64(round.AggregationDeadline) + 1)
			round = h.round(round.Id)
			require.Equal(t, knowledge.VerificationPhase_VERIFICATION_PHASE_EXPIRED, round.Phase)
			require.Equal(t, knowledge.Verdict_VERDICT_INCONCLUSIVE, round.Verdict)
			require.Len(t, round.Commits, 1)
			require.Empty(t, round.Reveals)
			claim = h.claim(claim.Id)
			require.Equal(t, knowledge.ClaimStatus_CLAIM_STATUS_INSUFFICIENT, claim.Status)
			require.Equal(t, []string{evidence.Id}, claim.ChallengeEvidenceIds)
			after := h.fact(target.Id)
			if conjecture {
				require.Equal(t, knowledge.FactStatus_FACT_STATUS_PROVISIONAL, after.Status)
			} else {
				require.Equal(t, knowledge.FactStatus_FACT_STATUS_ACTIVE, after.Status)
			}
			require.Equal(t, before.Confidence, after.Confidence)
			require.Equal(t, before.Energy, after.Energy)
			require.Equal(t, before.CorroborationCount, after.CorroborationCount)
			require.Equal(t, before.LastCorroboratedBlock, after.LastCorroboratedBlock)
			history := h.app.KnowledgeKeeper.GetStatusHistory(h.committed(), target.Id)
			require.Equal(t, claim.Id, history[len(history)-1].CauseId)
			require.Contains(t, history[len(history)-1].CauseEventType, "starved_panel")
			h.assertHistoryDurable(claim, target.Id)
		})
	}
}

func (h *feedbackTransport) assertHistoryDurable(challenge *knowledge.Claim, target string) {
	h.t.Helper()
	before := h.app.KnowledgeKeeper.ExportGenesis(h.committed())
	require.NoError(h.t, before.Validate())
	require.NotEmpty(h.t, before.CompletedRounds)
	require.NotEmpty(h.t, before.CompletedRoundRecords)
	require.NotEmpty(h.t, before.StatusTransitions)
	require.NotEmpty(h.t, before.StatusTransitionSequences)
	exact := h.knowledgeState()
	h.coldLoadLocalFixture() // still explicitly rejects production startup lineage
	require.Equal(h.t, exact, h.knowledgeState())
	require.True(h.t, proto.Equal(challenge, h.claim(challenge.Id)))
	require.True(h.t, proto.Equal(before, h.app.KnowledgeKeeper.ExportGenesis(h.committed())))
	wire, err := proto.MarshalOptions{Deterministic: true}.Marshal(before)
	require.NoError(h.t, err)
	digest := sha256.Sum256(wire)
	h.t.Logf("local-only committed export height=%d sha256=%x challenge=%s target=%s", h.height, digest, challenge.Id, target)
	imported := &knowledge.GenesisState{}
	require.NoError(h.t, proto.Unmarshal(wire, imported))
	// Export/import is a durability test, NOT a rollback or migration recipe.
	copy := newFeedbackTransportAt(h.t, h.height+1, false, nil, 10, func(g *knowledge.GenesisState) {
		proto.Reset(g)
		proto.Merge(g, imported)
	})
	after := copy.app.KnowledgeKeeper.ExportGenesis(copy.committed())
	// Compare canonical protobuf content, not Go's internal size/descriptor caches.
	durable := func(g *knowledge.GenesisState) []byte {
		value := &knowledge.GenesisState{
			Facts: g.Facts, PendingClaims: g.PendingClaims, CompletedRounds: g.CompletedRounds,
			CompletedRoundRecords: g.CompletedRoundRecords, FactRelations: g.FactRelations,
			StatusTransitions: g.StatusTransitions, StatusTransitionSequences: g.StatusTransitionSequences,
			CascadeEvents: g.CascadeEvents, FactUseReceipts: g.FactUseReceipts, FactUsePruning: g.FactUsePruning,
		}
		wire, err := proto.MarshalOptions{Deterministic: true}.Marshal(value)
		require.NoError(h.t, err)
		return wire
	}
	require.Equal(h.t, durable(before), durable(after), "exact claims, rounds, completion metadata, relations, transitions, sequences, cascades and receipts")
	require.True(h.t, proto.Equal(challenge, copy.claim(challenge.Id)))
}

func TestToKFeedbackSignedCorrectionExport(t *testing.T) {
	h := newCorrectionTransport(t)
	supply := h.app.BankKeeper.GetSupply(h.committed(), "uzrn")
	_, evidence := h.submitReviewedClaim("The local calibration log records a reproducible counterexample.", knowledge.ClaimType_CLAIM_TYPE_ASSERTION, nil, feedbackAcceptVotes, knowledge.Verdict_VERDICT_ACCEPT)
	_, original := h.submitReviewedClaim("All local specimens have the incorrectly reported property A.", knowledge.ClaimType_CLAIM_TYPE_ASSERTION, nil, feedbackAcceptVotes, knowledge.Verdict_VERDICT_ACCEPT)
	h.submit(7, h.sign(7, "report", original.Id, nil), "", "")
	h.submit(7, h.sign(7, "rate", original.Id, map[string]any{"useful": false}), "", "")
	require.Equal(t, knowledge.FactUseRating_FACT_USE_RATING_NOT_USEFUL, h.receipt(7, original.Id).Receipt.Rating)
	_, child := h.submitReviewedClaim("A local downstream conclusion relies on the original measurement.", knowledge.ClaimType_CLAIM_TYPE_ASSERTION,
		[]map[string]any{{"targetFactId": original.Id, "relation": knowledge.RelationType_RELATION_TYPE_REQUIRES, "inference": knowledge.InferenceType_INFERENCE_TYPE_DEDUCTIVE, "inferenceStrengthBps": "1000000"}}, feedbackAcceptVotes, knowledge.Verdict_VERDICT_ACCEPT)
	challenge := h.challengeWithEvidence(original, evidence, feedbackAcceptVotes, knowledge.Verdict_VERDICT_ACCEPT)
	require.Equal(t, knowledge.FactStatus_FACT_STATUS_DISPROVEN, h.fact(original.Id).Status)
	require.Equal(t, knowledge.FactStatus_FACT_STATUS_CONTESTED, h.fact(child.Id).Status)
	require.NotEmpty(t, h.app.KnowledgeKeeper.ExportGenesis(h.committed()).CascadeEvents)
	relation := []map[string]any{{"targetFactId": original.Id, "relation": knowledge.RelationType_RELATION_TYPE_SUPERSEDES}}
	rejected, noFact := h.submitReviewedClaim("An unsubstantiated local replacement must be separately rejected.", knowledge.ClaimType_CLAIM_TYPE_ASSERTION, relation, []string{"reject", "reject", "reject", "reject"}, knowledge.Verdict_VERDICT_REJECT)
	require.Nil(t, noFact)
	for _, fact := range h.app.KnowledgeKeeper.ExportGenesis(h.committed()).Facts {
		require.NotEqual(t, rejected.Id, fact.ClaimId)
	}
	edges := &knowledge.QueryFactRelationsResponse{}
	h.query("FactRelations", &knowledge.QueryFactRelationsRequest{FactId: original.Id, Relation: knowledge.RelationType_RELATION_TYPE_SUPERSEDES, Direction: "incoming"}, edges)
	require.Empty(t, edges.Relations, "a rejected replacement must not create a canonical edge")
	replacementClaim, replacement := h.submitReviewedClaim("The corrected local statement limits property A to measured specimens.", knowledge.ClaimType_CLAIM_TYPE_ASSERTION, relation, feedbackAcceptVotes, knowledge.Verdict_VERDICT_ACCEPT)
	require.NotEqual(t, original.Id, replacement.Id)
	require.NotEqual(t, challenge.Id, replacementClaim.Id)
	h.query("FactRelations", &knowledge.QueryFactRelationsRequest{FactId: original.Id, Relation: knowledge.RelationType_RELATION_TYPE_SUPERSEDES, Direction: "incoming"}, edges)
	require.Len(t, edges.Relations, 1)
	require.Equal(t, replacement.Id, edges.Relations[0].SourceFactId)
	require.Equal(t, original.Id, edges.Relations[0].TargetFactId)
	require.Equal(t, original.Content, h.fact(original.Id).Content)
	require.Equal(t, knowledge.FactStatus_FACT_STATUS_DISPROVEN, h.fact(original.Id).Status, "SUPERSEDES is an edge, not a rewrite of the disproven record")
	h.submit(7, h.sign(7, "report", replacement.Id, nil), "", "")
	h.submit(7, h.sign(7, "rate", replacement.Id, nil), "", "")
	require.Equal(t, knowledge.FactUseRating_FACT_USE_RATING_USEFUL, h.receipt(7, replacement.Id).Receipt.Rating)
	before, root := h.knowledgeState(), h.app.LastCommitID()
	queried := &knowledge.QueryFactResponse{}
	h.query("Fact", &knowledge.QueryFactRequest{Id: original.Id, TrackQuery: true, Querier: feedbackLocalAddress(7).String()}, queried)
	require.Equal(t, original.Content, queried.Fact.Content)
	require.Equal(t, before, h.knowledgeState())
	require.Equal(t, root, h.app.LastCommitID())
	require.Equal(t, knowledge.FactStatus_FACT_STATUS_CONTESTED, h.fact(child.Id).Status, "accepting a correction must not revive contested descendants")
	require.Equal(t, supply, h.app.BankKeeper.GetSupply(h.committed(), "uzrn"), "signed reviews settle existing fees/stakes without new issuance")
	h.assertHistoryDurable(challenge, original.Id)
}

func (h *feedbackTransport) governParams(p *knowledge.Params, expected govv1.ProposalStatus) {
	h.t.Helper()
	wire, err := proto.Marshal(&knowledge.MsgUpdateParams{Authority: h.app.KnowledgeKeeper.GetAuthority(), Params: p})
	require.NoError(h.t, err)
	result := h.submit(7, h.sign(7, "proposal", "", map[string]any{"paramsMessage": base64.StdEncoding.EncodeToString(wire)}), "", "")
	var id uint64
	for _, event := range result.Events {
		for _, attr := range event.Attributes {
			if attr.Key == "proposal_id" {
				_, _ = fmt.Sscan(attr.Value, &id)
			}
		}
	}
	require.NotZero(h.t, id)
	h.submit(7, h.sign(7, "vote", "", map[string]any{"proposalId": id}), "", "")
	h.advance(h.height + 3)
	// Query SDK governance over the same transport, not a direct keeper result.
	req, err := gogoproto.Marshal(&govv1.QueryProposalRequest{ProposalId: id})
	require.NoError(h.t, err)
	response, err := h.client.Query(context.Background(), &abci.RequestQuery{Path: "/cosmos.gov.v1.Query/Proposal", Data: req, Height: h.height})
	require.NoError(h.t, err)
	require.Zero(h.t, response.Code, response.Log)
	var proposal govv1.QueryProposalResponse
	require.NoError(h.t, gogoproto.Unmarshal(response.Value, &proposal))
	require.Equal(h.t, expected, proposal.Proposal.Status, "proposal %d: %s", id, proposal.Proposal.FailedReason)
	if expected == govv1.StatusFailed {
		require.Contains(h.t, proposal.Proposal.FailedReason, "ever_reported", "must fail for the economic latch, not an unrelated proposal error")
	}
}
