package bank

import (
	"testing"
	"time"
)

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

	id, err := b.CreateAccount("Jane Doe", "12345678", validDOB)

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
