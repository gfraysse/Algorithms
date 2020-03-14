/*
  Copyright "Guillaume Fraysse <gfraysse dot spam plus code at gmail dot com>"

TODO: 
- everything

How-to run: 
  go run awerbuch-saks.go 2>&1 |tee /tmp/tmp.log

Terminology
* A scheduler is any computing device which runs the Awerbuch-Saks algorithm

Parameters:
- Number of jobs is set with NB_JOBS global variable
- Number of CS entries is set with NB_ITERATIONS global variable
*/ 

/*
    Go implementation of Awerbuch-Saks mutual exclusion algorithm, dynamic job scheduling
    Algorithm by Baruch Awerbuch and Michael Saks 1990

References : 
 * https://doi.org/10.1109/FSCS.1990.89525 : Awerbuch, Baruch, and Mike Saks. "A dining philosophers algorithm with polynomial response time." Proceedings [1990] 31st Annual Symposium on Foundations of Computer Science. IEEE, 1990.
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
var NB_JOBS int = 4
var NB_ITERATIONS int = 10
var CURRENT_ITERATION int = 0

/*
// Debug function
func displayJobs() {
	for i := 0; i < len(jobs); i++ {
		log.Print("Job #", jobs[i].id, ", last=", jobs[i].last, ", next=", jobs[i].next)
	}
}
*/
type Request struct {
	nodeId         int
}

type Position struct {
	level int
	slot  int
}

type JobSet struct {
	jobs     []Job
	position Position
}

type Job struct {
	// From the algorithm
	id         int
	compete    JobSet
	position   Position
	imbalance  []int
	// Implementation specific
	nbCS       int // the number of time the node entered its Critical Section
	queue      []Request
	channel    chan string
	messages   []chan string
}

func (j *Job) String() string {
	var val string
	val = fmt.Sprintf("Job #%d, highestSeqNumber=%d, outstandingReplyCount=%d \n",
		j.id,
		j.highestSeqNumber,
		j.outstandingReplyCount)
	return val
}

func (j *Job) enterCS() {
	log.Print("Job #", j.id, " ######################### enterCS")
	CURRENT_ITERATION ++
	j.nbCS ++
	// log.Print(n)
	time.Sleep(500 * time.Millisecond)
}

func (j *Job) releaseCS() {
	log.Print("Job #", j.id," releaseCS #########################")	
	// log.Print(n)
}

func obstructs(p1 Position, p2 Position) {
	return false
}

func (j *Job) schedule(compete JobSet) {
	var L int = 0
	for i := 0; i < len(j.compete); i ++ {
		j.compete[j].position.level = 0
		j.compete[j].position.slot = 0
		if j.compete[j].position.level + 1 > L {
			L = j.compete[j].position.level + 1
		}
	}
	j.position.level = L
	j.position.slot = 0
	j.announce()
}

func (j *Job) done() {
	j.position.level = 0
	j.position.slot = -1
	j.rebalance()
}

func (j *Job) report(k int, P Position) {
}

func (j *Job) advance() {
	if j.position.slot > 0 {
		j.position.slot --
	} else {
		j.position.level --
		j.proper = 0
		j.same = 0
		j.filled = 0
		j.free = 0
		j.position.slot = minFreeAndProper(j)
	}
}

func (j *Job) rebalance() {
	for i := 0; i < len(j.compete); i ++ {
		k = j.compete[i]
		if imbalance(k) == -1 {
			inform(k)
		}
	}
}

func (j *Job) announce() {
	if j.position.level == 0 && j.position.slot == 0 {
		sendExecute()		
	} else {
		for i := 0; i < len(j.compete); i ++ {
			if obstructs(j.position, j.compete[i].position) {
				inform(j.compete[i])
			}
		}
	}
}

func (j *Job) inform(k int) {
	sendReport(k, j.id, j.position)
	j.imbalance[k] ++
}

func (j *Job) sendReport(k int, jobId int, position int) {
	for i := 0; i < len(j.messages); i++ {
		if i == k {
			var content = fmt.Sprintf("REP%d%d", j.id, position)
			log.Print("Job #", j.id, ",  REPORT ", content, " with position #", position, " to Job #", k)	
			j.messages[i] <- content
		}
	}
}

func (j *Job) sendReply(destJobId int) {
	var content = fmt.Sprintf("REP%d", j.id)
	log.Print("Job #", j.id, ", SENDING reply ", content, " to Job #", destJobId)	
	j.messages[destJobId] <- content
}

