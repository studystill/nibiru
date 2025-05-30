package precompile

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/NibiruChain/nibiru/v2/app/keepers"
	"github.com/NibiruChain/nibiru/v2/x/common/asset"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	oraclekeeper "github.com/NibiruChain/nibiru/v2/x/oracle/keeper"
)

var _ vm.PrecompiledContract = (*precompileOracle)(nil)

// Precompile address for "Oracle.sol", the contract that enables queries for exchange rates
var PrecompileAddr_Oracle = gethcommon.HexToAddress("0x0000000000000000000000000000000000000801")

func (p precompileOracle) Address() gethcommon.Address {
	return PrecompileAddr_Oracle
}

func (p precompileOracle) RequiredGas(input []byte) (gasPrice uint64) {
	return requiredGas(input, p.ABI())
}

func (p precompileOracle) ABI() *gethabi.ABI {
	return embeds.SmartContract_Oracle.ABI
}

const (
	OracleMethod_queryExchangeRate        PrecompileMethod = "queryExchangeRate"
	OracleMethod_chainLinkLatestRoundData PrecompileMethod = "chainLinkLatestRoundData"
)

// Run runs the precompiled contract
func (p precompileOracle) Run(
	evm *vm.EVM,
	trueCaller gethcommon.Address,
	// Note that we use "trueCaller" here to differentiate between a delegate
	// caller ("parent.CallerAddress" in geth) and "contract.CallerAddress"
	// because these two addresses may differ.
	contract *vm.Contract,
	readonly bool,
	// isDelegatedCall: Flag to add conditional logic specific to delegate calls
	isDelegatedCall bool,
) (bz []byte, err error) {
	defer func() {
		err = ErrPrecompileRun(err, p)
	}()
	startResult, err := OnRunStart(evm, contract.Input, p.ABI(), contract.Gas)
	if err != nil {
		return nil, err
	}
	method, args, ctx := startResult.Method, startResult.Args, startResult.CacheCtx

	switch PrecompileMethod(method.Name) {
	case OracleMethod_queryExchangeRate:
		bz, err = p.queryExchangeRate(ctx, method, args)
	// For "@chainlink/contracts/src/v0.8/shared/interfaces/AggregatorV3Interface.sol"
	case OracleMethod_chainLinkLatestRoundData:
		bz, err = p.chainLinkLatestRoundData(ctx, method, args)

	default:
		// Note that this code path should be impossible to reach since
		// "[decomposeInput]" parses methods directly from the ABI.
		err = fmt.Errorf("invalid method called with name \"%s\"", method.Name)
		return
	}
	contract.UseGas(
		startResult.CacheCtx.GasMeter().GasConsumed(),
		evm.Config.Tracer,
		tracing.GasChangeCallPrecompiledContract,
	)
	if err != nil {
		return nil, err
	}

	return bz, err
}

func PrecompileOracle(keepers keepers.PublicKeepers) NibiruCustomPrecompile {
	return precompileOracle{
		oracleKeeper: keepers.OracleKeeper,
	}
}

type precompileOracle struct {
	oracleKeeper oraclekeeper.Keeper
}

// Implements "IOracle.queryExchangeRate"
//
//	```solidity
//	function queryExchangeRate(
//	    string memory pair
//	)
//	    external
//	    view
//	    returns (uint256 price, uint64 blockTimeMs, uint64 blockHeight);
//	```
func (p precompileOracle) queryExchangeRate(
	ctx sdk.Context,
	method *gethabi.Method,
	args []any,
) (bz []byte, err error) {
	pair, err := p.parseQueryExchangeRateArgs(args)
	if err != nil {
		return nil, err
	}
	assetPair, err := asset.TryNewPair(pair)
	if err != nil {
		return nil, err
	}

	priceAtBlock, err := p.oracleKeeper.ExchangeRates.Get(ctx, assetPair)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(
		priceAtBlock.ExchangeRate.BigInt(),
		uint64(priceAtBlock.BlockTimestampMs),
		priceAtBlock.CreatedBlock,
	)
}

func (p precompileOracle) parseQueryExchangeRateArgs(args []any) (
	pair string,
	err error,
) {
	if e := assertNumArgs(args, 1); e != nil {
		err = e
		return
	}

	pair, ok := args[0].(string)
	if !ok {
		err = ErrArgTypeValidation("string pair", args[0])
		return
	}

	return pair, nil
}

// Implements "IOracle.chainLinkLatestRoundData"
//
//	```solidity
//	interface IOracle {
//	  function chainLinkLatestRoundData(
//	    string memory pair
//	  )
//	      external
//	      view
//	      returns (
//	          uint80 roundId,
//	          int256 answer,
//	          uint256 startedAt,
//	          uint256 updatedAt,
//	          uint80 answeredInRound
//	      );
//	  // ...
//	}
//	```
func (p precompileOracle) chainLinkLatestRoundData(
	ctx sdk.Context,
	method *gethabi.Method,
	args []any,
) (bz []byte, err error) {
	pair, err := p.parseQueryExchangeRateArgs(args)
	if err != nil {
		return nil, err
	}
	assetPair, err := asset.TryNewPair(pair)
	if err != nil {
		return nil, err
	}

	priceAtBlock, err := p.oracleKeeper.ExchangeRates.Get(ctx, assetPair)
	if err != nil {
		return nil, err
	}

	roundId := new(big.Int).SetUint64(priceAtBlock.CreatedBlock)
	answer := priceAtBlock.ExchangeRate.BigInt() // 18 decimals
	timestampSeconds := big.NewInt(priceAtBlock.BlockTimestampMs / 1000)
	answeredInRound := big.NewInt(420) // for no reason in particular / unused
	return method.Outputs.Pack(
		roundId,
		answer,
		timestampSeconds, // startedAt (seconds)
		timestampSeconds, // updatedAt (seconds)
		answeredInRound,
	)
}
