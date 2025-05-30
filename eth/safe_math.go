package eth

import (
	fmt "fmt"
	math "math"
	"math/big"

	sdkioerrors "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const maxBitLen = 256

// SafeNewIntFromBigInt constructs Int from big.Int, return error if more than 256bits
func SafeNewIntFromBigInt(i *big.Int) (sdkmath.Int, error) {
	if !IsValidInt256(i) {
		return sdkmath.NewInt(0), fmt.Errorf("big int out of bound: %s", i)
	} else if i == nil {
		return sdkmath.Int{}, fmt.Errorf("received nil pointer for *big.Int")
	}
	return sdkmath.NewIntFromBigInt(i), nil
}

// IsValidInt256 check the bound of 256 bit number
func IsValidInt256(i *big.Int) bool {
	return i == nil || i.BitLen() <= maxBitLen
}

// SafeInt64 checks for overflows while casting a uint64 to int64 value.
func SafeInt64(value uint64) (int64, error) {
	if value > uint64(math.MaxInt64) {
		return 0, sdkioerrors.Wrapf(sdkerrors.ErrInvalidHeight, "uint64 value %v cannot exceed %v", value, int64(math.MaxInt64))
	}

	return int64(value), nil // #nosec G701 -- checked for int overflow already
}
