package fdlib

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	//"os"
	"time"
)

type failDetector struct {
}

var (
	netDebug            = false
	hbmDebug            = false
	NotifyCh            chan FailureDetected
	chCapacity          uint8
	conn                *net.UDPConn
	epochNonce          uint64
	logger              *log.Logger
	initFlag                                    = false
	startRespondingFlag                         = false
	monitorMap          map[string]*NodeCluster = make(map[string]*NodeCluster)
	RTTRecordMap        map[string]*RTTRecord   = make(map[string]*RTTRecord)
)

type NodeCluster struct {
	conn    *net.UDPConn
	nodeMap map[string]*Node
	alive   bool
}

type Node struct {
	nodeChan      chan AckMessage
	lostMsgThresh uint64
	alive         bool
}

type HBeatMessageRecord struct {
	SeqNum    uint64
	timeStamp time.Time
}

type RTTRecord struct {
	RTT time.Duration
}

type HBeatMessage struct {
	EpochNonce uint64
	SeqNum     uint64
}

type AckMessage struct {
	HBEatEpochNonce uint64
	HBEatSeqNum     uint64
}

type FailureDetected struct {
	UDPIpPort string
	Timestamp time.Time
}

type FD interface {
	StartResponding(LocalIpPort string) (err error)
	StopResponding()
	AddMonitor(LocalIpPort string, RemoteIpPort string, LostMsgThresh uint64) (err error)
	RemoveMonitor(RemoteIpPort string)
	StopMonitoring()
}

func (fd failDetector) StartResponding(LocalIpPort string) (err error) {
	conn, err = getConnection(LocalIpPort)
	if err != nil {
		return err
	}
	if startRespondingFlag {
		return errors.New("Mutiple invocations on StartResponding not allowed")
	}
	startRespondingFlag = true
	go listenHbm(conn)
	fmt.Println("starting heartBeat responds...")
	return nil
}

func (fd failDetector) StopResponding() {
	startRespondingFlag = false
	if conn != nil {
		conn.Close()
	}
}

func (fd failDetector) AddMonitor(LocalIpPort string, RemoteIpPort string, LostMsgThresh uint64) (err error) {
	node := &Node{lostMsgThresh: LostMsgThresh, nodeChan: make(chan AckMessage), alive: true}
	var RTT *RTTRecord
	if _, ok := RTTRecordMap[RemoteIpPort]; ok {
		RTT = RTTRecordMap[RemoteIpPort]
	} else {
		rtt, _ := time.ParseDuration("1")
		RTT = &RTTRecord{RTT: rtt}
		RTTRecordMap[RemoteIpPort] = RTT
	}

	if nodeCluster, ok := monitorMap[LocalIpPort]; ok {
		nodeCluster.nodeMap[RemoteIpPort] = node
		go monitorNode(node, nodeCluster.conn, RemoteIpPort, RTT)
	} else {
		conn, err := getConnection(LocalIpPort)
		if err != nil {
			return err
		}
		nodeCluster = &NodeCluster{conn: conn, nodeMap: make(map[string]*Node), alive: true}
		monitorMap[LocalIpPort] = nodeCluster
		nodeCluster.nodeMap[RemoteIpPort] = node
		if netDebug {
			fmt.Println("starting to recieve acks on port: " + LocalIpPort)
		}
		go monitorNodeCluster(nodeCluster)
		go monitorNode(node, conn, RemoteIpPort, RTT)
	}
	return nil
}

func monitorNodeCluster(nodeCluster *NodeCluster) {
	defer (*nodeCluster).conn.Close()
	for (*nodeCluster).alive {
		var network bytes.Buffer
		buf := make([]byte, 1024)
		var ack AckMessage
		n, addr, err := (*nodeCluster).conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("udp read error", err)
			return
		}
		network.Write(buf[0:n])
		dec := gob.NewDecoder(&network)
		dec.Decode(&ack)
		if ack.HBEatEpochNonce != epochNonce {
			continue
		}
		if netDebug { // netDebug
			fmt.Println("Incoming ack from: " + addr.String())
			fmt.Print("Incoming ack with sequence number: ")
			fmt.Println(ack)
		}
		if node, ok := (*nodeCluster).nodeMap[addr.String()]; ok {
			node.nodeChan <- ack
		}
		network.Reset()
	}
}

func monitorNode(node *Node, listeningConn *net.UDPConn, RemoteIpPort string, nodeRTT *RTTRecord) {
	var lostMsg uint64 = 0
	var hBeatRecordMap map[uint64]HBeatMessageRecord = make(map[uint64]HBeatMessageRecord)
	var seqNum uint64
	var start time.Time
	seqNum, start = sendHbm(listeningConn, RemoteIpPort, hBeatRecordMap)
	var timeout time.Duration = nodeRTT.RTT
	for lostMsg < (*node).lostMsgThresh && (*node).alive {
		select {
		case ack := <-(*node).nodeChan:
			if !(*node).alive {
				return
			}
			if ack.HBEatSeqNum == seqNum {
				timeout = (*nodeRTT).RTT
				delete(hBeatRecordMap, ack.HBEatSeqNum)
				seqNum, start = sendHbm(listeningConn, RemoteIpPort, hBeatRecordMap)
				lostMsg = 0
				if hbmDebug {
					fmt.Println("Ack received before timeout",start)
				}
			} else {
				// sanity check
				if _, ok := hBeatRecordMap[ack.HBEatSeqNum]; ok {
					delete(hBeatRecordMap, ack.HBEatSeqNum)
					lostMsg = 0
					if hbmDebug {
						fmt.Println("Time out Ack Recieved", start)
					}
				}

			}
		case <-time.After(timeout):
			if !(*node).alive {
				return
			}
			lostMsg = lostMsg + 1
			seqNum, start = sendHbm(listeningConn, RemoteIpPort, hBeatRecordMap)
			if lostMsg >= (*node).lostMsgThresh {
				NotifyCh <- FailureDetected{UDPIpPort: RemoteIpPort, Timestamp: time.Now()}
				return
			}
		}
	}
}

