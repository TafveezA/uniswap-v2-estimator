# Uniswap V2 Swap Estimator API

A REST API that estimates swap amounts for Uniswap V2 pairs by fetching real-time blockchain state and performing off-chain calculations.

## Features

- Single `/estimate` endpoint for swap calculations
- Real-time blockchain data fetching from Ethereum mainnet
- Pure Uniswap V2 math implementation with 0.3% fee calculation
- Performance-optimized big integer mathematics
- Environment variable support with `.env` file
- No external Uniswap SDK dependencies

## Requirements

- Go 1.21+
- Ethereum node endpoint Infura.

## Quick Start

### 1. Setup
```bash
mkdir uniswap-v2-estimator && cd uniswap-v2-estimator
go mod init uniswap-v2-estimator
# Copy main.go from artifacts
go mod tidy
```

### 2. Configure Environment
Create `.env` file:
```env
ETH_NODE_URL=https://mainnet.infura.io/v3/YOUR-PROJECT-ID
PORT=1337
```

Get free API key:
- **Infura**: https://infura.io/ → Create project → Copy Project ID


### 3. Run
```bash
go run main.go
```

## API Usage

### Endpoint
```
GET /estimate?pool=POOL_ADDRESS&src=SRC_TOKEN&dst=DST_TOKEN&src_amount=AMOUNT
```

### Example
```bash
curl "http://localhost:1337/estimate?pool=0x0d4a11d5eeaac28ec3f61d100daf4d40471f1852&src=0xdAC17F958D2ee523a2206206994597C13D831ec7&dst=0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2&src_amount=10000000"
```

**Response:**
```json
{"dst_amount": "6241000000000000"}
```

## Example Usage

```bash
# 10 USDT → WETH
curl "http://localhost:1337/estimate?pool=0x0d4a11d5eeaac28ec3f61d100daf4d40471f1852&src=0xdAC17F958D2ee523a2206206994597C13D831ec7&dst=0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2&src_amount=10000000"

# 1 USDT → WETH
curl "http://localhost:1337/estimate?pool=0x0d4a11d5eeaac28ec3f61d100daf4d40471f1852&src=0xdAC17F958D2ee523a2206206994597C13D831ec7&dst=0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2&src_amount=1000000"

# 0.1 WETH → USDT
curl "http://localhost:1337/estimate?pool=0x0d4a11d5eeaac28ec3f61d100daf4d40471f1852&src=0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2&dst=0xdAC17F958D2ee523a2206206994597C13D831ec7&src_amount=100000000000000000"
```

## How It Works

Uses Uniswap V2 formula:
```
amountOut = (amountIn × 997 × reserveOut) / (reserveIn × 1000 + amountIn × 997)
```

1. Fetches pool reserves via `getReserves()`
2. Determines token ordering via `token0()`
3. Applies 0.3% fee calculation (997/1000)
4. Returns estimated output amount

## Popular Pools

| Pair | Pool Address |
|------|-------------|
| **USDT/WETH** | `0x0d4a11d5eeaac28ec3f61d100daf4d40471f1852` |
| **USDC/WETH** | `0xb4e16d0168e52d35cacd2c6185b44281ec28c9dc` |
| **DAI/WETH** | `0xa478c2975ab1ea89e8196811f51a7b7ade33eb11` |

## Token Addresses

- **USDT**: `0xdAC17F958D2ee523a2206206994597C13D831ec7` (6 decimals)
- **USDC**: `0xa0b86a33e6bb3abe3bc6aaed3a7e4c98a50b9b68` (6 decimals)
- **DAI**: `0x6b175474e89094c44da98b954eedeac495271d0f` (18 decimals)
- **WETH**: `0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2` (18 decimals)

## Build Binary

```bash
go build -o uniswap-estimator main.go
./uniswap-estimator
```

## Troubleshooting

**"ETH_NODE_URL environment variable is required"**
```bash
echo "ETH_NODE_URL=https://mainnet.infura.io/v3/YOUR-PROJECT-ID" > .env
```

**Connection errors**
- Verify your API key is correct
- Check internet connection
- Try different provider (Infura/Alchemy)

---

Built with Go + Ethereum + Uniswap V2 Protocol