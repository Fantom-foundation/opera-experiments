// Lachesis Consensus Algorithm by FANTOM Lab.
// 2019. 03. 13 (Wed) Last modified.

package main

import (
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/boltdb/bolt"
)

const dbFile = "Operachain_%s.db"
const blocksBucket = "blocks"

//Operachain(Node) structrue info.

type Operachain struct {
	Db           *bolt.DB
	MakeMutex    *sync.RWMutex
	MyAddress    string
	KnownAddress []string
	KnownHeight  map[string]int
	MyTip        []byte
	KnownTips    map[string][]byte
	SendConn     map[string]net.Conn
	MyName       string
	MyGraph      *Graph
	UpdateChk    bool
}

//OpenOperachain is initialization of Operachain
func OpenOperachain(name string) *Operachain {
	dbFile := fmt.Sprintf(dbFile, name)
	if dbExists(dbFile) == false {
		fmt.Println("No existing operachain found. Create one first.")
		oc := CreateOperachain(name)
		oc.UpdateAddress()
		return oc
	}
	address := fmt.Sprintf("localhost:%s", name)

	var tip []byte
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panic(err)
	}

	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		tip = b.Get([]byte("l"))
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
	//genesis block create

	oc := Operachain{
		Db:          db,
		MakeMutex:   new(sync.RWMutex),
		MyAddress:   address,
		MyTip:       tip,
		KnownHeight: make(map[string]int),
		KnownTips:   make(map[string][]byte),
		SendConn:    make(map[string]net.Conn),
		MyName:      name,
	}
	oc.UpdateAddress()
	oc.UpdateState()

	return &oc
}

