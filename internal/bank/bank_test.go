package bank

import (
	"mybank/internal/model"
	"testing"
	"time"
)

func TestCreateAccountRejectsInvalidDOB(t *testing.T) {
	b := NewBank()

	_, err := b.CreateAccount("Alex Doe", "123", "not-a-date", model.AccountTypeAssets, model.CurrencyTypeKSH)
	if err == nil {
		t.Fatal("expected an error for an invalid DOB")
	}
	if len(b.accounts) != 0 {
		t.Fatalf("expected no account to be created, got %d", len(b.accounts))
	}
}

func TestCreateAccountRejectsFutureDOB(t *testing.T) {
	b := NewBank()
	futureDOB := time.Now().AddDate(0, 0, 2).Format("2006-01-02")

	_, err := b.CreateAccount("Alex Doe", "123", futureDOB, model.AccountTypeAssets, model.CurrencyTypeKSH)
	if err == nil {
		t.Fatal("expected an error for a future DOB")
	}
	if len(b.accounts) != 0 {
		t.Fatalf("expected no account to be created, got %d", len(b.accounts))
	}
}

func TestCreateAccountRejectsUnderageCustomer(t *testing.T) {
	b := NewBank()
	underageDOB := time.Now().AddDate(-17, 0, 0).Format("2006-01-02")

	_, err := b.CreateAccount("Alex Doe", "123", underageDOB, model.AccountTypeAssets, model.CurrencyTypeKSH)
	if err == nil {
		t.Fatal("expected an error for a customer under 18")
	}
	if len(b.accounts) != 0 {
		t.Fatalf("expected no account to be created, got %d", len(b.accounts))
	}
}

func TestCreateAccountAcceptsAdultCustomer(t *testing.T) {
	b := NewBank()
	adultDOB := time.Now().AddDate(-20, 0, 0).Format("2006-01-02")

	id, err := b.CreateAccount("Alex Doe", "123", adultDOB, model.AccountTypeAssets, model.CurrencyTypeKSH)
	if err != nil {
		t.Fatalf("expected account creation to succeed, got error: %v", err)
	}
	if id != 1 {
		t.Fatalf("expected account ID 1, got %d", id)
	}
	if _, ok := b.accounts[id]; !ok {
		t.Fatalf("expected account %d to be stored", id)
	}
}

func TestValidateAge_InvalidDOB(t *testing.T) {
	err := ValidateAge("12/05/2000")

	if err == nil {
		t.Fatal("expected an error for invalid DOB format, got nil")
	}
}
func TestValidateAge_FutureDOB(t *testing.T) {
	futureDOB := time.Now().AddDate(1, 0, 0).Format("2006-01-02")

	err := ValidateAge(futureDOB)

	if err == nil {
		t.Fatal("expected an error for future DOB, got nil")
	}
}

func TestValidateAge_UnderageCustomer(t *testing.T) {
	underageDOB := time.Now().AddDate(-17, 0, 0).Format("2006-01-02")

	err := ValidateAge(underageDOB)

	if err == nil {
		t.Fatal("expected an error for an underage customer, got nil")
	}
}

func TestCreateAccount_ValidCustomer(t *testing.T) {
	b := NewBank()

	validDOB := time.Now().AddDate(-25, 0, 0).Format("2006-01-02")

	id, err := b.CreateAccount("Jane Doe", "12345678", validDOB, model.AccountTypeAssets, model.CurrencyTypeKSH)

	if err != nil {
		t.Fatalf("expected account creation to succeed, got error: %v", err)
	}

	if id != 1 {
		t.Errorf("expected account ID 1, got %d", id)
	}

	if _, exists := b.accounts[id]; !exists {
		t.Error("expected account to be stored in the bank")
	}
}
