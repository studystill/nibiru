package keeper

import (
	"sort"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/NibiruChain/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/x/common/set"
	testutilevents "github.com/NibiruChain/nibiru/v2/x/common/testutil"

	"github.com/NibiruChain/nibiru/v2/x/common/asset"
	"github.com/NibiruChain/nibiru/v2/x/common/denoms"
	"github.com/NibiruChain/nibiru/v2/x/oracle/types"
)

func TestQueryParams(t *testing.T) {
	input := CreateTestFixture(t)
	ctx := sdk.WrapSDKContext(input.Ctx)

	querier := NewQuerier(input.OracleKeeper)
	res, err := querier.Params(ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)

	params, err := input.OracleKeeper.Params.Get(input.Ctx)
	require.NoError(t, err)

	require.Equal(t, params, res.Params)
}

func TestQueryExchangeRate(t *testing.T) {
	input := CreateTestFixture(t)
	ctx := sdk.WrapSDKContext(input.Ctx)
	querier := NewQuerier(input.OracleKeeper)

	rate := sdkmath.LegacyNewDec(1700)
	input.OracleKeeper.ExchangeRates.Insert(
		input.Ctx,
		asset.Registry.Pair(denoms.ETH, denoms.NUSD),
		types.ExchangeRateAtBlock{
			ExchangeRate:     rate,
			CreatedBlock:     uint64(input.Ctx.BlockHeight()),
			BlockTimestampMs: input.Ctx.BlockTime().UnixMilli(),
		},
	)

	// empty request
	_, err := querier.ExchangeRate(ctx, nil)
	require.Error(t, err)

	// Query to grpc
	res, err := querier.ExchangeRate(ctx, &types.QueryExchangeRateRequest{
		Pair: asset.Registry.Pair(denoms.ETH, denoms.NUSD),
	})
	require.NoError(t, err)
	require.Equal(t, rate, res.ExchangeRate)
}

func TestQueryMissCounter(t *testing.T) {
	input := CreateTestFixture(t)
	ctx := sdk.WrapSDKContext(input.Ctx)
	querier := NewQuerier(input.OracleKeeper)

	missCounter := uint64(1)
	input.OracleKeeper.MissCounters.Insert(input.Ctx, ValAddrs[0], missCounter)

	// empty request
	_, err := querier.MissCounter(ctx, nil)
	require.Error(t, err)

	// Query to grpc
	res, err := querier.MissCounter(ctx, &types.QueryMissCounterRequest{
		ValidatorAddr: ValAddrs[0].String(),
	})
	require.NoError(t, err)
	require.Equal(t, missCounter, res.MissCounter)
}

func TestQueryExchangeRates(t *testing.T) {
	input := CreateTestFixture(t)
	ctx := sdk.WrapSDKContext(input.Ctx)
	querier := NewQuerier(input.OracleKeeper)

	rate := sdkmath.LegacyNewDec(1700)
	input.OracleKeeper.ExchangeRates.Insert(
		input.Ctx,
		asset.Registry.Pair(denoms.BTC, denoms.NUSD),
		types.ExchangeRateAtBlock{
			ExchangeRate:     rate,
			CreatedBlock:     uint64(input.Ctx.BlockHeight()),
			BlockTimestampMs: input.Ctx.BlockTime().UnixMilli(),
		},
	)
	input.OracleKeeper.ExchangeRates.Insert(
		input.Ctx,
		asset.Registry.Pair(denoms.ETH, denoms.NUSD),
		types.ExchangeRateAtBlock{
			ExchangeRate:     rate,
			CreatedBlock:     uint64(input.Ctx.BlockHeight()),
			BlockTimestampMs: input.Ctx.BlockTime().UnixMilli(),
		},
	)

	res, err := querier.ExchangeRates(ctx, &types.QueryExchangeRatesRequest{})
	require.NoError(t, err)

	require.Equal(t, types.ExchangeRateTuples{
		{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD), ExchangeRate: rate},
		{Pair: asset.Registry.Pair(denoms.ETH, denoms.NUSD), ExchangeRate: rate},
	}, res.ExchangeRates)
}

