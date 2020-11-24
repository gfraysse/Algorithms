/*
  Copyright "Guillaume Fraysse <gfraysse dot spam plus code at gmail dot com>"

TODO: 
- finalize implementation

How-to run: 
  go run rhee.go 2>&1  |tee /tmp/tmp.log

Parameters:
- Number of nodes is set with NB_NODES global variable
- Number of iterations is hardcoded in main function
*/ 

/*
    Go implementation of Injong Rhee mutual exclusion algorithm, uses Chandy-Misra Dining Philosophers algorithm as a subroutine
    Algorithm by Injong Rhee 1995

References : 
 * https://doi.org/10.1109/ICDCS.1995.500015: Injong Rhee, "A fast distributed modular algorithm for resource allocation," Proceedings of 15th International Conference on Distributed Computing Systems, Vancouver, BC, Canada, 1995, pp. 161-168, doi: 10.1109/ICDCS.1995.500015.
*/

package Rhee

/*
  logrus package is used for logs: https://github.com/sirupsen/logrus
*/
import (
	"bytes" // for go routine ID getGID
	"fmt"
	// "log"
	"ChandyMisra"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"runtime" // for debugging purpose
	"sync"
	// "strings"
	"strconv"
	"time"
)

/* global variable declaration */
var NB_NODES          int = 4
var REQUEST_SIZE      int = 2
var NB_MSG            int = 0
var NB_ITERATIONS     int = 10
var CURRENT_ITERATION int = 0

var STATE_THINKING    int = 0
var STATE_HUNGRY      int = 1
var STATE_EATING      int = 2

// Message types
var REQ_TYPE     int = 0
var REP_TYPE     int = 1
var REPORT_TYPE  int = 2
var SELECT_TYPE  int = 3
var RELEASE_TYPE int = 4
var MARKED_TYPE  int = 5
var GRANT_TYPE   int = 6
var ADV_TYPE     int = 7
var DEC_TYPE     int = 8

var Logger = log.New()

// Debug function
/*
func displayNodes() {
	for i := 0; i < NB_NODES; i++ {
		for j := 0; j < NB_NODES - 1; j++ {
			Logger.Info("  P#", nodes[i].id, ", fork #", nodes[i].forkId[j], ", status=", nodes[i].forkStatus[j], ", clean=", nodes[i].forkClean[j])
		}
	}
}
*/

type Request struct {
	requesterNodeId int
	requestId      int
	messageType    int
	resourceId     []int
}

type Node struct {
	Philosopher ChandyMisra.Philosopher 
	// id              int
	// initialized     bool
	// forkId          []int
	// forkClean       []bool
	// forkStatus      []bool
	// state           int
	// nbCS            int
	// queue           []RequestFork
	Messages       []chan []byte
	// // messages        []chan string
	requestIdCounter          int
}

////////////////////////////////////////////////////////////
// Debug functions
////////////////////////////////////////////////////////////
// get ID of go routine
func getGID() uint64 {
    b := make([]byte, 64)
    b = b[:runtime.Stack(b, false)]
    b = bytes.TrimPrefix(b, []byte("goroutine "))
    b = b[:bytes.IndexByte(b, ' ')]
    n, _ := strconv.ParseUint(string(b), 10, 64)
    return n
}

////////////////////////////////////////////////////////////
// Utility functions
////////////////////////////////////////////////////////////
func UnmarshalRequest(text []byte, request *Request) error {
	request.requesterNodeId = int(text[0])
	request.requestId      = int(text[1])
	request.messageType    = int(text[2])


	// if request.messageType == REQ_CT_TYPE {
	// } else if request.messageType == REQ_TYPE || request.messageType == INQUIRE_TYPE || request.messageType == ACK1_TYPE {
	request.resourceId = make ([]int, REQUEST_SIZE)
	
	for i := 0; i < REQUEST_SIZE; i++ {
		var val int = int(text[3 + i])
		if val == 255 {
			val = -1
		}
		request.resourceId[i] = val
	}
	// }
	
	return nil
}

