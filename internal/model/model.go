package model

import (
	"time"
)

type AccountStatus string

const (
	AccountStatusActive AccountStatus = "ACTIVE"
	AccountStatusFrozen AccountStatus = "FOZEN"
	AccountStatusDomant AccountStatus = "DOMANT"
)

type CurrencyType string

const (
	CurrencyTypeKSH CurrencyType = "KSH"
	CurrencyTypeUSD CurrencyType = "USD"
)

// starting with the models.
// This is where we declare the structs that hold the user data.
// user details for creating new account.
type AccountHolder struct {
	FullName string
	Idno     string
	Age      string
	Gender   string
	krapin   string
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
	//History  []Transactions
}

// User transaction details.
type Transctions struct {
	ID        int
	Amount    float64
	Type      string
	Timestamp time.Time
	Details   string
}

// Create New account for the user.
// This function updates the account struct.
func NewAccount(id int, holder AccountHolder) *Account {
	return &Account{
		ID:     id,
		Holder: holder,
	}
}
