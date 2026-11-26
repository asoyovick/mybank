package model

import (
	"errors"
	"time"
)

// Domain-Specific sentinel errors.
// Defining errors here lets the service and CLI layers react to specific failures.
var (
	ErrAccountNotFound   = errors.New("account not found")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrAccountIsDormant  = errors.New("account is dormant")
	ErrInvalidAmount     = errors.New("amount cannot be less than one")
	ErrCurrencyMismatch  = errors.New("currency mismatch between accounts")
)

type AccountType string

// Custom string types provide compile-time type safety.
// you cannot accidentally pass a currency into a function expecting AccountStatus.
const (
	AccountTypeAssets    AccountType = "ASSETS"
	AccountTypeLiability AccountType = "LIABILITY"
	AccountTypeRevenue   AccountType = "REVENUE"
)

type AccountStatus string

const (
	AccountStatusActive  AccountStatus = "ACTIVE"
	AccountStatusFrozen  AccountStatus = "FROZEN"
	AccountStatusDormant AccountStatus = "DORMANT"
)

type CurrencyType string

const (
	CurrencyTypeKSH CurrencyType = "KSH"
	CurrencyTypeUSD CurrencyType = "USD"
)

type EntryType string

const (
	Deposit  EntryType = "CREDIT"
	Withdraw EntryType = "DEBIT"
)

// starting with the models.
// This is where we declare the structs that hold the user data.
// user details for creating new account.
type AccountHolder struct {
	FullName string
	Idno     string
	Age      string
	Gender   string
	KRAPin   string
	Email    string
}

// user details and account info.
type Account struct {
	ID            int
	Holder        AccountHolder
	Balance       float64
	AccountNumber string
	Currency      CurrencyType
	Status        AccountStatus
	Type          AccountType
	History       []Transaction
}

type Transaction struct {
	ID        int
	AccountID int
	Amount    float64
	EntryType EntryType
	Timestamp time.Time
	Details   string
}

// Create New account for the user.
// This function updates the account struct.
func NewAccount(id int, holder AccountHolder, accNum string, accType AccountType, currency CurrencyType) *Account {
	return &Account{
		ID:            id,
		Holder:        holder,
		AccountNumber: accNum,
		Type:          accType,
		Balance:       0.0,
		Currency:      currency,
		Status:        AccountStatusActive,
		History:       make([]Transaction, 0),
	}
}
