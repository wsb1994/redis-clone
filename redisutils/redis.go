package redisutils

import (
	"context"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	okResponse   = "+OK\r\n"
	genericError = "-Error message\r\n"
	CRLF         = "\r\n"
	pongResponse = "+Pong\r\n"

	nilResponse = "(nil)"
)

// commands
const (
	commandPING = "ping"
	commandSET  = "set"
	commandGET  = "get"

	setLengthNoExpiry   = 3
	setLengthWithExpiry = 5

	expirySeconds = "EX"
	expiryMillis  = "PX"
)

type EvictableMap struct {
	DataStore sync.Map
}

var DataStore = &EvictableMap{}

type EvictionHeapSchedule struct {
	EvictionSchedule []EvictionEntry
	Mtx              *sync.Mutex
}

var EvictionSchedule = &EvictionHeapSchedule{
	EvictionSchedule: []EvictionEntry{},
	Mtx:              &sync.Mutex{},
}

type EvictionEntry struct {
	Key       string
	Timestamp time.Time
}

func (e EvictionHeapSchedule) Len() int {
	return len(e.EvictionSchedule)
}

func (e EvictionHeapSchedule) Swap(i, j int) {
	e.EvictionSchedule[i], e.EvictionSchedule[j] = e.EvictionSchedule[j], e.EvictionSchedule[i]
}

func (e EvictionHeapSchedule) Less(i, j int) bool {
	return e.EvictionSchedule[i].Timestamp.Before(e.EvictionSchedule[j].Timestamp)
}

type setCommand struct {
	key          string
	value        string
	expiryType   string
	expiryLength int
	isExpiry     bool
}

type getCommand struct {
	key   string
	value string
}

func HandlerLoop(ctx context.Context, conn net.Conn, dataToParse string) {
	commandList := strings.Fields(dataToParse)
	command := strings.ToLower(commandList[0])
	switch command {

	case commandPING:
		pingReply(conn)
	case commandSET:
		setReply(conn, commandList)
	case commandGET:
		getReply(conn, commandList)

	}

}

func getReply(conn net.Conn, commandlist []string) {
	if len(commandlist) != 2 {
		errorResult := []byte(genericError)
		conn.Write(errorResult)
	}
	results, ok := DataStore.DataStore.Load(commandlist[1])
	if ok != true {
		conn.Write([]byte(nilResponse))
	} else {
		conn.Write([]byte(results.(string)))
	}
}

func pingReply(conn net.Conn) {
	data := []byte(pongResponse)
	conn.Write(data)
}

func setReply(conn net.Conn, commandList []string) {
	if len(commandList) != setLengthNoExpiry {
		if len(commandList) != setLengthWithExpiry {
			errorResult := []byte(genericError)
			conn.Write(errorResult)
		}
	}
	items := len(commandList)
	setCommandParams := &setCommand{
		key:          "",
		value:        "",
		expiryType:   "",
		expiryLength: 0,
	}
	switch items {
	// fill command block
	case setLengthNoExpiry:
		setCommandParams.key = commandList[1]
		setCommandParams.value = commandList[2]
		setCommandParams.isExpiry = false
		// fill command block
	case setLengthWithExpiry:

		setCommandParams.key = commandList[1]
		setCommandParams.value = commandList[2]
		setCommandParams.expiryType = commandList[3]
		integer, _ := strconv.Atoi(commandList[4])
		setCommandParams.expiryLength = integer
		setCommandParams.isExpiry = true
	}
	var timestamp time.Time

	// execute valid command blocks based off of previous info
	if setCommandParams.isExpiry {
		// determine timestamp
		if setCommandParams.expiryType == expirySeconds {
			timestamp = time.Now().Add(time.Duration(setCommandParams.expiryLength * int(time.Second)))
		}

		if setCommandParams.expiryType == expiryMillis {
			timestamp = time.Now().Add(time.Duration(setCommandParams.expiryLength * int(time.Millisecond)))
		}

		DataStore.DataStore.Store(setCommandParams.key, setCommandParams.value)
		evictionNotice := &EvictionEntry{
			Key:       setCommandParams.key,
			Timestamp: timestamp,
		}

		EvictionSchedule.Mtx.Lock()
		EvictionSchedule.EvictionSchedule = append(EvictionSchedule.EvictionSchedule, *evictionNotice)
		EvictionSchedule.Mtx.Unlock()

		conn.Write([]byte(okResponse))

	}
	if setCommandParams.isExpiry == false {
		DataStore.DataStore.Store(setCommandParams.key, setCommandParams.value)
		conn.Write([]byte(okResponse))
	}
}
