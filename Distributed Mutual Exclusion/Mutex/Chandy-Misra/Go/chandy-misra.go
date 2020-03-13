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
var NB_NODES int = 4
var NB_ITERATIONS int = 10
var CURRENT_ITERATION int = 0
var STATE_THINKING int = 0
var STATE_EATING int = 2

/*
// Debug function
func displayNodes() {
	for i := 0; i < len(nodes); i++ {
		log.Print("Node #", nodes[i].id, ", last=", nodes[i].last, ", next=", nodes[i].next)
	}
}
*/
type Request struct {
	philosopherId int
	forkId        int
}

type Philosopher struct {
	id           int
	firstForkId  int
	secondForkId int
	firstForkClean  bool
	secondForkClean bool
	firstForkStatus  bool
	secondForkStatus bool
	firstForkRequested  bool
	secondForkRequested bool
	state        int
	nbCS         int
	queue      []Request
	channel    chan string
	messages   []chan string
}

func (p *Philosopher) String() string {
	var val string
	val = fmt.Sprintf("Philosopher #%d, state=%d, first fork=%d/%v/%v, second fork=%d/%v/%v \n",
		p.id,
		p.state,
		p.firstForkId,
		p.firstForkClean,
		p.firstForkStatus,
		p.secondForkId,
		p.secondForkClean,
		p.secondForkStatus)
	return val
}

func (p *Philosopher) enterCS() {
	log.Print("Philosopher #", p.id, " ######################### enterCS")
	p.state = STATE_EATING
	p.nbCS ++
	CURRENT_ITERATION ++
	// log.Print(p)
	time.Sleep(500 * time.Millisecond)
}

func (p *Philosopher) releaseCS() {
	log.Print("Philosopher #", p.id," releaseCS #########################")	
	p.state = STATE_THINKING
	p.firstForkClean  = false
	p.secondForkClean  = false
	// log.Print(p)
}

func (p *Philosopher) requestFork(philosopherId int, forkId int) {
	for i := 0; i < len(p.messages); i++ {
		if i == philosopherId {
			var content = fmt.Sprintf("REQ%d%d", p.id, forkId)
			// log.Print(p)
			log.Print("Philosopher #", p.id, ", SENDING request ", content, " for fork #", forkId, " to Philosopher #", philosopherId)	
			p.messages[i] <- content
		}
	}
}

func (p *Philosopher) sendFork(philosopherId int, forkId int) {
	for i := 0; i < len(p.messages); i++ {
		if i == philosopherId {
			var content = fmt.Sprintf("REP%d%d", p.id, forkId)
			log.Print("Philosopher #", p.id, ", SENDING reply ", content, " with fork #", forkId, " to Philosopher #", philosopherId)	
			p.messages[i] <- content
		}
	}
}
func (p *Philosopher) waitForReplies() {	
	// log.Print("Philosopher #", p.id," waitForReplies")	
	for {
		select {
		case msg := <-p.messages[p.id]:
			if (strings.Contains(msg, "REQ")) {
				var requester, err = strconv.Atoi(msg[3:4])
				if err != nil {
					log.Fatal(err)
				}
				var forkId, err2 = strconv.Atoi(msg[4:])
				if err2 != nil {
					log.Fatal(err2)
				}
				// Only send fork if we have it and it is clean,
				// otherwise store the request to send it after leaving CS
				if (forkId == p.firstForkId) {
					if(p.firstForkClean == true) {
						// keep the fork
						log.Print("Philosopher #", p.id,", fork1 #", forkId, " is clean, I keep it for now")
						var r Request
						r.philosopherId = requester
						r.forkId = forkId
						p.queue = append(p.queue, r)

					} else {
						p.firstForkStatus = false
						p.firstForkRequested = false
						p.sendFork(requester, forkId)
					}
				} else {
					if(p.secondForkClean == true) {
						// keep the fork
						log.Print("Philosopher #", p.id,", fork2 #", forkId, " is clean, I keep it for now")
						var r Request
						r.philosopherId = requester
						r.forkId = forkId
						p.queue = append(p.queue, r)
					} else {
						p.secondForkStatus = false
						p.secondForkRequested = false
						p.sendFork(requester, forkId)
					}
				}
			}  else if (strings.Contains(msg, "REP")) {
				var sender, err = strconv.Atoi(msg[3:4])
				if err != nil {
					log.Fatal(err)
				}
				var forkId, err2 = strconv.Atoi(msg[4:])
				if err2 != nil {
					log.Fatal(err2)
				}
				log.Print("Philosopher #", p.id, ", RECEIVED reply from Philosopher #", sender, " with fork #", forkId, ",", msg)
				if (forkId == p.firstForkId) {
					p.firstForkStatus = true
					p.firstForkClean  = true
					p.firstForkRequested  = false
					log.Print("Philosopher #", p.id, " received fork1")
				} else {
					p.secondForkStatus = true
					p.secondForkClean  = true
					p.secondForkRequested  = false
					log.Print("Philosopher #", p.id, " received fork2")
				}
				// log.Print(p)
			} else {
				log.Fatal("WTF")
			}
		}
	}
	log.Print(p)
}

