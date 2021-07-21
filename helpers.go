package main

import (
    "github.com/algorand/go-algorand-sdk/crypto"
    "io/ioutil"
    "errors"
    "gopkg.in/yaml.v2"
    "bytes"
    "github.com/algorand/go-algorand-sdk/encoding/msgpack"
    "github.com/algorand/go-algorand-sdk/types"
    "os"
    "log"
)

func LoadConfig(configFile string, loadWallet bool) (err error) {
    fileBytes, err := ioutil.ReadFile(configFile)
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
    if config.Datadir == "" {
        homedir, err := os.UserHomeDir()
        if err != nil {
            return err
        }
        config.Datadir = homedir + "/.vixi"
    }
    // Go doesn't support the ~, so replace it with the homedir
    if config.Datadir[0] == '~' {
        homedir, err := os.UserHomeDir()
        if err != nil {
            return err
        }
        config.Datadir = homedir + config.Datadir[1:]
    }

    // Create data directory
    err = os.Mkdir(config.Datadir, 0755)
    if err != nil {
        // ignore the error, the directory already exists
    }

    if loadWallet {
        accounts, err := GetWalletAccounts()
        if err != nil {
            return err
        }
        if len(accounts) == 0 {
            return errors.New("No accounts found. Import or create a key to continue. See ./vixi wallet")
        }
        if config.MMAddress == "" {
            config.MMAddress = accounts[0].Address.String()
        }
        if config.ClientAddress == "" {
            config.ClientAddress = accounts[0].Address.String()
        }
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
    fileBytes, err := ioutil.ReadFile(config.Datadir + "/wallet.dat")
    if err != nil {
        return account, errors.New("Wallet doesn't exist. Import or create a key to continue. See ./vixi wallet")
    }
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
    fileBytes, err := ioutil.ReadFile(config.Datadir + "/wallet.dat")
    if err != nil {
        log.Println(err)
        return accounts, errors.New("Wallet doesn't exist. Import or create a key to continue. See ./vixi wallet")
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
