/* 
go run rhee_main.go --algo=Rhee
 or
go run rhee_main.go --algo=ChandyMisra #default
*/

package main

import (
	"ChandyMisra"
	"flag"
	"log"
	"Rhee"
	"strings"
	"sync"
)

func mainRhee() {	
	var wg sync.WaitGroup
	Rhee.Init()

	for i := 0; i < Rhee.NB_NODES; i++ {
		wg.Add(1)
		go Rhee.Nodes[i].Rhee(&wg)
	}
	wg.Wait()
	for i := 0; i < Rhee.NB_NODES; i++ {
		Rhee.Logger.Info("Node #", Rhee.Nodes[i].Philosopher.Id," entered CS ", Rhee.Nodes[i].Philosopher.NbCS," time")	
	}
	Rhee.Logger.Info(Rhee.NB_MSG, " messages sent")
}


func mainCM() {
	ChandyMisra.Init()
	
	var wg sync.WaitGroup
	for i := 0; i < ChandyMisra.NB_NODES; i++ {
		wg.Add(1)
		go ChandyMisra.Philosophers[i].ChandyMisra(&wg)
	}
	wg.Wait()
	for i := 0; i < ChandyMisra.NB_NODES; i++ {
		log.Print("Philosopher #", ChandyMisra.Philosophers[i].Id," entered CS ", ChandyMisra.Philosophers[i].NbCS," time")	
	}
	log.Print(ChandyMisra.NB_MSG, " messages sent")
}

func main() {
	algoPtr := flag.String("algo", "Rhee", "algorithm to run")
	flag.Parse()
	log.Println("algo:", *algoPtr)
	if strings.EqualFold(*algoPtr, "Rhee") == true {
		mainRhee()
	} else {
		mainCM()
	}
}
