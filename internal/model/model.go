package model

import (
	"time"
)

// starting with the models.
// This is where we declare the structs that hold the user data.

// user details for creating new account.
type AccountHolder struct {
	FullName string
	Idno     string
	Age      string
}

// user details and account info.
type Account struct {
	ID       int
	Holder   AccountHolder
	Ballance string
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
