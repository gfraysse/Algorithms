/*
  Copyright "Guillaume Fraysse <gfraysse dot spam plus code at gmail dot com>"

TODO: 
- finalize implementation
- code duplication: WaitForReplies, RequestCMCS, enterCMCSIfICan between this and ChandyMisra.go files, find best workaround for Go lack of OO overrinding of methods

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
	"ChandyMisra"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"runtime" // for debugging purpose
	"strconv"
	"sync"
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

var Nodes []Node

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

var SEND_FORK    int = 9
var REQUEST_FORK int = 10

var EMPTY int = -1

var Logger = log.New()

// Debug function
/*
func displayNodes() {
	for i := 0; i < NB_NODES; i++ {
		for j := 0; j < NB_NODES - 1; j++ {
			Logger.Info("  P#", Nodes[i].id, ", fork #", Nodes[i].forkId[j], ", status=", Nodes[i].forkStatus[j], ", clean=", Nodes[i].forkClean[j])
		}
	}
}
*/

type Request struct {
	requesterNodeId int
	requestId      int
	messageType    int
	resourceId     []int
	occupied       []int
	position       int
}

type Node struct {
	Philosopher          ChandyMisra.Philosopher
	inCMCS               bool
	inRheeCS             bool
	// From paper: variables for resource managers
	has_received_advance [] bool
	rm_critical          bool
	// From paper: variables for users
	has_dec_sent         [] bool
	req_report           bool
	occupant             [] int
	// variables for implementation
	Messages             []chan []byte
	nbRheeCS             int
	requestIdCounter     int
	nbMarkedRcv          int
	positionSelected     map[int]int
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
	request.requestId       = int(text[1])
	request.messageType     = int(text[2])
	request.position        = int(text[3])
	var lenOccupied = int(text[4 + REQUEST_SIZE])

	request.resourceId = make ([]int, REQUEST_SIZE)
	request.occupied = make ([]int, lenOccupied)
	
	for i := 0; i < REQUEST_SIZE; i++ {
		var val int = int(text[4 + i])
		if val == 255 {
			val = -1
		}
		request.resourceId[i] = val
	}

	if lenOccupied > 0 {
		for i := 0; i < lenOccupied; i++ {
			var val int = int(text[4 + REQUEST_SIZE + 1 + i])
			if val == 255 {
				val = -1
			}
			request.occupied[i] = val
		}
	}
	return nil
}

func MarshalRequest(request Request) ([]byte, error) {
	var ret = make ([]byte, 4 + REQUEST_SIZE + 1 + len(request.occupied))

	ret[0] = byte(request.requesterNodeId)
	ret[1] = byte(request.requestId)
	ret[2] = byte(request.messageType)
	ret[3] = byte(request.position)

	for i := 0; i < REQUEST_SIZE; i++ {
		ret[4 + i] = byte(request.resourceId[i])
	}

	ret[4 + REQUEST_SIZE] = byte(len(request.occupied))
	for i := 0; i < len(request.occupied); i++ {
		ret[4 + REQUEST_SIZE + 1 + i] = byte(request.occupied[i])
	}

	return ret, nil
}

