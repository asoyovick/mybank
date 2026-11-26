package bank

import (
	"fmt"
	"mybank/internal/model"
	"strings"
	"time"
)

// Bank manages all customer accounts in memory.
type Bank struct {
	accounts map[int]*model.Account
	nextID   int
}

// NewBank initializes and returns an empty new bank instance.
func NewBank() *Bank {
	return &Bank{
		accounts: make(map[int]*model.Account),
		nextID:   1,
	}
}

func ValidateAge(dob string) error {
	dob = strings.TrimSpace(dob)

	parsedDOB, err := time.Parse("2006-01-02", dob)
	if err != nil {
		return fmt.Errorf("invalid date of birth: please use YYYY-MM-DD format")
	}

	today := time.Now()

	if parsedDOB.After(today) {
		return fmt.Errorf("date of birth cannot be in the future")
	}

	age := today.Year() - parsedDOB.Year()

	if today.Month() < parsedDOB.Month() ||
		(today.Month() == parsedDOB.Month() && today.Day() < parsedDOB.Day()) {
		age--
	}

	if age < 18 {
		return fmt.Errorf("customer must be at least 18 years old")
	}

	return nil
}

// CreateAccount initiates a new account with profile details and assigns a unique ID.
func (b *Bank) CreateAccount(fullname, idno, dob string, accType model.AccountType, currency model.CurrencyType) (int, error) {
	fullname = strings.TrimSpace(fullname)
	idno = strings.TrimSpace(idno)
	dob = strings.TrimSpace(dob)

	if fullname == "" || idno == "" {
		return 0, fmt.Errorf("full name and ID number are required")
	}

	if err := ValidateAge(dob); err != nil {
		return 0, err
	}

	holder := model.AccountHolder{
		FullName: fullname,
		Idno:     idno,
		Age:      dob,
	}

	id := b.nextID
	b.nextID++

	accNum := fmt.Sprintf("ACC%06d", id)

	b.accounts[id] = model.NewAccount(id, holder, accNum, accType, currency)
	return id, nil
}
