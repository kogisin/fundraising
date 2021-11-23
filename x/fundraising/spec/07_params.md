<!-- order: 8 -->

# Parameters

The `fundraising` module contains the following parameters:

| Key                        | Type      | Example             |
| -------------------------- | --------- | ------------------- |
| FundraisingCreationFee     | sdk.Coins | TBD                 |
| ExtendedPeriod             | uint32    | 3600 * 24           |

## FundraisingCreationFee

`FundraisingCreationFee` is the fee required to pay to create a fundraising. This fee prevents from spamming attack.

## ExtendedPeriod

`ExtendedPeriod` is the extended period that determines how long the extended auction round is.