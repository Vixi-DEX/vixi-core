package main

import (
    "bufio"
    "bytes"
    "fmt"
    "strconv"
    "strings"
    "log"
    "os"
    "net/http"
    "math"
    "errors"
    "io/ioutil"
    "github.com/algorand/go-algorand-sdk/encoding/msgpack"
    "github.com/algorand/go-algorand-sdk/client/v2/algod"
    "github.com/algorand/go-algorand-sdk/crypto"
    "github.com/algorand/go-algorand-sdk/mnemonic"
    "github.com/algorand/go-algorand-sdk/types"
    "crypto/ed25519"
    "context"
    "encoding/base64"
    stdjson "encoding/json"
    "text/tabwriter"
    "golang.org/x/term"
    "syscall"
)

func quoteget(pair TradingPair, side TradeSide, quantityFloat float64, toAddress string) (quote AlgorandQuote, err error) {
    var decimals = 6
    quantity := int(quantityFloat * math.Pow10(decimals))
    url := fmt.Sprintf("%s/quote?address=%s&baseAssetId=%s&quoteAssetId=%s&quantity=%d&side=%s&encoding=msgpack", BOOTSTRAP_MM_URL, toAddress, pair.BaseAsset.AssetId, pair.QuoteAsset.AssetId, quantity, side.String())
    resp, err := http.Get(url)
    if err != nil {
        log.Print(err)
        return quote, errors.New("Could not get quote")
    }
    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        log.Print(err)
        return quote, errors.New("Could not read HTTP Body")
    }
    err = msgpack.Decode(body, &quote)
    if err != nil {
        errmsg := string(body)
        return quote, errors.New(errmsg)
    }
    return quote, nil
}

func quoteaccept(quote AlgorandQuote) (groupId types.Digest, err error) {
    var algodClient *algod.Client
    algodClient, err = algod.MakeClient(config.AlgodUrl, config.AlgodToken)
    clientAccount, err := GetWalletAccount(config.ClientAddress)
    if err != nil {
        return groupId, err
    }
    groupId = quote[0].Txn.Header.Group

    stx0 := msgpack.Encode(quote[0])
    _, stx1, err := crypto.SignTransaction(clientAccount.PrivateKey, quote[1].Txn)
    if err != nil {
        log.Print(err)
        return groupId, errors.New("Could not sign transaction")
    }
    stx2 := msgpack.Encode(quote[2])

    var signedGroup []byte
    signedGroup = append(signedGroup, stx0...)
    signedGroup = append(signedGroup, stx1...)
    signedGroup = append(signedGroup, stx2...)

    _, err = algodClient.SendRawTransaction(signedGroup).Do(context.Background())
    if err != nil {
        log.Print(err)
        return groupId, errors.New("Could not broadcast transaction")
    }

    return groupId, nil
}

func DisplayHelpText(command string) {
    if command == "" {
        fmt.Println("VIXI is an atomic swap DEX on Algorand")
        fmt.Println("")
        fmt.Println("Available Commands:")
        fmt.Println("  wallet   Create and manage Algorand accounts")
        fmt.Println("  quote    Get, view, and trade RFQ-style quotes")
        fmt.Println("  mm       Market making related tasks")
        return
    }

    if command == "wallet" {
        fmt.Println("Create and manage Algorand accounts")
        fmt.Println("")
        fmt.Println("Available Commands:")
        fmt.Println("  new      Create a new keypair")
        fmt.Println("  import   Import a key with a mnemonic")
        fmt.Println("  list     View existing keys")
        return
    }

    if command == "quote" {
        fmt.Println("Get, view, and trade RFQ-style quotes")
        fmt.Println("")
        fmt.Println("Available Commands:")
        fmt.Println("  pairs    View available trading pairs")
        fmt.Println("  get      Get a tradeable quote")
        fmt.Println("  accept   Accept the last generated quote")
        return
    }

    if command == "mm" {
        fmt.Println("Market making management")
        fmt.Println("")
        fmt.Println("Available Commands:")
        fmt.Println("  start       Start the market making daemon")
        fmt.Println("  setbid      Set the bid for a quoted pair")
        fmt.Println("  setask      Set the ask for a quoted pair")
        fmt.Println("  orderbook   View the current orderbook for a pair")
        return
    }

    // Default
    DisplayHelpText("")
    return
}