func TestQueryExchangeRateTwap(t *testing.T) {
	input := CreateTestFixture(t)
	querier := NewQuerier(input.OracleKeeper)

	rate := sdkmath.LegacyNewDec(1700)
	input.OracleKeeper.SetPrice(input.Ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), rate)
	testutilevents.RequireContainsTypedEvent(
		t,
		input.Ctx,
		&types.EventPriceUpdate{
			Pair:        asset.Registry.Pair(denoms.BTC, denoms.NUSD).String(),
			Price:       rate,
			TimestampMs: input.Ctx.BlockTime().UnixMilli(),
		},
	)

	ctx := sdk.WrapSDKContext(input.Ctx.
		WithBlockTime(input.Ctx.BlockTime().Add(time.Second)).
		WithBlockHeight(input.Ctx.BlockHeight() + 1),
	)

	_, err := querier.ExchangeRateTwap(ctx, &types.QueryExchangeRateRequest{Pair: asset.Registry.Pair(denoms.ETH, denoms.NUSD)})
	require.Error(t, err)

	res, err := querier.ExchangeRateTwap(ctx, &types.QueryExchangeRateRequest{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD)})
	require.NoError(t, err)
	require.Equal(t, sdkmath.LegacyMustNewDecFromStr("1700"), res.ExchangeRate)
}

func TestQueryDatedExchangeRate(t *testing.T) {
	input := CreateTestFixture(t)
	querier := NewQuerier(input.OracleKeeper)

	initialTime := input.Ctx.BlockTime()
	initialHeight := input.Ctx.BlockHeight()
	t.Logf("Set initial block time and height\ninitialTimeSeconds: %d, intialHeight: %d",
		initialTime.Unix(),
		initialHeight,
	)

	// Pair 1: BTC/NUSD
	pairBTC := asset.Registry.Pair(denoms.BTC, denoms.NUSD)
	rateBTC1 := sdkmath.LegacyNewDec(1700)
	rateBTC2 := sdkmath.LegacyNewDec(1800)

	// Pair 2: ETH/NUSD
	pairETH := asset.Registry.Pair(denoms.ETH, denoms.NUSD)
	rateETH1 := sdkmath.LegacyNewDec(100)
	rateETH2 := sdkmath.LegacyNewDec(110)

	// --- Set first price for BTC/NUSD ---
	input.OracleKeeper.SetPrice(input.Ctx, pairBTC, rateBTC1)
	testutilevents.RequireContainsTypedEvent(
		t,
		input.Ctx,
		&types.EventPriceUpdate{
			Pair:        pairBTC.String(),
			Price:       rateBTC1,
			TimestampMs: input.Ctx.BlockTime().UnixMilli(),
		},
	)

	blockTime := initialTime.Add(1 * time.Second)
	blockHeight := initialHeight + 1
	t.Logf("Advance time and block height\nblockTimeSeconds: %d, blockHeight: %d",
		blockTime.Unix(),
		blockHeight,
	)
	input.Ctx = input.Ctx.
		WithBlockTime(blockTime).
		WithBlockHeight(blockHeight)

	t.Log("Set first price for ETH")
	input.OracleKeeper.SetPrice(input.Ctx, pairETH, rateETH1)
	testutilevents.RequireContainsTypedEvent(
		t,
		input.Ctx,
		&types.EventPriceUpdate{
			Pair:        pairETH.String(),
			Price:       rateETH1,
			TimestampMs: input.Ctx.BlockTime().UnixMilli(),
		},
	)

	blockTime = initialTime.Add(2 * time.Second)
	blockHeight = initialHeight + 2
	t.Logf("Advance time and block height\nblockTimeSeconds: %d, blockHeight: %d",
		blockTime.Unix(),
		blockHeight,
	)
	input.Ctx = input.Ctx.
		WithBlockTime(blockTime).
		WithBlockHeight(blockHeight)

	t.Log("Set second price for BTC")
	input.OracleKeeper.SetPrice(input.Ctx, pairBTC, rateBTC2)
	testutilevents.RequireContainsTypedEvent(
		t,
		input.Ctx,
		&types.EventPriceUpdate{
			Pair:        pairBTC.String(),
			Price:       rateBTC2,
			TimestampMs: input.Ctx.BlockTime().UnixMilli(),
		},
	)

	t.Log("Advance time and block height")
	input.Ctx = input.Ctx.
		WithBlockTime(initialTime.Add(3 * time.Second)).
		WithBlockHeight(initialHeight + 3)

	t.Log("Set second price for ETH")
	input.OracleKeeper.SetPrice(input.Ctx, pairETH, rateETH2)
	testutilevents.RequireContainsTypedEvent(
		t,
		input.Ctx,
		&types.EventPriceUpdate{
			Pair:        pairETH.String(),
			Price:       rateETH2,
			TimestampMs: input.Ctx.BlockTime().UnixMilli(),
		},
	)

	// Wrap context for querying
	ctx := sdk.WrapSDKContext(input.Ctx)

	t.Log("Query latest snapshot for BTC")
	resBTC, err := querier.ExchangeRate(
		ctx,
		&types.QueryExchangeRateRequest{Pair: pairBTC},
	)
	require.NoError(t, err)
	res := resBTC
	require.Equal(t, rateBTC2.String(), resBTC.ExchangeRate.String())
	require.Equal(t, initialTime.Add(2*time.Second).UnixMilli(), res.BlockTimestampMs)
	require.Equal(t, uint64(initialHeight+2), res.BlockHeight)

	t.Log("Query latest snapshot for ETH")
	resETH, err := querier.ExchangeRate(
		ctx,
		&types.QueryExchangeRateRequest{Pair: pairETH},
	)
	require.NoError(t, err)
	res = resETH
	require.Equal(t, rateETH2, res.ExchangeRate)
	require.Equal(t, initialTime.Add(3*time.Second).UnixMilli(), res.BlockTimestampMs)
	require.Equal(t, uint64(initialHeight+3), res.BlockHeight)

	t.Run("Query a pair with no snapshots (should return an error)", func(t *testing.T) {
		pairATOM := asset.Registry.Pair(denoms.ATOM, denoms.NUSD)
		_, err = querier.ExchangeRate(ctx, &types.QueryExchangeRateRequest{Pair: pairATOM})
		require.Error(t, err)
	})
}

