package main

import (
	"./fdlib"
	"os"
        "log"
        "net/smtp"
)



func main() {
	// Local (127.0.0.1) hardcoded IPs to simplify testing.
	localIpPort := os.Getenv("LOCAL_RESP_IP_PORT")
	toMonitorIpPort := os.Getenv("REMOTE_MONITOR_IP_PORT") // TODO: change this to remote node
	var lostMsgThresh uint8 = 5

	// TODO: generate a new random epoch nonce on each run
	var epochNonce uint64 = 12345
	var chCapacity uint8 = 5

	// Initialize fdlib. Note the use of multiple assignment:
	// https://gobyexample.com/multiple-return-values
	fd, notifyCh, err := fdlib.Initialize(epochNonce, chCapacity)
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

	log.Println("Started responding to heartbeats.")

	// Add a monitor for a remote node.
	localIpPortMon := "127.0.0.1:9090"
	err = fd.AddMonitor(localIpPortMon, toMonitorIpPort, lostMsgThresh)
	if checkError(err) != nil {
		return
	}

	log.Println("Started to monitor node: ", toMonitorIpPort)

	// Wait indefinitely, blocking on the notify channel, to detect a
	// failure.
	select {
	case notify := <-notifyCh:
		log.Println("Detected a failure of", notify)
		sendFailerEmail()
	}
}

// If error is non-nil, print it out and return it.
func checkError(err error) error {
	if err != nil {
		log.Println(os.Stderr, "Error ", err.Error())
		return err
	}
	return nil
}

// send server failer notification
func sendFailerEmail() {
        from := os.Getenv("NOTIFICATION_EMAIL")
        pass := os.Getenv("NOTIFICATION_EMAIL_PASSWORD")
        to := os.Getenv("SUBJECT_EMAIL")
	msg := "From: " + from + "\n" +
                "To: " + to + "\n" +
                "Subject: Server Failer\n\n" + os.Getenv("CONTEXT")

        err := smtp.SendMail("smtp.gmail.com:587",
                smtp.PlainAuth("", from, pass, "smtp.gmail.com"),
                from, []string{to}, []byte(msg))

        if err != nil {
                log.Printf("smtp error: %s", err)
                return
        }
        log.Print("server failer notfication sent")
}


