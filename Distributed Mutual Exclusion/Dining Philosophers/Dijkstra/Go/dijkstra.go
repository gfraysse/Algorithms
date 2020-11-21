/*
  Copyright "Guillaume Fraysse <gfraysse dot spam plus code at gmail dot com>"

How-to run: 
  go run dijkstra.go 2>&1 |tee /tmp/tmp.log
or 
  go to the Go Playground: https://play.golang.org/p/KrlXVfFF-_x

Terminology
* A scheduler is any computing device which runs the Dijkstra's incremental algorithm

Parameters:
- Number of nodes is set with NB_NODES global variable
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
var NB_NODES          int = 4
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
func displayNodes() {
	for i := 0; i < len(nodes); i++ {
		log.Print("Node #", nodes[i].id, ", last=", nodes[i].last, ", next=", nodes[i].next)
	}
}
*/
type Request struct {
	requesterNodeId int
	requestId       int
	messageType     int
	resourceId      []int
}

type Node struct {
	// From the algorithm
	id              int
	replyReceived   []bool
	resourcePresent bool
	pendingRequests []Request
	// Implementation specific
	nbCS          int // the number of time the node entered its Critical Section
	messages      []chan []byte
}


func (node *Node) String() string {
	var val string
	val = fmt.Sprintf("Node #%d\n",
		node.id)
	return val
}

func (node *Node) enterCS() {
	log.Print("Node #", node.id, " ######################### enterCS")
	CURRENT_ITERATION ++
	node.nbCS ++
	// log.Print(n)
}

func (node *Node) executeCSCode() {
	log.Print("Node #", node.id, " ######################### executeCSCode")
	// log.Print(n)
	time.Sleep(500 * time.Millisecond)
}

func (node *Node) releaseCS() {
	log.Print("Node #", node.id," releaseCS #########################")
	// log.Print(n)
}

