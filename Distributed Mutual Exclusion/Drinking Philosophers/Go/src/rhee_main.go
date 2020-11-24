package main

import (
	"Rhee"
	"sync"
)

func main() {
	var nodes []Rhee.Node

	nodes = make([]Rhee.Node, Rhee.NB_NODES)
	var wg sync.WaitGroup
	var messages = make([]chan []byte, Rhee.NB_NODES)
	
	Rhee.Logger.Info("nb_process #", Rhee.NB_NODES)
	
	for i := 0; i < Rhee.NB_NODES; i++ {
		nodes[i].Philosopher.Id = i
		nodes[i].Philosopher.NbCS = 0
		nodes[i].Philosopher.State = Rhee.STATE_THINKING
		nodes[i].Philosopher.ForkId  = make([]int, Rhee.NB_NODES - 1)
		nodes[i].Philosopher.ForkStatus  = make([]bool, Rhee.NB_NODES - 1)
		nodes[i].Philosopher.ForkClean  = make([]bool, Rhee.NB_NODES - 1)
		var idx int = 0
		for j := 0; j < Rhee.NB_NODES; j++ {
			if j == nodes[i].Philosopher.Id {
				// skip my own ID
				continue
			} else {
				nodes[i].Philosopher.ForkId[idx] = j
				idx++
			}
		}
		// Initially forks are in the hand of the nodes with id lower than the fork id to make graphs acyclic
		// Initially all forks are dirty
		for j := 0; j < Rhee.NB_NODES - 1; j++ {
			if nodes[i].Philosopher.ForkId[j] < i {
				nodes[i].Philosopher.ForkStatus[j]    = false
			} else {
				nodes[i].Philosopher.ForkStatus[j]    = true
			}
			nodes[i].Philosopher.ForkClean[j]     = false
		}

		messages[i] = make(chan []byte)
		// messages[i] = make(chan string)
		nodes[i].Philosopher.Initialized = true
	}

	for i := 0; i < Rhee.NB_NODES; i++ {
		nodes[i].Messages = messages
	}

	for i := 0; i < Rhee.NB_NODES; i++ {
		wg.Add(1)
		go nodes[i].Rhee(&wg)
	}
	wg.Wait()
	for i := 0; i < Rhee.NB_NODES; i++ {
		Rhee.Logger.Info("Node #", nodes[i].Philosopher.Id," entered CS ", nodes[i].Philosopher.NbCS," time")	
	}
	Rhee.Logger.Info(Rhee.NB_MSG, " messages sent")
}

