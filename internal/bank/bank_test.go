package bank

import (
	"testing"
	"time"
)

func TestCreateAccountRejectsInvalidDOB(t *testing.T) {
	b := NewBank()

	_, err := b.CreateAccount("Alex Doe", "123", "not-a-date")
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

	_, err := b.CreateAccount("Alex Doe", "123", futureDOB)
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

	_, err := b.CreateAccount("Alex Doe", "123", underageDOB)
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

	id, err := b.CreateAccount("Alex Doe", "123", adultDOB)
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
