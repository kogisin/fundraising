package keeper

import (
	"fmt"
	"strconv"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/tendermint/fundraising/x/fundraising/types"
)

// GetNextAuctionIdWithUpdate increments auction id by one and set it.
func (k Keeper) GetNextAuctionIdWithUpdate(ctx sdk.Context) uint64 {
	id := k.GetLastAuctionId(ctx) + 1
	k.SetAuctionId(ctx, id)
	return id
}

// DistributeSellingCoin releases designated selling coin from the selling reserve account.
func (k Keeper) DistributeSellingCoin(ctx sdk.Context, auction types.AuctionI) error {
	sellingReserveAddress := auction.GetSellingReserveAddress()

	var inputs []banktypes.Input
	var outputs []banktypes.Output

	totalBidCoin := sdk.NewCoin(auction.GetSellingCoin().Denom, sdk.ZeroInt())

	// Distribute coins to all bidders from the selling reserve account
	for _, bid := range k.GetBidsByAuctionId(ctx, auction.GetId()) {
		exchangedSellingAmt := bid.Coin.Amount.ToDec().QuoTruncate(bid.Price).TruncateInt()
		exchangedSellingCoin := sdk.NewCoin(auction.GetSellingCoin().Denom, exchangedSellingAmt)

		inputs = append(inputs, banktypes.NewInput(sellingReserveAddress, sdk.NewCoins(exchangedSellingCoin)))
		outputs = append(outputs, banktypes.NewOutput(bid.GetBidder(), sdk.NewCoins(exchangedSellingCoin)))

		totalBidCoin = totalBidCoin.Add(exchangedSellingCoin)
	}

	balance := k.bankKeeper.GetBalance(ctx, sellingReserveAddress, auction.GetSellingCoin().Denom)
	remainingCoin := balance.Sub(totalBidCoin)

	// Send remaining coin to the auctioneer
	inputs = append(inputs, banktypes.NewInput(sellingReserveAddress, sdk.NewCoins(remainingCoin)))
	outputs = append(outputs, banktypes.NewOutput(auction.GetAuctioneer(), sdk.NewCoins(remainingCoin)))

	// Send all at once
	if err := k.bankKeeper.InputOutputCoins(ctx, inputs, outputs); err != nil {
		return err
	}

	return nil
}

// DistributePayingCoin releases the selling coin from the vesting reserve account.
func (k Keeper) DistributePayingCoin(ctx sdk.Context, auction types.AuctionI) error {
	lenVestingQueue := len(k.GetVestingQueuesByAuctionId(ctx, auction.GetId()))

	for i, vq := range k.GetVestingQueuesByAuctionId(ctx, auction.GetId()) {
		if vq.IsVestingReleasable(ctx.BlockTime()) {
			vestingReserveAddress := auction.GetVestingReserveAddress()

			if err := k.bankKeeper.SendCoins(ctx, vestingReserveAddress, auction.GetAuctioneer(), sdk.NewCoins(vq.PayingCoin)); err != nil {
				return sdkerrors.Wrap(err, "failed to release paying coin to the auctioneer")
			}

			vq.Released = true
			k.SetVestingQueue(ctx, vq)

			// Set finished status when vesting schedule is ended
			if i == lenVestingQueue-1 {
				if err := auction.SetStatus(types.AuctionStatusFinished); err != nil {
					return err
				}

				k.SetAuction(ctx, auction)
			}
		}
	}

	return nil
}

// ReserveSellingCoin reserves the selling coin to the selling reserve account.
func (k Keeper) ReserveSellingCoin(ctx sdk.Context, auctionId uint64, auctioneerAddr sdk.AccAddress, sellingCoin sdk.Coin) error {
	if err := k.bankKeeper.SendCoins(ctx, auctioneerAddr, types.SellingReserveAddress(auctionId), sdk.NewCoins(sellingCoin)); err != nil {
		return sdkerrors.Wrap(err, "failed to reserve selling coin")
	}
	return nil
}

