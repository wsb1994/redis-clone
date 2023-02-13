package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"sort"
	"time"

	"example.com/m/v2/redisutils"
)

func main() {

	ln, err := net.Listen("tcp", ":6379")
	if err != nil {
		// handle error
	}
	defer ln.Close()
	// handle evicting from the string library
	go evictionLoop()
	for {
		conn, err := ln.Accept()

		if err != nil {
			// handle error
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	//defer conn.Close()
	buf := make([]byte, 512)
	var results []byte
	for {
		// Set a timeout for the read operation
		err := conn.SetReadDeadline(time.Now().Add(time.Millisecond * 256))
		if err != nil {
			fmt.Println("Error setting deadline:", err)
			break
		}

		len, err := conn.Read(buf)
		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				// Read timed out, break out of the loop
				break
			}
			fmt.Println("Error reading:", err)
			break
		}

		s := string(buf[:len])
		fmt.Println(s)
		results = append(results, buf[:len]...)
	}

	redisutils.HandlerLoop(context.Background(), conn, string(results))
}
func evictionLoop() {
	// I was too lazy to heapify here, sue me. Golang basic collections are horrible and make this annoying so I got annoyed and moved on lol
	for {
		if len(redisutils.EvictionSchedule.EvictionSchedule) != 0 {
			redisutils.EvictionSchedule.Mtx.Lock()
			sort.Sort(redisutils.EvictionSchedule)

			if redisutils.EvictionSchedule.EvictionSchedule[0].Timestamp.Before(time.Now()) {
				redisutils.DataStore.DataStore.Delete(redisutils.EvictionSchedule.EvictionSchedule[0].Key)
				redisutils.EvictionSchedule.EvictionSchedule = redisutils.EvictionSchedule.EvictionSchedule[1:]
			}
			redisutils.EvictionSchedule.Mtx.Unlock()

		}
	}
}

func readBytesFromTCPConnection(conn net.Conn) ([]byte, error) {
	result := make([]byte, 4096)
	reader := bufio.NewReader(conn)
	for {

		_, err := reader.Peek(1)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		for {

			n, err := conn.Read(result)
			if err != nil {
				if err != io.EOF {
					fmt.Println("read error:", err)
				}
				break
			}
			//fmt.Println("got", n, "bytes.")
			result = append(result, result[:n]...)

		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

	}

	return result, nil
}
