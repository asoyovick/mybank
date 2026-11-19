package bank

//handle the bank engine.
import (
	"fmt"
	"mybank/internal/model"
	"strings"
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

// Initiates New account with profile details, and asigns a unique ID.
func (b *Bank) CreateAccount(fullname, idno, dob string) (int, error) {
	dob = strings.TrimSpace(dob)

	if err := ValidateAge(dob); err != nil {
		return 0, err
	}

	holder := model.AccountHolder{
		FullName: fullname,
		IDNumber: idno,
		DOB:      dob,
	}

	id := b.nextID
	b.nextID++

	b.accounts[id] = model.NewAccount(id, holder)
	return id, nil
}
