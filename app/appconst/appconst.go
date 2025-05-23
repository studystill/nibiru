// Copyright (c) 2023-2024 Nibi, Inc.
package appconst

import (
	"fmt"
	"math/big"
	"runtime"

	db "github.com/cometbft/cometbft-db"
)

const (
	BinaryName = "nibiru"
	BondDenom  = "unibi"
	// AccountAddressPrefix: Bech32 prefix for Nibiru accounts.
	AccountAddressPrefix = "nibi"
)

var (
	DefaultDBBackend     db.BackendType = db.PebbleDBBackend
	HavePebbleDBBuildTag bool
)

// Runtime version vars
var (
	AppVersion = ""
	GitCommit  = ""
	BuildDate  = ""

	GoVersion = ""
	GoArch    = ""
)

func init() {
	if len(AppVersion) == 0 {
		AppVersion = "dev"
	}

	GoVersion = runtime.Version()
	GoArch = runtime.GOARCH
}

func RuntimeVersion() string {
	return fmt.Sprintf(
		"Version %s (%s)\nCompiled at %s using Go %s (%s)",
		AppVersion,
		GitCommit,
		BuildDate,
		GoVersion,
		GoArch,
	)
}

// EIP 155 Chain IDs exported for tests.
const (
	ETH_CHAIN_ID_MAINNET int64 = 6900

	ETH_CHAIN_ID_TESTNET_1 int64 = 7210
	ETH_CHAIN_ID_TESTNET_2 int64 = 6911
	ETH_CHAIN_ID_TESTNET_3 int64 = 6912

	ETH_CHAIN_ID_DEVNET_1 int64 = 6920
	ETH_CHAIN_ID_DEVNET_2 int64 = 6921
	ETH_CHAIN_ID_DEVNET_3 int64 = 6922

	ETH_CHAIN_ID_LOCALNET_0 int64 = 6930
	ETH_CHAIN_ID_LOCALNET_1 int64 = 6931
	ETH_CHAIN_ID_LOCALNET_2 int64 = 6932
	ETH_CHAIN_ID_LOCALNET_3 int64 = 6933

	ETH_CHAIN_ID_DEFAULT int64 = 6930
)

var knownEthChainIDMap = map[string]int64{
	"cataclysm-1": ETH_CHAIN_ID_MAINNET,

	"nibiru-testnet-1": ETH_CHAIN_ID_TESTNET_1,
	"nibiru-testnet-2": ETH_CHAIN_ID_TESTNET_2,
	"nibiru-testnet-3": ETH_CHAIN_ID_TESTNET_3,

	"nibiru-devnet-1": ETH_CHAIN_ID_DEVNET_1,
	"nibiru-devnet-2": ETH_CHAIN_ID_DEVNET_2,
	"nibiru-devnet-3": ETH_CHAIN_ID_DEVNET_3,

	"nibiru-localnet-0": ETH_CHAIN_ID_LOCALNET_0,
	"nibiru-localnet-1": ETH_CHAIN_ID_LOCALNET_1,
	"nibiru-localnet-2": ETH_CHAIN_ID_LOCALNET_2,
	"nibiru-localnet-3": ETH_CHAIN_ID_LOCALNET_3,
}

// GetEthChainID: Maps the given chain ID from the block's `sdk.Context` to an
// EVM Chain ID (`*big.Int`).
func GetEthChainID(ctxChainID string) (ethChainID *big.Int) {
	ethChainIdInt, found := knownEthChainIDMap[ctxChainID]
	if !found {
		ethChainID = big.NewInt(ETH_CHAIN_ID_DEFAULT)
	} else {
		ethChainID = big.NewInt(ethChainIdInt)
	}
	return ethChainID
}