func (p *Philosopher) requestCS() {
	// log.Print("Philosopher #", p.id, " requestCS")

	for {
		time.Sleep(100 * time.Millisecond)
		if (p.firstForkStatus == false && p.firstForkRequested != true) {
			p.firstForkRequested = true
			if (p.id != 0) {
				p.requestFork(p.id - 1, p.firstForkId)
			} else {
				p.requestFork(3, p.firstForkId)
			}
		} else {
			p.firstForkClean = true
		}
		if (p.secondForkStatus == false && p.secondForkRequested != true) {
			p.secondForkRequested = true
			if (p.id != 3) {
				p.requestFork(p.id + 1, p.secondForkId)
			} else {
				p.requestFork(0, p.secondForkId)
			}
		} else {
			p.secondForkClean = true
		}

		if (p.firstForkStatus == true && p.secondForkStatus == true && p.firstForkClean == true && p.secondForkClean == true) {
			if (p.state == STATE_EATING) {
				// log.Print("** Philosopher #", p.id, " is already eating **")
			} else {
				p.enterCS()
				p.releaseCS()
				for i := 0; i < len(p.queue); i++ {
					var r Request
					r = p.queue[i]
					if (r.forkId == p.firstForkId) {
						p.firstForkStatus = false
						p.firstForkRequested = false
					} else {
						p.secondForkStatus = false
						p.secondForkRequested = false
					}
					p.sendFork(r.philosopherId, r.forkId)
					
				}
				p.queue = nil
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
	var philosophers = make([]Philosopher, NB_NODES)
	var wg sync.WaitGroup
	var messages  = make([]chan string, NB_NODES)
	
	log.Print("nb_process #", NB_NODES)
	
	for i := 0; i < NB_NODES; i++ {
		philosophers[i].id = i
		philosophers[i].nbCS = 0
		philosophers[i].state = STATE_THINKING
		if (i == NB_NODES - 1) {
			philosophers[i].secondForkId  = 0
			philosophers[i].firstForkId = NB_NODES - 1
		} else {
			philosophers[i].firstForkId  = i
			philosophers[i].secondForkId = i + 1
		}
		// Initially forks are in the hand of the philosophers with the same id
		// Initially all forks are dirty
		philosophers[i].firstForkStatus  = true
		philosophers[i].firstForkClean  = false
		philosophers[i].firstForkRequested = false
		
		philosophers[i].secondForkStatus = false
		philosophers[i].secondForkClean  = false
		philosophers[i].secondForkRequested = false

		// Break the symetry of the initialization otherwise the system can deadlock at startup
		if (i == NB_NODES - 1) {
			philosophers[i].secondForkStatus = true
		} else if (i == 0) {
			philosophers[i].firstForkStatus  = false
		}
		
		philosophers[i].channel = messages[i]
		messages[i] = make(chan string)
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
}
