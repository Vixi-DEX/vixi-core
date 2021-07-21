# VIXI Core

VIXI Core is the core protocol client for the VIXI DEX. 

# Installation

```
go get
go build
```

# Wallet Setup

Before you can use the VIXI DEX, you will have to import or create a new keypair. 
Attempting to use the DEX without a keypair will result in the following error.

```
$ ./vixi quote get USDC-BRZ Buy 1
Wallet doesn't exist. Import or create a key to continue. See ./vixi wallet
```

Import a keypair

```
$ ./vixi wallet import
Enter mnemonic: 
```

Or create a new one

```
$ ./vixi wallet new
Don't forget to backup your mnemonic! Each address has a unique mnemonic

Address: TCKNML4BPALCWNR7L32F2PTGYODDSKDBUGMKH5NUYLH3YBMIIGYZKYMDAE
Mnemonic: gallery stove purpose arm element multiply clinic army giant priority must half mix provide edge shiver cliff moral antique pear federal adjust piano able catalog
```

# Configuration

The default configurations shipped with VIXI are sufficient to get started. 
If you wish to customize your software, copy the `config.yaml.EXAMPLE` file into your data directory and make your edits there. 

```
cp config.yaml.EXAMPLE ~/.vixi/config.yaml
```

Market makers in particular should look into the `algod_url` and `algod_token` settings to hook up their own Algorand participation node.

See the [config.yaml.EXAMPLE](config.yaml.EXAMPLE) file for a full explanation of each setting

# Commands

You can view the list of available commands and help text by running the base executable with no arguments

```
$ ./vixi
VIXI is an atomic swap DEX on Algorand

Available Commands:
  wallet   Create and manage Algorand accounts
  quote    Get, view, and trade RFQ-style quotes
  mm       Market making related tasks
```

## Wallet

```
$ ./vixi wallet
Create and manage Algorand accounts

Available Commands:
  new      Create a new keypair
  import   Import a key with a mnemonic
  list     View existing keys
```

## Quote

```
$ ./vixi quote
Get, view, and trade RFQ-style quotes

Available Commands:
  pairs    View available trading pairs
  get      Get a tradeable quote
  accept   Accept the last generated quote
```

## Market Making

```
$ ./vixi mm
Market making management

Available Commands:
  start       Start the market making daemon
  setbid      Set the bid for a quoted pair
  setask      Set the ask for a quoted pair
  orderbook   View the current orderbook for a pair
```