func (j *Job) waitForReplies() {	
	// log.Print("Job #", j.id," waitForReplies")	
	for {
		select {
		case msg := <-j.messages[j.id]:
			if (strings.Contains(msg, "REQ")) {
				// requester is the variable j in the paper
				var requester, err = strconv.Atoi(msg[3:4])
				if err != nil {
					log.Fatal(err)
				}
				// k is seqNumber,
				// k is the name of the variable in the paper
				var k, err2 = strconv.Atoi(msg[4:])
				if err2 != nil {
					log.Fatal(err2)
				}

				if k > j.highestSeqNumber {
					j.highestSeqNumber = k
				}
				var defer_it bool = j.isRequestingCS && ((k > j.seqNumber) || (k == j.seqNumber && requester > j.id))
				if defer_it {
					j.replyDeferred[requester] = true
				} else {
					j.sendReply(requester)
				}
			}  else if (strings.Contains(msg, "REP")) {
				var sender, err = strconv.Atoi(msg[3:4])
				if err != nil {
					log.Fatal(err)
				}
				log.Print("Job #", j.id, ", RECEIVED reply from Job #", sender, ",", msg)
				j.outstandingReplyCount --
			} else {
				log.Fatal("WTF")
			}
		}
	}
	// log.Print(n)
	// log.Print("Job #", j.id, " end waitForReplies")
}

func (j *Job) requestCS() {
	// log.Print("Job #", j.id, " requestCS")

	for {
		time.Sleep(100 * time.Millisecond)
		// Mutex on shared variable
		j.isRequestingCS = true
		j.seqNumber = j.highestSeqNumber + 1
		// end mutex on shared variable
		j.outstandingReplyCount = NB_JOBS - 1

		for j := 0; j < NB_JOBS; j ++ {
			if (j != j.id) {
				j.sendRequest(j.seqNumber, j.id, j)
			}
		}
		for {
			time.Sleep(100 * time.Millisecond)
			if (j.outstandingReplyCount == 0) {
				j.enterCS()
				j.releaseCS()
				break
			}
		} 
	}	
	// log.Print("Job #", j.id," END")	
}

func (j *Job) AwerbuchSaks(wg *sync.WaitGroup) {
	log.Print("Job #", j.id)

	go j.requestCS()
	go j.waitForReplies()
	for {
		time.Sleep(100 * time.Millisecond)
		if CURRENT_ITERATION > NB_ITERATIONS {
			break
		}
	}

	log.Print("Job #", j.id," END after ", NB_ITERATIONS," CS entries")	
	wg.Done()
}

func main() {
	var jobs = make([]Job, NB_JOBS)
	var wg sync.WaitGroup
	var messages = make([]chan string, NB_JOBS)
	
	log.Print("nb_process #", NB_JOBS)

	// Initialization
	for i := 0; i < NB_JOBS; i++ {
		jobs[i].id = i
		jobs[i].nbCS = 0 

		jobs[i].channel = messages[i]
		messages[i] = make(chan string)
	}
	for i := 0; i < NB_JOBS; i++ {
		jobs[i].messages = messages
	}

	// start
	for i := 0; i < NB_JOBS; i++ {
		wg.Add(1)
		go jobs[i].AwerbuchSaks(&wg)
	}

	// end
	wg.Wait()
	for i := 0; i < NB_JOBS; i++ {
		log.Print("Job #", jobs[i].id," entered CS ", jobs[i].nbCS, " time")	
	}
}
/* Pseudo-code for original article
program RECEIVE(C)

C = Schedule(j, Compete):
  forall k in Compete : Position(k) <- (bit(j, k), infinity)
  L <- max{Level(k) + 1) | k in Compete}
  Position <- (L, 0)
  ANNOUNCE

C = Done:
  Position <-(0,-1)
  REBALANCE

C = Report(k, P):
  Compete <- Compete + {k}
  Imbalance(k) <- Imbalance(k) - 1
  Position(k) <- P
  if (Imbalance(k) == 0)
    if (P == (0, -1)) Compete <- Compete - {k}
    while ((forall k in Compete) Imbalance(k) <= 0)
    and (Position > (0,O))
      ADVANCE
      REBALANCE
      ANNOUNCE
  else Imbalance(k) = -1
    if (P != Position + 1) then
      INFORM (k)

procedure ADVANCE
  if (Slot > 0) Slot <- Slot - 1
  else
    Level <- Level - 1
    Proper <- {T | T congruent (2 . ID[Level]) mod 4}
    Same <- {k | k in Competel & Level(k) == Level}
    Filled <- {Slot(k) | k in Same}
    Free <- {T >= 0 & {T, T + 1) notin Filled}
    Slot <- min {T | T in Free & Proper }

procedure REBALANCE
  forall k in Compete:
    if (Imbalance(k) == -1) INFORM (k)

procedure ANNOUNCE
  if (Position = (0, 0)) then
    SEND Execute
  else forall k in Compete
    if (Position(k) obstructs Position)
      INFORM (k)

procedure INFORM (k)
  SEND Report(j, Position) to p_k
  Imbalance(k) <- Imbalance(k) + 1
*/