func main() {
    if len(os.Args) < 2 {
        DisplayHelpText("")
        return
    }
    command := os.Args[1]
    if len(os.Args) < 3 {
        DisplayHelpText(command)
        return
    }
    subcommand := os.Args[2]

    loadWallet := (command != "wallet")
    err := LoadConfig(loadWallet)
    if err != nil {
        fmt.Println(err)
        return
    }

    if command == "quote" && subcommand == "get" {
        if len(os.Args) < 6 {
            fmt.Println("HELP: ./vixi quote get [PAIR] [SIDE] [QUANTITY]")
            return
        }
        displayName := os.Args[3]
        pair, err := DisplayNameToTradingPair(displayName)
        if err != nil {
            fmt.Println("Invalid pair")
            return
        }
        side, err := parseTradeSide(os.Args[4])
        if err != nil {
            fmt.Println("Invalid side")
            return
        }
        quantity, _ := strconv.ParseFloat(os.Args[5], 64)
        if (quantity <= 0 || quantity > 1e12) {
            fmt.Println("Invalid quantity")
            return
        }
        toAddress := config.ClientAddress
        quote, err := quoteget(pair, side, quantity, toAddress)
        if err != nil {
            fmt.Println(err)
            return
        }
        msgpackQuote := msgpack.Encode(quote)
        err = ioutil.WriteFile(config.Datadir + "/latest.quote", msgpackQuote, 0666)
        if err != nil {
            fmt.Println("Error writing quote to disk")
            log.Print(err)
            return
        }

        // Determine quantity of quote currency
        var quoteQuantity float64
        var quoteIndex uint8
        if side == Buy {
            quoteIndex = 1
        } else {
            quoteIndex = 0
        }
        if pair.QuoteAsset.AssetId == "algorand" {
            quoteQuantity = float64(quote[quoteIndex].Txn.PaymentTxnFields.Amount) / math.Pow10(6)
        } else {
            quoteQuantity = float64(quote[quoteIndex].Txn.AssetTransferTxnFields.AssetAmount) / math.Pow10(int(pair.QuoteAsset.Decimals))
        }
        var price float64 = quoteQuantity / quantity

        fmt.Println("Pair            ", displayName)
        fmt.Println("Side            ", side)
        fmt.Println("Base Quantity   ", quantity, pair.BaseAsset.Ticker)
        fmt.Println("Quote Quantity  ", quoteQuantity, pair.QuoteAsset.Ticker)
        fmt.Println(fmt.Sprintf("Price            %.4f", price))
        fmt.Println("Counterparty    ", quote[0].Txn.Header.Sender)

        return
    }

    if command == "quote" && subcommand == "accept" {
        fileBytes, err := ioutil.ReadFile(config.Datadir + "/latest.quote")
        if err != nil {
            log.Print("Error reading quote from disk")
            log.Print(err)
            return
        }
        var quote AlgorandQuote
        msgpack.Decode(fileBytes, &quote)
        groupId, err := quoteaccept(quote)
        if err != nil {
            log.Fatal("Atomic Swap failed")
            log.Fatal(err)
            return
        }
        groupIdString := base64.StdEncoding.EncodeToString(groupId[:])
        fmt.Println("Broadcasted transaction group: ", groupIdString)
        return
    }

    if command == "quote" && subcommand == "pairs" {
        url := fmt.Sprintf("%s/pairs", BOOTSTRAP_MM_URL)
        resp, err := http.Get(url)
        if err != nil {
            fmt.Println("Failed to connect to market maker")
            log.Println(err)
            return
        }
        defer resp.Body.Close()
        body, err := ioutil.ReadAll(resp.Body)
        if err != nil {
            fmt.Println("Could not parse HTTP response")
            fmt.Println(err)
            return
        }
        var pairs []TradingPair
        stdjson.Unmarshal(body, &pairs)

        w := new(tabwriter.Writer)
        w.Init(os.Stdout, 8, 8, 0, '\t', 0)
        defer w.Flush()
        fmt.Fprintln(w, "Display Name\tBase Asset ID\tQuote Asset ID\tDecimals")
        fmt.Fprintln(w, "---\t---\t---\t---")
        for _, pair := range pairs {
            fmt.Fprintf(w, "%s\t%s\t%s\t%d\n", pair.DisplayName, pair.BaseAsset.AssetId, pair.QuoteAsset.AssetId, pair.Precision)
        }
        return
    }

    if command == "mm" && subcommand == "start" {
        log.Print("Starting server on ", config.ServerUrl)
        runserver()
        return
    }
    if command == "mm" && (subcommand == "setbid" || subcommand == "setask") {
        if len(os.Args) < 7 {
            fmt.Println("HELP: ./vixi mm", os.Args[2], "[PAIR] [POSITION] [QUANTITY] [PRICE]")
            return
        }
        var bidask TradeSide
        if os.Args[2] == "setbid" {
            bidask = 0
        } else if os.Args[2] == "setask" {
            bidask = 1
        }
        displayName := os.Args[3]
        pair, err := DisplayNameToTradingPair(displayName)
        if err != nil {
            fmt.Println("Unsupported pair")
            return
        }
        position64, err := strconv.ParseUint(os.Args[4], 10, 16)
        position := uint16(position64)
        if err != nil {
            fmt.Println("Could not parse position")
            return
        }
        quantityFloat, err := strconv.ParseFloat(os.Args[5], 64)
        if err != nil {
            fmt.Println("Could not parse quantity")
            return
        }
        quantity := uint64(quantityFloat * math.Pow10(int(pair.BaseAsset.Decimals)))
        price, err := strconv.ParseFloat(os.Args[6], 64)
        if err != nil {
            fmt.Println("Could not parse price")
            return
        }
        url := fmt.Sprintf("%s/mm/setbidask", BOOTSTRAP_MM_URL)
        var args SetBidAskArgs = SetBidAskArgs{bidask, position, pair.BaseAsset.AssetId, pair.QuoteAsset.AssetId, quantity, price}
        jsonargs, err := stdjson.Marshal(args)
        if err != nil {
            fmt.Println("Could not encode json args")
            return
        }
        resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonargs))
        if err != nil {
            log.Print(err)
            log.Println("Is your market maker daemon running? Run ./vixi mm start")
            return
        }
        defer resp.Body.Close()
        body, err := ioutil.ReadAll(resp.Body)
        if err != nil {
            fmt.Println("Could not parse HTTP response")
            fmt.Println(err)
            return
        }
        fmt.Println(string(body))
        return
    }

    if command == "mm" && subcommand == "orderbook" {
        if len(os.Args) < 4 {
            fmt.Println("HELP: ./vixi mm orderbook [PAIR]")
            return
        }
        displayName := os.Args[3]
        pair, err := DisplayNameToTradingPair(displayName)
        if err != nil {
            fmt.Println("Invalid pair")
            return
        }
        url := fmt.Sprintf("%s/mm/orderbook?baseAssetId=%s&quoteAssetId=%s", BOOTSTRAP_MM_URL, pair.BaseAsset.AssetId, pair.QuoteAsset.AssetId)
        resp, err := http.Get(url)
        if err != nil {
            log.Print(err)
            return
        }
        defer resp.Body.Close()
        body, err := ioutil.ReadAll(resp.Body)
        if err != nil {
            fmt.Println("Could not parse HTTP response")
            fmt.Println(err)
            return
        }
        var orderbook GetOrderBookResponse
        stdjson.Unmarshal(body, &orderbook)

        w := new(tabwriter.Writer)
        w.Init(os.Stdout, 8, 8, 0, '\t', 0)
        defer w.Flush()
        fmt.Fprintln(w, "Bid Quantity\tBid Price\t|\tAsk Quantity\tAsk Price")
        fmt.Fprintln(w, "---\t---\t|\t---\t---")
        for i := 0; i < len(orderbook.Bids); i++ {
            bidSizeFloat := float64(orderbook.Bids[i].Quantity) / math.Pow10(int(pair.BaseAsset.Decimals))
            askSizeFloat := float64(orderbook.Asks[i].Quantity) / math.Pow10(int(pair.BaseAsset.Decimals))
            fmt.Fprintf(w, "%.2f\t%.4f\t|\t%.2f\t%.4f\n", bidSizeFloat, orderbook.Bids[i].Price, askSizeFloat, orderbook.Asks[i].Price)
        }
        return
    }

    if command == "wallet" && subcommand == "create" {
        fmt.Print("Enter a password: ")
        password, err := term.ReadPassword(int(syscall.Stdin))
        if err != nil {
            fmt.Println(err)
            return
        }
        fmt.Print("\nConfirm Password: ")
        confirmPassword, err := term.ReadPassword(int(syscall.Stdin))
        if err != nil {
            fmt.Println(err)
            return
        }
        if string(password) != string(confirmPassword) {
            fmt.Println("Passwords do not match")
        }
        fmt.Println("TODO: Actual wallet creation")
        return
    }

    if command == "wallet" && subcommand == "new" {
        keypair := crypto.GenerateAccount()
        keypairMsgpack := msgpack.Encode(keypair)
        keypairMsgpack = append(keypairMsgpack, '\n')
        f, err := os.OpenFile(config.Datadir + "/wallet.dat", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
        if err != nil {
            log.Println(err)
            return
        }
        defer f.Close()
        _, err = f.Write(keypairMsgpack)
        if err != nil {
            log.Println(err)
            return
        }
        phrase, err := mnemonic.FromPrivateKey(keypair.PrivateKey)
        if err != nil {
            log.Println(err)
            return
        }
        fmt.Println("Don't forget to backup your mnemonic! Each address has a unique mnemonic")
        fmt.Println("")
        fmt.Print("Address: ")
        fmt.Print(keypair.Address)
        fmt.Print("\nMnemonic: ")
        fmt.Print(phrase, "\n")
        return
    }

    if command == "wallet" && subcommand == "list" {
        accounts, err := GetWalletAccounts()
        if err != nil {
            log.Println(err)
            return
        }
        for _, account := range accounts {
            fmt.Println(account.Address)
        }
        return
    }

    if command == "wallet" && subcommand == "import" {
        fmt.Print("Enter mnemonic: ")
        reader := bufio.NewReader(os.Stdin)
        phrase, err := reader.ReadString('\n')
        if err != nil {
            log.Println("Could not read input")
            return
        }
        phrase = strings.TrimSuffix(phrase, "\n")
        privkey, err := mnemonic.ToPrivateKey(phrase)
        if err != nil {
            log.Println("Invalid mnemonic")
            return
        }
        address, err := crypto.GenerateAddressFromSK(privkey)
        if err != nil {
            log.Println("Could not generate address from mnemonic")
            return
        }
        pubkey := ed25519.PublicKey(privkey[32:])
        var keypair crypto.Account = crypto.Account{pubkey,privkey,address}
        keypairMsgpack := msgpack.Encode(keypair)
        keypairMsgpack = append(keypairMsgpack, '\n')
        f, err := os.OpenFile(config.Datadir + "/wallet.dat", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
        if err != nil {
            log.Println(err)
        }
        defer f.Close()
        _, err = f.Write(keypairMsgpack)
        if err != nil {
            log.Println(err)
        }
        fmt.Println("Imported", address)
        return
    }

    // Default
    DisplayHelpText("")
    return
}
