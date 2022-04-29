<!-- order: 8 -->

# Parameters

The `fundraising` module contains the following parameters:

| Key                        | Type      | Example                                        |
| -------------------------- | --------- | ---------------------------------------------- |
| AuctionCreationFee         | sdk.Coins | [{"denom":"stake","amount":"100000000"}]       |
| PlaceBidFee                | sdk.Coins | [{"denom":"stake","amount":"0"}]               |
| ExtendedPeriod             | uint32    | 3600 * 24                                      |
| | |

## AuctionCreationFee

`AuctionCreationFee` is the fee required to pay to create an auction. This fee prevents from spamming attack.

## PlaceBidFee

`PlaceBidFee` is the fee required to pay when placing a bid for an auction. This fee prevents from spamming attack.

## ExtendedPeriod

`ExtendedPeriod` is the extended period that determines how long the extended auction round is.
