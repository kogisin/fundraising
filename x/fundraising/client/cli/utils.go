package cli

import (
	"encoding/json"
	"io/ioutil"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/tendermint/fundraising/x/fundraising/types"
)

// FixedPriceAuctionRequest defines CLI request for a fixed price auction.
type FixedPriceAuctionRequest struct {
	StartPrice       sdk.Dec                 `json:"start_price"`
	SellingCoin      sdk.Coin                `json:"selling_coin"`
	PayingCoinDenom  string                  `json:"paying_coin_denom"`
	VestingSchedules []types.VestingSchedule `json:"vesting_schedules"`
	StartTime        time.Time               `json:"start_time"`
	EndTime          time.Time               `json:"end_time"`
}

// EnglishAuctionRequest defines CLI request for an english auction.
type EnglishAuctionRequest struct {
	StartPrice       sdk.Dec                 `json:"start_price"`
	SellingCoin      sdk.Coin                `json:"selling_coin"`
	PayingCoinDenom  string                  `json:"paying_coin_denom"`
	VestingSchedules []types.VestingSchedule `json:"vesting_schedules"`
	MaximumBidPrice  sdk.Dec                 `json:"maximum_bid_price"`
	Extended         uint32                  `json:"extended"`
	ExtendRate       sdk.Dec                 `json:"extend_rate"`
	StartTime        time.Time               `json:"start_time"`
	EndTime          time.Time               `json:"end_time"`
}

// ParseFixedPriceAuctionRequest reads the file and parses FixedPriceAuctionRequest.
func ParseFixedPriceAuctionRequest(fileName string) (req FixedPriceAuctionRequest, err error) {
	contents, err := ioutil.ReadFile(fileName)
	if err != nil {
		return req, err
	}

	if err = json.Unmarshal(contents, &req); err != nil {
		return req, err
	}

	return req, nil
}

// String returns a human readable string representation of the request.
func (req FixedPriceAuctionRequest) String() string {
	result, err := json.Marshal(&req)
	if err != nil {
		panic(err)
	}
	return string(result)
}

// ParseEnglishAuctionRequest reads the file and parses EnglishAuctionRequest.
func ParseEnglishAuctionRequest(fileName string) (req EnglishAuctionRequest, err error) {
	contents, err := ioutil.ReadFile(fileName)
	if err != nil {
		return req, err
	}

	if err = json.Unmarshal(contents, &req); err != nil {
		return req, err
	}

	return req, nil
}

// String returns a human readable string representation of the request.
func (req EnglishAuctionRequest) String() string {
	result, err := json.Marshal(&req)
	if err != nil {
		panic(err)
	}
	return string(result)
}
