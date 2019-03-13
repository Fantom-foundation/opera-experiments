// Lachesis Consensus Algorithm by FANTOM Lab.
// 2019. 03. 13 (Wed) Last modified.

package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"time"

	"github.com/boltdb/bolt"
)

//HeightMsg is structrue for sending message of request blocks
type HeightMsg struct {
	Heights  map[string]int
	AddrFrom string
}

//BlockMsg is structrue for sending message of sending blocks
type BlockMsg struct {
	AddrFrom string
	Block    []byte
}

//BlocksMsg is structrue for sending message of sending blocks
type BlocksMsg struct {
	AddrFrom string
	Blocks   []Block
}

//EndMsg is structure for sending close packet
type EndMsg struct {
	AddrFrom string
}

func (oc *Operachain) receiveServer() {
	ln, err := net.Listen("tcp", oc.MyAddress)
	if err != nil {
		log.Panic(err)
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Panic(err)
		}

		go oc.handleConnection(conn)
	}
}

//Sync is process for request other nodes blocks
func (oc *Operachain) Sync() {
	for {
		//requestVersion
		time.Sleep(time.Second)

		for _, node := range oc.KnownAddress {
			if node != oc.MyAddress {
				payload := gobEncode(HeightMsg{oc.KnownHeight, oc.MyAddress})
				reqeust := append(commandToBytes("rstBlocks"), payload...)
				oc.sendData(node, reqeust)
			}
		}
	}
}

func (oc *Operachain) handleConnection(conn net.Conn) {
	defer conn.Close()

	request, err := ioutil.ReadAll(conn)
	if err != nil {
		log.Panic(err)
	}

	command := bytesToCommand(request[:commandLength])
	//fmt.Printf("Received %s command\n", command)
	switch command {
	case "rstBlocks":
		oc.handleRstBlocks(request)
	case "getBlocks":
		oc.handleGetBlocks(request)
	default:
		fmt.Println("Unknown command!")
	}
}

func (oc *Operachain) handleRstBlocks(request []byte) {
	var buff bytes.Buffer
	var payload HeightMsg

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)

	if err != nil {
		log.Panic(err)
	}

	var blocksData []Block

	for _, addr := range oc.KnownAddress {
		node := FindName(addr)
		if oc.KnownHeight[node] > payload.Heights[node] {
			ptr := oc.KnownTips[node]
			oci := oc.Iterator(ptr)
			for {
				if oci.currentHash == nil {
					break
				}
				block := oci.Show()

				if block.Height > payload.Heights[node] {
					blocksData = append(blocksData, *block)
				} else {
					break
				}

				oci.NextSelf()

			}
		}
	}

	data := BlocksMsg{oc.MyAddress, blocksData}
	payload2 := gobEncode(data)
	reqeust2 := append(commandToBytes("getBlocks"), payload2...)
	oc.sendData(payload.AddrFrom, reqeust2)

	/*
		if chk {
			data := EndMsg{oc.myAddress}
			payload2 := gobEncode(data)
			reqeust := append(commandToBytes("endBlocks"), payload2...)
			oc.sendData(payload.AddrFrom, reqeust)
		}
	*/
}

func (oc *Operachain) handleGetBlocks(request []byte) {
	var buff bytes.Buffer
	var payload BlocksMsg

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	oc.MakeMutex.Lock()

	blocksData := payload.Blocks
	chk := false
	for _, block := range blocksData {
		oc.AddBlock(&block)
		chk = true
	}

	if chk {
		//fmt.Println("Received a new block")

		oc.MakeBlock(FindName(payload.AddrFrom))
	}

	oc.MakeMutex.Unlock()
}

//MakeBlock creates a new block
func (oc *Operachain) MakeBlock(name string) {
	var newHeight int
	var tip []byte
	err := oc.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		tip = b.Get([]byte("l"))
		tipData := b.Get(tip)
		tipBlock := DeserializeBlock(tipData)
		newHeight = tipBlock.Height + 1

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	newBlock := NewBlock(oc.MyName, tip, oc.KnownTips[name], newHeight)

	oc.AddBlock(newBlock)

	oc.MyGraph.Tip = oc.BuildGraph(newBlock.Hash, oc.MyGraph.ChkVertex, oc.MyGraph)
	fmt.Println("create new block")
	//oc.UpdateChk = true
}