func UnmarshalRequest(text []byte, request *Request) error {
	request.requesterNodeId = int(text[0])
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

	ret[0] = byte(request.requesterNodeId)
	ret[1] = byte(request.requestId)
	ret[2] = byte(request.messageType)
	for i := 0; i < REQUEST_SIZE; i++ {
		ret[3 + i] = byte(request.resourceId[i])
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

func (node *Node) rcv() {	
	// log.Print("Node #", node.id," rcv")	
	for {
		select {
		case msg := <-node.messages[node.id]:
			var request Request
			err := UnmarshalRequest(msg, &request)
			if err != nil {
				log.Fatal(err)
			}			
			if (request.messageType == REQ_TYPE) {
				var requester = request.requesterNodeId
				var res = request.resourceId
				log.Print("Node #", node.id, "<-REQ#", request.requestId, ", Requester #", requester, ", nb of res:", len(res))
				if node.resourcePresent == true {
					node.resourcePresent = false
					// log.Print("Node #", node.id, ", Resource available, replying")
					var next_res = getNextResourceForReq(request, node.id)
					if next_res == NO_NEXT {
						log.Print("Node #", node.id, "                               with an ack")
						var ack Request = request
						ack.messageType = REP_TYPE
						// Messages are sent in a different subroutine
						go node.sendAck(ack)
					} else {
						log.Print("Node #", node.id, "                               forwarding to next req #", next_res)
						// Messages are sent in a different subroutine
						go node.sendRequest(request, next_res)
					}
				} else {
					log.Print("Node #", node.id, ", Resource NOT available, appending request to pendinglist")
					node.pendingRequests = append(node.pendingRequests, request)
				}
			}  else if (request.messageType == FREE_TYPE) {
				var requester = request.requesterNodeId
				log.Print("Node #", node.id, "<- FREE for REQ#", request.requestId, ", requester =", requester, ",", msg)
				node.resourcePresent = true
				// log.Print(len(node.pendingRequests), " pending requests on node#", node.id)
				
				if len(node.pendingRequests) > 0 {
					var r Request = node.pendingRequests[0]
					node.pendingRequests = node.pendingRequests[1:]

					log.Print("Node #", node.id, ", Continuing REQ#", r.requestId)
					var next_res = getNextResourceForReq(r, node.id)
					// log.Print("next_res=", next_res)
					if next_res == NO_NEXT {
						var ack Request = r
						ack.messageType = REP_TYPE
						// Messages are sent in a different subroutine
						go node.sendAck(ack)
					} else {
						// Messages are sent in a different subroutine
						go node.sendRequest(r, next_res)
					}
				}
			}  else if (request.messageType == REP_TYPE) {
				var requester = request.requesterNodeId
				log.Print("Node #", node.id, "<- REPLY for REQ#", request.requestId, ", requester =", requester, ",", msg)
				node.enterCS()
				node.executeCSCode()
				node.releaseCS()
				go node.freeResources(request)
				go node.requestCS()				
			} else {
				log.Fatal("Fatal Error")
			}
		}
	}
	// log.Print(n)
	// log.Print("Node #", node.id, " end rcv")
}

func (node *Node) freeResources(r Request) {
	var freeRequest Request = r
	freeRequest.messageType = FREE_TYPE
	content, err := MarshalRequest(freeRequest)
	if err != nil {
		log.Fatal(err)
	}			
	for i := 0; i < REQUEST_SIZE; i++ {
		log.Print("Node #", node.id, ",  FREE #", r.requestId, ":", content, " for resources #", r.resourceId[0], ", #", r.resourceId[1], " to Node #", r.resourceId[i])	
		node.messages[r.resourceId[i]] <- content		
	}
}

func (node *Node) sendAck(r Request) {
	content, err := MarshalRequest(r)
	if err != nil {
		log.Fatal(err)
	}			
	// var content = fmt.Sprintf("REQ%d%d%d", node.id, request.resourceId[0], request.resourceId[1])
	log.Print("Node #", node.id, ",  REPLY#", r.requestId, ":", content, " for resources #", r.resourceId[0], ", #", r.resourceId[1], " to Node #", r.requesterNodeId)	
	node.messages[r.requesterNodeId] <- content
}

func (node *Node) sendRequest(request Request, destination int) {
	content, err := MarshalRequest(request)
	if err != nil {
		log.Fatal(err)
	}			
	// var content = fmt.Sprintf("REQ%d%d%d", node.id, request.resourceId[0], request.resourceId[1])
	log.Print("Node #", node.id, ",  REQUEST #", request.requestId, ":", content, " for resources #", request.resourceId[0], ", #", request.resourceId[1], " to Node #", destination)	
	node.messages[destination] <- content
}

func (node *Node) requestCS() {
	// log.Print("Node #", node.id, " requestCS")

	var request Request
	request.messageType = REQ_TYPE
	request.requesterNodeId = node.id
	request.requestId = REQUEST_ID
	REQUEST_ID += 1
	// node.replyReceived[request.requestId] = false
	
	var resources = make([]int, NB_NODES)
	request.resourceId = make([]int, REQUEST_SIZE)
	// initialize a resources array with all
	// resources id, when selecting randomly
	// a resource we just remove it from the
	// array so that the array alsways contains
	// available resources
	for j := 0; j < NB_NODES; j++ {
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
	go node.sendRequest(request, destination)
	
	// log.Print("Node #", node.id," END")	
}

func (node *Node) Dijkstra(wg *sync.WaitGroup) {
	// log.Print("Node #", node.id)

	go node.requestCS()
	go node.rcv()
	for {
		
		time.Sleep(100 * time.Millisecond)
		if CURRENT_ITERATION > NB_ITERATIONS {
			break
		}
	}

	log.Print("Node #", node.id," END after ", NB_ITERATIONS," CS entries")	
	wg.Done()
}

func main() {
	var nodes = make([]Node, NB_NODES)
	var wg sync.WaitGroup
	var messages = make([]chan []byte, NB_NODES)
	
	log.Print("nb_process #", NB_NODES)

	// Initialization
	for i := 0; i < NB_NODES; i++ {
		nodes[i].id = i
		nodes[i].resourcePresent = true
		nodes[i].nbCS = 0 

		messages[i] = make(chan []byte)
		nodes[i].replyReceived   = make([]bool, NB_NODES * NB_ITERATIONS)
		nodes[i].pendingRequests = nil
	}
	for i := 0; i < NB_NODES; i++ {
		nodes[i].messages = messages
	}

	// start
	for i := 0; i < NB_NODES; i++ {
		wg.Add(1)
		go nodes[i].Dijkstra(&wg)
	}

	// end
	wg.Wait()
	for i := 0; i < NB_NODES; i++ {
		log.Print("Node #", nodes[i].id," entered CS ", nodes[i].nbCS, " time")	
	}
}
