package main

import "./fdlib"
import "fmt"
import "os"

func main() {
	localIpPort := os.Getenv("LOCAL_RESP_IP_PORT")

	var epochNonce uint64 = 12345
	var chCapacity uint8 = 5

	fd, _, err := fdlib.Initialize(epochNonce, chCapacity)
	if checkError(err) != nil {
		return
	}

	defer fd.StopMonitoring()
	defer fd.StopResponding()

	err = fd.StartResponding(localIpPort)
	if checkError(err) != nil {
		return
	}

	fmt.Println("Started responding to heartbeats.")

	// Wait indefinitely
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
