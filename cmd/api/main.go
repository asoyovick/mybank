package main

import (
	"bufio"
	"fmt"
	"mybank/internal/bank"
	"mybank/internal/model"
	"os"
	"strings"
)

func main() {
	b := bank.NewBank()
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("\n--- Welcome to MyBank CLI ---")
		fmt.Println("1. Create Account")
		fmt.Println("2. Exit")
		fmt.Println("3. Deposit")
		fmt.Print("Choose an option: ")

		input, _ := reader.ReadString('\n')
		choice := strings.TrimSpace(input)

		switch choice {
		case "1":
			fmt.Print("Enter full name: ")
			nameInput, _ := reader.ReadString('\n')
			fullname := strings.TrimSpace(nameInput)

			fmt.Print("Enter ID number: ")
			idInput, _ := reader.ReadString('\n')
			idno := strings.TrimSpace(idInput)

			fmt.Print("Enter your Date of Birth (YYYY-MM-DD): ")
			dobInput, _ := reader.ReadString('\n')
			dob := strings.TrimSpace(dobInput)

			fmt.Print("Enter Currency (KSH or USD): ")
			currInput, _ := reader.ReadString('\n')
			currency := model.CurrencyType(strings.ToUpper(strings.TrimSpace(currInput)))
			if currency != model.CurrencyTypeKSH && currency != model.CurrencyTypeUSD {
				fmt.Println("Invalid currency. Please choose KSH or USD.")
				continue
			}

			fmt.Print("Enter Account Type (ASSETS, LIABILITY, REVENUE): ")
			typeInput, _ := reader.ReadString('\n')
			accType := model.AccountType(strings.ToUpper(strings.TrimSpace(typeInput)))
			if accType != model.AccountTypeAssets && accType != model.AccountTypeLiability && accType != model.AccountTypeRevenue {
				fmt.Println("Invalid account type.")
				continue
			}

			id, err := b.CreateAccount(fullname, idno, dob, accType, currency)
			if err != nil {
				fmt.Println("Account creation failed:", err)
				continue
			}
			fmt.Printf("Account created successfully! User ID: %d | Name: %s | Currency: %s\n", id, fullname, currency)

		case "2":
			fmt.Println("Goodbye!")
			return

		case "3":
			fmt.Println("Deposit feature coming soon!")

		default:
			fmt.Println("Invalid option. Please enter 1, 2, or 3.")
		}
	}
}
