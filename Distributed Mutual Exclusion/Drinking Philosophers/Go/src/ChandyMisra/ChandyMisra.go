/*
  Copyright "Guillaume Fraysse <gfraysse dot spam plus code at gmail dot com>"

How-to run: 
  go run drinking_philosophers_chandy-misra.go 2>&1 |tee /tmp/tmp.log

Parameters:
- Number of nodes is set with NB_NODES global variable
- Number of iterations is hardcoded in main function
*/ 

/*
    Go implementation of Chandy-Misra mutual exclusion algorithm

References :
* https://www.cs.utexas.edu/users/misra/scannedPdf.dir/DrinkingPhil.pdf: Chandy, K.M.; Misra, J. (1984). The Drinking Philosophers Problem. ACM Transactions on Programming Languages and Systems.
* https://en.wikipedia.org/wiki/Dining_philosophers_problem#Chandy/Misra_solution

Message complexity is O(Log(n)) where N is the number of philosophers
*/

package ChandyMisra

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"strings"
	"strconv"
	"time"
)

/* global variable declaration */
var NB_NODES          int = 4
var NB_MSG            int = 0
var NB_ITERATIONS     int = 50
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
			log.Print("  P#", philosophers[i].Id, ", fork #", philosophers[i].ForkId[j], ", status=", philosophers[i].ForkStatus[j], ", clean=", philosophers[i].ForkClean[j])
		}
	}
}
*/

func checkSanity() {
	for i := 0; i < NB_NODES; i++ {
		// Sanity check, it I don't have a fork, check if the owner actually has it, if I have it check it is not owned by the other
		for j := 0; j < NB_NODES - 1; j ++ {
			var idx int = philosophers[i].ForkId[j]
			for k := 0; k < NB_NODES - 1; k ++ {
				if philosophers[i].ForkId[j] == philosophers[idx].ForkId[k] {
					if philosophers[i].ForkStatus[j] == philosophers[idx].ForkStatus[k] {
						log.Print("ERR Sanity Check expected philosopher #", i, " fork#", j, " status = ", philosophers[i].ForkStatus[j], ", philosopher#", idx, ", fork #", k, " status=", philosophers[idx].ForkStatus[k])
					}
					break
				}
			}
		}
	}
}

type ForkRequest struct {
	PhilosopherId int
	ForkId        int
}

type Philosopher struct {
	Id              int
	Initialized     bool
	ForkId          []int
	ForkClean       []bool
	ForkStatus      []bool
	State           int
	NbCS            int
	Queue           []ForkRequest
	Messages        []chan string
}

func (p *Philosopher) String() string {
	var val string
	val = fmt.Sprintf("Philosopher #%d, state=%d, first fork=%d/%v/%v, second fork=%d/%v/%v, third fork=%d/%v/%v, my fork=%d/%v\n",
		p.Id,
		p.State,
		p.ForkId[0],
		p.ForkClean[0],
		p.ForkStatus[0],
		p.ForkId[1],
		p.ForkClean[1],
		p.ForkStatus[1],
		p.ForkId[2],
		p.ForkClean[2],
		p.ForkStatus[2])
	return val
}

func (p *Philosopher) EnterCS() {
	log.Print("Philosopher #", p.Id, " ######################### enterCS")
	p.State = STATE_EATING
	p.NbCS ++
	CURRENT_ITERATION ++
	time.Sleep(500 * time.Millisecond)
	checkSanity()
}

func (p *Philosopher) ReleaseCS() {
	log.Print("Philosopher #", p.Id," ReleaseCS #########################")	
	p.State = STATE_THINKING
	for i := 0; i < NB_NODES - 1; i ++ {
		p.ForkClean[i] = false
	}
	checkSanity()
}

func (p *Philosopher) requestFork(philosopherId int) {
	for i := 0; i < len(p.Messages); i++ {
		if i == philosopherId {
			var content = fmt.Sprintf("REQ%.2d", p.Id)
			// log.Print(p)
 			// log.Print("Philosopher #", p.Id, ", SENDING request ", content, " to Philosopher #", i)	
			log.Print(p.Id, " --", i, "--> ", i)	
			p.Messages[i] <- content
			NB_MSG ++
		}
	}
}

func (p *Philosopher) sendFork(philosopherId int) {
	for i := 0; i < len(p.Messages); i++ {
		if i == philosopherId {
			var content = fmt.Sprintf("REP%.2d", p.Id)
			// log.Print("Philosopher #", p.Id, ", SENDING fork ", content, " to Philosopher #", philosopherId)	
			log.Print(p.Id,": ", p.Id, " ====> ", philosopherId)	
			p.Messages[i] <- content
			NB_MSG ++
		}
	}
}

