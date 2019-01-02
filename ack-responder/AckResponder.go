package main

import "./fdlib"
import "fmt"
import "os"

func main() {
	// Local (127.0.0.1) hardcoded IPs to simplify testing.
	localIpPort := os.Getenv("LOCAL_RESP_IP_PORT")

	// TODO: generate a new random epoch nonce on each run
	var epochNonce uint64 = 12345
	var chCapacity uint8 = 5

	// Initialize fdlib. Note the use of multiple assignment:
	// https://gobyexample.com/multiple-return-values
	fd, _, err := fdlib.Initialize(epochNonce, chCapacity)
	if checkError(err) != nil {
		return
	}

	// Stop monitoring and stop responding on exit.
	// Defers are really cool, check out: https://blog.golang.org/defer-panic-and-recover
	defer fd.StopMonitoring()
	defer fd.StopResponding()

	err = fd.StartResponding(localIpPort)
	if checkError(err) != nil {
		return
	}

	fmt.Println("Started responding to heartbeats.")

	// Wait indefinitely, blocking on the notify channel, to detect a
	// failure.
	for {
	}
}

// If error is non-nil, print it out and return it.
func checkError(err error) error {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error ", err.Error())
		return err
	}
	return nil
}
