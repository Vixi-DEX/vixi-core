package main

import (
    "math"
    "fmt"
    "net/http"
    "github.com/algorand/go-algorand-sdk/client/v2/algod"
    "github.com/algorand/go-algorand-sdk/future"
    "github.com/algorand/go-algorand-sdk/types"
    //"github.com/algorand/go-algorand-sdk/transaction"
    "github.com/algorand/go-algorand-sdk/crypto"
    "github.com/algorand/go-algorand-sdk/encoding/json"
    "github.com/algorand/go-algorand-sdk/encoding/msgpack"
    "github.com/gorilla/mux"
    "strconv"
    "context"
    "errors"
    "log"
    "strings"
    stdjson "encoding/json"
)

var algodClient *algod.Client

func GetAlgorandQuote(baseAssetId string, quoteAssetId string, side TradeSide, quantity uint64, toAddress string) (quote AlgorandQuote, err error) {
    pair, err := getTradingPair(baseAssetId, quoteAssetId)
    if err != nil {
        log.Print(err)
        return quote, err
    }

    quoteQuantity, _, err := calculateQuoteQuantity(quantity, *pair, side)
    if err != nil {
        log.Print(err)
        return quote, err
    }

    var txn types.Transaction
    note := []byte("Algodex Swap")
    if side == Buy {
        txn, err = createPaymentTxn(baseAssetId, config.MMAddress, toAddress, quantity, note)
    } else {
        txn, err = createPaymentTxn(quoteAssetId, config.MMAddress, toAddress, quoteQuantity, note)
    }
    if err != nil {
        log.Print(err)
        return quote, errors.New("Could not create payment transaction")
    }

    var txn2 types.Transaction
    if side == Buy {
        txn2, err = createPaymentTxn(quoteAssetId, toAddress, config.MMAddress, quoteQuantity, note)
    } else {
        txn2, err = createPaymentTxn(baseAssetId, toAddress, config.MMAddress, quantity, note)
    }
    if err != nil {
        log.Print(err)
        return quote, errors.New("Could not create asset transfer transaction")
    }

    feeQuantity := quantity * uint64(PROTOCOL_FEE_BIPS) / 10000
    if feeQuantity == 0 {
        feeQuantity = 1
    }
    feeNote := []byte("Algodex Fee")
    feeTxn, err := createPaymentTxn(baseAssetId, config.MMAddress, ALGORAND_FEE_ADDRESS, feeQuantity, feeNote)
    if err != nil {
        log.Print(err)
        return quote, errors.New("Could not create fee transaction")
    }

    txns := []types.Transaction{txn,txn2,feeTxn}
    gid, err := crypto.ComputeGroupID(txns)
    if err != nil {
        log.Print(err)
        return quote, errors.New("Could not group transactions")
    }
    txns[0].Group = gid
    txns[1].Group = gid
    txns[2].Group = gid

    mmAccount, err := GetWalletAccount(config.MMAddress)
    _, stx0, err := crypto.SignTransaction(mmAccount.PrivateKey, txns[0])
    if err != nil {
        log.Print(err)
        return quote, errors.New("Could not sign transaction")
    }
    err = msgpack.Decode(stx0, &quote[0])
    if err != nil {
        log.Print(err)
        return quote, errors.New("Could not decode signed transaction")
    }

    authAddr, err := types.DecodeAddress(toAddress)
    if err != nil {
        log.Print(err)
        return quote, errors.New("Invalid toAddress")
    }
    quote[1].Txn = txns[1]
    quote[1].AuthAddr = authAddr

    _, stx2, err := crypto.SignTransaction(mmAccount.PrivateKey, txns[2])
    if err != nil {
        log.Print(err)
        return quote, errors.New("Could not sign transaction")
    }
    err = msgpack.Decode(stx2, &quote[2])
    if err != nil {
        log.Print(err)
        return quote, errors.New("Could not decode signed transaction")
    }

    return quote, nil
}

func quote(w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query()
    baseAssetId := query.Get("baseAssetId")
    quoteAssetId := query.Get("quoteAssetId")
    quantityStr := query.Get("quantity")
    toAddress := query.Get("address")
    values, exists := query["encoding"]
    var encoding string
    if exists {
        encoding = values[0]
    } else {
        encoding = "json"
    }
    side, err := parseTradeSide(query.Get("side"))
    if err != nil {
        fmt.Fprintf(w, "Invalid side")
        return
    }
    _ , err = types.DecodeAddress(toAddress)
    if err != nil {
        fmt.Fprintf(w, "Invalid address")
        return
    }
    quantity, _ := strconv.ParseUint(quantityStr, 10, 64)
    if (quantity <= 0 || quantity > 1e12) {
        fmt.Fprintf(w, "Invalid quantity")
        return
    }

    quote, err  := GetAlgorandQuote(baseAssetId, quoteAssetId, side, quantity, toAddress)
    if err != nil {
        fmt.Fprintf(w, "Error: %s", err)
        return
    }

    var encodedQuote []byte
    if encoding == "msgpack" {
        encodedQuote = msgpack.Encode(quote)
    } else if encoding == "goal" {
        stx1 := msgpack.Encode(quote[0])
        stx2 := msgpack.Encode(quote[1])
        encodedQuote = append(encodedQuote, stx1...)
        encodedQuote = append(encodedQuote, stx2...)
    } else {
        encodedQuote = json.Encode(quote)
    }
    if err != nil {
        fmt.Fprintf(w, "Error: %s", err)
        return
    }

    w.Write(encodedQuote)
}