// ReleaseSellingCoin releases the selling coin to the auctioneer.
func (k Keeper) ReleaseSellingCoin(ctx sdk.Context, auction types.AuctionI) error {
	sellingReserveAddr := auction.GetSellingReserveAddress()
	auctioneerAddr := auction.GetAuctioneer()
	releaseCoin := k.bankKeeper.GetBalance(ctx, sellingReserveAddr, auction.GetSellingCoin().Denom)

	if err := k.bankKeeper.SendCoins(ctx, sellingReserveAddr, auctioneerAddr, sdk.NewCoins(releaseCoin)); err != nil {
		return sdkerrors.Wrap(err, "failed to release selling coin")
	}
	return nil
}

// ReserveCreationFee reserves the auction creation fee to the fee collector account.
func (k Keeper) ReserveCreationFee(ctx sdk.Context, auctioneerAddr sdk.AccAddress) error {
	params := k.GetParams(ctx)

	feeCollectorAddr, err := sdk.AccAddressFromBech32(params.FeeCollectorAddress)
	if err != nil {
		return err
	}

	if err := k.bankKeeper.SendCoins(ctx, auctioneerAddr, feeCollectorAddr, params.AuctionCreationFee); err != nil {
		return sdkerrors.Wrap(err, "failed to reserve auction creation fee")
	}
	return nil
}

// CreateFixedPriceAuction sets fixed price auction.
func (k Keeper) CreateFixedPriceAuction(ctx sdk.Context, msg *types.MsgCreateFixedPriceAuction) (*types.FixedPriceAuction, error) {
	if ctx.BlockTime().After(msg.EndTime) {
		return &types.FixedPriceAuction{}, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "end time must be set prior to the current time")
	}

	nextId := k.GetNextAuctionIdWithUpdate(ctx)

	if err := k.ReserveCreationFee(ctx, msg.GetAuctioneer()); err != nil {
		return &types.FixedPriceAuction{}, err
	}

	if err := k.ReserveSellingCoin(ctx, nextId, msg.GetAuctioneer(), msg.SellingCoin); err != nil {
		return &types.FixedPriceAuction{}, err
	}

	allowedBidders := []types.AllowedBidder{} // it is nil when an auction is created
	winningPrice := sdk.ZeroDec()             // TODO: makes sense to have start price?
	numWinningBidders := uint64(0)            // initial value is 0
	remainingSellingCoin := msg.SellingCoin   // it is starting with selling coin amount
	endTimes := []time.Time{msg.EndTime}      // it is an array data type to handle BatchAuction

	baseAuction := types.NewBaseAuction(
		nextId,
		types.AuctionTypeFixedPrice,
		allowedBidders,
		msg.Auctioneer,
		types.SellingReserveAddress(nextId).String(),
		types.PayingReserveAddress(nextId).String(),
		msg.StartPrice,
		msg.SellingCoin,
		msg.PayingCoinDenom,
		types.VestingReserveAddress(nextId).String(),
		msg.VestingSchedules,
		winningPrice,
		numWinningBidders,
		remainingSellingCoin,
		msg.StartTime,
		endTimes,
		types.AuctionStatusStandBy,
	)

	// Update status if the start time is already passed over the current time
	if baseAuction.IsAuctionStarted(ctx.BlockTime()) {
		baseAuction.Status = types.AuctionStatusStarted
	}

	auction := types.NewFixedPriceAuction(baseAuction)
	k.SetAuction(ctx, auction)

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeCreateFixedPriceAuction,
			sdk.NewAttribute(types.AttributeKeyAuctionId, strconv.FormatUint(nextId, 10)),
			sdk.NewAttribute(types.AttributeKeyAuctioneerAddress, msg.Auctioneer),
			sdk.NewAttribute(types.AttributeKeyStartPrice, msg.StartPrice.String()),
			sdk.NewAttribute(types.AttributeKeySellingReserveAddress, auction.GetSellingReserveAddress().String()),
			sdk.NewAttribute(types.AttributeKeyPayingReserveAddress, auction.GetPayingReserveAddress().String()),
			sdk.NewAttribute(types.AttributeKeyVestingReserveAddress, auction.GetVestingReserveAddress().String()),
			sdk.NewAttribute(types.AttributeKeySellingCoin, msg.SellingCoin.String()),
			sdk.NewAttribute(types.AttributeKeyPayingCoinDenom, msg.PayingCoinDenom),
			sdk.NewAttribute(types.AttributeKeyStartTime, msg.StartTime.String()),
			sdk.NewAttribute(types.AttributeKeyEndTime, msg.EndTime.String()),
			sdk.NewAttribute(types.AttributeKeyAuctionStatus, types.AuctionStatusStandBy.String()),
		),
	})

	return auction, nil
}

