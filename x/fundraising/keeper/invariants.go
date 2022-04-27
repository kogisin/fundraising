package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/fundraising/x/fundraising/types"
)

// RegisterInvariants registers all fundraising invariants.
func RegisterInvariants(ir sdk.InvariantRegistry, k Keeper) {
	ir.RegisterRoute(types.ModuleName, "selling-pool-reserve-amount",
		SellingPoolReserveAmountInvariant(k))
	ir.RegisterRoute(types.ModuleName, "paying-pool-reserve-amount",
		PayingPoolReserveAmountInvariant(k))
	ir.RegisterRoute(types.ModuleName, "vesting-pool-reserve-amount",
		VestingPoolReserveAmountInvariant(k))
	ir.RegisterRoute(types.ModuleName, "auction-status-states",
		AuctionStatusStatesInvariant(k))
}

// AllInvariants runs all invariants of the fundraising module.
func AllInvariants(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		for _, inv := range []func(Keeper) sdk.Invariant{
			SellingPoolReserveAmountInvariant,
			PayingPoolReserveAmountInvariant,
			VestingPoolReserveAmountInvariant,
			AuctionStatusStatesInvariant,
		} {
			res, stop := inv(k)(ctx)
			if stop {
				return res, stop
			}
		}
		return "", false
	}
}

// SellingPoolReserveAmountInvariant checks an invariant that the total amount of selling coin for an auction
// must equal or greater than the selling reserve account balance.
func SellingPoolReserveAmountInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		msg := ""
		count := 0

		for _, auction := range k.GetAuctions(ctx) {
			if auction.GetStatus() == types.AuctionStatusStarted {
				sellingReserveAddr := auction.GetSellingReserveAddress()
				sellingCoinDenom := auction.GetSellingCoin().Denom
				spendable := k.bankKeeper.SpendableCoins(ctx, sellingReserveAddr)
				sellingReserve := sdk.NewCoin(sellingCoinDenom, spendable.AmountOf(sellingCoinDenom))
				if !sellingReserve.IsGTE(auction.GetSellingCoin()) {
					msg += fmt.Sprintf("\tselling reserve balance %s\n"+
						"\tselling pool reserve: %v\n"+
						"\ttotal selling coin: %v\n",
						sellingReserveAddr.String(), sellingReserve, auction.GetSellingCoin())
					count++
				}
			}
		}
		broken := count != 0

		return sdk.FormatInvariant(types.ModuleName, "selling pool reserve amount and selling coin amount", msg), broken
	}
}

// PayingPoolReserveAmountInvariant checks an invariant that the total bid amount
// must equal or greater than the paying reserve account balance.
func PayingPoolReserveAmountInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		msg := ""
		count := 0

		for _, auction := range k.GetAuctions(ctx) {
			totalBidCoin := sdk.NewCoin(auction.GetPayingCoinDenom(), sdk.ZeroInt())

			if auction.GetStatus() == types.AuctionStatusStarted {
				for _, bid := range k.GetBidsByAuctionId(ctx, auction.GetId()) {
					bidAmt := bid.ConvertToPayingAmount(auction.GetPayingCoinDenom())
					totalBidCoin = totalBidCoin.Add(sdk.NewCoin(auction.GetPayingCoinDenom(), bidAmt))
				}
			}

			payingReserveAddr := auction.GetPayingReserveAddress()
			payingCoinDenom := auction.GetPayingCoinDenom()
			spendable := k.bankKeeper.SpendableCoins(ctx, payingReserveAddr)
			payingReserve := sdk.NewCoin(payingCoinDenom, spendable.AmountOf(payingCoinDenom))
			if !payingReserve.IsGTE(totalBidCoin) {
				msg += fmt.Sprintf("\tpaying reserve balance %s\n"+
					"\tpaying pool reserve: %v\n"+
					"\ttotal bid coin: %v\n",
					payingReserveAddr.String(), payingReserve, totalBidCoin)
				count++
			}
		}
		broken := count != 0

		return sdk.FormatInvariant(types.ModuleName, "paying pool reserve amount and total bids amount", msg), broken
	}
}

