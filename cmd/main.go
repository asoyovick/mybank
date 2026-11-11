package main

import (
	"bufio"
	"fmt"
	"mybank/internal/bank"
	"os"
	"strings"
)

func main() {

	b := bank.NewBank()
	reader := bufio.NewReader(os.Stdin)

	// display the options for the user in the cli.

	fmt.Println("Enter 1 to Create Account and continue")
	//fmt.Scan(&reader)
	fmt.Println("Enter 2 to exit")
	//	fmt.Scan(&reader)
	fmt.Println("enter 3 to deposit")
	//fmt.Scan(&reader)
	// read the users input.
	input, _ := reader.ReadString('\n')

	// remove trailing space from the string.
	choice := strings.TrimSpace(input)

	if choice == "1" {
		fmt.Print("Enter full name:")
		// read the users full name.
		nameInput, _ := reader.ReadString('\n')
		// remove white space from the name.
		fullname := strings.TrimSpace(nameInput)
		fmt.Println("Enter ID number: ")

		idInput, _ := reader.ReadString('\n')
		idno := strings.TrimSpace(idInput)

		fmt.Println("Enter your Date of Birth: ")
		dobInput, _ := reader.ReadString('\n')
		dob := strings.TrimSpace(dobInput)
		// call the bank engine to create an account.
		id, err := b.CreateAccount(fullname, idno, dob)
		if err != nil {
			fmt.Println("Account creation failed: ", err)
			return
		}
		fmt.Printf("Account created successfully for user ID: %d for %s\n", id, fullname)
	} else {
		fmt.Println("User registered successfully")
	}
}