func (p *Philosopher) waitForReplies() {	
	log.Print("Philosopher #", p.Id," waitForReplies")	
	for {
		select {
		case msg := <-p.Messages[p.Id]:
			checkSanity()
			if (strings.Contains(msg, "REQ")) {
				var requester, err = strconv.Atoi(msg[3:5])
				if err != nil {
					log.Fatal(err)
				}
				for i := 0; i < NB_NODES - 1; i ++ {
					if requester == p.ForkId[i] {
						if p.ForkStatus[i] == true {
							if p.ForkClean[i] == true {
								// keep the fork
								log.Print("Philosopher #", p.Id,", fork#", i, " is clean, I keep it for now")
								var r ForkRequest
								r.PhilosopherId = requester
								r.ForkId        = requester
								p.Queue         = append(p.Queue, r)
							} else {
								p.ForkStatus[i]    = false
								p.ForkStatus[i]    = false
								p.sendFork(requester)
							}						
							break
						} else {
							log.Print("Philosopher #", p.Id,", DOES NOT own fork#", i)
						}
					}
				}
			}  else if (strings.Contains(msg, "REP")) {
				var sender, err = strconv.Atoi(msg[3:5])
				if err != nil {
					log.Fatal(err)
				}
				log.Print("Philosopher #", p.Id, ", RECEIVED fork from Philosopher #", sender, ", ", msg)
				log.Print(sender, ": ", p.Id, " <==== ", sender)	
				for i := 0; i < NB_NODES - 1; i ++ {
					if (sender == p.ForkId[i]) {
						p.ForkStatus[i]    = true
						p.ForkClean[i]     = true
						break
					}
				}
				var hasSentReq bool = false
				log.Print("Philosopher #", p.Id, ", checking if forks are missing")
				for j := 0; j < NB_NODES - 1; j++ {
					if p.ForkStatus[j] == false {
						p.requestFork(p.ForkId[j])
						hasSentReq = true
						break
					}
				}
				if hasSentReq == false {
					log.Print("Philosopher #", p.Id, ", has not requested any fork")
				}
					
			} else {
				log.Fatal("WTF")
			}
		}
	}
}

func (p *Philosopher) RequestCS() {
	log.Print("Philosopher #", p.Id, " RequestCS")

	for {
		time.Sleep(100 * time.Millisecond)
		if p.State == STATE_THINKING {
			rand.Seed(time.Now().UnixNano())
			var proba int = rand.Intn(2)
			// log.Print("Proba =", proba)
			if proba < 1 {
				p.State = STATE_HUNGRY
				log.Print("Philosopher #", p.Id, " wants to enter CS")
				for j := 0; j < NB_NODES - 1; j++ {
					if p.ForkStatus[j] == false {
						p.requestFork(p.ForkId[j])
						break
					} else {
						p.ForkClean[j] = true
					}
				}
			} else {
				// log.Print("Philosopher #", p.Id, " DOES NOT want to enter CS")
			}
		} else if p.State == STATE_HUNGRY {
			var allGreen = true

			for i := 0; i < NB_NODES - 1; i ++ {
				allGreen = allGreen && p.ForkStatus[i] && p.ForkClean[i]
				if allGreen == false {
					// log.Print("Philosopher #", p.Id, " waiting for fork", p.ForkId[i])
					// displayNodes()
					break
				}
			}
			if (allGreen == true) {
				if (p.State == STATE_EATING) {
					log.Print("** Philosopher #", p.Id, " is already eating **")
				} else {
					p.EnterCS()
					p.ReleaseCS()
					for i := 0; i < len(p.Queue); i++ {
						var r ForkRequest
						r = p.Queue[i]
						for j := 0; j < NB_NODES - 1; j++ {
							if (r.PhilosopherId == p.ForkId[j] && p.ForkStatus[j] == true) {
								p.ForkStatus[j] = false
								p.sendFork(r.PhilosopherId)
								break
							}
						}
					}
					p.Queue = nil
					for j := 0; j < NB_NODES - 1; j++ {
						if (p.ForkStatus[j] == true) {
							p.ForkStatus[j] = false
							p.sendFork(p.ForkId[j])
						}
					}
				}
			}
		} else {
			log.Print("already thinking")
		}
	}

	log.Print("Philosopher #", p.Id," END")	
}

func (p *Philosopher) ChandyMisra(wg *sync.WaitGroup) {
	log.Print("Philosopher #", p.Id)

	go p.RequestCS()
	go p.waitForReplies()
	for {
		time.Sleep(100 * time.Millisecond)
		if CURRENT_ITERATION == NB_ITERATIONS {
			break
		}
	}

	log.Print("Philosopher #", p.Id," END after ", NB_ITERATIONS," CS entries")	
	wg.Done()
}

func main() {
	// var philosophers = make([]Philosopher, NB_NODES)
	philosophers = make([]Philosopher, NB_NODES)
	var wg sync.WaitGroup
	var messages  = make([]chan string, NB_NODES)
	
	log.Print("nb_process #", NB_NODES)
	
	for i := 0; i < NB_NODES; i++ {
		philosophers[i].Id = i
		philosophers[i].NbCS = 0
		philosophers[i].State = STATE_THINKING
		philosophers[i].ForkId  = make([]int, NB_NODES - 1)
		philosophers[i].ForkStatus  = make([]bool, NB_NODES - 1)
		philosophers[i].ForkClean  = make([]bool, NB_NODES - 1)
		var idx int = 0
		for j := 0; j < NB_NODES; j++ {
			if j == philosophers[i].Id {
				// skip my own ID
				continue
			} else {
				philosophers[i].ForkId[idx] = j
				idx++
			}
		}
		// Initially forks are in the hand of the philosophers with id lower than the fork id to make graphs acyclic
		// Initially all forks are dirty
		for j := 0; j < NB_NODES - 1; j++ {
			if philosophers[i].ForkId[j] < i {
				philosophers[i].ForkStatus[j]    = false
			} else {
				philosophers[i].ForkStatus[j]    = true
			}
			philosophers[i].ForkClean[j]     = false
		}

		messages[i] = make(chan string)
		philosophers[i].Initialized = true
	}

	for i := 0; i < NB_NODES; i++ {
		philosophers[i].Messages = messages
	}

	for i := 0; i < NB_NODES; i++ {
		wg.Add(1)
		go philosophers[i].ChandyMisra(&wg)
	}
	wg.Wait()
	for i := 0; i < NB_NODES; i++ {
		log.Print("Philosopher #", philosophers[i].Id," entered CS ", philosophers[i].NbCS," time")	
	}
	log.Print(NB_MSG, " messages sent")
}