func MarshalRequest(request Request) ([]byte, error) {
	var ret = make ([]byte, 3 + REQUEST_SIZE)

	ret[0] = byte(request.requesterNodeId)
	ret[1] = byte(request.requestId)
	ret[2] = byte(request.messageType)

	// if request.messageType == REQ_CT_TYPE {
	// } else if request.messageType == REQ_TYPE || request.messageType == INQUIRE_TYPE || request.messageType == ACK1_TYPE {
	for i := 0; i < REQUEST_SIZE; i++ {
		ret[3 + i] = byte(request.resourceId[i])
	}
	//}
		
	return ret, nil
}

func (r *Request) String() string {
	var val string
	var res string = ""

	for i := 0; i < len(r.resourceId); i ++ {
		res +=  strconv.Itoa(r.resourceId[i]) + ", "
	}

	val = fmt.Sprintf("Request #=%d, requester node=%d, messageType=%d, resources={%s}",
		r.requestId,
		r.requesterNodeId,
		r.messageType,
		res)
	return val
}

////////////////////////////////////////////////////////////
// Node class
////////////////////////////////////////////////////////////
func (n *Node) String() string {
	var val string
	val = fmt.Sprintf("Node #%d, state=%d, first fork=%d/%v/%v, second fork=%d/%v/%v, third fork=%d/%v/%v, my fork=%d/%v\n",
		n.Philosopher.Id,
		n.Philosopher.State)
	return val
}

func (n *Node) enterCS() {
	Logger.Info("Node #", n.Philosopher.Id, " ######################### Node.enterCS")
	n.Philosopher.State = STATE_EATING
	n.Philosopher.NbCS ++
	CURRENT_ITERATION ++
}

func (n *Node) executeCSCode() {
	Logger.Debug("Node #", n.Philosopher.Id, " ######################### Node.executeCSCode")
	Logger.Debug(n)
	time.Sleep(500 * time.Millisecond)
}

func (n *Node) releaseCS() {
	Logger.Info("Node #", n.Philosopher.Id," Node.releaseCS #########################")
	n.Philosopher.ReleaseCS()
}

func (n *Node) sendRequest(request Request, destination int) {
        content, err := MarshalRequest(request)
        if err != nil {
                log.Fatal(err)
        }                       
        log.Print("Node #", n.Philosopher.Id, ",  REQUEST #", request.requestId, ":", content, " for resources #", request.resourceId[0], ", #", request.resourceId[1], " to Node #", destination)       
        n.Messages[destination] <- content
}

func (n *Node) buildRequest() Request {
	var request Request
	request.messageType = REQ_TYPE
	var resources = make([]int, NB_NODES)
	for j := 0; j < NB_NODES; j++ {
		resources[j] = j
	}
	request.requesterNodeId = n.Philosopher.Id
	request.requestId = n.requestIdCounter
	n.requestIdCounter ++
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
	return request
}

