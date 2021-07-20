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
    if err != nil {
        return errors.New("Could not read config.yaml")
    }
    err = yaml.Unmarshal(fileBytes, &config)
    if err != nil {
        return errors.New("Error parsing config.yaml")
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
