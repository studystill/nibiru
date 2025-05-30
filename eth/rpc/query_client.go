// Copyright (c) 2023-2024 Nibi, Inc.
package rpc

import (
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/types/tx"
	gethcommon "github.com/ethereum/go-ethereum/common"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/proto/tendermint/crypto"

	"github.com/cosmos/cosmos-sdk/client"

	"github.com/NibiruChain/nibiru/v2/x/evm"
)

// QueryClient defines a gRPC Client used for:
//   - TM transaction simulation
//   - EVM module queries
type QueryClient struct {
	tx.ServiceClient
	evm.QueryClient
}

// NewQueryClient creates a new gRPC query client
//
// TODO:🔗 https://github.com/NibiruChain/nibiru/issues/1857
// test(eth): Test GetProof (rpc/types/query_client.go) in a similar manner to
// cosmos-sdk/client/rpc/rpc_test.go using a network after EVM is wired into the
// app keepers:
func NewQueryClient(clientCtx client.Context) *QueryClient {
	return &QueryClient{
		ServiceClient: tx.NewServiceClient(clientCtx),
		QueryClient:   evm.NewQueryClient(clientCtx),
	}
}

// GetProof performs an ABCI query with the given key and returns a merkle proof. The desired
// tendermint height to perform the query should be set in the client context. The query will be
// performed at one below this height (at the IAVL version) in order to obtain the correct merkle
// proof. Proof queries at height less than or equal to 2 are not supported.
// Issue: https://github.com/cosmos/cosmos-sdk/issues/6567
func (QueryClient) GetProof(
	clientCtx client.Context, storeKey string, key []byte,
) ([]byte, *crypto.ProofOps, error) {
	height := clientCtx.Height
	// ABCI queries at height less than or equal to 2 are not supported.
	// Base app does not support queries for height less than or equal to 1, and
	// the base app uses 0 indexing.
	//
	// Ethereum uses 1 indexing for the initial block height, therefore <= 2 means
	// <= (Eth) height 3.
	if height <= 2 {
		return nil, nil, fmt.Errorf(
			"proof queries at ABCI block height <= 2 are not supported")
	}

	abciReq := abci.RequestQuery{
		Path:   fmt.Sprintf("store/%s/key", storeKey),
		Data:   key,
		Height: height,
		Prove:  true,
	}

	abciRes, err := clientCtx.QueryABCI(abciReq)
	if err != nil {
		reqJsonMap := make(map[string]string)
		reqJsonMap["path"] = abciReq.Path
		reqJsonMap["storeKey"] = storeKey
		reqJsonMap["key"] = gethcommon.Bytes2Hex(abciReq.Data)
		reqJsonMap["height"] = fmt.Sprintf("%d", abciReq.Height)
		reqJsonMap["prove"] = fmt.Sprintf("%v", abciReq.Prove)
		reqJson, _ := json.Marshal(reqJsonMap)
		return nil, nil, fmt.Errorf(
			"error in ABCI query for merkle proof: request %s: %w", reqJson, err)
	}

	return abciRes.Value, abciRes.ProofOps, nil
}
