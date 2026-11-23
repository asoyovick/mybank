package model

import (
	"time"
)

// Domain-Specific sentinel errors.
// Defining errors here lets the service and HTTP layers to react to specific failures.
var (
	ErrAccountNotFound = errors.New("account not found")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrAccountIsDormant = errors.New("account is dormant")
	ErrInvadlidAmount = errors.New("ammount cannot be less than one")
	ErrCurrencyMismatch = errors.New("currency mismatch between accounts")

)
type AccountType string
// Custom string types provide compile-time type safety.
// you cannot accidentally pass a currency into a function expecting AccountStatus.
const (
	AccountTypeAssets   AccountType = "ASSETS"
	AccountTypeLiabilty AccountType = "LIABILITY"
	AccountTypeRevenue  AccountType = "REVENUE"
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
	Type 		AccountType
	History  []Transactions
}

// User transaction details.
}
}

type Transctions struct {
	ID        int
	AccountID int
	Amount    float64
	entrytype EntryType
	Timestamp time.Time
	Details   string
}

// Create New account for the user.
// This function updates the account struct.
func NewAccount(id int, holder AccountHolder, accNum string, accType AccountType, currency CurrencyType) *Account {
	return &Account{
		ID:     id,
		Holder: holder,
		Type:	accType,
		Balance:	0.0,
		Currency:	currency,
		Status:		AccountStatusActive,
		History:	make([]Transaction, 0),
	}
}
