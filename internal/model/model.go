package model

import (
	"fmt"
	"sync"
	"time"
)

// starting with the models.
// This is where we declare the structs that hold the user data.

// user details for creating new account.
type AccountHolder struct {
	FullName string
	Idno     string
	Age      time.Time
}

// user details and account info.
type Account struct {
	ID       int
	Holder   AccountHolder
	Ballance float64
	History  []Transactions
	mu       sync.Mutex
}

// User transaction details.
type Transactions struct {
	ID        int
	Amount    float64
	Type      string
	Timestamp time.Time
	Details   string
}

// Create New account for the user.
// This function updates the account struct.
func NewAccount(id int, holder AccountHolder, initialDeposit float64) *Account {
	var history []Transactions
	if initialDeposit > 0 {
		history = append(history, Transactions{
			ID:        1,
			Type:      "INITIAL DEPOSIT",
			Amount:    initialDeposit,
			Timestamp: time.Now(),
			Details:   fmt.Sprint("Initial deposit of $%.2f", initialDeposit),
		})
	}
	return &Account{
		ID:     id,
		Holder: holder,
	}
}