// CreateBatchAuction sets batch auction.
func (k Keeper) CreateBatchAuction(ctx sdk.Context, msg *types.MsgCreateBatchAuction) (*types.BatchAuction, error) {
	if ctx.BlockTime().After(msg.EndTime) {
		return &types.BatchAuction{}, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "end time must be set prior to the current time")
	}

	nextId := k.GetNextAuctionIdWithUpdate(ctx)

	if err := k.ReserveCreationFee(ctx, msg.GetAuctioneer()); err != nil {
		return &types.BatchAuction{}, err
	}

	if err := k.ReserveSellingCoin(ctx, nextId, msg.GetAuctioneer(), msg.SellingCoin); err != nil {
		return &types.BatchAuction{}, err
	}

	allowedBidders := []types.AllowedBidder{} // it is nil when an auction is created
	winningPrice := sdk.ZeroDec()             // TODO: makes sense to have start price?
	numWinningBidders := uint64(0)            // initial value is 0
	remainingSellingCoin := msg.SellingCoin   // it is starting with selling coin amount
	endTimes := []time.Time{msg.EndTime}      // it is an array data type to handle BatchAuction

	baseAuction := types.NewBaseAuction(
		nextId,
		types.AuctionTypeBatch,
		allowedBidders,
		msg.Auctioneer,
		types.SellingReserveAddress(nextId).String(),
		types.PayingReserveAddress(nextId).String(),
		msg.StartPrice,
		msg.SellingCoin,
		msg.PayingCoinDenom,
		types.VestingReserveAddress(nextId).String(),
		msg.VestingSchedules,
		winningPrice,
		numWinningBidders,
		remainingSellingCoin,
		msg.StartTime,
		endTimes,
		types.AuctionStatusStandBy,
	)

	// Update status if the start time is already passed the current time
	if baseAuction.IsAuctionStarted(ctx.BlockTime()) {
		baseAuction.Status = types.AuctionStatusStarted
	}

	auction := types.NewBatchAuction(
		baseAuction,
		msg.MaxExtendedRound,
		msg.ExtendedRoundRate,
	)
	k.SetAuction(ctx, auction)

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeCreateFixedPriceAuction,
			sdk.NewAttribute(types.AttributeKeyAuctionId, strconv.FormatUint(nextId, 10)),
			sdk.NewAttribute(types.AttributeKeyAuctioneerAddress, msg.Auctioneer),
			sdk.NewAttribute(types.AttributeKeyStartPrice, auction.GetStartPrice().String()),
			sdk.NewAttribute(types.AttributeKeySellingReserveAddress, auction.GetSellingReserveAddress().String()),
			sdk.NewAttribute(types.AttributeKeyPayingReserveAddress, auction.GetPayingReserveAddress().String()),
			sdk.NewAttribute(types.AttributeKeyVestingReserveAddress, auction.GetVestingReserveAddress().String()),
			sdk.NewAttribute(types.AttributeKeySellingCoin, auction.GetSellingCoin().String()),
			sdk.NewAttribute(types.AttributeKeyPayingCoinDenom, auction.GetPayingCoinDenom()),
			sdk.NewAttribute(types.AttributeKeyStartTime, auction.GetStartTime().String()),
			sdk.NewAttribute(types.AttributeKeyEndTime, msg.EndTime.String()),
			sdk.NewAttribute(types.AttributeKeyAuctionStatus, auction.GetStatus().String()),
			sdk.NewAttribute(types.AttributeKeyMaxExtendedRound, fmt.Sprint(msg.MaxExtendedRound)),
			sdk.NewAttribute(types.AttributeKeyExtendedRoundRate, msg.ExtendedRoundRate.String()),
		),
	})

	return auction, nil
}

