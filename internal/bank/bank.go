package bank

//handle the bank engine.
import (
	"mybank/internal/model"
)

// create bank register for managing all accounts.
type Bank struct {
	accounts map[int]*model.Account
	nextID   int
}

// Initialize and return an empty new bank instance.
func NewBank() *Bank {
	return &Bank{
		accounts: make(map[int]*model.Account),
		nextID:   1,
	}
}

// Initiates New account with profile details, and asigns a unique ID.
func (b *Bank) CreateAccount(fullname, idno string, dob string) int {
	id := b.nextID
	b.nextID++
	// Construct accountHolder Profile
	holder := model.AccountHolder{
		FullName: fullname,
		Idno:     idno,
		Age:      dob,
	}
	b.accounts[id] = model.NewAccount(id, holder)
	return id
}
