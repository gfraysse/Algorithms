/*
  Copyright "Guillaume Fraysse <gfraysse dot spam plus code at gmail dot com>"

How-to run: 
  go build chandy-misra.go 
  ./chandy-misra 2>&1 |tee /tmp/tmp.log

Parameters:
- Number of nodes is set with NB_NODES global variable
- Number of iterations is hardcoded in main function
*/ 

/*
    Go implementation of Chandy-Misra mutual exclusion algorithm

References :
* https://www.cs.utexas.edu/users/misra/scannedPdf.dir/DrinkingPhil.pdf: Chandy, K.M.; Misra, J. (1984). The Drinking Philosophers Problem. ACM Transactions on Programming Languages and Systems.
* https://en.wikipedia.org/wiki/Dining_philosophers_problem#Chandy/Misra_solution

Complexity is O(Log(n))
*/

package main

import (
	"fmt"
	"log"
	"sync"
	"strings"
	"strconv"
	"time"
)

/* global variable declaration */
var NB_NODES          int = 4
var NB_MSG            int = 0
var NB_ITERATIONS     int = 10
var CURRENT_ITERATION int = 0

var STATE_THINKING    int = 0
var STATE_HUNGRY      int = 1
var STATE_EATING      int = 2

var philosophers []Philosopher

// Debug function
/*
func displayNodes() {
	for i := 0; i < NB_NODES; i++ {
		for j := 0; j < NB_NODES - 1; j++ {
			log.Print("  P#", philosophers[i].id, ", fork #", philosophers[i].forkId[j], ", status=", philosophers[i].forkStatus[j])
		}
	}
}
*/

func checkSanity() {
	for i := 0; i < NB_NODES; i++ {
		// Sanity check, it I don't have a fork, check if the owner actually has it, if I have it check it is not owned by the other
		for j := 0; j < NB_NODES - 1; j ++ {
			var idx int = philosophers[i].forkId[j]
			for k := 0; k < NB_NODES - 1; k ++ {
				if philosophers[i].forkId[j] == philosophers[idx].forkId[k] {
					if philosophers[i].forkStatus[j] == philosophers[idx].forkStatus[k] {
						log.Print("ERR Sanity Check expected philosopher #", i, " fork#", j, " status = ", philosophers[i].forkStatus[j], ", philosopher#", idx, ", fork #", k, " status=", philosophers[idx].forkStatus[k])
					}
					break
				}
			}
		}
	}
}

type Request struct {
	philosopherId int
	forkId        int
}

type Philosopher struct {
	id                 int
	initialized        bool
	forkId           []int
	forkClean        []bool
	forkStatus       []bool
	forkRequestToken []bool
	state           int
	nbCS            int
	queue           []Request
	channel         chan string
	messages        []chan string
}

func (p *Philosopher) String() string {
	var val string
	val = fmt.Sprintf("Philosopher #%d, state=%d, first fork=%d/%v/%v, second fork=%d/%v/%v, third fork=%d/%v/%v, my fork=%d/%v\n",
		p.id,
		p.state,
		p.forkId[0],
		p.forkClean[0],
		p.forkStatus[0],
		p.forkId[1],
		p.forkClean[1],
		p.forkStatus[1],
		p.forkId[2],
		p.forkClean[2],
		p.forkStatus[2])
	return val
}

func (p *Philosopher) enterCS() {
	log.Print("Philosopher #", p.id, " ######################### enterCS")
	p.state = STATE_EATING
	p.nbCS ++
	CURRENT_ITERATION ++
	time.Sleep(500 * time.Millisecond)
	checkSanity()
}

func (p *Philosopher) releaseCS() {
	log.Print("Philosopher #", p.id," releaseCS #########################")	
	p.state = STATE_THINKING
	for i := 0; i < NB_NODES - 1; i ++ {
		p.forkClean[i] = false
	}
	checkSanity()
}

func (p *Philosopher) requestFork(philosopherId int) {
	for i := 0; i < len(p.messages); i++ {
		if i == philosopherId {
			var content = fmt.Sprintf("REQ%d%d", p.id)
			// log.Print(p)
			// log.Print("Philosopher #", p.id, ", SENDING request ", content, " for fork#", forkId, " to Philosopher #", i)	
			log.Print(p.id, " --", i, "--> ", i)	
			p.messages[i] <- content
			NB_MSG ++
		}
	}
}

func (p *Philosopher) sendFork(philosopherId int) {
	for i := 0; i < len(p.messages); i++ {
		if i == philosopherId {
			var content = fmt.Sprintf("REP%d", p.id)
			// log.Print("Philosopher #", p.id, ", SENDING fork ", content, " with fork#", forkId," to Philosopher #", philosopherId)	
			log.Print(p.id,": ", p.id, " ====> ", philosopherId)	
			p.messages[i] <- content
			NB_MSG ++
		}
	}
}

