package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

const pairABI = `[
	{
		"constant": true,
		"inputs": [],
		"name": "getReserves",
		"outputs": [
			{"name": "reserve0", "type": "uint112"},
			{"name": "reserve1", "type": "uint112"},
			{"name": "blockTimestampLast", "type": "uint32"}
		],
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [],
		"name": "token0",
		"outputs": [{"name": "", "type": "address"}],
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [],
		"name": "token1",
		"outputs": [{"name": "", "type": "address"}],
		"type": "function"
	}
]`

type EthereumClient struct {
	client *ethclient.Client
	abi    abi.ABI
}

type PoolReserves struct {
	Reserve0 *big.Int
	Reserve1 *big.Int
}

type SwapEstimator struct {
	ethClient *EthereumClient
}

type EstimateResponse struct {
	DstAmount string `json:"dst_amount"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func NewEthereumClient(nodeURL string) (*EthereumClient, error) {
	client, err := ethclient.Dial(nodeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum node: %w", err)
	}

	parsedABI, err := abi.JSON(strings.NewReader(pairABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	return &EthereumClient{
		client: client,
		abi:    parsedABI,
	}, nil
}

func (ec *EthereumClient) GetReserves(ctx context.Context, pairAddr common.Address) (*PoolReserves, error) {

	data, err := ec.abi.Pack("getReserves")
	if err != nil {
		return nil, fmt.Errorf("failed to pack getReserves call: %w", err)
	}

	result, err := ec.client.CallContract(ctx, ethereum.CallMsg{
		To:   &pairAddr,
		Data: data,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call getReserves: %w", err)
	}

	// Unpack the result
	unpacked, err := ec.abi.Unpack("getReserves", result)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack getReserves result: %w", err)
	}

	if len(unpacked) < 2 {
		return nil, fmt.Errorf("unexpected getReserves result length")
	}

	reserve0, ok := unpacked[0].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("failed to cast reserve0 to *big.Int")
	}

	reserve1, ok := unpacked[1].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("failed to cast reserve1 to *big.Int")
	}

	return &PoolReserves{
		Reserve0: reserve0,
		Reserve1: reserve1,
	}, nil
}

func (ec *EthereumClient) GetToken0(ctx context.Context, pairAddr common.Address) (common.Address, error) {
	data, err := ec.abi.Pack("token0")
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to pack token0 call: %w", err)
	}

	result, err := ec.client.CallContract(ctx, ethereum.CallMsg{
		To:   &pairAddr,
		Data: data,
	}, nil)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to call token0: %w", err)
	}

	unpacked, err := ec.abi.Unpack("token0", result)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to unpack token0 result: %w", err)
	}

	if len(unpacked) == 0 {
		return common.Address{}, fmt.Errorf("empty token0 result")
	}

	token0Addr, ok := unpacked[0].(common.Address)
	if !ok {
		return common.Address{}, fmt.Errorf("failed to cast token0 to common.Address")
	}

	return token0Addr, nil
}

func NewSwapEstimator(ethClient *EthereumClient) *SwapEstimator {
	return &SwapEstimator{
		ethClient: ethClient,
	}
}

func (se *SwapEstimator) EstimateSwap(ctx context.Context, poolAddr, srcToken, dstToken common.Address, srcAmount *big.Int) (*big.Int, error) {

	reserves, err := se.ethClient.GetReserves(ctx, poolAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to get reserves: %w", err)
	}

	token0, err := se.ethClient.GetToken0(ctx, poolAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to get token0: %w", err)
	}

	var reserveIn, reserveOut *big.Int

	if token0 == srcToken {
		reserveIn = reserves.Reserve0
		reserveOut = reserves.Reserve1
	} else if token0 == dstToken {
		reserveIn = reserves.Reserve1
		reserveOut = reserves.Reserve0
	} else {
		return nil, fmt.Errorf("token addresses don't match pool tokens")
	}

	return calculateSwapAmount(srcAmount, reserveIn, reserveOut), nil
}

func calculateSwapAmount(amountIn, reserveIn, reserveOut *big.Int) *big.Int {

	amountInWithFee := new(big.Int).Mul(amountIn, big.NewInt(997))
	numerator := new(big.Int).Mul(amountInWithFee, reserveOut)

	denominator := new(big.Int).Mul(reserveIn, big.NewInt(1000))
	denominator.Add(denominator, amountInWithFee)

	amountOut := new(big.Int).Div(numerator, denominator)
	return amountOut
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}


func (se *SwapEstimator) estimateHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	poolStr := r.URL.Query().Get("pool")
	srcStr := r.URL.Query().Get("src")
	dstStr := r.URL.Query().Get("dst")
	srcAmountStr := r.URL.Query().Get("src_amount")

	if poolStr == "" || srcStr == "" || dstStr == "" || srcAmountStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Missing required parameters: pool, src, dst, src_amount"})
		return
	}

	poolAddr := common.HexToAddress(poolStr)
	srcAddr := common.HexToAddress(srcStr)
	dstAddr := common.HexToAddress(dstStr)

	srcAmount, ok := new(big.Int).SetString(srcAmountStr, 10)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid src_amount format"})
		return
	}

	dstAmount, err := se.EstimateSwap(r.Context(), poolAddr, srcAddr, dstAddr, srcAmount)
	if err != nil {
		log.Printf("Error estimating swap: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to estimate swap"})
		return
	}

	response := EstimateResponse{
		DstAmount: dstAmount.String(),
	}
	json.NewEncoder(w).Encode(response)
}

func main() {

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	nodeURL := os.Getenv("ETH_NODE_URL")
	if nodeURL == "" {
		log.Fatal("ETH_NODE_URL environment variable is required")
	}

	ethClient, err := NewEthereumClient(nodeURL)
	if err != nil {
		log.Fatal("Failed to create Ethereum client:", err)
	}

	estimator := NewSwapEstimator(ethClient)

	r := mux.NewRouter()
	r.HandleFunc("/health", healthHandler).Methods("GET")
    r.HandleFunc("/estimate", estimator.estimateHandler).Methods("GET")

	port := os.Getenv("PORT")
	if port == "" {
		port = "1337"
	}

	log.Printf("Starting server on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