// CancelAuction cancels the auction in an event when the auctioneer needs to modify the auction.
// However, it can only be canceled when the auction has not started yet.
func (k Keeper) CancelAuction(ctx sdk.Context, msg *types.MsgCancelAuction) (types.AuctionI, error) {
	auction, found := k.GetAuction(ctx, msg.AuctionId)
	if !found {
		return nil, sdkerrors.Wrap(sdkerrors.ErrNotFound, "auction not found")
	}

	if auction.GetAuctioneer().String() != msg.Auctioneer {
		return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "failed to verify ownership of the auction")
	}

	if auction.GetStatus() != types.AuctionStatusStandBy {
		return nil, sdkerrors.Wrap(types.ErrInvalidAuctionStatus, "auction cannot be canceled due to current status")
	}

	if err := k.ReleaseSellingCoin(ctx, auction); err != nil {
		return nil, err
	}

	_ = auction.SetRemainingSellingCoin(sdk.NewCoin(auction.GetSellingCoin().Denom, sdk.ZeroInt()))
	_ = auction.SetStatus(types.AuctionStatusCancelled)

	k.SetAuction(ctx, auction)

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeCancelAuction,
			sdk.NewAttribute(types.AttributeKeyAuctionId, strconv.FormatUint(auction.GetId(), 10)),
		),
	})

	return auction, nil
}

// AddAllowedBidders is a function for an external module and it simply adds new bidder(s) to AllowedBidder list.
// Note that it doesn't do auctioneer verification because the module is generalized for broader use cases.
// It is designed to delegate to an external module to add necessary verification and logics depending on their use case.
func (k Keeper) AddAllowedBidders(ctx sdk.Context, auctionId uint64, bidders []types.AllowedBidder) error {
	auction, found := k.GetAuction(ctx, auctionId)
	if !found {
		return sdkerrors.Wrapf(sdkerrors.ErrNotFound, "auction %d is not found", auctionId)
	}

	if len(bidders) == 0 {
		return types.ErrEmptyAllowedBidders
	}

	if err := types.ValidateAllowedBidders(bidders); err != nil {
		return err
	}

	// Append new bidders from the existing ones
	allowedBidders := auction.GetAllowedBidders()
	allowedBidders = append(allowedBidders, bidders...)

	if err := auction.SetAllowedBidders(allowedBidders); err != nil {
		return err
	}
	k.SetAuction(ctx, auction)

	return nil
}

// UpdateAllowedBidder is a function for an external module and it simply updates the bidder's maximum bid amount.
// Note that it doesn't do auctioneer verification because the module is generalized for broader use cases.
// It is designed to delegate to an external module to add necessary verification and logics depending on their use case.
func (k Keeper) UpdateAllowedBidder(ctx sdk.Context, auctionId uint64, bidder sdk.AccAddress, maxBidAmount sdk.Int) error {
	auction, found := k.GetAuction(ctx, auctionId)
	if !found {
		return sdkerrors.Wrapf(sdkerrors.ErrNotFound, "auction %d is not found", auctionId)
	}

	if maxBidAmount.IsNil() {
		return types.ErrInvalidMaxBidAmount
	}

	if !maxBidAmount.IsPositive() {
		return types.ErrInvalidMaxBidAmount
	}

	if _, found := auction.GetAllowedBiddersMap()[bidder.String()]; !found {
		return sdkerrors.Wrapf(sdkerrors.ErrNotFound, "bidder %s is not found", bidder.String())
	}

	_ = auction.SetMaxBidAmount(bidder.String(), maxBidAmount)

	k.SetAuction(ctx, auction)

	return nil
}

