<!-- order: 2 -->

# State

The `fundraising` module keeps track of the auction and bid states.

## Auction Interface

The auction interface exposes methods to read and write standard auction information.

Note that all of these methods operate on a auction struct that confirms to the interface. In order to write the auction to the store, the auction keeper is required.

```go
// AuctionI is an interface that exposes common methods to read and write
// standard auction information.
type AuctionI interface {
	GetId() uint64
	SetId(uint64) error

	GetType() AuctionType
	SetType(AuctionType) error

	GetAuctioneer() string
	SetAuctioneer(string) error

	GetSellingPoolAddress() string
	SetSellingPoolAddress(string) error

	GetPayingPoolAddress() string
	SetPayingPoolAddress(string) error

	GetStartPrice() sdk.Dec
	SetStartPrice(sdk.Dec) error

	GetSellingCoin() sdk.Coin
	SetSellingCoin(sdk.Coin) error

	GetPayingCoinDenom() string
	SetPayingCoinDenom(string) error

	GetVestingAddress() string
	SetVestingAddress(string) error

	GetVestingSchedules() []VestingSchedule
	SetVestingSchedules([]VestingSchedule) error

	GetStartTime() time.Time
	SetStartTime(time.Time) error

	GetEndTimes() []time.Time
	SetEndTimes([]time.Time) error

	GetStatus() AuctionStatus
	SetStatus(AuctionStatus) error
}
```

## Base Auction

A base auction is the simplest and most common auction type that just stores all requisite fields directly in a struct.

```go
// BaseAuction defines a base auction type. It contains all the necessary fields
// for basic auction functionality. Any custom auction type should extend this
// type for additional functionality (e.g. english auction, fixed ratio auction).
type BaseAuction struct {
	Id                 uint64            // id of the auction
	Type               AuctionType       // supporting auction types are english and fixed price
	Auctioneer         string            // the account that is in charge of the action
	SellingPoolAddress string            // an escrow account to collect selling tokens for the auction
	PayingPoolAddress  string            // an escrow account to collect paying tokens for the auction
	StartPrice         sdk.Dec           // starting price of the auction
	SellingCoin        sdk.Coin          // selling coin for the auction
	PayingCoinDenom    string            // the paying coin denom that bidders use to bid for
	VestingAddress     string            // the vesting account that releases the paying amount of coins based on the schedules
	VestingSchedules   []VestingSchedule // vesting schedules for the auction
	StartTime          time.Time         // start time of the auction
	EndTime            []time.Time       // end times of the auction since extended round(s) can occur
	Status             AuctionStatus     // the auction status
}

// AuctionMetadata defines the metadata for the auction
type AuctionMetadata struct {
	AuctionId        uint64   // id of the auction
	WinningPrice     sdk.Dec  // the winning price of the auction
	TotalSellingCoin sdk.Coin // the total amount of coin that is sold when the auction is finished
}
```

## Vesting

```go
// VestingSchedule defines the vesting schedule for the owner of an auction.
type VestingSchedule struct {
	Time   time.Time // vesting time
	Weight sdk.Dec   // vesting weight for the schedule
}

// VestingQueue defines the vesting queue 
type VestingQueue struct {
	AuctionId  uint64    // id of the auction
	Auctioneer string    // the account that is in charge of the auction
	PayingCoin sdk.Coin  // the paying amount of coin 
	Time       time.Time // timestamp of the vesting schedule
}
```


## Auction Type

```go
// AuctionType is the type of an auction.
type AuctionType uint32

const (
	// AUCTION_TYPE_UNSPECIFIED defines an invalid auction type
	TypeNil AuctionType = 0
	// AUCTION_TYPE_ENGLISH defines the English auction type
	TypeEnglish AuctionType = 1
	// AUCTION_TYPE_FIXED_PRICE defines the fixed price auction type
	TypeFixedPrice AuctionType = 1
)

// EnglishAuction defines the english auction type 
type EnglishAuction struct {
	*BaseAuction

	MaximumBidPrice sdk.Dec // the maximum bid price for the auction
	Extended        uint32  // a number of extended rounds
	ExtendRate      sdk.Dec // rate that decides if the auction needs another round
}

// FixedPriceAuction defines the fixed price auction type
type FixedPriceAuction struct {
	*BaseAuction
}
```

## Auction Status

```go
// AuctionStatus is the status of an auction
type AuctionStatus uint32

const (
	// AUCTION_STATUS_UNSPECIFIED defines an invalid auction status
	StatusNil AuctionStatus = 0
	// AUCTION_STATUS_STANDY_BY defines an auction status before it opens
	StatusStandBy AuctionStatus = 1
	// AUCTION_STATUS_STARTED defines an auction status that is started
	StatusStarted AuctionStatus = 2
	// AUCTION_STATUS_VESTING defines an auction status that is in distribution based on the vesting schedules
	StatusVesting AuctionStatus = 3
	// AUCTION_STATUS_FINISHED defines an auction status that is finished
	StatusFinished AuctionStatus = 4
	// AUCTION_STATUS_CANCELLED defines an auction sttus that is cancelled
	StatusCancelled AuctionStatus = 5
)
```

## Bid

```go
// Bid defines a standard bid for an auction.
type Bid struct {
	Sequence  uint64   // a number to track first come first served based auction type
	AuctionId uint64   // id of the auction
	Bidder    string   // the account that bids for the auction
	Price     sdk.Dec  // increasing bid price is only possible
	Coin      sdk.Coin // paying amount of coin that the bidder bids
	Height    int64    // block height
	isWinner  bool     // winner gets determined when an auction ends; default value is false
}
```

## Parameters

- ModuleName: `farming`
- RouterKey: `farming`
- StoreKey: `farming`
- QuerierRoute: `farming`

## Stores

Stores are KVStores in the multi-store. The key to find the store is the first parameter in the list.

### prefix key for the latest auction id

- `AuctionIdKey: 0x11 -> uint64`

### prefix key for the latest sequence number

- `SequencePrefix: 0x12 -> uint64`

### prefix key for auction id to find an auction

- `AuctionKeyPrefix: 0x21 | AuctionId -> ProtocolBuffer(Auction)`

### prefix key for auctioneer address to find an auction id

- `AuctioneerKeyPrefix: 0x22 | AuctioneerAddrLen (1 byte) | AuctioneerAddr -> uint64`

### prefix key for the auction id with the sequence number to find the bid

- `SequenceKeyPrefix: 0x31 | Sequence -> ProtocolBuffer(Bid)`

### prefix key for auction id with bidder address to find the sequence number

- `BidKeyPrefix: 0x32 | AuctionId | BidderAddrLen (1 byte) | BidderAddr -> uint64`

### prefix key for bidder address to find the sequence number

- `BidderKeyPrefix: 0x33 | BidderAddrLen (1 byte) | BidderAddr -> uint64`