// Lachesis Consensus Algorithm by FANTOM Lab.
// 2019. 03. 13 (Wed) Last modified.

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"log"
	"time"

	"github.com/boltdb/bolt"
)

//Block is sturcture of event block
type Block struct {
	Timestamp     int64
	Signature     string
	PrevSelfHash  []byte
	PrevOtherHash []byte
	Hash          []byte
	Height        int
}

//NewBlock is creation of event block
func NewBlock(name string, PrevSelfHash []byte, PrevOtherHash []byte, height int) *Block {
	block := &Block{time.Now().Unix(), name, PrevSelfHash, PrevOtherHash, []byte{}, height}
	data := prepareData(block)
	hash := sha256.Sum256(data)
	block.Hash = hash[:]

	return block
}

func prepareData(b *Block) []byte {
	data := bytes.Join(
		[][]byte{
			[]byte(b.Signature),
			IntToHex(b.Timestamp),
			b.PrevSelfHash,
			b.PrevOtherHash,
		},
		[]byte{},
	)

	return data
}

// Serialize serializes the block
func (b *Block) Serialize() []byte {
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)
	err := encoder.Encode(Block{b.Timestamp, b.Signature, b.PrevSelfHash, b.PrevOtherHash, b.Hash, b.Height})
	if err != nil {
		log.Panic(err)
		fmt.Println(err)
	}

	return result.Bytes()
}

// DeserializeBlock deserializes a block
func DeserializeBlock(d []byte) *Block {
	var block Block

	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&block)
	if err != nil {
		log.Panic(err)
	}

	return &block
}

// OperachainIterator is used to iterate over blockchain blocks
type OperachainIterator struct {
	currentHash []byte
	db          *bolt.DB
}

// NextSelf returns next self parent block starting from the tip
func (i *OperachainIterator) NextSelf() {
	var block *Block

	err := i.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		encodedBlock := b.Get(i.currentHash)
		block = DeserializeBlock(encodedBlock)

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	i.currentHash = block.PrevSelfHash
}

// NextOther returns next other parent block starting from the tip
func (i *OperachainIterator) NextOther() {
	var block *Block

	err := i.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		encodedBlock := b.Get(i.currentHash)
		block = DeserializeBlock(encodedBlock)

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	i.currentHash = block.PrevOtherHash
}

// Show represent Current Block
func (i *OperachainIterator) Show() *Block {
	var block *Block

	err := i.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		encodedBlock := b.Get(i.currentHash)
		block = DeserializeBlock(encodedBlock)

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	return block
}
