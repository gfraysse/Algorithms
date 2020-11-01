/*
  Copyright "Guillaume Fraysse <gfraysse dot spam plus code at gmail dot com>"

TODO: 
- finalize implementation

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
	"math/rand"
	"sync"
	"time"
)

/* global variable declaration */
var NB_JOBS           int = 4
var REQUEST_SIZE      int = 2
var NB_ITERATIONS     int = 10
var CURRENT_ITERATION int = 0

var MAX_SLOTS         = 50

var REQ_TYPE   int = 0
var REP_TYPE   int = 1
/*
// Debug function
func displayJobs() {
	for i := 0; i < len(jobs); i++ {
		log.Print("Job #", jobs[i].id, ", last=", jobs[i].last, ", next=", jobs[i].next)
	}
}
*/
type Request struct {
	requesterJobId int
	requestId      int
	messageType    int
	resourceId     []int
}

type Position struct {
	level int
	slot  int
}

type JobSet struct {
	jobid     int
	position  Position
	resources []int
}

type Job struct {
	// From the algorithm
	id         int
	compete    []JobSet
	position   Position
	positions   []Position
	imbalance  []int
	// Implementation specific
	nbCS       int // the number of time the node entered its Critical Section
	queue      []Request
	messages   []chan []byte
}

func UnmarshalRequest(text []byte, request *Request) error {
	request.requesterJobId = int(text[0])
	request.requestId      = int(text[1])
	request.messageType    = int(text[2])
	request.resourceId = make ([]int, REQUEST_SIZE)
	
	for i := 0; i < REQUEST_SIZE; i++ {
		request.resourceId[i] = int(text[3 + i])
	}
	
	return nil
}

func MarshalRequest(request Request) ([]byte, error) {
	var ret = make ([]byte, 3 + REQUEST_SIZE)

	ret[0] = byte(request.requesterJobId)
	ret[1] = byte(request.requestId)
	ret[2] = byte(request.messageType)
	for i := 0; i < REQUEST_SIZE; i++ {
		ret[3 + i] = byte(request.resourceId[i])
	}
	return ret, nil
}

func removeFromJobSet(slice []JobSet, i int) []JobSet {
	for j := 0; j< len(slice); j ++{
		if slice[j].jobid == i {
			return append(slice[:j], slice[j+1:]...)
		}
	}
    return slice
}

func (job *Job) String() string {
	var val string
	val = fmt.Sprintf("Job #%d Position.level=%d, Position.slot=%d\n",
		job.id,
		job.position.level,
		job.position.slot)
	return val
}

func (job *Job) enterCS() {
	log.Print("Job #", job.id, " ######################### enterCS")
	CURRENT_ITERATION ++
	job.nbCS ++
	// log.Print(n)
	time.Sleep(500 * time.Millisecond)
}

func (job *Job) releaseCS() {
	log.Print("Job #", job.id," releaseCS #########################")	
	// log.Print(n)
}

// does p2 obstruct p1 ?
func obstructs(p1 Position, p2 Position) bool {
	if (p1.level == p2.level) {
		if (p2.slot == p1.slot || p2.slot == p1.slot +1) {
			return true
		}
	}
	if (p2.level == p1.level + 1 && p2.slot == 0) {
		return true
	}
	if (p1.level == p2.level + 1 && p1.slot == 0 && p2.slot > 1) {
		return true
	}
	
	return false
}

func (job *Job) schedule(receivedCompete []JobSet) {
	log.Print("Job #", job.id," schedule")	
	var L int = 0
	for i := 0; i < len(receivedCompete); i ++ {
		job.compete[receivedCompete[i].jobid].position.level = 0		
		job.compete[receivedCompete[i].jobid].position.slot = MAX_SLOTS
		if receivedCompete[i].position.level + 1 > L {
			L = receivedCompete[i].position.level + 1
		}
		job.compete[receivedCompete[i].jobid].resources = receivedCompete[i].resources
	}
	job.position.level = L
	job.position.slot = 0
	job.announce()
}

