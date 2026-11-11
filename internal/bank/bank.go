package bank

//handle the bank engine.
import (
	"fmt"
	"mybank/internal/model"
	"time"
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
func (b *Bank) CreateAccount(fullname, idno string, dob string) (int, error) {
	// Construct accountHolder Profile
	holder := model.AccountHolder{
		FullName: fullname,
		Idno:     idno,
		Age:      dob,
	}
	layout := "2006-01-02"

	parsedDOB, err := time.Parse(layout, dob)

	if err != nil {
		return 0, fmt.Errorf("invalid date of birth. Please use YYYY-MM-DD format (e.g. 2000-05-20)")
	}
	today := time.Now()

	if parsedDOB.After(time.Now()) {
		return 0, fmt.Errorf("date of birth cannot be in the future")
	}
	age := today.Year() - parsedDOB.Year()

	if today.Month() < parsedDOB.Month() || (today.Month() == parsedDOB.Month() && today.Day() < parsedDOB.Day()) {
		age--
	}
	if age < 18 {
		return 0, fmt.Errorf("Client must 18 years old")
	}
	id := b.nextID
	b.nextID++

	b.accounts[id] = model.NewAccount(id, holder)
	return id, nil
}
