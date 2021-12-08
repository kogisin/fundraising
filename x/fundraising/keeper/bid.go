package keeper

import (
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/tendermint/fundraising/x/fundraising/types"
)

// GetBid returns a bid for the given auction id and sequence number.
// A bidder can have as many bids as they want, so sequence is required to get the bid.
func (k Keeper) GetBid(ctx sdk.Context, auctionId uint64, sequence uint64) (bid types.Bid, found bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.GetBidKey(auctionId, sequence))
	if bz == nil {
		return bid, false
	}
	k.cdc.MustUnmarshal(bz, &bid)
	return bid, true
}

// SetBid sets a bid with the given arguments.
func (k Keeper) SetBid(ctx sdk.Context, auctionId uint64, sequence uint64, bidderAcc sdk.AccAddress, bid types.Bid) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&bid)
	store.Set(types.GetBidKey(auctionId, sequence), bz)
	store.Set(types.GetBidIndexKey(bidderAcc, auctionId, sequence), []byte{})
}

// GetBidsByAuctionId returns all bids registered in the store.
func (k Keeper) GetBidsByAuctionId(ctx sdk.Context, auctionId uint64) []types.Bid {
	bids := []types.Bid{}
	k.IterateBidsByAuctionId(ctx, auctionId, func(bid types.Bid) (stop bool) {
		bids = append(bids, bid)
		return false
	})
	return bids
}

// GetBidsByBidder returns all bids that are created by a bidder.
func (k Keeper) GetBidsByBidder(ctx sdk.Context, bidderAcc sdk.AccAddress) []types.Bid {
	bids := []types.Bid{}
	k.IterateBidsByBidder(ctx, bidderAcc, func(bid types.Bid) (stop bool) {
		bids = append(bids, bid)
		return false
	})
	return bids
}

// IterateBidsByAuctionId iterates through all bids stored in the store
// and invokes callback function for each item.
// Stops the iteration when the callback function returns true.
func (k Keeper) IterateBidsByAuctionId(ctx sdk.Context, auctionId uint64, cb func(bid types.Bid) (stop bool)) {
	store := ctx.KVStore(k.storeKey)
	iter := sdk.KVStorePrefixIterator(store, types.GetBidAuctionIDKey(auctionId))
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var bid types.Bid
		k.cdc.MustUnmarshal(iter.Value(), &bid)
		if cb(bid) {
			break
		}
	}
}

// IterateBidsByBidder iterates through all bids by a bidder stored in the store
// and invokes callback function for each item.
// Stops the iteration when the callback function returns true.
func (k Keeper) IterateBidsByBidder(ctx sdk.Context, bidderAcc sdk.AccAddress, cb func(bid types.Bid) (stop bool)) {
	store := ctx.KVStore(k.storeKey)
	iter := sdk.KVStorePrefixIterator(store, types.GetBidIndexByBidderPrefix(bidderAcc))
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		auctionId, sequence := types.ParseBidIndexKey(iter.Key())
		bid, _ := k.GetBid(ctx, auctionId, sequence)
		if cb(bid) {
			break
		}
	}
}

// PlaceBid places bid for the auction.
func (k Keeper) PlaceBid(ctx sdk.Context, msg *types.MsgPlaceBid) error {
	auction, found := k.GetAuction(ctx, msg.AuctionId)
	if !found {
		return sdkerrors.Wrapf(sdkerrors.ErrNotFound, "auction %d is not found", msg.AuctionId)
	}

	if auction.GetStatus() != types.AuctionStatusStarted {
		return sdkerrors.Wrapf(types.ErrInvalidAuctionStatus, "unable to bid because the auction is in %s", auction.GetStatus().String())
	}

	bidAmt := msg.Price.Mul(msg.Coin.Amount.ToDec()).TruncateInt()
	balanceAmt := k.bankKeeper.GetBalance(ctx, msg.GetBidder(), auction.GetPayingCoinDenom()).Amount

	// The bidder must have greater than or equal to the bid amount
	if balanceAmt.Sub(bidAmt).IsNegative() {
		return sdkerrors.ErrInsufficientFunds
	}

	// The bidder cannot bid more than the remaining coin to sell
	if !auction.GetRemainingCoin().Amount.Sub(bidAmt).IsPositive() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "request coin must be lower than or equal to the remaining total selling coin")
	}

	if auction.GetType() == types.AuctionTypeFixedPrice {
		if !msg.Price.Equal(auction.GetStartPrice()) {
			return sdkerrors.Wrap(types.ErrInvalidStartPrice, "bid price must be equal to the start price of the auction")
		}

		// Bidder cannot bid more than the total selling coin
		remaining := auction.GetRemainingCoin().Sub(msg.Coin)
		if err := auction.SetRemainingCoin(remaining); err != nil {
			return err
		}

		k.SetAuction(ctx, auction)
	}

	sequenceId := k.GetNextSequenceWithUpdate(ctx)

	bid := types.Bid{
		AuctionId: auction.GetId(),
		Sequence:  sequenceId,
		Bidder:    msg.Bidder,
		Price:     msg.Price,
		Coin:      msg.Coin,
		Height:    uint64(ctx.BlockHeader().Height),
		Eligible:  false, // it becomes true when a bidder receives succesfully during distribution in endblocker
	}

	k.SetBid(ctx, bid.AuctionId, bid.Sequence, msg.GetBidder(), bid)

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypePlaceBid,
			sdk.NewAttribute(types.AttributeKeyAuctionId, strconv.FormatUint(auction.GetId(), 10)),
			sdk.NewAttribute(types.AttributeKeyBidderAddress, msg.GetBidder().String()),
			sdk.NewAttribute(types.AttributeKeyBidPrice, msg.Price.String()),
			sdk.NewAttribute(types.AttributeKeyBidCoin, msg.Coin.String()),
			sdk.NewAttribute(types.AttributeKeyBidAmount, bidAmt.String()),
		),
	})

	return nil
}