package main

import (
    "github.com/algorand/go-algorand-sdk/crypto"
    "io/ioutil"
    "errors"
    "gopkg.in/yaml.v2"
    "bytes"
    "github.com/algorand/go-algorand-sdk/encoding/msgpack"
    "github.com/algorand/go-algorand-sdk/types"
)

func LoadConfig() (err error) {
    fileBytes, err := ioutil.ReadFile("config.yaml")
    if err == nil { // If the file exists
        err = yaml.Unmarshal(fileBytes, &config)
        if err != nil {
            return errors.New("Error parsing config.yaml")
        }
    }
    // Defaults
    if config.ServerUrl == "" {
        config.ServerUrl = "localhost:8082"
    }
    if config.AlgodUrl == "" {
        config.AlgodUrl = "https://algoexplorerapi.io"
    }
    accounts, err := GetWalletAccounts()
    if err != nil {
        return err
    }
    if len(accounts) == 0 {
        return errors.New("You must either create or import a key before using VIXI")
    }
    if config.MMAddress == "" {
        config.MMAddress = accounts[0].Address.String()
    }
    if config.ClientAddress == "" {
        config.ClientAddress = accounts[0].Address.String()
    }
    return nil
}

func DisplayNameToTradingPair(displayName string) (pair TradingPair, err error) {
    for _, pair := range SUPPORTED_PAIRS {
        if displayName == pair.DisplayName {
            return pair, nil
        }
    }
    return pair, errors.New("Unsupported pair")
}

func GetWalletAccount(addressString string) (account crypto.Account, err error) {
    address, err := types.DecodeAddress(addressString)
    if err != nil {
        return account, errors.New("Invalid Address")
    }
    fileBytes, err := ioutil.ReadFile("wallet.dat")
    lines := bytes.Split(fileBytes, []byte("\n"))
    for _, line := range lines {
        if len(line) == 0 {
            continue
        }
        var account crypto.Account
        err = msgpack.Decode(line, &account)
        if err != nil {
            return account, errors.New("Wallet file is corrupted")
        }
        if account.Address == address {
            return account, nil
        }
    }
    return account, errors.New("Address not found in wallet")

}

func GetWalletAccounts() (accounts []crypto.Account, err error) {
    fileBytes, err := ioutil.ReadFile("wallet.dat")
    if err != nil {
        return accounts, errors.New("Could not read wallet.dat")
    }
    lines := bytes.Split(fileBytes, []byte("\n"))
    for _, line := range lines {
        if len(line) == 0 {
            continue
        }
        var account crypto.Account
        err = msgpack.Decode(line, &account)
        if err != nil {
            return accounts, errors.New("Wallet file is corrupted")
        }
        accounts = append(accounts, account)
    }
    return accounts, nil
}
