package main

import (
    "errors"
    "github.com/algorand/go-algorand-sdk/types"
)

type TradeSide uint8;
const (
    Buy TradeSide = 0
    Sell TradeSide = 1
)

func parseTradeSide(s string) (side TradeSide, err error) {
    switch s {
        case "Buy": return 0, nil
        case "Sell": return 1, nil
        default: return 127, errors.New("Invalid side")
    }
}
func (s TradeSide) String() string {
    switch s {
        case 0: return "Buy"
        case 1: return "Sell"
        default: return ""
    }
}

// By Index:
// 0. Signed by MM
// 1. Fee transaction
// 2. Signed by client
type AlgorandQuote [3]types.SignedTxn

type LimitOrder struct {
    Quantity uint64
    Price float64
}
type Asset struct {
    AssetId string
    Ticker string
    Chain string
    Decimals uint8
}

type TradingPair struct {
    DisplayName string
    BaseAsset Asset
    QuoteAsset Asset
    Precision uint8
    Orderbook OrderBook `json:"-"`
}
type OrderBook struct {
    Bid [16]LimitOrder
    Ask [16]LimitOrder
}
var SUPPORTED_ASSETS []Asset = []Asset{ Asset{"algorand", "ALGO", "algorand", 6 },
                                        Asset{"algorand_31566704", "USDC", "algorand", 6},
                                        Asset{"algorand_312769", "USDT", "algorand", 6},
                                        Asset{"algorand_112866019", "BRZ", "algorand", 4},
                                        Asset{"algorand_283867985", "VIXI", "algorand", 6} }

var SUPPORTED_PAIRS []TradingPair = []TradingPair{ TradingPair{"ALGO-USDC", SUPPORTED_ASSETS[0], SUPPORTED_ASSETS[1], 4, OrderBook{} },
                                                   TradingPair{"ALGO-USDT", SUPPORTED_ASSETS[0], SUPPORTED_ASSETS[2], 4, OrderBook{} },
                                                   TradingPair{"USDC-BRZ", SUPPORTED_ASSETS[1], SUPPORTED_ASSETS[3], 4, OrderBook{} },
                                                   TradingPair{"USDT-BRZ", SUPPORTED_ASSETS[2], SUPPORTED_ASSETS[3], 4, OrderBook{} } }

// API Request Args
type SetBidAskArgs struct {
    Side TradeSide
    Position uint16
    BaseAssetId string
    QuoteAssetId string
    Quantity uint64
    Price float64
}

// API responses
type GetOrderBookResponse struct {
    Bids []LimitOrder
    Asks []LimitOrder
}

// Config Variables
type ConfigVars struct {
    ServerUrl string `yaml:"server_url"`
    AlgodUrl string `yaml:"algod_url"`
    AlgodToken string `yaml:"algod_token"`
    MMAddress string `yaml:"mm_address"`
    ClientAddress string `yaml:"client_address"`
    Datadir string `yaml:"datadir"`
}
var config ConfigVars

const ALGORAND_FEE_ADDRESS string = "LJ3G2M3UELFQJSHINTIRLLN3PCB6MXBCGRLQRVHYPWLDVAMJT4R4UK63JE"
const PROTOCOL_FEE_BIPS uint8 = 3
const BOOTSTRAP_MM_URL string = "http://18.231.186.3:8082"
