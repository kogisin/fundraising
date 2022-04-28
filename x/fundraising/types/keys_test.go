package types_test

import (
	"testing"
	time "time"

	"github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/crypto"

	"github.com/tendermint/fundraising/x/fundraising/types"
)

type keysTestSuite struct {
	suite.Suite
}

func TestKeysTestSuite(t *testing.T) {
	suite.Run(t, new(keysTestSuite))
}

func (s *keysTestSuite) TestGetAuctionKey() {
	s.Require().Equal([]byte{0x21, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}, types.GetAuctionKey(0))
	s.Require().Equal([]byte{0x21, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x9}, types.GetAuctionKey(9))
	s.Require().Equal([]byte{0x21, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xa}, types.GetAuctionKey(10))
}

func (s *keysTestSuite) TestGetAllowedBidderKey() {
	testCases := []struct {
		auctionId  uint64
		bidderAddr sdk.AccAddress
		expected   []byte
	}{
		{
			uint64(1),
			sdk.AccAddress(crypto.AddressHash([]byte("bidder1"))),
			[]byte{0x22, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x14, 0x20, 0x5c, 0xa, 0x82,
				0xa, 0xf1, 0xed, 0x98, 0x39, 0x6a, 0x35, 0xfe, 0xe3, 0x5d, 0x5, 0x2c, 0xd7, 0x96, 0x5a, 0x37},
		},
		{
			uint64(3),
			sdk.AccAddress(crypto.AddressHash([]byte("bidder2"))),
			[]byte{0x22, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x3, 0x14, 0xa, 0xaf, 0x72, 0x3a,
				0xd0, 0x8c, 0x17, 0x88, 0x2e, 0xf6, 0x7a, 0x5, 0x31, 0xb3, 0x46, 0xdd, 0x22, 0xb3, 0x62, 0x1e},
		},
		{
			uint64(11),
			sdk.AccAddress(crypto.AddressHash([]byte("bidder3"))),
			[]byte{0x22, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xb, 0x14, 0xe, 0x99, 0x7b, 0x9b, 0x5c, 0xef, 0x81,
				0x2f, 0xc6, 0x3f, 0xb6, 0x8b, 0x27, 0x42, 0x8a, 0xab, 0x7a, 0x58, 0xbc, 0x5e},
		},
	}

	for _, tc := range testCases {
		key := types.GetAllowedBidderKey(tc.auctionId, tc.bidderAddr)
		s.Require().Equal(tc.expected, key)
	}
}

func (s *keysTestSuite) TestGetBidKey() {
	testCases := []struct {
		auctionId uint64
		bidId     uint64
		expected  []byte
	}{
		{
			uint64(5),
			uint64(10),
			[]byte{0x31, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x5, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xa},
		},
		{
			uint64(2),
			uint64(7),
			[]byte{0x31, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x7},
		},
		{
			uint64(3),
			uint64(5),
			[]byte{0x31, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x3, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x5},
		},
	}

	for _, tc := range testCases {
		key := types.GetBidKey(tc.auctionId, tc.bidId)
		s.Require().Equal(tc.expected, key)
	}
}

func (s *keysTestSuite) TestBidIndexKey() {
	testCases := []struct {
		bidderAddr sdk.AccAddress
		auctionId  uint64
		bidId      uint64
		expected   []byte
	}{
		{
			sdk.AccAddress(crypto.AddressHash([]byte("bidder1"))),
			uint64(1),
			uint64(1),
			[]byte{0x32, 0x14, 0x20, 0x5c, 0xa, 0x82, 0xa, 0xf1, 0xed,
				0x98, 0x39, 0x6a, 0x35, 0xfe, 0xe3, 0x5d, 0x5, 0x2c, 0xd7,
				0x96, 0x5a, 0x37, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
				0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
		},
		{
			sdk.AccAddress(crypto.AddressHash([]byte("bidder2"))),
			uint64(3),
			uint64(12),
			[]byte{0x32, 0x14, 0xa, 0xaf, 0x72, 0x3a, 0xd0, 0x8c, 0x17,
				0x88, 0x2e, 0xf6, 0x7a, 0x5, 0x31, 0xb3, 0x46, 0xdd, 0x22,
				0xb3, 0x62, 0x1e, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x3,
				0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xc},
		},
		{
			sdk.AccAddress(crypto.AddressHash([]byte("bidder3"))),
			uint64(12),
			uint64(2),
			[]byte{0x32, 0x14, 0xe, 0x99, 0x7b, 0x9b, 0x5c, 0xef, 0x81,
				0x2f, 0xc6, 0x3f, 0xb6, 0x8b, 0x27, 0x42, 0x8a, 0xab, 0x7a,
				0x58, 0xbc, 0x5e, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xc,
				0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
		},
	}

	for _, tc := range testCases {
		key := types.GetBidIndexKey(tc.bidderAddr, tc.auctionId, tc.bidId)
		s.Require().Equal(tc.expected, key)

		auctionId, bidId := types.ParseBidIndexKey(key)
		s.Require().Equal(tc.auctionId, auctionId)
		s.Require().Equal(tc.bidId, bidId)
	}
}

func (s *keysTestSuite) TestVestingQueueKey() {
	testCases := []struct {
		auctionId uint64
		timestamp time.Time
		expected  []byte
	}{
		{
			uint64(1),
			types.MustParseRFC3339("2021-12-01T00:00:00Z"),
			[]byte{0x41, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x32, 0x30, 0x32, 0x31,
				0x2d, 0x31, 0x32, 0x2d, 0x30, 0x31, 0x54, 0x30, 0x30, 0x3a, 0x30, 0x30,
				0x3a, 0x30, 0x30, 0x2e, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30},
		},
		{
			uint64(5),
			types.MustParseRFC3339("2022-01-05T00:00:00Z"),
			[]byte{0x41, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x5, 0x32, 0x30, 0x32, 0x32,
				0x2d, 0x30, 0x31, 0x2d, 0x30, 0x35, 0x54, 0x30, 0x30, 0x3a, 0x30, 0x30,
				0x3a, 0x30, 0x30, 0x2e, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30},
		},
		{
			uint64(11),
			types.MustParseRFC3339("2022-07-11T00:00:00Z"),
			[]byte{0x41, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xb, 0x32, 0x30, 0x32, 0x32, 0x2d,
				0x30, 0x37, 0x2d, 0x31, 0x31, 0x54, 0x30, 0x30, 0x3a, 0x30, 0x30, 0x3a, 0x30,
				0x30, 0x2e, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30},
		},
	}

	for _, tc := range testCases {
		key := types.GetVestingQueueKey(tc.auctionId, tc.timestamp)
		s.Require().Equal(tc.expected, key)
	}
}