func TestCalcTwap(t *testing.T) {
	tests := []struct {
		name               string
		pair               asset.Pair
		priceSnapshots     []types.PriceSnapshot
		currentBlockTime   time.Time
		currentBlockHeight int64
		lookbackInterval   time.Duration
		assetAmount        sdkmath.LegacyDec
		expectedPrice      sdkmath.LegacyDec
		expectedErr        error
	}{
		// expected price: (9.5 * (35 - 30) + 8.5 * (30 - 20) + 9.0 * (20 - 5)) / 30 = 8.916666
		{
			name: "spot price twap calc, t=(5,35]",
			pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			priceSnapshots: []types.PriceSnapshot{
				{
					Pair:        asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					Price:       sdkmath.LegacyMustNewDecFromStr("90000.0"),
					TimestampMs: time.UnixMilli(1).UnixMilli(),
				},
				{
					Pair:        asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					Price:       sdkmath.LegacyMustNewDecFromStr("9.0"),
					TimestampMs: time.UnixMilli(10).UnixMilli(),
				},
				{
					Pair:        asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					Price:       sdkmath.LegacyMustNewDecFromStr("8.5"),
					TimestampMs: time.UnixMilli(20).UnixMilli(),
				},
				{
					Pair:        asset.Registry.Pair(denoms.BTC, denoms.NUSD),
					Price:       sdkmath.LegacyMustNewDecFromStr("9.5"),
					TimestampMs: time.UnixMilli(30).UnixMilli(),
				},
			},
			currentBlockTime:   time.UnixMilli(35),
			currentBlockHeight: 3,
			lookbackInterval:   30 * time.Millisecond,
			expectedPrice:      sdkmath.LegacyMustNewDecFromStr("8.900000000000000000"),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			input := CreateTestFixture(t)
			querier := NewQuerier(input.OracleKeeper)
			ctx := input.Ctx

			newParams := types.Params{
				VotePeriod:         types.DefaultVotePeriod,
				VoteThreshold:      types.DefaultVoteThreshold,
				MinVoters:          types.DefaultMinVoters,
				RewardBand:         types.DefaultRewardBand,
				Whitelist:          types.DefaultWhitelist,
				SlashFraction:      types.DefaultSlashFraction,
				SlashWindow:        types.DefaultSlashWindow,
				MinValidPerWindow:  types.DefaultMinValidPerWindow,
				TwapLookbackWindow: tc.lookbackInterval,
				ValidatorFeeRatio:  types.DefaultValidatorFeeRatio,
			}

			input.OracleKeeper.Params.Set(ctx, newParams)
			ctx = ctx.WithBlockTime(time.UnixMilli(0))
			for _, reserve := range tc.priceSnapshots {
				ctx = ctx.WithBlockTime(time.UnixMilli(reserve.TimestampMs))
				input.OracleKeeper.SetPrice(ctx, asset.Registry.Pair(denoms.BTC, denoms.NUSD), reserve.Price)
			}

			ctx = ctx.WithBlockTime(tc.currentBlockTime).WithBlockHeight(tc.currentBlockHeight)

			price, err := querier.ExchangeRateTwap(sdk.WrapSDKContext(ctx), &types.QueryExchangeRateRequest{Pair: asset.Registry.Pair(denoms.BTC, denoms.NUSD)})
			require.NoError(t, err)

			require.EqualValuesf(t, tc.expectedPrice, price.ExchangeRate,
				"expected %s, got %s", tc.expectedPrice.String(), price.ExchangeRate.String())
		})
	}
}