//Send is an async function for writing message responses to the
//network. Send waits on a buffered channel, and casts response
//strings to arrays of bytes.
func sendHbm(conn *net.UDPConn, RemoteIpPort string, hBeatRecordMap map[uint64]HBeatMessageRecord) (seqNum uint64, timeStamp time.Time) {
	time.Sleep(3000*time.Millisecond)
	seqNum = rand.Uint64()
	addr, err := net.ResolveUDPAddr("udp", RemoteIpPort)
	if err != nil {
		fmt.Printf("Cannot resolve UDPAddress: Error %s\n", err)
		return
	}
	hbm := HBeatMessage{EpochNonce: epochNonce, SeqNum: seqNum}
	var network bytes.Buffer
	enc := gob.NewEncoder(&network)
	err = enc.Encode(hbm)
	if err != nil {
		fmt.Printf("Cannot encode to HBeatMessage: Error %s\n", err)
		return
	}
	_, err = conn.WriteToUDP(network.Bytes(), addr)
	if err != nil {
		fmt.Println("Error writing to UDP", err)
	}
	if netDebug { //net debug
		fmt.Print("sent hbm to:" + addr.String() + " seqNum: ")
		fmt.Println(seqNum)
	}
	record := HBeatMessageRecord{SeqNum: seqNum, timeStamp: time.Now()}
	hBeatRecordMap[seqNum] = record
	return record.SeqNum, record.timeStamp
}

func (fd failDetector) RemoveMonitor(RemoteIpPort string) {
	for keyc, nodeCluster := range monitorMap {
		for keyn, node := range nodeCluster.nodeMap {
			if keyn == RemoteIpPort {
				node.alive = false
				delete(nodeCluster.nodeMap, RemoteIpPort)
			}
		}
		if len(nodeCluster.nodeMap) == 0 {
			nodeCluster.alive = false
			nodeCluster.conn.Close()
			delete(monitorMap, keyc)
		}
	}
}

func (fd failDetector) StopMonitoring() {
	for keyc, nodeCluster := range monitorMap {
		for keyn, node := range nodeCluster.nodeMap {
			node.alive = false
			delete(nodeCluster.nodeMap, keyn)
		}
		nodeCluster.alive = false
		nodeCluster.conn.Close()
		delete(monitorMap, keyc)
	}
}

// The constructor for a new FD object instance. Note that notifyCh
// can only be received on by the client that receives it from
// initialize:
// https://www.golang-book.com/books/intro/10
func Initialize(EpochNonce uint64, ChCapacity uint8) (fd FD, notifyCh <-chan FailureDetected, err error) {
	if !initFlag {
		fd = failDetector{}
		chCapacity = ChCapacity
		epochNonce = EpochNonce
		NotifyCh = make(chan FailureDetected, ChCapacity)
		initFlag = true
		return fd, NotifyCh, nil
	} else {
		return nil, nil, errors.New("Mutiple invocations on Initialize not allowed")
	}
}

//get connection returChCapacity a udp conn which the server listens on.
func getConnection(ip string) (conn *net.UDPConn, err error) {
	lAddr, err := net.ResolveUDPAddr("udp", ip)
	if err != nil {
		return nil, err
	}
	l, err := net.ListenUDP("udp", lAddr)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return l, nil
}

//Send is an async function for writing message responses to the
//network. Send waits on a buffered channel, and casts response
//strings to arrays of bytes.
func sendAck(conn *net.UDPConn, addr *net.UDPAddr, hbm HBeatMessage) {
	var network bytes.Buffer
	ack := AckMessage{HBEatEpochNonce: hbm.EpochNonce, HBEatSeqNum: hbm.SeqNum}
	enc := gob.NewEncoder(&network)
	err := enc.Encode(ack)
	if err != nil {
		logger.Printf("Cannot decode to uint32: Error %s\n", err)
	}
	if netDebug { //net debug
		fmt.Println("sending ack to:" + addr.String())
	}
	conn.WriteToUDP(network.Bytes(), addr)
}

//Listen is an async function for reading messages from the network.
//Incomming messages which are correctly read, and decoded are passed
//to the main loop via a buffered channel.
func listenHbm(conn *net.UDPConn) {
	defer conn.Close()
	var network bytes.Buffer
	buf := make([]byte, 1024)
	for startRespondingFlag {
		var hbm HBeatMessage
		n, addr, err := conn.ReadFromUDP(buf)
		if netDebug { //net debug
			fmt.Println("incoming hbm from: " + addr.String())
		}
		if err != nil {
			return
		}
		//transfer from byte array to buffer because the canonical
		//io.Copy(&network,conn) requires size checks.
		network.Write(buf[0:n])
		dec := gob.NewDecoder(&network)
		err = dec.Decode(&hbm)
		//fmt.Println(hbm)
		if err != nil {
			logger.Printf("Cannot decode to uint32: Error %s\n", err)
			return
		}
		sendAck(conn, addr, hbm)
		network.Reset()
	}
}