func (r *Request) String() string {
	var val string
	var res string = ""
	var occ string = ""

	for i := 0; i < len(r.resourceId); i ++ {
		res +=  strconv.Itoa(r.resourceId[i]) + ", "
	}

	for i := 0; i < len(r.occupied); i ++ {
		occ +=  strconv.Itoa(r.occupied[i]) + ", "
	}

	val = fmt.Sprintf("Request #%d, requester node=%d, position=%d messageType=%d, resources={%s}, occupied={%s}",
		r.requestId,
		r.requesterNodeId,
		r.position,
		r.messageType,
		res,
		occ)
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

func (n *Node) EnterCMCS(request Request) {
	Logger.Info("Node #", n.Philosopher.Id, " ######################### Node.EnterCMCS, req=", request.String())
	n.inCMCS = true
	n.Philosopher.EnterCS()
}

func (n *Node) ExecuteCMCSCode(request Request) {
	Logger.Debug("Node #", n.Philosopher.Id, " ######################### Node.executeCMCSCode")

	if n.req_report == false {
		n.req_report = true
		n.positionSelected[request.requestId] = 0
		n.nbMarkedRcv = 0
		for k := 0; k < len(request.resourceId); k++ {
			go n.sendReport(request.resourceId[k], request)
		}
	}	
}

func (n *Node) ReleaseCMCS(request Request) {
	Logger.Info("Node #", n.Philosopher.Id," Node.releaseCMCS #########################, req=", request.String())
	n.inCMCS = false
	n.Philosopher.ReleaseCS()
	
	for i := 0; i < len(n.Philosopher.Queue); i++ {
		var r ChandyMisra.ForkRequest
		r = n.Philosopher.Queue[i]
		for j := 0; j < NB_NODES - 1; j++ {
			if (r.PhilosopherId == n.Philosopher.ForkId[j] && n.Philosopher.ForkStatus[j] == true) {
				n.Philosopher.ForkStatus[j] = false
				go n.SendFork(r.PhilosopherId, request)
				break
			}
		}
	}
	n.Philosopher.Queue = nil
	for j := 0; j < NB_NODES - 1; j++ {
		if (n.Philosopher.ForkStatus[j] == true) {
			n.Philosopher.ForkStatus[j] = false
			go n.SendFork(n.Philosopher.ForkId[j], request)
		}
	}
}

func (n *Node) EnterCS(request Request) {
	Logger.Info("Node #", n.Philosopher.Id, " ######################### Node.EnterCS")
	n.nbRheeCS ++
}

func (n *Node) ExecuteCSCode(request Request) {
	Logger.Debug("Node #", n.Philosopher.Id, " ######################### Node.ExecuteCSCode")
	Logger.Debug(n)
	time.Sleep(500 * time.Millisecond)
}

func (n *Node) ReleaseCS(request Request) {
	Logger.Info("Node #", n.Philosopher.Id," Node.ReleaseCS #########################")
	for i := 0; i < len(request.resourceId); i++ {
		n.sendRelease(request.resourceId[i], request)
	}
}

func (n *Node) sendRequest(request Request, destination int) {
        content, err := MarshalRequest(request)
        if err != nil {
                log.Fatal("sendRequest", err)
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

func (n *Node) advance_one_position(p int, request Request) {
	Logger.Debug("Node #", n.Philosopher.Id,"** Node.advance_one_position **");
	n.occupant[p - 1] = n.occupant[p]
	n.occupant[p] = EMPTY
	n.has_received_advance[p] = false
	n.has_dec_sent[p] = false
	if p - 1 == 1 {
		go n.sendGrant(n.occupant[p - 1], request)
	}
	if n.has_dec_sent[p - 1] == false && n.occupant[p - 2] == EMPTY {
		n.has_dec_sent[p] = true
		go n.sendDec(p - 1, n.occupant[p - 1], request)
	}
}

func (n *Node) adjust_queue(p int, request Request) {
	Logger.Debug("Node #", n.Philosopher.Id,"** Node.adjust_queue ** p=", p);
	if p == 0 {
		return
	}
	
	for n.occupant[p] != EMPTY && n.occupant[p - 1] == EMPTY {
		if n.has_dec_sent[p] == false {
			n.has_dec_sent[p] = true
			go n.sendDec(p, n.occupant[p], request)
		}
		if n.has_received_advance[p] == true {
			n.advance_one_position(p, request)			
		}
		p ++
	}
}

func (n *Node) sendReport(dst int, request Request) {
	request.messageType = REPORT_TYPE
	request.requesterNodeId = n.Philosopher.Id

	content, err := MarshalRequest(request)
	if err != nil {
		Logger.Fatal("sendReport", err)
	}			
	Logger.Debug("Node #", n.Philosopher.Id, ", send REPORT #", request.requestId, ":", content, " to Node #", dst, ", routine #", getGID())
	n.Messages[dst] <- content
}

func (n *Node) sendSelect(position int, dst int, request Request) {
	request.messageType = SELECT_TYPE
	request.requesterNodeId = n.Philosopher.Id
	request.position = position

	content, err := MarshalRequest(request)
	if err != nil {
		Logger.Fatal("sendSelect", err)
	}			
	Logger.Debug("Node #", n.Philosopher.Id, ", send SELECT #", request.requestId, " with position=", position, ":", content, " to Node #", dst, ", routine #", getGID())
	n.Messages[dst] <- content
}

func (n *Node) sendRelease(dst int, request Request) {
	request.messageType = RELEASE_TYPE
	request.requesterNodeId = n.Philosopher.Id

	content, err := MarshalRequest(request)
	if err != nil {
		Logger.Fatal("sendRelease", err)
	}			
	Logger.Debug("Node #", n.Philosopher.Id, ", send RELEASE #", request.requestId, ":", content, " to Node #", dst, ", routine #", getGID())
	n.Messages[dst] <- content
}

func (n *Node) sendMarked(occupied []int, dst int, request Request) {
	request.messageType = MARKED_TYPE
	request.requesterNodeId = n.Philosopher.Id
	request.occupied = occupied

	content, err := MarshalRequest(request)
	if err != nil {
		Logger.Fatal("sendMarked", err)
	}			
	Logger.Debug("Node #", n.Philosopher.Id, ", send MARKED #", request.requestId, ":", content, " to Node #", dst, " with occupied", occupied, ", routine #", getGID())	
	n.Messages[dst] <- content
}

func (n *Node) sendGrant(dst int, request Request) {
	request.messageType = GRANT_TYPE
	request.requesterNodeId = n.Philosopher.Id

	content, err := MarshalRequest(request)
	if err != nil {
		Logger.Fatal("sendGrant", err)
	}			
	Logger.Debug("Node #", n.Philosopher.Id, ", send GRANT #", request.requestId, ":", content, " to Node #", dst, ", routine #", getGID())
	n.Messages[dst] <- content
}

func (n *Node) sendAdv(position int, dst int, request Request) {
	request.messageType = ADV_TYPE
	request.requesterNodeId = n.Philosopher.Id
	request.position = position

	content, err := MarshalRequest(request)
	if err != nil {
		Logger.Fatal("sendAdv", err)
	}			
	Logger.Debug("Node #", n.Philosopher.Id, ", send ADV #", request.requestId, " with position=", position, ":", content, " to Node #", dst, ", routine #", getGID())
	n.Messages[dst] <- content
}

func (n *Node) sendDec(p int, dst int, request Request) {
	request.messageType = DEC_TYPE
	request.requesterNodeId = n.Philosopher.Id

	content, err := MarshalRequest(request)
	if err != nil {
		Logger.Fatal("sendDec", err)
	}			
	Logger.Debug("Node #", n.Philosopher.Id, ", send DEC #", request.requestId, ":", content, " to Node #", dst, ", routine #", getGID())
	n.Messages[dst] <- content
}

func (n *Node) receiveReport(request Request) {
	var occupied []int
	Logger.Debug("Node #", n.Philosopher.Id,"** Node.receiveReport **, req=", request.String());
	if n.rm_critical == true {
		Logger.Fatal("Node #", n.Philosopher.Id, " already rm_critical !")
	}
	
	if n.rm_critical == false {		
		n.rm_critical = true
		for i := 0; i < len(n.occupant); i ++ {
			if n.occupant[i] != EMPTY {
				if i > 0 {
					occupied = append(occupied, i - 1)
				}
				occupied = append(occupied, i)
			}
		}
		go n.sendMarked(occupied, request.requesterNodeId, request)
	} else {
		Logger.Info("Node #", n.Philosopher.Id, " receiveReport n.rm_critical = true")
	}
}

func (n *Node) receiveSelect(request Request) {
	Logger.Debug("Node #", n.Philosopher.Id,"** Node.receiveSelect **, req=", request.String());
	n.rm_critical = false
	// Logger.Debug("len(n.occupant)=", len(n.occupant))
	n.occupant[request.position] = request.requestId
	if request.position == 0 {
		n.sendGrant(request.requesterNodeId, request)		
	}
	n.adjust_queue(request.position, request)
}

func (n *Node) receiveRelease(request Request) {
	Logger.Debug("Node #", n.Philosopher.Id, "** Node.receiveRelease **");
	if n.rm_critical == true {
		n.occupant[0] = EMPTY
		n.adjust_queue(1, request)
	} else {
		Logger.Fatal("receiveRelease but not rm_critical")
	}
}

func (n *Node) receiveMarked(request Request) {
	var position_selected int = 0;
	n.nbMarkedRcv ++
	if len(request.occupied) > 0 {
		var maxPosition int = 0;
		for i := 0; i < len(request.occupied); i ++ {
			if request.occupied[i] > maxPosition {
				maxPosition = i
			}		
		}
		position_selected = maxPosition + 1
	}
	if position_selected > n.positionSelected[request.requestId] {
		n.positionSelected[request.requestId] = position_selected
	}
	if n.nbMarkedRcv == len(request.resourceId) {
		Logger.Debug("Node #", n.Philosopher.Id, " ALL MARKED RECEIVED")
		for k := 0; k < len(request.resourceId); k++ {
			go n.sendSelect(n.positionSelected[request.requestId], request.resourceId[k], request)
		}
		n.req_report = false
		n.ReleaseCMCS(request)
	} else {
		Logger.Debug("Node #", n.Philosopher.Id, " is still expecting ", len(request.resourceId) - n.nbMarkedRcv, " MARKED")
	}
}

func (n *Node) receiveGrant(request Request) {
	Logger.Debug("Node #", n.Philosopher.Id,"** Node.receiveGrant **");
	n.inRheeCS = true
	n.EnterCS(request)
	n.ExecuteCSCode(request)
	n.ReleaseCS(request)
	go n.requestCS()
}

func (n *Node) receiveAdv(request Request) {
	Logger.Debug("Node #", n.Philosopher.Id,"** Node.receiveAdv **");
	if n.rm_critical == true {
		n.has_received_advance[request.position] = true
		n.adjust_queue(request.position, request)
	} else {
		Logger.Fatal("receiveAdv but not rm_critical")
	}
}

func (n *Node) receiveDec(request Request) {
	Logger.Debug("Node #", n.Philosopher.Id,"** Node.receiveDec **");
	for k := 0; k < len(request.resourceId); k++ {
		go n.sendAdv(request.position, request.resourceId[k], request)
	}
}


func (n *Node) enterCMCSIfICan(request Request) {
	var hasSentReq bool = false
	log.Print("Node #", n.Philosopher.Id, ", checking if forks are missing")
	for j := 0; j < NB_NODES - 1; j++ {
		if n.Philosopher.ForkStatus[j] == false {
			go n.RequestFork(n.Philosopher.ForkId[j], request)
			hasSentReq = true
			break
		}
	}
	if hasSentReq == false {
		log.Print("Node #", n.Philosopher.Id, ", has not requested any fork")
		if n.Philosopher.State == STATE_HUNGRY {
			var allGreen = true
			
			for i := 0; i < NB_NODES - 1; i ++ {
				allGreen = allGreen && n.Philosopher.ForkStatus[i] && n.Philosopher.ForkClean[i]
				if allGreen == false {
					// log.Print("Node #", n.Philosopher.Id, " waiting for fork", n.Philosopher.ForkId[i])
					// displayNodes()
					break
				}
			}
			if (allGreen == true) {
				if (n.Philosopher.State == STATE_EATING) {
					log.Print("** Node #", n.Philosopher.Id, " is already eating **")
				} else {
					n.EnterCMCS(request)
					n.ExecuteCMCSCode(request)
					// n.ReleaseCMCS(request)
					// n.Philosopher.RequestCS()								
				}
			}
		} else {
			log.Print("NOT HUNGRY")
		}					
	}
}

func (n *Node) rcv() {	
	Logger.Debug("Node #", n.Philosopher.Id," rcv", ", routine #", getGID())	
	for {
		select {
		case msg := <-n.Messages[n.Philosopher.Id]:
			var request Request
			err := UnmarshalRequest(msg, &request)
			if err != nil {
				Logger.Fatal("rcv", err)
			}			
			Logger.Debug(request.String())
			var requester = request.requesterNodeId
			if (request.messageType == REQ_TYPE) {
				var res = request.resourceId
				Logger.Debug("Node #", n.Philosopher.Id, "<-REQ#", request.requestId, ", Requester #", requester, ", nb of res:", len(res), " res ", res)
				go n.handleRequest(request)
			} else
			if (request.messageType == REP_TYPE) {
				Logger.Info("Node #", n.Philosopher.Id, ", received REPLY from Node #", requester, ",", msg)
			} else if (request.messageType == REPORT_TYPE) {
				Logger.Info("Node #", n.Philosopher.Id, ", received REPORT from Node #", requester, ",", msg)
				go n.receiveReport(request)
			} else if (request.messageType == SELECT_TYPE) {
				Logger.Info("Node #", n.Philosopher.Id, ", received SELECT from Node #", requester, ",", msg)
				go n.receiveSelect(request)
			} else if (request.messageType == RELEASE_TYPE) {
				Logger.Info("Node #", n.Philosopher.Id, ", received RELEASE from Node #", requester, ",", msg)
				go n.receiveRelease(request)
			} else if (request.messageType == MARKED_TYPE) {
				Logger.Info("Node #", n.Philosopher.Id, ", received MARKED from Node #", requester, ",", msg)
				go n.receiveMarked(request)
			} else if (request.messageType == GRANT_TYPE) {
				Logger.Info("Node #", n.Philosopher.Id, ", received GRANT from Node #", requester, ",", msg)
				go n.receiveGrant(request)
			} else if (request.messageType == ADV_TYPE) {
				Logger.Info("Node #", n.Philosopher.Id, ", received ADV from Node #", requester, ",", msg)
				go n.receiveAdv(request)
			} else if (request.messageType == DEC_TYPE) {
				Logger.Info("Node #", n.Philosopher.Id, ", received DEC from Node #", requester, ",", msg)
				go n.receiveDec(request)
			} else if (request.messageType == REQUEST_FORK) {
				Logger.Info("Node #", n.Philosopher.Id, ", received REQUEST_FORK from Node #", requester, ",", msg)
				for i := 0; i < NB_NODES - 1; i ++ {
					if requester == n.Philosopher.ForkId[i] {
						if n.Philosopher.ForkStatus[i] == true {
							if n.Philosopher.ForkClean[i] == true {
								// keep the fork
								log.Print("Node #", n.Philosopher.Id,", fork#", i, " is clean, I keep it for now")
								var r ChandyMisra.ForkRequest
								r.PhilosopherId = requester
								r.ForkId        = requester
								n.Philosopher.Queue         = append(n.Philosopher.Queue, r)
							} else {
								n.Philosopher.ForkStatus[i]    = false
								n.Philosopher.ForkStatus[i]    = false
								go n.SendFork(requester, request)
							}						
							break
						} else {
							log.Print("Node #", n.Philosopher.Id,", DOES NOT own fork#", i)
						}
					}
				}
				n.enterCMCSIfICan(request)
			} else if (request.messageType == SEND_FORK) {
				Logger.Info("Node #", n.Philosopher.Id, ", received SEND_FORK from Node #", requester, ",", msg)
				log.Print(requester, ": ", n.Philosopher.Id, " <==== ", requester)	
				for i := 0; i < NB_NODES - 1; i ++ {
					if (requester == n.Philosopher.ForkId[i]) {
						n.Philosopher.ForkStatus[i]    = true
						n.Philosopher.ForkClean[i]     = true
						break
					}
				}
				n.enterCMCSIfICan(request)
			} else {
				Logger.Fatal("Unknown message type=", request.messageType)
			}
		}
	}
	Logger.Debug(n)
	Logger.Debug("Node #", n.Philosopher.Id, " end rcv")
}

func (n *Node) handleRequest(request Request) {
	if n.Philosopher.State == STATE_THINKING {
		n.RequestCMCS(request)
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
				n.EnterCMCS(request)
				n.ExecuteCMCSCode(request)
				// n.ReleaseCMCS(request)
				// go n.requestCS()
			}
		}
	} else {
		Logger.Info("already thinking")
	}
}

func (n *Node) RequestCMCS(request Request) {
	log.Print("Node #", n.Philosopher.Id, " RequestCMCS")

	if n.Philosopher.State == STATE_THINKING {
		n.Philosopher.State = STATE_HUNGRY
		log.Print("Node #", n.Philosopher.Id, " wants to enter CS")
		for j := 0; j < NB_NODES - 1; j++ {
			if n.Philosopher.ForkStatus[j] == false {
				n.RequestFork(n.Philosopher.ForkId[j], request)
				break
			} else {
				n.Philosopher.ForkClean[j] = true
			}
		}
	} else {
		log.Print("already eating")
	}

	log.Print("Node #", n.Philosopher.Id," END - RequestCMCS")	
}

func (n *Node) RequestFork(dst int, request Request) {
	request.messageType = REQUEST_FORK
	request.requesterNodeId = n.Philosopher.Id

	content, err := MarshalRequest(request)
	if err != nil {
		Logger.Fatal("RequestFort", err)
	}			
	// Logger.Debug("Node #", n.Philosopher.Id, ", send REQUEST_FORK #", request.requestId, ":", content, " to Node #", dst, ", routine #", getGID())	
	Logger.Info(n.Philosopher.Id, " --", dst, "--> ", dst)	
	n.Messages[dst] <- content
	NB_MSG ++
}

func (n *Node) SendFork(dst int, request Request) {
	request.messageType = SEND_FORK
	request.requesterNodeId = n.Philosopher.Id

	content, err := MarshalRequest(request)
	if err != nil {
		Logger.Fatal("SendFork", err)
	}			
	// Logger.Debug("Node #", n.Philosopher.Id, ", send SEND_FORK #", request.requestId, ":", content, " to Node #", dst, ", routine #", getGID())	
	Logger.Info(n.Philosopher.Id,": ", n.Philosopher.Id, " ====> ", dst)	
	n.Messages[dst] <- content
	NB_MSG ++
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

	Logger.Info("Node #", n.Philosopher.Id," END requestCS")	
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

func Init() {
	Logger.SetLevel(log.DebugLevel)
	// Logger.SetLevel(log.InfoLevel)
	Logger.Print("Rhee.Init")	
	ChandyMisra.Init()

	Nodes = make([]Node, NB_NODES)
	var messages = make([]chan []byte, NB_NODES)
	var philosopherMessages = make([]chan string, NB_NODES)
	var occupants = make([]int, NB_NODES *2 )
	var has_received_advance = make([]bool, NB_NODES)
	var has_dec_sent = make([]bool, NB_NODES)

	Logger.Info("nb_process #", NB_NODES)
	
	for i := 0; i < NB_NODES; i++ {		
		ChandyMisra.InitPhilosopher(&Nodes[i].Philosopher, i , NB_NODES)
		philosopherMessages[i] = make(chan string)
		messages[i] = make(chan []byte)
		Nodes[i].occupant = make([]int, NB_NODES)
		Nodes[i].has_received_advance = make([]bool, NB_NODES)
		Nodes[i].has_dec_sent = make([]bool, NB_NODES)
		occupants[i] = EMPTY
		has_received_advance[i] = false
		has_dec_sent[i] = false
	}

	for i := 0; i < NB_NODES; i++ {
		Nodes[i].Philosopher.Messages = philosopherMessages
		Nodes[i].positionSelected = make(map[int]int)
		Nodes[i].Messages = messages
		Nodes[i].rm_critical = false
		Nodes[i].req_report = false
		copy(Nodes[i].occupant, occupants)
		copy(Nodes[i].has_received_advance, has_received_advance)
		copy(Nodes[i].has_dec_sent, has_dec_sent)
	} 
}