func (k Keeper) ExecuteStandByStatus(ctx sdk.Context, auction types.AuctionI) {
	if auction.IsAuctionStarted(ctx.BlockTime()) {
		if err := auction.SetStatus(types.AuctionStatusStarted); err != nil {
			panic(err)
		}
		k.SetAuction(ctx, auction)
	}
}

func (k Keeper) ExecuteStartedStatus(ctx sdk.Context, auction types.AuctionI) {
	ctx, writeCache := ctx.CacheContext()

	// Do nothing when the auction is still in started status
	if !auction.IsAuctionFinished(ctx.BlockTime()) {
		return
	}

	switch auction.GetType() {
	case types.AuctionTypeFixedPrice:
		if err := k.DistributeSellingCoin(ctx, auction); err != nil {
			panic(err)
		}

		if err := k.SetVestingSchedules(ctx, auction); err != nil {
			panic(err)
		}

	case types.AuctionTypeBatch:
		k.CalculateBatchResult(ctx, auction.GetId(), auction.GetRemainingSellingCoin().Amount)

		k.FinishBatchAuction(ctx, auction)
	}

	writeCache()
}

func (k Keeper) ExecuteVestingStatus(ctx sdk.Context, auction types.AuctionI) {
	if err := k.DistributePayingCoin(ctx, auction); err != nil {
		panic(err)
	}
}

type BatchAuctionResult struct {
	Bids       []types.Bid
	Price      sdk.Dec
	SoldAmount sdk.Int
}

func (k Keeper) CalculateBatchResult(ctx sdk.Context, auctionId uint64, remainingAmt sdk.Int) BatchAuctionResult {
	bids := k.GetBidsByAuctionId(ctx, auctionId)
	bids = types.SortByBidPrice(bids)

	result := BatchAuctionResult{
		Bids:       []types.Bid{},
		Price:      bids[0].Price,
		SoldAmount: sdk.ZeroInt(),
	}

	// Iterate from the highest bid price and find the last bid price and total sold amount
	// It doesn't calculate a partial amount of coins that the last bid can match
	for _, b := range bids {
		currPrice := b.Price
		accumulatedAmt := sdk.ZeroInt()

		for _, b := range bids {
			if b.Price.LT(currPrice) {
				continue
			}

			if b.Type == types.BidTypeBatchWorth {
				amt := b.Coin.Amount.ToDec().QuoTruncate(currPrice).TruncateInt()
				accumulatedAmt = accumulatedAmt.Add(amt)
			} else {
				accumulatedAmt = accumulatedAmt.Add(b.Coin.Amount)
			}
		}

		if accumulatedAmt.GT(remainingAmt) {
			break
		}

		// TODO: set winner status to true
		b.SetWinner(true)
		k.SetBid(ctx, b)

		result.Bids = append(result.Bids, b)
		result.Price = currPrice
		result.SoldAmount = accumulatedAmt
	}

	k.SetWinningBidsLen(ctx, auctionId, len(result.Bids))

	return result
}

func (k Keeper) FinishBatchAuction(ctx sdk.Context, auction types.AuctionI) {
	ba := auction.(*types.BatchAuction)

	if ba.MaxExtendedRound == 0 {
		// Distribute
	} else {
		// Compare last and current winning bids len and
		// determine if it needs another round
		lastWinningBidsLen := k.GetWinningBidsLen(ctx, ba.Id)
		sdk.NewDec(lastWinningBidsLen)

		ts := ba.GetEndTimes()
		ts = append(ts, ctx.BlockTime())

		_ = ba.SetEndTimes(ts)
		k.SetAuction(ctx, ba)
	}

	// if ba.MaxExtendedRound == 0 {
	// 	// Store the last end time
	// 	// Store auction id -> len(LastWinningBids)
	// } else {
	// 	// GetLastWinningBidsByAuctionId() and compare with current winningBids length
	// 	// Determint if it needs an extended round
	// 	// YES
	// 	// - Store the last time
	// 	// - Store auction id -> len(LastWinningBids)
	// 	// NO -
	// 	// - Set remaining coin
	// 	// - Distribute - vesting (use multisend)
	// }
}