// CreateOperachain creates a new blockchain DB
func CreateOperachain(name string) *Operachain {
	address := FindAddr(name)

	dbFile := fmt.Sprintf(dbFile, name)

	var tip []byte

	genesis := NewBlock(name, nil, nil, 1)

	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panic(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucket([]byte(blocksBucket))
		if err != nil {
			log.Panic(err)
		}

		err = b.Put(genesis.Hash, genesis.Serialize())
		if err != nil {
			log.Panic(err)
		}

		err = b.Put([]byte("l"), genesis.Hash)
		if err != nil {
			log.Panic(err)
		}
		tip = genesis.Hash

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	heights := make(map[string]int)
	tips := make(map[string][]byte)

	heights[name] = genesis.Height
	tips[name] = genesis.Hash

	oc := Operachain{
		Db:          db,
		MakeMutex:   new(sync.RWMutex),
		MyAddress:   address,
		MyTip:       tip,
		KnownHeight: heights,
		KnownTips:   tips,
		SendConn:    make(map[string]net.Conn),
		MyName:      name,
	}

	return &oc
}

// UpdateAddress initializes IP
func (oc *Operachain) UpdateAddress() {
	for _, node := range DNSaddress {
		if node != oc.MyAddress {
			if !oc.nodeIsKnown(node) {
				oc.KnownAddress = append(oc.KnownAddress, node)
			}
		} else {
			oc.KnownAddress = append(oc.KnownAddress, node)
			err := oc.Db.View(func(tx *bolt.Tx) error {
				b := tx.Bucket([]byte(blocksBucket))
				tip := b.Get([]byte("l"))
				tipData := b.Get(tip)
				tipBlock := DeserializeBlock(tipData)

				nodeName := FindName(node)
				oc.KnownTips[nodeName] = tip
				oc.KnownHeight[nodeName] = tipBlock.Height
				return nil
			})
			if err != nil {
				log.Panic(err)
			}
		}
	}
}

//UpdateState initializes state of Operachain
func (oc *Operachain) UpdateState() {
	err := oc.Db.View(func(tx *bolt.Tx) error {
		chk := make(map[string]bool)
		var mylist []string

		mylist = append(mylist, string(oc.MyTip))
		chk[string(oc.MyTip)] = true

		for len(mylist) > 0 {
			currentHash := mylist[len(mylist)-1]
			mylist = mylist[:len(mylist)-1]

			oci := oc.Iterator([]byte(currentHash))
			block := oci.Show()
			blockAddr := fmt.Sprintf("localhost:%s", block.Signature)
			if !oc.nodeIsKnown(blockAddr) {
				oc.KnownAddress = append(oc.KnownAddress, blockAddr)
				oc.KnownHeight[block.Signature] = block.Height
				oc.KnownTips[block.Signature] = block.Hash
			} else {
				if oc.KnownHeight[block.Signature] < block.Height {
					oc.KnownHeight[block.Signature] = block.Height
					oc.KnownTips[block.Signature] = block.Hash
				}
			}

			if block.PrevSelfHash != nil {
				if !chk[string(block.PrevSelfHash)] {
					mylist = append(mylist, string(block.PrevSelfHash))
					chk[string(block.PrevSelfHash)] = true
				}
			}
			if block.PrevOtherHash != nil {
				if !chk[string(block.PrevOtherHash)] {
					mylist = append(mylist, string(block.PrevOtherHash))
					chk[string(block.PrevOtherHash)] = true
				}
			}
		}

		return nil
	})

	if err != nil {
		log.Panic(err)
	}
}

// AddBlock saves the block into the blockchain
func (oc *Operachain) AddBlock(block *Block) {
	err := oc.Db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		blockInDb := b.Get(block.Hash)

		if blockInDb != nil {
			return nil
		}

		blockData := block.Serialize()
		err := b.Put(block.Hash, blockData)
		if err != nil {
			log.Panic(err)
		}

		if block.Signature == oc.MyName {
			err = b.Put([]byte("l"), block.Hash)
			if err != nil {
				log.Panic(err)
			}
			oc.MyTip = block.Hash
			oc.KnownTips[block.Signature] = block.Hash
			oc.KnownHeight[block.Signature] = block.Height
		} else {
			blockAddr := FindAddr(block.Signature)
			if oc.nodeIsKnown(blockAddr) {
				if oc.KnownHeight[block.Signature] < block.Height {
					oc.KnownHeight[block.Signature] = block.Height
					oc.KnownTips[block.Signature] = block.Hash
				}
			} else {
				oc.KnownAddress = append(oc.KnownAddress, blockAddr)
				oc.KnownHeight[block.Signature] = block.Height
				oc.KnownTips[block.Signature] = block.Hash
			}
		}

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

}

// Iterator returns a OperachainIter
func (oc *Operachain) Iterator(setPtr []byte) *OperachainIterator {
	var ptr []byte
	if setPtr == nil {
		ptr = oc.MyTip
	} else {
		ptr = setPtr
	}

	oci := &OperachainIterator{ptr, oc.Db}
	return oci
}

func (oc *Operachain) nodeIsKnown(addr string) bool {
	for _, node := range oc.KnownAddress {
		if node == addr {
			return true
		}
	}

	return false
}

//PrintChain initializes state of Operachain
func (oc *Operachain) PrintChain() {
	err := oc.Db.View(func(tx *bolt.Tx) error {
		chk := make(map[string]bool)
		var mylist []string

		mylist = append(mylist, string(oc.MyTip))
		chk[string(oc.MyTip)] = true

		for len(mylist) > 0 {
			currentHash := mylist[0]
			mylist = mylist[1:]

			oci := oc.Iterator([]byte(currentHash))
			block := oci.Show()

			fmt.Printf("============ Block %x ============\n", block.Hash)
			fmt.Printf("Signature: %s\n", block.Signature)
			fmt.Printf("Height: %d\n", block.Height)
			fmt.Printf("Prev.S block: %x\n", block.PrevSelfHash)
			fmt.Printf("Prev.O block: %x\n", block.PrevOtherHash)
			fmt.Printf("\n\n")

			if block.PrevSelfHash != nil {
				if !chk[string(block.PrevSelfHash)] {
					mylist = append(mylist, string(block.PrevSelfHash))
					chk[string(block.PrevSelfHash)] = true
				}
			}
			if block.PrevOtherHash != nil {
				if !chk[string(block.PrevOtherHash)] {
					mylist = append(mylist, string(block.PrevOtherHash))
					chk[string(block.PrevOtherHash)] = true
				}
			}
		}

		return nil
	})

	if err != nil {
		log.Panic(err)
	}
}