func TestQueryActives(t *testing.T) {
	input := CreateTestFixture(t)
	ctx := sdk.WrapSDKContext(input.Ctx)
	queryClient := NewQuerier(input.OracleKeeper)

	rate := sdkmath.LegacyNewDec(1700)
	targetPairs := set.New[asset.Pair]()
	for _, bankDenom := range []string{denoms.BTC, denoms.ETH, denoms.NIBI} {
		pair := asset.Registry.Pair(bankDenom, denoms.NUSD)
		targetPairs.Add(pair)
		input.OracleKeeper.ExchangeRates.Insert(
			input.Ctx,
			pair,
			types.ExchangeRateAtBlock{
				ExchangeRate:     rate,
				CreatedBlock:     uint64(input.Ctx.BlockHeight()),
				BlockTimestampMs: input.Ctx.BlockTime().UnixMilli(),
			},
		)
	}

	res, err := queryClient.Actives(ctx, &types.QueryActivesRequest{})
	require.NoError(t, err)
	for _, pair := range res.Actives {
		require.True(t, targetPairs.Has(pair))
	}
}

func TestQueryFeederDelegation(t *testing.T) {
	input := CreateTestFixture(t)
	ctx := sdk.WrapSDKContext(input.Ctx)
	querier := NewQuerier(input.OracleKeeper)

	input.OracleKeeper.FeederDelegations.Insert(input.Ctx, ValAddrs[0], Addrs[1])

	// empty request
	_, err := querier.FeederDelegation(ctx, nil)
	require.Error(t, err)

	res, err := querier.FeederDelegation(ctx, &types.QueryFeederDelegationRequest{
		ValidatorAddr: ValAddrs[0].String(),
	})
	require.NoError(t, err)

	require.Equal(t, Addrs[1].String(), res.FeederAddr)
}

func TestQueryAggregatePrevote(t *testing.T) {
	input := CreateTestFixture(t)
	ctx := sdk.WrapSDKContext(input.Ctx)
	querier := NewQuerier(input.OracleKeeper)

	prevote1 := types.NewAggregateExchangeRatePrevote(types.AggregateVoteHash{}, ValAddrs[0], 0)
	input.OracleKeeper.Prevotes.Insert(input.Ctx, ValAddrs[0], prevote1)
	prevote2 := types.NewAggregateExchangeRatePrevote(types.AggregateVoteHash{}, ValAddrs[1], 0)
	input.OracleKeeper.Prevotes.Insert(input.Ctx, ValAddrs[1], prevote2)

	// validator 0 address params
	res, err := querier.AggregatePrevote(ctx, &types.QueryAggregatePrevoteRequest{
		ValidatorAddr: ValAddrs[0].String(),
	})
	require.NoError(t, err)
	require.Equal(t, prevote1, res.AggregatePrevote)

	// empty request
	_, err = querier.AggregatePrevote(ctx, nil)
	require.Error(t, err)

	// validator 1 address params
	res, err = querier.AggregatePrevote(ctx, &types.QueryAggregatePrevoteRequest{
		ValidatorAddr: ValAddrs[1].String(),
	})
	require.NoError(t, err)
	require.Equal(t, prevote2, res.AggregatePrevote)
}

func TestQueryAggregatePrevotes(t *testing.T) {
	input := CreateTestFixture(t)
	ctx := sdk.WrapSDKContext(input.Ctx)
	querier := NewQuerier(input.OracleKeeper)

	prevote1 := types.NewAggregateExchangeRatePrevote(types.AggregateVoteHash{}, ValAddrs[0], 0)
	input.OracleKeeper.Prevotes.Insert(input.Ctx, ValAddrs[0], prevote1)
	prevote2 := types.NewAggregateExchangeRatePrevote(types.AggregateVoteHash{}, ValAddrs[1], 0)
	input.OracleKeeper.Prevotes.Insert(input.Ctx, ValAddrs[1], prevote2)
	prevote3 := types.NewAggregateExchangeRatePrevote(types.AggregateVoteHash{}, ValAddrs[2], 0)
	input.OracleKeeper.Prevotes.Insert(input.Ctx, ValAddrs[2], prevote3)

	expectedPrevotes := []types.AggregateExchangeRatePrevote{prevote1, prevote2, prevote3}
	sort.SliceStable(expectedPrevotes, func(i, j int) bool {
		return expectedPrevotes[i].Voter <= expectedPrevotes[j].Voter
	})

	res, err := querier.AggregatePrevotes(ctx, &types.QueryAggregatePrevotesRequest{})
	require.NoError(t, err)
	require.Equal(t, expectedPrevotes, res.AggregatePrevotes)
}

