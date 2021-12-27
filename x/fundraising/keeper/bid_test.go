package keeper_test

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/tendermint/fundraising/x/fundraising/types"

	_ "github.com/stretchr/testify/suite"
)

func (suite *KeeperTestSuite) TestBidIterators() {
	// create a fixed price auction with already started status
	suite.SetAuction(suite.ctx, suite.sampleFixedPriceAuctions[1])

	auction, found := suite.keeper.GetAuction(suite.ctx, 2)
	suite.Require().True(found)

	for _, bid := range suite.sampleFixedPriceBids {
		suite.PlaceBid(suite.ctx, bid)
	}

	bids := suite.keeper.GetBids(suite.ctx)
	suite.Require().Len(bids, 4)

	bidsById := suite.keeper.GetBidsByAuctionId(suite.ctx, auction.GetId())
	suite.Require().Len(bidsById, 4)

	bidsByBidder := suite.keeper.GetBidsByBidder(suite.ctx, suite.addrs[0])
	suite.Require().Len(bidsByBidder, 2)
}

func (suite *KeeperTestSuite) TestBidSequence() {
	suite.SetAuction(suite.ctx, suite.sampleFixedPriceAuctions[1])

	for _, bid := range suite.sampleFixedPriceBids {
		suite.PlaceBid(suite.ctx, bid)
	}

	auction, found := suite.keeper.GetAuction(suite.ctx, 2)
	suite.Require().True(found)

	bidsById := suite.keeper.GetBidsByAuctionId(suite.ctx, auction.GetId())
	suite.Require().Len(bidsById, 4)
	suite.Require().Equal(uint64(5), suite.keeper.GetNextSequenceWithUpdate(suite.ctx, auction.GetId()))

	// create a new auction with auction
	suite.SetAuction(suite.ctx, types.NewFixedPriceAuction(
		&types.BaseAuction{
			Id:                    3,
			Type:                  types.AuctionTypeFixedPrice,
			Auctioneer:            suite.addrs[4].String(),
			SellingReserveAddress: types.SellingReserveAcc(3).String(),
			PayingReserveAddress:  types.PayingReserveAcc(3).String(),
			StartPrice:            sdk.MustNewDecFromStr("0.5"),
			SellingCoin:           sdk.NewInt64Coin(denom3, 1_000_000_000_000),
			PayingCoinDenom:       denom4,
			VestingReserveAddress: types.VestingReserveAcc(3).String(),
			VestingSchedules:      []types.VestingSchedule{},
			WinningPrice:          sdk.ZeroDec(),
			RemainingCoin:         sdk.NewInt64Coin(denom3, 1_000_000_000_000),
			StartTime:             types.ParseTime("2021-12-10T00:00:00Z"),
			EndTimes:              []time.Time{types.ParseTime("2022-12-20T00:00:00Z")},
			Status:                types.AuctionStatusStarted,
		},
	))

	auction, found = suite.keeper.GetAuction(suite.ctx, 3)
	suite.Require().True(found)

	// sequence must start with 1
	bidsById = suite.keeper.GetBidsByAuctionId(suite.ctx, auction.GetId())
	suite.Require().Len(bidsById, 0)
	suite.Require().Equal(uint64(1), suite.keeper.GetNextSequenceWithUpdate(suite.ctx, auction.GetId()))
}
