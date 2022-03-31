package keeper_test

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/tendermint/fundraising/x/fundraising/types"

	_ "github.com/stretchr/testify/suite"
)

func (s *KeeperTestSuite) TestBenchmark_CalculateFixedPriceAllocation() {
	a := s.createFixedPriceAuction(
		s.addr(0),
		parseDec("0.5"),
		parseCoin("50_000_000_000_000denom1"),
		"denom2",
		[]types.VestingSchedule{},
		time.Now().AddDate(0, 0, -1),
		time.Now().AddDate(0, 0, -1).AddDate(0, 2, 0),
		true,
	)

	auction, found := s.keeper.GetAuction(s.ctx, a.GetId())
	s.Require().True(found)

	s.placeBidFixedPrice(auction.GetId(), s.addr(1), auction.GetStartPrice(), parseCoin("15_000_000denom2"), true)
	s.placeBidFixedPrice(auction.GetId(), s.addr(2), auction.GetStartPrice(), parseCoin("22_000_000denom2"), true)

	// A number of FixedPriceAuction: 1
	// A number of bids: 100, 1000, 3000, 5000, 10000, 50000
	// CalculateFixedPriceAllocation
}

func (s *KeeperTestSuite) TestBenchmark_CalculateBatchAllocation() {
	a := s.createBatchAuction(
		s.addr(0),
		parseDec("1"),
		parseDec("0.1"),
		parseCoin("1_000_000_000denom1"),
		"denom2",
		[]types.VestingSchedule{},
		1,
		sdk.MustNewDecFromStr("0.2"),
		time.Now().AddDate(0, 0, -1),
		time.Now().AddDate(0, 0, -1).AddDate(0, 2, 0),
		true,
	)
	s.Require().Equal(types.AuctionStatusStarted, a.GetStatus())

	// TODO: randomize placeBidBatchMany, placeBidBatchWorth
	// s.placeBidBatchMany(a.Id, s.addr(1), parseDec("1"), parseCoin("500_000_000denom1"), sdk.NewInt(1000_000_000), true)
	// s.placeBidBatchMany(a.Id, s.addr(2), parseDec("0.9"), parseCoin("500_000_000denom1"), sdk.NewInt(1000_000_000), true)
	// s.placeBidBatchMany(a.Id, s.addr(3), parseDec("0.8"), parseCoin("500_000_000denom1"), sdk.NewInt(1000_000_000), true)

	// _, found := s.keeper.GetAuction(s.ctx, a.Id)
	// s.Require().True(found)
}
