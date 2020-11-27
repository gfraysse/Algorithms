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

func mainRhee(nbNodes int, nbIterations int, requestSize int) {	
	var wg sync.WaitGroup
	Rhee.Init(nbNodes, nbIterations, requestSize)

	for i := 0; i < nbNodes; i++ {
		wg.Add(1)
		go Rhee.Nodes[i].Rhee(&wg)
	}
	wg.Wait()
	for i := 0; i < nbNodes; i++ {
		Rhee.Logger.Info("Node #", Rhee.Nodes[i].Philosopher.Id," entered CS ", Rhee.Nodes[i].Philosopher.NbCS," time")	
	}
	Rhee.Logger.Info(Rhee.NB_MSG, " messages sent")
}


func mainCM(nbNodes int, nbIterations int) {
	ChandyMisra.Init(nbNodes, nbIterations)
	
	var wg sync.WaitGroup
	for i := 0; i < nbNodes; i++ {
		wg.Add(1)
		go ChandyMisra.Philosophers[i].ChandyMisra(&wg)
	}
	wg.Wait()
	for i := 0; i < nbNodes; i++ {
		log.Print("Philosopher #", ChandyMisra.Philosophers[i].Id," entered CS ", ChandyMisra.Philosophers[i].NbCS," time")	
	}
	log.Print(ChandyMisra.NB_MSG, " messages sent")
}

func main() {
	algoPtr := flag.String("algo", "Rhee", "algorithm to run")
	nbNodesPtr := flag.Int("nodes", 4, "number of nodes in the system")
	requestSizePtr := flag.Int("requestSize", 2, "size of requests")
	nbIterationsPtr := flag.Int("nbIterations", 10, "total number of Critical Section requests")
	flag.Parse()
	log.Println("algo:", *algoPtr)
	if strings.EqualFold(*algoPtr, "Rhee") == true {
		mainRhee(*nbNodesPtr, *nbIterationsPtr, *requestSizePtr)
	} else {
		mainCM(*nbNodesPtr, *nbIterationsPtr)
	}
}