// VestingPoolReserveAmountInvariant checks an invariant that the total vesting amount
// must be equal or greater than the vesting reserve account balance.
func VestingPoolReserveAmountInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		msg := ""
		count := 0

		for _, auction := range k.GetAuctions(ctx) {
			totalPayingCoin := sdk.NewCoin(auction.GetPayingCoinDenom(), sdk.ZeroInt())

			if auction.GetStatus() == types.AuctionStatusVesting {
				for _, queue := range k.GetVestingQueuesByAuctionId(ctx, auction.GetId()) {
					if !queue.Released {
						totalPayingCoin = totalPayingCoin.Add(queue.PayingCoin)
					}
				}
			}

			vestingReserveAddr := auction.GetVestingReserveAddress()
			payingCoinDenom := auction.GetPayingCoinDenom()
			spendable := k.bankKeeper.SpendableCoins(ctx, vestingReserveAddr)
			vestingReserve := sdk.NewCoin(payingCoinDenom, spendable.AmountOf(payingCoinDenom))
			if !vestingReserve.IsGTE(totalPayingCoin) {
				msg += fmt.Sprintf("\tvesting reserve balance %s\n"+
					"\tvesting pool reserve: %v\n"+
					"\ttotal paying coin: %v\n",
					vestingReserveAddr.String(), vestingReserve, totalPayingCoin)
				count++
			}
		}
		broken := count != 0

		return sdk.FormatInvariant(types.ModuleName, "vesting pool reserve amount and total paying amount", msg), broken
	}
}

// AuctionStatusStatesInvariant checks an invariant that states are properly set depending on the auction status.
func AuctionStatusStatesInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		msg := ""
		count := 0

		for _, auction := range k.GetAuctions(ctx) {
			_, found := k.GetAuction(ctx, auction.GetId())
			if !found {
				msg += fmt.Sprintf("\tauction %d not found\n", auction.GetId())
				count++
			}

			switch auction.GetStatus() {
			case types.AuctionStatusStandBy:
				if !ctx.BlockTime().Before(auction.GetStartTime()) {
					msg += fmt.Sprintf("\texpected status for auction %d is %s\n", auction.GetId(), types.AuctionStatusStandBy)
					msg += fmt.Sprintf("\tcurrent time %s\n", ctx.BlockTime())
					msg += fmt.Sprintf("\tstart time %s\n", auction.GetStartTime())
					msg += fmt.Sprintf("\tend time %s\n\n", auction.GetEndTimes()[0])
					count++
				}
			case types.AuctionStatusStarted:
				if !auction.ShouldAuctionStarted(ctx.BlockTime()) {
					msg += fmt.Sprintf("\texpected status for auction %d is %s\n", auction.GetId(), types.AuctionStatusStarted)
					msg += fmt.Sprintf("\tcurrentTime: %s\n", ctx.BlockTime())
					msg += fmt.Sprintf("\tstartTime: %s\n", auction.GetStartTime())
					msg += fmt.Sprintf("\tendTime: %s\n\n", auction.GetEndTimes()[0])
					count++
				}
			case types.AuctionStatusVesting:
				lenVestingSchedules := len(auction.GetVestingSchedules())
				lenVestingQueues := len(k.GetVestingQueuesByAuctionId(ctx, auction.GetId()))

				if lenVestingSchedules != lenVestingQueues {
					msg += fmt.Sprintf("\texpected vesting queue length %d but got %d\n", lenVestingSchedules, lenVestingQueues)
					count++
				}
			case types.AuctionStatusFinished:
				if auction.GetType() == types.AuctionTypeFixedPrice {
					if !auction.ShouldAuctionClosed(ctx.BlockTime()) {
						msg += fmt.Sprintf("\texpected status for auction %d is %s\n", auction.GetId(), types.AuctionStatusFinished)
						count++
					}
				}
			case types.AuctionStatusCancelled:
				if !auction.GetRemainingSellingCoin().IsZero() {
					msg += fmt.Sprintf("\texpected remaining coin is 0 for auction %d but got %v\n",
						auction.GetId(), auction.GetRemainingSellingCoin())
					count++
				}
			default:
				panic("invalid auction status")
			}
		}

		broken := count != 0
		return sdk.FormatInvariant(types.ModuleName, "auction status states", msg), broken
	}
}
