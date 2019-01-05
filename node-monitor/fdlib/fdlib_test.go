package fdlib

import (
	"testing"
	"time"
	"fmt"
)
var (
        fd FD
        notifyCh <-chan FailureDetected
)

// Fdlib Factory test
func TestInitialize(t *testing.T) {
	var epochNonce uint64 = 12345
        var chCapacity uint8 = 128
	var err error = nil
        fd, notifyCh, err = Initialize(epochNonce, chCapacity)
        if err != nil {
		t.Errorf("%+v", err)
        }
	_, _, err = Initialize(epochNonce, chCapacity)
	if err == nil {
		t.Errorf("error should of occered when Initialized is called again")
	}
}

func TestTimeout(t *testing.T) {
	var lostMsgThresh uint64 = 3
	fd.AddMonitor("127.0.0.1:41234","127.0.0.1:53252",lostMsgThresh)
	select {
        case _ = <-notifyCh:
		fmt.Println("\nTestTimeout( ... pass")
		return
        case <-time.After(time.Duration(int(lostMsgThresh)*4) * time.Second):
		t.Errorf("TestTimeout( failed should alert timeout")
        }
}



func TestStartRespStopResp(t *testing.T) {
	var lostMsgThresh uint64 = 12
	err := fd.StartResponding("127.0.0.1:19999")
	defer fd.StopMonitoring()
        if err != nil{
                t.Errorf("%+v", err)
        }
	fd.AddMonitor("127.0.0.1:20000","127.0.0.1:19999",lostMsgThresh)
	select {
        case _ = <-notifyCh:
		t.Errorf("TestStartRespStopResp( ... fail ... no notifications should of occured")
        case <-time.After(time.Duration(4 * time.Second)):
                // do nothing
        }
	defer fd.StopMonitoring()
	fd.StopResponding();
	select {
        case _ = <-notifyCh:
		fmt.Println("\nTestStartRespStopResp( ... pass")
		return
        case <-time.After(8 * time.Second):
                t.Errorf("TestStartRespStopResp( ... fail ... timeouts should of not happened")
        }
}


func TestMonitorOtherNode(t *testing.T) {
	err := fd.StartResponding("127.0.0.1:19999")
	if err != nil{
	            t.Errorf("%+v", err)
	}
	defer fd.StopResponding();
	var lostMsgThresh uint8 = 100
	var localMonitorPort = "127.0.0.1:20000"// TODO Change when testing on different machine
	var monitoringIp = "127.0.0.1:19999"
	fd.AddMonitor(localMonitorPort,monitoringIp,lostMsgThresh)
	defer fd.RemoveMonitor(monitoringIp)
	select {
        case _ = <-notifyCh:
		t.Errorf("TestMonitorOtherNode( ... fail ... timeouts should of not happened")
        case <-time.After(time.Duration(60 * time.Second)):
		fmt.Println("\nTestMonitorOtherNode( ... pass")
		fd.StopResponding();
		return
        }
}



func TestMonitorOtherNodeAndMe(t *testing.T) {
        err := fd.StartResponding("198.162.33.14:20042")
        if err != nil{
                    t.Errorf("%+v", err)
        }
	defer fd.StopResponding();
        var lostMsgThresh uint8 = 250
        var localMonitorPort = "198.162.33.14:20000"// TODO Change when testing on different machine
        var monitoringIp = "198.162.33.23:9999"
        fd.AddMonitor(localMonitorPort,monitoringIp,lostMsgThresh)
        defer fd.RemoveMonitor(monitoringIp)
	//localMonitorPort = "198.162.33.46:20001"
	monitoringIp = "198.162.33.14:20042"
	fd.AddMonitor(localMonitorPort,monitoringIp,lostMsgThresh)
	defer fd.RemoveMonitor(monitoringIp)

        select {
        case _ = <-notifyCh:
                t.Errorf("TestMonitorOtherNode( ... fail ... timeouts should of not happened")
        case <-time.After(time.Duration(30 * time.Second)):
                fmt.Println("\nTestMonitorOtherNodeAndMe( ... pass")
                return
        }
}


func TestStartResponding(t *testing.T) {
	err := fd.StartResponding("127.0.0.1:19999")
	if err != nil{
		t.Errorf("%+v", err)
	}
	fd.StopResponding()
	time.Sleep(3 * time.Second)
	err = fd.StartResponding("127.0.0.1:19999")
	if err != nil {
		t.Errorf("%+v", err)
	}
	fd.AddMonitor("127.0.0.1:20000","127.0.0.1:44444",128)
	fd.AddMonitor("127.0.0.1:21111","127.0.0.1:55555",128)
	fd.AddMonitor("127.0.0.1:22222","127.0.0.1:55556",128)
	fd.AddMonitor("127.0.0.1:33333","127.0.0.1:55557",128)
	time.Sleep(4 * time.Second)
	fd.RemoveMonitor("127.0.0.1:44444")
	fd.RemoveMonitor("127.0.0.1:55555")
	fd.RemoveMonitor("127.0.0.1:55556")
	fd.RemoveMonitor("127.0.0.1:55557")
	fd.AddMonitor("127.0.0.1:20000","127.0.0.1:44444",128)
        fd.AddMonitor("127.0.0.1:21111","127.0.0.1:55555",128)
        fd.AddMonitor("127.0.0.1:22222","127.0.0.1:55556",128)
        fd.AddMonitor("127.0.0.1:33333","127.0.0.1:55557",128)
	time.Sleep(4 * time.Second)
	fd.AddMonitor("127.0.0.1:20000","127.0.0.1:44445",128)
	fd.AddMonitor("127.0.0.1:20001","127.0.0.1:44445",128)
	fd.RemoveMonitor("127.0.0.1:44445")
	time.Sleep(12 * time.Second)
}





