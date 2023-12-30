package main

import "os"

func main() {
	err := moltServiceCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
