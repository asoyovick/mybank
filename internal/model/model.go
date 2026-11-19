package model

import "time"

type AccountStatus string

const (
	AccountStatusActive AccountStatus = "ACTIVE"
	AccountStatusFrozen AccountStatus = "FROZEN"
	AccountStatusClosed AccountStatus = "CLOSED"
)

type AccountType string

const (
	AccountTypeAsset     AccountType = "ASSET"
	AccountTypeLiability AccountType = "LIABILITY"
	AccountTypeRevenue   AccountType = "REVENUE"
)

// AccountHolder contains the personal details of an account owner.
type AccountHolder struct {
	FullName string
	IDNumber string
	DOB      string
}

// Account represents a bank account.
type Account struct {
	ID      int
	Holder  AccountHolder
	Balance string
	// History []Transaction
}

// Transaction represents a single account transaction.
type Transaction struct {
	ID        int
	Amount    float64
	Type      string
	Timestamp time.Time
	Details   string
}

// NewAccount creates and returns a new account for the given holder.
func NewAccount(id int, holder AccountHolder) *Account {
	return &Account{
		ID:     id,
		Holder: holder,
	}
}