func getpairsroute(w http.ResponseWriter, r *http.Request) {
    jsonPairs := json.Encode(SUPPORTED_PAIRS)
    w.Write(jsonPairs)
}

func setbidaskroute(w http.ResponseWriter, r *http.Request) {
    decoder := stdjson.NewDecoder(r.Body)
    var args SetBidAskArgs
    err := decoder.Decode(&args)
    if err != nil {
        fmt.Fprintf(w, "Error: %s", err)
        return
    }
    pair, err := getTradingPair(args.BaseAssetId, args.QuoteAssetId)
    if err != nil {
        fmt.Fprintf(w, "Error: %s", err)
        return
    }
    if args.Side == Buy {
        pair.Orderbook.Bid[args.Position] = LimitOrder{args.Quantity,args.Price}
        fmt.Fprintf(w, "Bid Set")
    } else if args.Side == Sell {
        pair.Orderbook.Ask[args.Position] = LimitOrder{args.Quantity,args.Price}
        fmt.Fprintf(w, "Ask Set")
    } else {
        fmt.Fprintf(w, "Invalid side")
    }
}

func getorderbookroute(w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query()
    baseAssetId := query.Get("baseAssetId")
    quoteAssetId := query.Get("quoteAssetId")
    values, exists := query["depth"]
    var depth uint64
    var err error
    if exists {
        depth, err = strconv.ParseUint(values[0], 10, 64)
        if err != nil {
            fmt.Fprintf(w, "Unsupported pair")
            return
        }
        if depth > 15 {
            depth = 15
        }
    } else {
        depth = 5
    }
    pair, err := getTradingPair(baseAssetId, quoteAssetId)
    if err != nil {
        fmt.Fprintf(w, "Unsupported pair")
        return
    }
    var response GetOrderBookResponse
    response.Bids = pair.Orderbook.Bid[0:2]
    response.Asks = pair.Orderbook.Ask[0:2]
    jsonresponse, err := stdjson.Marshal(response)
    if err != nil {
        fmt.Fprintf(w, "Error: %s", err)
        return
    }
    w.Write(jsonresponse)
}

func getTradingPair(baseAssetId string, quoteAssetId string) (pair *TradingPair, err error) {
    for i := 0; i < len(SUPPORTED_PAIRS); i++ {
        if baseAssetId == SUPPORTED_PAIRS[i].BaseAsset.AssetId && quoteAssetId == SUPPORTED_PAIRS[i].QuoteAsset.AssetId {
            return &SUPPORTED_PAIRS[i], nil
        }
    }
    return pair, errors.New("Unsupported pair")
}

func createPaymentTxn(assetId string, fromAddress string, toAddress string, quantity uint64, note []byte) (txn types.Transaction, err error) {
    txParams, _ := algodClient.SuggestedParams().Do(context.Background())
    if assetId == "algorand" {
        txn, err = future.MakePaymentTxn(fromAddress, toAddress, quantity, note, "", txParams)
    } else {
        algorandAssetId, err := strconv.ParseUint(strings.Split(assetId, "_")[1], 10, 64)
        if err != nil {
            log.Print(err)
            return txn, errors.New("Invalid asset id")
        }
        txn, err = future.MakeAssetTransferTxn(fromAddress, toAddress, quantity, note, txParams, "", algorandAssetId)
    }
    if err != nil {
        log.Print(err)
        return txn, err
    }
    return txn, err
}

// TODO: Use the full bid/ask ladders instead of just the tip
func calculateQuoteQuantity(baseQuantity uint64, pair TradingPair, side TradeSide) (quoteQuantity uint64, price float64, err error) {
    if side == Buy {
        if pair.Orderbook.Ask[0].Quantity < baseQuantity {
            return quoteQuantity, price, errors.New("Insufficient liquidity")
        }
        price = pair.Orderbook.Ask[0].Price
    } else if side == Sell {
        if pair.Orderbook.Bid[0].Quantity < baseQuantity {
            return quoteQuantity, price, errors.New("Insufficient liquidity")
        }
        price = pair.Orderbook.Bid[0].Price
    } else {
        return quoteQuantity, price, errors.New("Invalid side")
    }
    quoteQuantity = uint64(float64(baseQuantity) * price * math.Pow10(int(pair.QuoteAsset.Decimals)) / math.Pow10(int(pair.BaseAsset.Decimals)))
    return quoteQuantity, price, nil
}

func LogRequests(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        log.Printf(
            "[%s] %s %s %s",
            r.Method,
            r.Host,
            r.URL.Path,
            r.URL.RawQuery,
        )
        next.ServeHTTP(w, r)
    })
}

func runserver() {
    var err error
    algodClient, err = algod.MakeClient(config.AlgodUrl, config.AlgodToken)
    if err != nil {
        log.Print(err)
        return
    }

    router := mux.NewRouter()
    router.Use(LogRequests)
    router.HandleFunc("/quote", quote).Methods("GET")
    router.HandleFunc("/pairs", getpairsroute).Methods("GET")
    router.HandleFunc("/mm/setbidask", setbidaskroute).Methods("POST")
    router.HandleFunc("/mm/orderbook", getorderbookroute).Methods("GET")
    err = http.ListenAndServe(config.ServerUrl, router)
    if err != nil {
        log.Fatal(err)
        return
    }
}