func (job *Job) done() {
	log.Print("Job #", job.id," done")	
	job.position.level = 0
	job.position.slot = -1
	job.rebalance()
}

func (job *Job) report(k int, P Position) {
	log.Print("Job #", job.id," report")	
	var jk JobSet
	jk.jobid = k
	job.compete = append (job.compete, jk)
	job.imbalance[k] --
	job.position = P
	if (job.imbalance[k] == 0) {
		if (P.level == 0 && P.slot == -1) {
			removeFromJobSet (job.compete, k)
			for i := 0; i < len(job.compete); i ++ {
				if(job.imbalance[i] <= 0 && job.position.level > 0 && job.position.slot > 0) {
					job.advance()
					job.rebalance()
					job.announce()
				}
			}
		}
	} else {
		job.imbalance[k] = -1
		if (P.level == job.position.level && P.slot != job.position.slot + 1) || (job.position.slot == 0 && P.level == job.position.level + 1) {
			job.inform(k)
		}
	}
}

func inSlice (slice []int, v int) bool {
	for i := 0; i < len(slice); i ++ {
		if slice[i] == v {
			return true
		}
	}
	return false
}

func minIntersect(s1 []int, s2 []int) int {
	var min int = MAX_SLOTS
	for i := 0; i < len(s1); i ++ {
		if inSlice(s2, s1[i]) {
			if s1[i] < min {
				min = s1[i]
			}
		}
	}
	return min
}

func (job *Job) advance() {
	log.Print("Job #", job.id," advance")	
	if job.position.slot > 0 {
		job.position.slot --
	} else {
		job.position.level --

		var s string = fmt.Sprintf("%b", job.id)
		var bit int = int(s[job.position.level] - '0')
		var n int = (bit * 2) % 4
		var proper []int
		for i := 0; i  < MAX_SLOTS; i ++ {
			if i % n == 0{
				proper = append(proper, i)
			}
		}
		var same []int
		var filled []int
		// var minFilled int = MAX_SLOTS + 1
		// var maxFilled int = 0
		for i := 0; i < len(job.compete); i ++ {
			if job.compete[i].position.level == job.position.level {
				same = append(same, job.compete[i].jobid)
				filled = append(filled, job.compete[i].position.slot)
				// if job.compete[i].slot > maxFilled {
				// 	maxFilled = job.compete[i].slot
				// }
				// if job.compete[i].slot < minFilled {
				// 	minFilled = job.compete[i].slot
				// }
			}
		}

		var free []int
		for i := 0; i <= MAX_SLOTS; i ++ {
			if !inSlice(filled, i) && !inSlice(filled, i + 1) {
				free = append(free, i)
			}
		}
		
		job.position.slot = minIntersect(free, proper)
	}
}

func (job *Job) rebalance() {
	log.Print("Job #", job.id," rebalance")	
	for i := 0; i < len(job.compete); i ++ {
		var k JobSet = job.compete[i]
		if job.imbalance[k.jobid] == -1 {
			job.inform(k.jobid)
		}
	}
}

func (job *Job) sendExecute() {
	// to myself ... so nothing to do but enter
	log.Print("Job #", job.id," sendExecute")	
	job.enterCS()
	job.releaseCS()	
}

func (job *Job) announce() {
	log.Print("Job #", job.id," announce")	
	log.Print(job)
	if job.position.level == 0 && job.position.slot == 0 {
		job.sendExecute()
	} else {
		for i := 0; i < len(job.compete); i ++ {
			if obstructs(job.position, job.compete[i].position) {
				job.inform(job.compete[i].jobid)
			}
		}
	}
}

func (job *Job) inform(k int) {
	log.Print("Job #", job.id," inform")	
	job.sendReport(k, job.id, job.position)
	job.imbalance[k] ++
}

