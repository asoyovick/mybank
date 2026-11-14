package model

import (
	"errors"
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
	ID      int
	Holder  AccountHolder
	Balance float64
	History []Transactions
	Mu      sync.Mutex
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
	// user Register the initial deposit.
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
		ID:      id,
		Holder:  holder,
		Balance: initialDeposit,
		History: history,
	}
}

// Deposit adds funds to the account under the mutex lock.
func (acc *Account) Deposit(amount float64) error {
	if amount <= 0 {
		return errors.New("deposit amount must be greater than zero")
	}
	acc.Mu.Lock()
	defer acc.Mu.Unlock()
	acc.Balance += amount
	acc.History = append(acc.History, Transactions{
		ID:        len(acc.History) + 1,
		Type:      "Deposit",
		Amount:    amount,
		Timestamp: time.Now(),
		Details:   fmt.Sprintf("Deposited $%.2f", amount),
	})
	return nil
}
