# VIXI Core

VIXI Core is the core protocol client for the VIXI DEX. 

# Installation

```
go get
go build
```

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
