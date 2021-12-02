package cli_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/tendermint/fundraising/x/fundraising/client/cli"
	"github.com/tendermint/fundraising/x/fundraising/types"
)

func TestParseFixedPriceAuction(t *testing.T) {
	okJSON := testutil.WriteToNewTempFile(t, `
{
  "start_price": "1.000000000000000000",
  "selling_coin": {
    "denom": "denom1",
    "amount": "1000000000000"
  },
  "paying_coin_denom": "denom2",
  "vesting_schedules": [
    {
      "release_time": "2022-01-01T00:00:00Z",
      "weight": "0.500000000000000000"
    },
    {
      "release_time": "2022-06-01T00:00:00Z",
      "weight": "0.250000000000000000"
    },
    {
      "release_time": "2022-12-01T00:00:00Z",
      "weight": "0.250000000000000000"
    }
  ],
  "start_time": "2021-11-01T00:00:00Z",
  "end_time": "2021-12-01T00:00:00Z"
}
`)

	expSchedules := []types.VestingSchedule{
		types.NewVestingSchedule(types.ParseTime("2022-01-01T00:00:00Z"), sdk.MustNewDecFromStr("0.50")),
		types.NewVestingSchedule(types.ParseTime("2022-06-01T00:00:00Z"), sdk.MustNewDecFromStr("0.25")),
		types.NewVestingSchedule(types.ParseTime("2022-12-01T00:00:00Z"), sdk.MustNewDecFromStr("0.25")),
	}

	auction, err := cli.ParseFixedPriceAuctionRequest(okJSON.Name())
	require.NoError(t, err)
	require.NotEmpty(t, auction.String())
	require.Equal(t, sdk.MustNewDecFromStr("1.0"), auction.StartPrice)
	require.Equal(t, sdk.NewInt64Coin("denom1", 1000000000000), auction.SellingCoin)
	require.Equal(t, "denom2", auction.PayingCoinDenom)
	require.EqualValues(t, expSchedules, auction.VestingSchedules)
}
