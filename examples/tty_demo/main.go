package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	fmt.Println("TTY Demo Program")
	fmt.Println("Type something and press Enter (type 'quit' to exit):")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := scanner.Text()
		if strings.ToLower(input) == "quit" {
			fmt.Println("Goodbye!")
			break
		}

		fmt.Printf("You typed: %s\n", input)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}
}
