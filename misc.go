package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

// if we got an error, panic and log it. otherwise do nothing
func check(e error) {
	if e != nil {
		log.Panicln("PANIC!:", e)
		// panic(e)
	}
}

// fetches an environment variable. if the variable is not set, it returns a default
func fetchEnvValue(key string, fallback string) string {
	value, isset := os.LookupEnv(key)
	if !isset {
		return fallback
	} else {
		return value
	}
}

// stolen from: https://gist.github.com/r0l1/3dcbb0c8f6cfe9c66ab8008f55f8f28b
// askForConfirmation asks the user for confirmation. A user must type in "yes" or "no" and
// then press enter. It has fuzzy matching, so "y", "Y", "yes", "YES", and "Yes" all count as
// confirmations. If the input is not recognized, it will ask again. The function does not return
// until it gets a valid response from the user.
func askForConfirmation(s string) bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s [y/n]: ", s)

		response, err := reader.ReadString('\n')
		check(err)

		response = strings.ToLower(strings.TrimSpace(response))

		if response == "y" || response == "yes" {
			return true
		} else if response == "n" || response == "no" {
			return false
		}
	}
}