func (job *Job) sendReport(k int, jobId int, position Position) {
	var report Request
	content, err := MarshalRequest(report)
	if err != nil {
		log.Fatal(err)
	}			
	for i := 0; i < len(job.messages); i++ {
		if i == k {
			// var content = fmt.Sprintf("REP%d%d%d", job.id, position.level, position.slot)
			log.Print("Job #", job.id, ",  REPORT ", content, " with position #", position.level, ".", position.slot, " to Job #", k)	
			job.messages[i] <- content
		}
	}
}

func (job *Job) waitForReplies() {	
	// log.Print("Job #", job.id," waitForReplies")	
	for {
		select {
		case msg := <-job.messages[job.id]:
			var request Request
			err := UnmarshalRequest(msg, &request)
			if err != nil {
				log.Fatal(err)
			}			
			if (request.messageType == REQ_TYPE) {
				var requester = request.requesterJobId
				var res = request.resourceId
				log.Print("Node #", job.id, "<-REQ#", request.requestId, ", Requester #", requester, ", nb of res:", len(res))
				var jobset JobSet
				jobset.jobid = 0
				jobset.resources = res
				job.schedule(job.compete)
			} else if (request.messageType == REP_TYPE) {
				var requester = request.requesterJobId
				log.Print("Job #", job.id, ", received REPORT from Job #", requester, ",", msg)
			} else {
				log.Fatal("Fatal Error")
			}
		}
	}
	// log.Print(n)
	// log.Print("Job #", job.id, " end waitForReplies")
}

func (job *Job) sendRequest(request Request) {
	content, err := MarshalRequest(request)
	if err != nil {
		log.Fatal(err)
	}			
	for i := 0; i < len(request.resourceId); i++ {
		for j := 0; j < len(job.messages); j++ {
			if i == j {
				// var content = fmt.Sprintf("REQ%d%d%d", job.id, request.resourceId[0], request.resourceId[1])
				log.Print("Job #", job.id, ",  REQUEST #", request.requestId, ":", content, " for resources #", request.resourceId[0], ", ", request.resourceId[1], " to Job #", j)	
				job.messages[j] <- content
			}
		}
	}
}

func (job *Job) requestCS() {
	// log.Print("Job #", job.id, " requestCS")

	for {
		time.Sleep(100 * time.Millisecond)

		for i := 0; i < NB_JOBS; i ++ {
			if (i != job.id) {
				var request Request
				var resources = make([]int, NB_JOBS)
				for j := 0; j < NB_JOBS; j++ {
					resources[j] = j
				}
				request.requesterJobId = job.id
				request.resourceId = make([]int, REQUEST_SIZE)
				for k := 0; k < REQUEST_SIZE; k++ {
					rand.Seed(time.Now().UnixNano())
					var idx int = rand.Intn(len(resources))
					request.resourceId[k] = resources[idx]
					// remove element from array to avoid requesting it twice
					// changes order, but who cares ?
					resources[idx] = resources[len(resources) - 1]
					resources[len(resources) - 1] = 0 
					resources = resources[:len(resources) - 1]
				}
				job.sendRequest(request)
			}
		}
		for {
			time.Sleep(100 * time.Millisecond)
			// if (job.outstandingReplyCount == 0) {
			// 	job.enterCS()
			// 	job.releaseCS()
			// 	break
			// }
		} 
	}	
	// log.Print("Job #", job.id," END")	
}

func (job *Job) AwerbuchSaks(wg *sync.WaitGroup) {
	log.Print("Job #", job.id)

	go job.requestCS()
	go job.waitForReplies()
	for {
		time.Sleep(100 * time.Millisecond)
		if CURRENT_ITERATION > NB_ITERATIONS {
			break
		}
	}

	log.Print("Job #", job.id," END after ", NB_ITERATIONS," CS entries")	
	wg.Done()
}

func main() {
	var jobs = make([]Job, NB_JOBS)
	var wg sync.WaitGroup
	var messages = make([]chan []byte, NB_JOBS)
	
	log.Print("nb_process #", NB_JOBS)

	// Initialization
	for i := 0; i < NB_JOBS; i++ {
		jobs[i].id = i
		jobs[i].nbCS = 0 

		messages[i] = make(chan []byte)
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
