package main

import (
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	switch os.Args[1] {
	case "hash":
		hash(os.Args[2])
	case "compare":
		compare(os.Args[2], os.Args[3])
	default:
		fmt.Println("Invalid command", os.Args[1])
	}
}

func hash(password string) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Printf("error hashing: %v", err)
		return
	}
	fmt.Println(string(hashedBytes))
}

func compare(password, hash_pwd string) {
	err := bcrypt.CompareHashAndPassword([]byte(hash_pwd), []byte(password))
	if err != nil {
		fmt.Printf("Incorrect password man: %s\n", password)
		return
	}
	fmt.Println("Correct pasword fella!")
}