func (n *Node) rcv() {	
	Logger.Debug("Node #", n.Philosopher.Id," rcv", ", routine #", getGID())	
	for {
		select {
		case msg := <-n.Messages[n.Philosopher.Id]:
			var request Request
			err := UnmarshalRequest(msg, &request)
			if err != nil {
				Logger.Fatal(err)
			}			
			var requester = request.requesterNodeId
			// if (request.messageType == REQ_TYPE) {
			// 	var res = request.resourceId
			// 	Logger.Debug("Node #", n.id, "<-REQ#", request.requestId, ", Requester #", requester, ", nb of res:", len(res), " res ", res)
			// 	go n.handleRequest(request)
			// } else
			if (request.messageType == REP_TYPE) {
				Logger.Info("Node #", n.Philosopher.Id, ", received REPLY from Node #", requester, ",", msg)
			// } else if (request.messageType == REQ_CT_TYPE) {
			// 	Logger.Info("Node #", n.id, ", received REQUEST Control Token from Node #", requester, ",", msg)
			// 	n.mutex.Lock()
			// 	n.handleCTRequest(request)
			// 	n.mutex.Unlock()
			// } else if (request.messageType == REP_CT_TYPE) {
			// 	Logger.Info("Node #", n.id, ", received REPLY Control Token from Node #", requester, ",", msg)
			// 	go n.receiveCT()
			// } else if (request.messageType == INQUIRE_TYPE) {
			// 	Logger.Info("Node #", n.id, ", received INQUIRE from Node #", requester, ",", msg)
			// 	go n.receiveInquire(request)
			// } else if (request.messageType == ACK1_TYPE) {
			// 	Logger.Info("Node #", n.id, ", received ACK1 from Node #", requester, ",", msg)
			// 	go n.receiveACK1(request)
			// } else if (request.messageType == ACK2_TYPE) {
			// 	Logger.Info("Node #", n.id, ", received ACK2 from Node #", requester, ",", msg)
			// 	go n.receiveACK2(request)
			} else {
				Logger.Fatal("Fatal Error")
			}
		}
	}
	Logger.Debug(n)
	Logger.Debug("Node #", n.Philosopher.Id, " end rcv")

	// Logger.Info("Node #", n.Philosopher.Id," rcv")	
	// for {
	// 	select {
	// 	case msg := <-n.Philosophers.Messages[n.Philosopher.Id]:
	// 		checkSanity()
	// 		if (strings.Contains(msg, "REQ")) {
	// 			var requester, err = strconv.Atoi(msg[3:5])
	// 			if err != nil {
	// 				log.Fatal(err)
	// 			}
	// 			for i := 0; i < NB_NODES - 1; i ++ {
	// 				if requester == n.forkId[i] {
	// 					if n.forkStatus[i] == true {
	// 						if n.forkClean[i] == true {
	// 							// keep the fork
	// 							Logger.Info("Node #", n.Philosopher.Id,", fork#", i, " is clean, I keep it for now")
	// 							var r RequestFork
	// 							r.PhilosopherId = requester
	// 							r.forkId        = requester
	// 							n.queue         = append(n.queue, r)
	// 						} else {
	// 							n.forkStatus[i]    = false
	// 							n.forkStatus[i]    = false
	// 							n.sendFork(requester)
	// 						}						
	// 						break
	// 					} else {
	// 						Logger.Info("Node #", n.Philosopher.Id,", DOES NOT own fork#", i)
	// 					}
	// 				}
	// 			}
	// 		}  else if (strings.Contains(msg, "REP")) {
	// 			var sender, err = strconv.Atoi(msg[3:5])
	// 			if err != nil {
	// 				log.Fatal(err)
	// 			}
	// 			Logger.Info("Node #", n.Philosopher.Id, ", RECEIVED fork from Node #", sender, ", ", msg)
	// 			Logger.Info(sender, ": ", n.Philosopher.Id, " <==== ", sender)	
	// 			for i := 0; i < NB_NODES - 1; i ++ {
	// 				if (sender == n.forkId[i]) {
	// 					n.forkStatus[i]    = true
	// 					n.forkClean[i]     = true
	// 					break
	// 				}
	// 			}
	// 			var hasSentReq bool = false
	// 			Logger.Info("Node #", n.Philosopher.Id, ", checking if forks are missing")
	// 			for j := 0; j < NB_NODES - 1; j++ {
	// 				if n.forkStatus[j] == false {
	// 					n.requestFork(n.forkId[j])
	// 					hasSentReq = true
	// 					break
	// 				}
	// 			}
	// 			if hasSentReq == false {
	// 				Logger.Info("Node #", n.Philosopher.Id, ", has not requested any fork")
	// 			}
					
	// 		} else {
	// 			log.Fatal("WTF")
	// 		}
	// 	}
	// }
}

func (n *Node) handleRequest(request Request) {
}