func TestQueryAggregateVote(t *testing.T) {
	input := CreateTestFixture(t)
	ctx := sdk.WrapSDKContext(input.Ctx)
	querier := NewQuerier(input.OracleKeeper)

	vote1 := types.NewAggregateExchangeRateVote(types.ExchangeRateTuples{{Pair: "", ExchangeRate: sdkmath.LegacyOneDec()}}, ValAddrs[0])
	input.OracleKeeper.Votes.Insert(input.Ctx, ValAddrs[0], vote1)
	vote2 := types.NewAggregateExchangeRateVote(types.ExchangeRateTuples{{Pair: "", ExchangeRate: sdkmath.LegacyOneDec()}}, ValAddrs[1])
	input.OracleKeeper.Votes.Insert(input.Ctx, ValAddrs[1], vote2)

	// empty request
	_, err := querier.AggregateVote(ctx, nil)
	require.Error(t, err)

	// validator 0 address params
	res, err := querier.AggregateVote(ctx, &types.QueryAggregateVoteRequest{
		ValidatorAddr: ValAddrs[0].String(),
	})
	require.NoError(t, err)
	require.Equal(t, vote1, res.AggregateVote)

	// validator 1 address params
	res, err = querier.AggregateVote(ctx, &types.QueryAggregateVoteRequest{
		ValidatorAddr: ValAddrs[1].String(),
	})
	require.NoError(t, err)
	require.Equal(t, vote2, res.AggregateVote)
}

func TestQueryAggregateVotes(t *testing.T) {
	input := CreateTestFixture(t)
	ctx := sdk.WrapSDKContext(input.Ctx)
	querier := NewQuerier(input.OracleKeeper)

	vote1 := types.NewAggregateExchangeRateVote(types.ExchangeRateTuples{{Pair: "", ExchangeRate: sdkmath.LegacyOneDec()}}, ValAddrs[0])
	input.OracleKeeper.Votes.Insert(input.Ctx, ValAddrs[0], vote1)
	vote2 := types.NewAggregateExchangeRateVote(types.ExchangeRateTuples{{Pair: "", ExchangeRate: sdkmath.LegacyOneDec()}}, ValAddrs[1])
	input.OracleKeeper.Votes.Insert(input.Ctx, ValAddrs[1], vote2)
	vote3 := types.NewAggregateExchangeRateVote(types.ExchangeRateTuples{{Pair: "", ExchangeRate: sdkmath.LegacyOneDec()}}, ValAddrs[2])
	input.OracleKeeper.Votes.Insert(input.Ctx, ValAddrs[2], vote3)

	expectedVotes := []types.AggregateExchangeRateVote{vote1, vote2, vote3}
	sort.SliceStable(expectedVotes, func(i, j int) bool {
		return expectedVotes[i].Voter <= expectedVotes[j].Voter
	})

	res, err := querier.AggregateVotes(ctx, &types.QueryAggregateVotesRequest{})
	require.NoError(t, err)
	require.Equal(t, expectedVotes, res.AggregateVotes)
}

func TestQueryVoteTargets(t *testing.T) {
	input := CreateTestFixture(t)
	ctx := sdk.WrapSDKContext(input.Ctx)
	querier := NewQuerier(input.OracleKeeper)

	// clear pairs
	for _, p := range input.OracleKeeper.WhitelistedPairs.Iterate(input.Ctx, collections.Range[asset.Pair]{}).Keys() {
		input.OracleKeeper.WhitelistedPairs.Delete(input.Ctx, p)
	}

	voteTargets := []asset.Pair{"denom1:denom2", "denom3:denom4", "denom5:denom6"}
	for _, target := range voteTargets {
		input.OracleKeeper.WhitelistedPairs.Insert(input.Ctx, target)
	}

	res, err := querier.VoteTargets(ctx, &types.QueryVoteTargetsRequest{})
	require.NoError(t, err)
	require.Equal(t, voteTargets, res.VoteTargets)
}
