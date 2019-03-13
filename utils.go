// Lachesis Consensus Algorithm by FANTOM Lab.
// 2019. 03. 13 (Wed) Last modified.

package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
)

func commandToBytes(command string) []byte {
	var bytes [commandLength]byte

	for i, c := range command {
		bytes[i] = byte(c)
	}

	return bytes[:]
}

func bytesToCommand(bytes []byte) string {
	var command []byte

	for _, b := range bytes {
		if b != 0x0 {
			command = append(command, b)
		}
	}

	return fmt.Sprintf("%s", command)
}

func (oc *Operachain) sendData(addr string, data []byte) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Printf("%s is not available\n", addr)
		//var updatedNodes []string

		//for _, node := range oc.knownAddress {
		//	if node != addr {
		//		updatedNodes = append(updatedNodes, node)
		//	}
		//}

		//oc.knownAddress = updatedNodes

		return
	}
	defer conn.Close()

	_, err = io.Copy(conn, bytes.NewReader(data))
	if err != nil {
		log.Panic(err)
	}
}

func gobEncode(data interface{}) []byte {
	var buff bytes.Buffer

	enc := gob.NewEncoder(&buff)
	err := enc.Encode(data)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}

func dbExists(dbFile string) bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}

	return true
}

// IntToHex converts an int64 to a byte array
func IntToHex(num int64) []byte {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, num)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}

// FindAddr find address based on name
func FindAddr(name string) string {
	addr := fmt.Sprintf("localhost:%s", name)
	return addr
}

// FindName find name based on address
func FindName(addr string) string {
	name := strings.Split(addr, ":")[1]
	return name
}
