/*
  Copyright "Guillaume Fraysse <gfraysse dot spam plus code at gmail dot com>"

How-to run: 
  go run ChandyMisra.go 2>&1 |tee /tmp/tmp.log

Parameters:
- Number of nodes
- Number of iterations
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
	"sync"
	"strings"
	"strconv"
	"time"
)

/* global variable declaration */
var NB_MSG            int = 0
var CURRENT_ITERATION int = 0

var STATE_THINKING    int = 0
var STATE_HUNGRY      int = 1
var STATE_EATING      int = 2

var Philosophers []Philosopher

// Debug function
/*
func displayNodes() {
	for i := 0; i < NB_NODES; i++ {
		for j := 0; j < NB_NODES - 1; j++ {
			log.Print("  P#", Philosophers[i].Id, ", fork #", Philosophers[i].ForkId[j], ", status=", Philosophers[i].ForkStatus[j], ", clean=", Philosophers[i].ForkClean[j])
		}
	}
}
*/

func checkSanity() {
	for i := 0; i < Philosophers[0].NbNodes; i++ {
		// Sanity check, it I don't have a fork, check if the owner actually has it, if I have it check it is not owned by the other
		for j := 0; j < Philosophers[0].NbNodes - 1; j ++ {
			var idx int = Philosophers[i].ForkId[j]
			for k := 0; k < Philosophers[0].NbNodes - 1; k ++ {
				if Philosophers[i].ForkId[j] == Philosophers[idx].ForkId[k] {
					if Philosophers[i].ForkStatus[j] == Philosophers[idx].ForkStatus[k] {
						log.Print("ERR Sanity Check expected philosopher #", i, " fork#", j, " status = ", Philosophers[i].ForkStatus[j], ", philosopher#", idx, ", fork #", k, " status=", Philosophers[idx].ForkStatus[k])
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
	Id           int
	Initialized  bool
	ForkId       []int
	ForkClean    []bool
	ForkStatus   []bool
	State        int
	NbCS         int
	Queue        []ForkRequest
	Messages     []chan string
	NbNodes      int
	NbIterations int
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
	log.Print("Philosopher #", p.Id, " ######################### Philosopher.EnterCS")
	p.State = STATE_EATING
	p.NbCS ++
	CURRENT_ITERATION ++
	checkSanity()
}

func (p *Philosopher) ExecuteCSCode() {
	log.Print("Philosopher #", p.Id, " ######################### Philosopher.ExecuteCSCode")
	time.Sleep(500 * time.Millisecond)
}

func (p *Philosopher) ReleaseCS() {
	log.Print("Philosopher #", p.Id," Philosopher.ReleaseCS #########################")	
	p.State = STATE_THINKING
	for i := 0; i < p.NbNodes - 1; i ++ {
		p.ForkClean[i] = false
	}
	checkSanity()
}

func (p *Philosopher) RequestFork(philosopherId int) {
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

func (p *Philosopher) SendFork(philosopherId int) {
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

func (p *Philosopher) enterCSIfICan() {	
	var hasSentReq bool = false
	log.Print("Philosopher #", p.Id, ", checking if forks are missing")
	for j := 0; j < p.NbNodes - 1; j++ {
		if p.ForkStatus[j] == false {
			go p.RequestFork(p.ForkId[j])
			hasSentReq = true
			break
		}
	}
	if hasSentReq == false {
		log.Print("Philosopher #", p.Id, ", has not requested any fork")
		if p.State == STATE_HUNGRY {
			var allGreen = true
			
			for i := 0; i < p.NbNodes - 1; i ++ {
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
					p.ExecuteCSCode()
					p.ReleaseCS()
					for i := 0; i < len(p.Queue); i++ {
						var r ForkRequest
						r = p.Queue[i]
						for j := 0; j < p.NbNodes - 1; j++ {
							if (r.PhilosopherId == p.ForkId[j] && p.ForkStatus[j] == true) {
								p.ForkStatus[j] = false
								go p.SendFork(r.PhilosopherId)
								break
							}
						}
					}
					p.Queue = nil
					for j := 0; j < p.NbNodes - 1; j++ {
						if (p.ForkStatus[j] == true) {
							p.ForkStatus[j] = false
							go p.SendFork(p.ForkId[j])
						}
					}
					p.RequestCS()								
				}
			}
		} else {
			log.Print("NOT HUNGRY")
		}
		
	}
}
func (p *Philosopher) WaitForReplies() {	
	log.Print("Philosopher #", p.Id," WaitForReplies")	
	for {
		select {
		case msg := <-p.Messages[p.Id]:
			checkSanity()
			if (strings.Contains(msg, "REQ")) {
				var requester, err = strconv.Atoi(msg[3:5])
				if err != nil {
					log.Fatal(err)
				}
				for i := 0; i < p.NbNodes - 1; i ++ {
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
								go p.SendFork(requester)
							}						
							break
						} else {
							log.Print("Philosopher #", p.Id,", DOES NOT own fork#", i)
						}
					}
				}
				p.enterCSIfICan()
			}  else if (strings.Contains(msg, "REP")) {
				var sender, err = strconv.Atoi(msg[3:5])
				if err != nil {
					log.Fatal(err)
				}
				log.Print("Philosopher #", p.Id, ", RECEIVED fork from Philosopher #", sender, ", ", msg)
				log.Print(sender, ": ", p.Id, " <==== ", sender)	
				for i := 0; i < p.NbNodes - 1; i ++ {
					if (sender == p.ForkId[i]) {
						p.ForkStatus[i]    = true
						p.ForkClean[i]     = true
						break
					}
				}
				p.enterCSIfICan()
			} else {
				log.Fatal("Unknown message", msg)
			}
		}
	}
}

func (p *Philosopher) RequestCS() {
	log.Print("Philosopher #", p.Id, " RequestCS")

	if p.State == STATE_THINKING {
		p.State = STATE_HUNGRY
		log.Print("Philosopher #", p.Id, " wants to enter CS")
		for j := 0; j < p.NbNodes - 1; j++ {
			if p.ForkStatus[j] == false {
				go p.RequestFork(p.ForkId[j])
				break
			} else {
				p.ForkClean[j] = true
			}
		}
	} else {
		log.Print("already eating")
	}
	
	log.Print("Philosopher #", p.Id," END RequestCS")	
}

func (p *Philosopher) ChandyMisra(wg *sync.WaitGroup) {
	log.Print("Philosopher #", p.Id)

	go p.RequestCS()
	go p.WaitForReplies()
	for {
		time.Sleep(100 * time.Millisecond)
		if CURRENT_ITERATION == p.NbIterations {
			break
		}
	}

	log.Print("Philosopher #", p.Id," END after ", p.NbIterations, " CS entries")	
	wg.Done()
}

func InitPhilosopher(p *Philosopher, id int, nbNodes int, nbIterations int) {
	p.Id = id
	p.NbCS = 0
	p.NbNodes = nbNodes
	p.NbIterations = nbIterations
	p.State = STATE_THINKING
	p.ForkId  = make([]int, nbNodes)
	p.ForkStatus  = make([]bool, nbNodes - 1)
	p.ForkClean  = make([]bool, nbNodes - 1)
	var idx int = 0
	for j := 0; j < nbNodes; j++ {
		if j == p.Id {
			// skip my own ID
			continue
		} else {
			p.ForkId[idx] = j
			idx++
		}
	}
	// Initially forks are in the hand of the Philosophers with id lower than the fork id to make graphs acyclic
	// Initially all forks are dirty
	for j := 0; j < nbNodes - 1; j++ {
		if p.ForkId[j] < id {
			p.ForkStatus[j]    = false
		} else {
			p.ForkStatus[j]    = true
		}
		p.ForkClean[j]     = false
	}
	
	p.Initialized = true
}

func Init(nbNodes int, nbIterations int) {
	log.Print("ChandyMisra.Init")	
	Philosophers = make([]Philosopher, nbNodes)
	var messages  = make([]chan string, nbNodes)
	
	log.Print("nb_process #", nbNodes)
	
	for i := 0; i < nbNodes; i++ {
		InitPhilosopher(&Philosophers[i], i , nbNodes, nbIterations)
		messages[i] = make(chan string)
	}

	for i := 0; i < nbNodes; i++ {
		Philosophers[i].Messages = messages
	}
}
