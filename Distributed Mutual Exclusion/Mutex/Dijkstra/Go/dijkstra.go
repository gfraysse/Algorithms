/*
  Copyright "Guillaume Fraysse <gfraysse dot spam plus code at gmail dot com>"

How-to run: 
  go run dijkstra.go 2>&1 |tee /tmp/tmp.log

Terminology
* A scheduler is any computing device which runs the Dijkstra's incremental algorithm

Parameters:
- Number of jobs is set with NB_JOBS global variable
- Number of CS entries is set with NB_ITERATIONS global variable
*/ 

/*
    Go implementation of Dijkstra dining philosophers algorithm
    Algorithm by Dijkstra 1971

References : 
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

var REQUEST_ID int = 0
var MAX_SLOTS  int = 50
var NO_NEXT    int = -1

var REQ_TYPE   int = 0
var REP_TYPE   int = 1
var FREE_TYPE  int = 2

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
	requester      int
	resourceId     []int
}

type Job struct {
	// From the algorithm
	id              int
	replyReceived   []bool
	resourcePresent bool
	pendingRequests []Request
	// Implementation specific
	nbCS          int // the number of time the node entered its Critical Section
	channel       chan []byte
	messages      []chan []byte
}


func (job *Job) String() string {
	var val string
	val = fmt.Sprintf("Job #%d\n",
		job.id)
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


func (job *Job) sendExecute() {
	// to myself ... so nothing to do but enter
	job.enterCS()
	job.releaseCS()	
}

func UnmarshalRequest(text []byte, request *Request) error {
	request.requesterJobId = int(text[0])
	request.requestId      = int(text[1])
	request.messageType    = int(text[2])
	request.requester      = int(text[3])
	request.resourceId = make ([]int, REQUEST_SIZE)
	
	for i := 0; i < REQUEST_SIZE; i++ {
		request.resourceId[i] = int(text[4 + i])
	}
	
	return nil
}

func MarshalRequest(request Request) ([]byte, error) {
	var ret = make ([]byte, 4 + REQUEST_SIZE)

	ret[0] = byte(request.requesterJobId)
	ret[1] = byte(request.requestId)
	ret[2] = byte(request.messageType)
	ret[3] = byte(request.requester)
	for i := 0; i < REQUEST_SIZE; i++ {
		ret[4 + i] = byte(request.resourceId[i])
	}
	return ret, nil
}

func getNextResourceForReq(request Request, current int) int {
	var next int = NO_NEXT

	for i := 0; i < REQUEST_SIZE; i++ {
		if request.resourceId[i] < current && request.resourceId[i] > next {
			next = request.resourceId[i]
		}
	}
	return next
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
				// var requester = request.requester
				// var res = request.resourceId
				// log.Print("Job #", job.id, ", Requester #", requester, ", nb of res:", len(res))
				if job.resourcePresent == true {
					var next_res = getNextResourceForReq(request, job.id)
					if next_res == NO_NEXT {
						var ack Request = request
						ack.messageType = REP_TYPE
						job.sendAck(ack)
					} else {
						job.sendRequest(request, next_res)
					}
				} else {
					log.Print("Job #", job.id, "Resource not available, appending request to pendinglist")
					job.pendingRequests = append(job.pendingRequests, request)
				}

			}  else if (request.messageType == REP_TYPE) {
				var requester = request.requester
				log.Print("Job #", job.id, ", received REPLY for REQ#", request.requestId, ", requester =", requester, ",", msg)
				job.replyReceived[request.requestId] = true
			}  else if (request.messageType == FREE_TYPE) {
				var requester = request.requester
				log.Print("Job #", job.id, ", received FREE for REQ#", request.requestId, ", requester =", requester, ",", msg)
				job.resourcePresent = true
				var r Request = job.pendingRequests[0]
				job.pendingRequests = job.pendingRequests[1:]

				var next_res = getNextResourceForReq(r, job.id)
				log.Print("next_res=", next_res)
				if next_res == NO_NEXT {
					var ack Request = r
					ack.messageType = REP_TYPE
					job.sendAck(ack)
				} else {
					job.sendRequest(r, next_res)
				}

			} else {
				log.Fatal("WTF")
			}
		}
	}
	// log.Print(n)
	// log.Print("Job #", job.id, " end waitForReplies")
}

func (job *Job) freeResources(r Request) {
	var freeRequest Request = r
	freeRequest.messageType = FREE_TYPE
	content, err := MarshalRequest(freeRequest)
	if err != nil {
		log.Fatal(err)
	}			
	for i := 0; i < REQUEST_SIZE; i++ {
		log.Print("Job #", job.id, ",  FREE #", r.requestId, ":", content, " for resources #", r.resourceId[0], ", #", r.resourceId[1], " to Job #", r.resourceId[i])	
		job.messages[r.resourceId[i]] <- content		
	}
}

func (job *Job) sendAck(r Request) {
	content, err := MarshalRequest(r)
	if err != nil {
		log.Fatal(err)
	}			
	// var content = fmt.Sprintf("REQ%d%d%d", job.id, request.resourceId[0], request.resourceId[1])
	log.Print("Job #", job.id, ",  REPLY#", r.requestId, ":", content, " for resources #", r.resourceId[0], ", #", r.resourceId[1], " to Job #", r.requester)	
	job.messages[r.requester] <- content
}

func (job *Job) sendRequest(request Request, destination int) {
	content, err := MarshalRequest(request)
	if err != nil {
		log.Fatal(err)
	}			
	// var content = fmt.Sprintf("REQ%d%d%d", job.id, request.resourceId[0], request.resourceId[1])
	log.Print("Job #", job.id, ",  REQUEST #", request.requestId, ":", content, " for resources #", request.resourceId[0], ", #", request.resourceId[1], " to Job #", destination)	
	job.messages[destination] <- content
}

func (job *Job) requestCS() {
	// log.Print("Job #", job.id, " requestCS")

	for {
		time.Sleep(100 * time.Millisecond)

		var request Request
		for i := 0; i < NB_JOBS; i ++ {
			if (i != job.id) {
				request.requesterJobId = job.id
				request.requestId = REQUEST_ID
				REQUEST_ID += 1
				request.requester = job.id
				job.replyReceived[request.requestId] = false
				
				var resources = make([]int, NB_JOBS)
				request.resourceId = make([]int, REQUEST_SIZE)
				// initialize a resources array with all
				// resources id, when selecting randomly
				// a resource we just remove it from the
				// array so that the array alsways contains
				// available resources
				for j := 0; j < NB_JOBS; j++ {
					resources[j] = j
				}
				var destination int = 0
				for k := 0; k < REQUEST_SIZE; k++ {
					rand.Seed(time.Now().UnixNano())
					var idx int = rand.Intn(len(resources))
					request.resourceId[k] = resources[idx]
					if resources[idx] > destination {
						destination = resources[idx]
					}
					// remove element from array to avoid requesting it twice
					// changes order, but who cares ?
					resources[idx] = resources[len(resources) - 1]
					resources[len(resources) - 1] = 0 
					resources = resources[:len(resources) - 1]
				}
				job.sendRequest(request, destination)
				break
			}
		}
		for {
			time.Sleep(100 * time.Millisecond)
			if (job.replyReceived[request.requestId] == true) {
				log.Print("Job #", job.id, " entering CS")
				job.enterCS()
				job.releaseCS()
				job.freeResources(request)
				break
			}
		} 
	}	
	// log.Print("Job #", job.id," END")	
}

func (job *Job) Dijkstra(wg *sync.WaitGroup) {
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
		jobs[i].resourcePresent = true
		jobs[i].nbCS = 0 

		jobs[i].channel = messages[i]
		messages[i] = make(chan []byte)
		jobs[i].replyReceived   = make([]bool, NB_JOBS * NB_ITERATIONS)
		jobs[i].pendingRequests = make([]Request, NB_JOBS * NB_ITERATIONS)
	}
	for i := 0; i < NB_JOBS; i++ {
		jobs[i].messages = messages
	}

	// start
	for i := 0; i < NB_JOBS; i++ {
		wg.Add(1)
		go jobs[i].Dijkstra(&wg)
	}

	// end
	wg.Wait()
	for i := 0; i < NB_JOBS; i++ {
		log.Print("Job #", jobs[i].id," entered CS ", jobs[i].nbCS, " time")	
	}
}
