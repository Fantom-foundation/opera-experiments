// Lachesis Consensus Algorithm by FANTOM Lab.
// 2019. 03. 13 (Wed) Last modified.

package main

import (
	"fmt"
	"os"
	"time"
)

const commandLength = 12
const nodeNumber = 4
const supraMajor = 3 // oversee 2/3
const subMajor = 2 // see 1/3

//DNSaddress nitailizes ip
var DNSaddress = []string{
	"localhost:3001",
	"localhost:3002",
	"localhost:3003",

	//"3003",
	//"3004",
}

func main() {
	//Initialize
	myName := os.Args[1]

	//Operachain start
	oc := OpenOperachain(myName)
	oc.MyGraph = oc.NewGraph()
	switch os.Args[2] {
	case "run":
		go oc.receiveServer()

		go oc.Sync()

		time.Sleep(time.Second * 10)
	case "print":
		oc.PrintChain()

	default:
		fmt.Println("Fault command!")
	}

}
