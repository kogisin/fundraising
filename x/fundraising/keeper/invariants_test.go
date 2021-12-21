package keeper_test

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/tendermint/fundraising/x/fundraising"
	"github.com/tendermint/fundraising/x/fundraising/keeper"
	"github.com/tendermint/fundraising/x/fundraising/types"

	_ "github.com/stretchr/testify/suite"
)

func (suite *KeeperTestSuite) TestSellingPoolReserveAmountInvariant() {
	k, ctx, auction := suite.keeper, suite.ctx, suite.sampleFixedPriceAuctions[1]

	k.SetAuction(suite.ctx, auction)

	_, broken := keeper.SellingPoolReserveAmountInvariant(k)(ctx)
	suite.Require().True(broken)

	err := k.ReserveSellingCoin(
		ctx,
		auction.GetId(),
		auction.GetAuctioneer(),
		auction.GetSellingCoin(),
	)
	suite.Require().NoError(err)

	_, broken = keeper.SellingPoolReserveAmountInvariant(k)(ctx)
	suite.Require().False(broken)
}

func (suite *KeeperTestSuite) TestPayingPoolReserveAmountInvariant() {
	k, ctx, auction := suite.keeper, suite.ctx, suite.sampleFixedPriceAuctions[1]

	k.SetAuction(suite.ctx, auction)
	err := k.ReserveSellingCoin(
		ctx,
		auction.GetId(),
		auction.GetAuctioneer(),
		auction.GetSellingCoin(),
	)
	suite.Require().NoError(err)

	for _, bid := range suite.sampleFixedPriceBids {
		bidderAcc, err := sdk.AccAddressFromBech32(bid.Bidder)
		suite.Require().NoError(err)
		k.SetBid(ctx, bid.AuctionId, bid.Sequence, bidderAcc, bid)

		err = k.ReservePayingCoin(ctx, bid.GetAuctionId(), bidderAcc, bid.Coin)
		suite.Require().NoError(err)
	}

	_, broken := keeper.PayingPoolReserveAmountInvariant(k)(ctx)
	suite.Require().False(broken)
}

func (suite *KeeperTestSuite) TestVestingPoolReserveAmountInvariant() {
	k, ctx, auction := suite.keeper, suite.ctx, suite.sampleFixedPriceAuctions[1]

	k.SetAuction(suite.ctx, auction)
	err := k.ReserveSellingCoin(
		ctx,
		auction.GetId(),
		auction.GetAuctioneer(),
		auction.GetSellingCoin(),
	)
	suite.Require().NoError(err)

	for _, bid := range suite.sampleFixedPriceBids {
		bidderAcc, err := sdk.AccAddressFromBech32(bid.Bidder)
		suite.Require().NoError(err)
		k.SetBid(ctx, bid.AuctionId, bid.Sequence, bidderAcc, bid)

		err = k.ReservePayingCoin(ctx, bid.GetAuctionId(), bidderAcc, bid.Coin)
		suite.Require().NoError(err)
	}

	// set the current block time a day before second auction so that it gets finished
	ctx = ctx.WithBlockTime(suite.sampleFixedPriceAuctions[1].GetEndTimes()[0].AddDate(0, 0, -1))
	fundraising.EndBlocker(ctx, k)

	// make first and second vesting queues over
	ctx = ctx.WithBlockTime(types.ParseTime("2022-04-02T00:00:00Z"))
	fundraising.EndBlocker(ctx, k)

	msg, broken := keeper.VestingPoolReserveAmountInvariant(k)(ctx)
	fmt.Println("msg: ", msg)
	fmt.Println("broken: ", broken)
	suite.Require().False(broken)
}