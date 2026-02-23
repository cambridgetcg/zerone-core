package ibc_test

import (
	"encoding/json"
	"os"
	"testing"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/suite"

	"cosmossdk.io/log"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"

	ibctesting "github.com/cosmos/ibc-go/v8/testing"

	zeroneapp "github.com/zerone-chain/zerone/app"
	ibcratelimittypes "github.com/zerone-chain/zerone/x/ibcratelimit/types"
)

func init() {
	ibctesting.DefaultTestingAppInit = SetupTestingApp
}

// SetupTestingApp creates a ZeroneApp for use with the ibctesting framework.
func SetupTestingApp() (ibctesting.TestingApp, map[string]json.RawMessage) {
	db := dbm.NewMemDB()
	app := zeroneapp.NewZeroneApp(
		log.NewNopLogger(),
		db,
		nil,  // traceStore
		true, // loadLatest
		simtestutil.NewAppOptionsWithFlagHome(os.TempDir()),
	)
	return app, app.DefaultGenesis()
}

// IBCTestSuite is the testify suite for IBC integration tests.
type IBCTestSuite struct {
	suite.Suite

	coordinator  *ibctesting.Coordinator
	chainA       *ibctesting.TestChain
	chainB       *ibctesting.TestChain
	transferPath *ibctesting.Path
}

// TestIBCTestSuite runs the IBC test suite.
func TestIBCTestSuite(t *testing.T) {
	suite.Run(t, new(IBCTestSuite))
}

// SetupTest initializes the coordinator, chains, and transfer path.
func (s *IBCTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 2)
	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))

	s.transferPath = ibctesting.NewTransferPath(s.chainA, s.chainB)
	s.coordinator.Setup(s.transferPath)
}

// GetZeroneApp casts a TestChain's app to *ZeroneApp.
func GetZeroneApp(chain *ibctesting.TestChain) *zeroneapp.ZeroneApp {
	app, ok := chain.App.(*zeroneapp.ZeroneApp)
	if !ok {
		panic("chain app is not *ZeroneApp")
	}
	return app
}

// SetupRateLimit configures a rate limit on chainA via the keeper directly.
func (s *IBCTestSuite) SetupRateLimit(channelID, denom, maxSend, maxRecv string, windowBlocks uint64) {
	app := GetZeroneApp(s.chainA)
	ctx := s.chainA.GetContext()

	// Ensure rate limiting is enabled.
	app.IBCRateLimitKeeper.SetParams(ctx, &ibcratelimittypes.Params{Enabled: true})

	app.IBCRateLimitKeeper.SetRateLimit(ctx, &ibcratelimittypes.RateLimit{
		ChannelId:    channelID,
		Denom:        denom,
		MaxSend:      maxSend,
		MaxRecv:      maxRecv,
		WindowBlocks: windowBlocks,
		CurrentSend:  "0",
		CurrentRecv:  "0",
		WindowStart:  uint64(ctx.BlockHeight()),
	})
}