func (p *Philosopher) waitForReplies() {	
	log.Print("Philosopher #", p.id," waitForReplies")	
	for {
		select {
		case msg := <-p.messages[p.id]:
			checkSanity()
			if (strings.Contains(msg, "REQ")) {
				var requester, err = strconv.Atoi(msg[3:4])
				if err != nil {
					log.Fatal(err)
				}
				for i := 0; i < NB_NODES - 1; i ++ {
					if requester == p.forkId[i] {
						if p.forkClean[i] == true && p.forkStatus[i] == true {
							// keep the fork
							log.Print("Philosopher #", p.id,", fork#", i, " is clean, I keep it for now")
							var r Request
							r.philosopherId = requester
							r.forkId        = requester
							p.queue         = append(p.queue, r)
						} else {
							p.forkStatus[i]    = false
							p.forkStatus[i]    = false
							p.sendFork(requester)
						}						
						break							
					}
				}
			}  else if (strings.Contains(msg, "REP")) {
				var sender, err = strconv.Atoi(msg[3:4])
				if err != nil {
					log.Fatal(err)
				}
				log.Print("Philosopher #", p.id, ", RECEIVED fork from Philosopher #", sender, ", ", msg)
				log.Print(sender, ": ", p.id, " <==== ", sender)	
				for i := 0; i < NB_NODES - 1; i ++ {
					if (sender == p.forkId[i]) {
						p.forkStatus[i]    = true
						p.forkClean[i]     = true
						p.forkRequestToken[i] = true
						break
					}
				}
			} else {
				log.Fatal("WTF")
			}
		}
	}
}

func (p *Philosopher) requestCS() {
	log.Print("Philosopher #", p.id, " requestCS")

	for {
		p.state = STATE_HUNGRY
		time.Sleep(100 * time.Millisecond)
		
		for j := 0; j < NB_NODES - 1; j++ {
			if p.forkStatus[j] == false && p.forkRequestToken[j] == true {
				p.forkRequestToken[j] = false
				p.requestFork(p.forkId[j])
				break
			} else {
				p.forkClean[j] = true
			}
		}
		var allGreen = true

		for i := 0; i < NB_NODES - 1; i ++ {
			allGreen = allGreen && p.forkStatus[i] && p.forkClean[i]
			if allGreen == false {
				break
			}
		}
		if (allGreen == true) {
			if (p.state == STATE_EATING) {
				log.Print("** Philosopher #", p.id, " is already eating **")
			} else {
				p.enterCS()
				p.releaseCS()
				for i := 0; i < len(p.queue); i++ {
					var r Request
					r = p.queue[i]
					for j := 0; j < NB_NODES - 1; j++ {
						if (r.philosopherId == p.forkId[j] && p.forkStatus[j] == true) {
							p.forkStatus[j] = false
							p.forkRequestToken[j] = true
							p.sendFork(r.philosopherId)
							break
						}
					}
				}
				p.queue = nil
				for j := 0; j < NB_NODES - 1; j++ {
					if (p.forkStatus[j] == true) {
						p.forkStatus[j] = false
						p.forkRequestToken[j] = true
						p.sendFork(p.forkId[j])
					}
				}
			}
		}
	}

	log.Print("Philosopher #", p.id," END")	
}

func (p *Philosopher) ChandyMisra(wg *sync.WaitGroup) {
	log.Print("Philosopher #", p.id)

	go p.requestCS()
	go p.waitForReplies()
	for {
		time.Sleep(100 * time.Millisecond)
		if CURRENT_ITERATION == NB_ITERATIONS {
			break
		}
	}

	log.Print("Philosopher #", p.id," END after ", NB_ITERATIONS," CS entries")	
	wg.Done()
}

func main() {
	// var philosophers = make([]Philosopher, NB_NODES)
	philosophers = make([]Philosopher, NB_NODES)
	var wg sync.WaitGroup
	var messages  = make([]chan string, NB_NODES)
	
	log.Print("nb_process #", NB_NODES)
	
	for i := 0; i < NB_NODES; i++ {
		philosophers[i].id = i
		philosophers[i].nbCS = 0
		philosophers[i].state = STATE_THINKING
		philosophers[i].forkId  = make([]int, NB_NODES - 1)
		philosophers[i].forkStatus  = make([]bool, NB_NODES - 1)
		philosophers[i].forkClean  = make([]bool, NB_NODES - 1)
		philosophers[i].forkRequestToken  = make([]bool, NB_NODES - 1)
		var idx int = 0
		for j := 0; j < NB_NODES; j++ {
			if j == philosophers[i].id {
				// skip my own ID
				continue
			} else {
				philosophers[i].forkId[idx] = j
				idx++
			}
		}
		// Initially forks are in the hand of the philosophers with id lower than the fork id to make graphs acyclic
		// Initially all forks are dirty
		for j := 0; j < NB_NODES - 1; j++ {
			if philosophers[i].forkId[j] < i {
				philosophers[i].forkStatus[j]    = false
				philosophers[i].forkRequestToken[j] = true
			} else {
				philosophers[i].forkStatus[j]    = true
				philosophers[i].forkRequestToken[j] = false
			}
			philosophers[i].forkClean[j]     = false
		}

		philosophers[i].channel = messages[i]
		messages[i] = make(chan string)
		philosophers[i].initialized = true
	}

	for i := 0; i < NB_NODES; i++ {
		philosophers[i].messages = messages
	}

	for i := 0; i < NB_NODES; i++ {
		wg.Add(1)
		go philosophers[i].ChandyMisra(&wg)
	}
	wg.Wait()
	for i := 0; i < NB_NODES; i++ {
		log.Print("Philosopher #", philosophers[i].id," entered CS ", philosophers[i].nbCS," time")	
	}
	log.Print(NB_MSG, " messages sent")
}