func (n *Node) requestCS() {
	Logger.Info("Node #", n.Philosopher.Id, " requestCS")

	var waitingTime int = rand.Intn(100)
	time.Sleep(time.Duration(waitingTime) * time.Millisecond)

	var request Request = n.buildRequest()
	
	var requester = request.requesterNodeId
	var res = request.resourceId
	Logger.Debug("Node #", n.Philosopher.Id, "<-REQ#", request.requestId, ", Requester #", requester, ", nb of res:", len(res), " res ", res)
	n.handleRequest(request)

	// for {
	// 	time.Sleep(100 * time.Millisecond)
	if n.Philosopher.State == STATE_THINKING {
		// rand.Seed(time.Now().UnixNano())
		// var proba int = rand.Intn(4)
		// Logger.Info("Proba =", proba)
		var proba = 0
		if proba < 1 {
			n.Philosopher.RequestCS()
		} else {
			Logger.Info("Node #", n.Philosopher.Id, " DOES NOT want to enter CS")
		}
	} else if n.Philosopher.State == STATE_HUNGRY {
		var allGreen = true

		for i := 0; i < NB_NODES - 1; i ++ {
			allGreen = allGreen && n.Philosopher.ForkStatus[i] && n.Philosopher.ForkClean[i]
			if allGreen == false {
				// Logger.Info("Node #", n.Philosopher.Id, " waiting for fork", n.forkId[i])
				// displayNodes()
				break
			}
		}
		if (allGreen == true) {
			if (n.Philosopher.State == STATE_EATING) {
				Logger.Info("** Node #", n.Philosopher.Id, " is already eating **")
			} else {
				n.enterCS()
				n.executeCSCode()
				n.releaseCS()
				go n.requestCS()
			}
		}
	} else {
			Logger.Info("already thinking")
	}
	//}
	Logger.Info("Node #", n.Philosopher.Id," END")	
}

func (n *Node) Rhee(wg *sync.WaitGroup) {
	Logger.Info("Node #", n.Philosopher.Id)

	go n.requestCS()
	go n.rcv()
	for {
		time.Sleep(100 * time.Millisecond)
		if CURRENT_ITERATION == NB_ITERATIONS {
			break
		}
	}

	Logger.Info("Node #", n.Philosopher.Id," END after ", NB_ITERATIONS," CS entries")	
	wg.Done()
}

// func main() {
// 	// var nodes = make([]Node, NB_NODES)
// 	nodes = make([]Node, NB_NODES)
// 	var wg sync.WaitGroup
// 	var messages = make([]chan []byte, NB_NODES)
// 	// var messages  = make([]chan string, NB_NODES)
	
// 	Logger.Info("nb_process #", NB_NODES)
	
// 	for i := 0; i < NB_NODES; i++ {
// 		nodes[i].id = i
// 		nodes[i].nbCS = 0
// 		nodes[i].state = STATE_THINKING
// 		nodes[i].forkId  = make([]int, NB_NODES - 1)
// 		nodes[i].forkStatus  = make([]bool, NB_NODES - 1)
// 		nodes[i].forkClean  = make([]bool, NB_NODES - 1)
// 		var idx int = 0
// 		for j := 0; j < NB_NODES; j++ {
// 			if j == nodes[i].id {
// 				// skip my own ID
// 				continue
// 			} else {
// 				nodes[i].forkId[idx] = j
// 				idx++
// 			}
// 		}
// 		// Initially forks are in the hand of the nodes with id lower than the fork id to make graphs acyclic
// 		// Initially all forks are dirty
// 		for j := 0; j < NB_NODES - 1; j++ {
// 			if nodes[i].forkId[j] < i {
// 				nodes[i].forkStatus[j]    = false
// 			} else {
// 				nodes[i].forkStatus[j]    = true
// 			}
// 			nodes[i].forkClean[j]     = false
// 		}

// 		messages[i] = make(chan []byte)
// 		// messages[i] = make(chan string)
// 		nodes[i].initialized = true
// 	}

// 	for i := 0; i < NB_NODES; i++ {
// 		nodes[i].messages = messages
// 	}

// 	for i := 0; i < NB_NODES; i++ {
// 		wg.Add(1)
// 		go nodes[i].Rhee(&wg)
// 	}
// 	wg.Wait()
// 	for i := 0; i < NB_NODES; i++ {
// 		Logger.Info("Node #", nodes[i].id," entered CS ", nodes[i].nbCS," time")	
// 	}
// 	Logger.Info(NB_MSG, " messages sent")
// }
