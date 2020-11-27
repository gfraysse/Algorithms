/*
  Copyright "Guillaume Fraysse <gfraysse dot spam plus code at gmail dot com>"

TODO: 
- finalize implementation: 
    - ends up on Fatal "receiveRelease but not rm_critical" or "Node #? already rm_critical !"
    - a queue of pending messages seems to be necessary, 
    - there might still be a bug in the implementation
- code duplication: WaitForReplies, RequestCMCS, enterCMCSIfICan between this and ChandyMisra.go files, have to do for now but need to find a workaround for Go lack of OO overriding of methods

How-to run: 
  go run rhee.go 2>&1  |tee /tmp/tmp.log

Parameters:
- Number of nodes 
- Number of iterations 
- Size of requests: a constant, all requests have the same size
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
	"encoding/gob"
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
var NB_MSG            int = 0
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
func displayNodes() {
	for i := 0; i < Nodes[0].Philosopher.NbNodes; i++ {
		Logger.Debug("  N#", Nodes[i].Philosopher.Id, ", inCMCS #", Nodes[i].InCMCS, ", inRheeCS=", Nodes[i].InRheeCS)
		for key, element := range Nodes[i].PositionSelected {
			Logger.Debug("Request:", key, "=>", "Position:", element)
		}

		for j, element := range Nodes[i].Occupant {
			Logger.Debug("occupant[", j, "] =", "request #", element)
		}
	}
}


type Request struct {
	RequesterNodeId int
	RequestId      int
	MessageType    int
	ResourceId     []int
	Occupied       []int
	Position       int
}

type Node struct {
	Philosopher          ChandyMisra.Philosopher
	InCMCS               bool
	InRheeCS             bool
	// From paper: variables for resource managers
	Has_received_advance [] bool
	Rm_critical          bool
	// From paper: variables for users
	Has_dec_sent         [] bool
	Req_report           bool
	Occupant             map[int] int
	// variables for implementation
	Messages             []chan bytes.Buffer
	nbRheeCS             int
	RequestIdCounter     int
	nbMarkedRcv          int
	nbGrantRcv           int
	PositionSelected     map[int]int
	requestSize          int
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
func UnmarshalRequest(b bytes.Buffer, request *Request) error {
	dec := gob.NewDecoder(&b)

	error := dec.Decode(request)

	return error
}

func MarshalRequest(request Request) (bytes.Buffer, error) {
	var buffer bytes.Buffer        
	enc := gob.NewEncoder(&buffer) 
	error := enc.Encode(request)

	return buffer, error
}

func (r *Request) String() string {
	var val string
	var res string = ""
	var occ string = ""

	for i := 0; i < len(r.ResourceId); i ++ {
		res +=  strconv.Itoa(r.ResourceId[i]) + ", "
	}

	for i := 0; i < len(r.Occupied); i ++ {
		occ +=  strconv.Itoa(r.Occupied[i]) + ", "
	}

	val = fmt.Sprintf("Request #%d, requester node=%d, position=%d messageType=%d, resources={%s}, occupied={%s}",
		r.RequestId,
		r.RequesterNodeId,
		r.Position,
		r.MessageType,
		res,
		occ)
	return val
}

////////////////////////////////////////////////////////////
// Node class
////////////////////////////////////////////////////////////
func (n *Node) String() string {
	var val string
	val = fmt.Sprintf("Node #%d, state=%d",
		n.Philosopher.Id,
		n.Philosopher.State)
	return val
}

func (n *Node) EnterCMCS(request Request) {
	Logger.Info("Node #", n.Philosopher.Id, " ######################### Node.EnterCMCS, req=", request.String())
	n.InCMCS = true
	n.Philosopher.EnterCS()
	displayNodes()
}

func (n *Node) ExecuteCMCSCode(request Request) {
	Logger.Debug("Node #", n.Philosopher.Id, " ######################### Node.executeCMCSCode")

	if n.Req_report == false {
		n.Req_report = true
		n.PositionSelected[request.RequestId] = 0
		n.nbMarkedRcv = 0
		n.nbGrantRcv = 0
		for k := 0; k < len(request.ResourceId); k++ {
			go n.sendReport(request.ResourceId[k], request)
		}
	}	
}

func (n *Node) ReleaseCMCS(request Request) {
	Logger.Info("Node #", n.Philosopher.Id," Node.releaseCMCS #########################, req=", request.String())
	n.InCMCS = false
	n.Philosopher.ReleaseCS()
	displayNodes()
	
	for i := 0; i < len(n.Philosopher.Queue); i++ {
		var r ChandyMisra.ForkRequest
		r = n.Philosopher.Queue[i]
		for j := 0; j < n.Philosopher.NbNodes - 1; j++ {
			if (r.PhilosopherId == n.Philosopher.ForkId[j] && n.Philosopher.ForkStatus[j] == true) {
				n.Philosopher.ForkStatus[j] = false
				go n.SendFork(r.PhilosopherId, request)
				break
			}
		}
	}
	n.Philosopher.Queue = nil
	for j := 0; j < n.Philosopher.NbNodes - 1; j++ {
		if (n.Philosopher.ForkStatus[j] == true) {
			n.Philosopher.ForkStatus[j] = false
			go n.SendFork(n.Philosopher.ForkId[j], request)
		}
	}
}

func (n *Node) EnterCS(request Request) {
	Logger.Info("Node #", n.Philosopher.Id, " ######################### Node.EnterCS")
	displayNodes()
	n.nbRheeCS ++
}

func (n *Node) ExecuteCSCode(request Request) {
	Logger.Debug("Node #", n.Philosopher.Id, " ######################### Node.ExecuteCSCode")
	// Logger.Debug(n)
	time.Sleep(500 * time.Millisecond)
}

func (n *Node) ReleaseCS(request Request) {
	Logger.Info("Node #", n.Philosopher.Id," Node.ReleaseCS #########################")
	displayNodes()
	for i := 0; i < len(request.ResourceId); i++ {
		n.sendRelease(request.ResourceId[i], request)
	}
}

func (n *Node) sendRequest(request Request, destination int) {
        content, err := MarshalRequest(request)
        if err != nil {
                log.Fatal("sendRequest", err)
        }                       
        log.Print("Node #", n.Philosopher.Id, ",  REQUEST #", request.RequestId, ":", content, " for resources #", request.ResourceId[0], ", #", request.ResourceId[1], " to Node #", destination)       
        n.Messages[destination] <- content
}

func (n *Node) buildRequest() Request {
	var request Request
	request.MessageType = REQ_TYPE
	var resources = make([]int, n.Philosopher.NbNodes)
	for j := 0; j < n.Philosopher.NbNodes; j++ {
		resources[j] = j
	}
	request.RequesterNodeId = n.Philosopher.Id
	request.RequestId = n.RequestIdCounter
	n.RequestIdCounter ++
	request.ResourceId = make([]int, n.requestSize)
	for k := 0; k < n.requestSize; k++ {
		rand.Seed(time.Now().UnixNano())
		var idx int = rand.Intn(len(resources))
		request.ResourceId[k] = resources[idx]
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
	n.Occupant[p - 1] = n.Occupant[p]
	n.Occupant[p] = EMPTY
	n.Has_received_advance[p] = false
	n.Has_dec_sent[p] = false
	if p - 1 == 1 {
		go n.sendGrant(n.Occupant[p - 1], request)
	}
	if n.Has_dec_sent[p - 1] == false && n.Occupant[p - 2] == EMPTY {
		n.Has_dec_sent[p] = true
		go n.sendDec(p - 1, n.Occupant[p - 1], request)
	}
}

func (n *Node) hasOccupantsAfterPosition(p int) bool {
	for o, r := range n.Occupant {
		if o >= p && r != EMPTY {
			Logger.Debug("Found occupant ", r, " after position ", p);
			return true
		}
	}
	Logger.Debug("Found no occupant after position ", p);
	return false
}

func (n *Node) adjust_queue(p int, request Request) {
	Logger.Debug("Node #", n.Philosopher.Id,"** Node.adjust_queue ** p=", p);
	if p == 0 {
		return
	}

	for n.hasOccupantsAfterPosition(p) && n.Occupant[p - 1] == EMPTY {
		if n.Occupant[p] == EMPTY {
			p++
			continue
		}				
		
		//	for n.Occupant[p] != EMPTY && n.Occupant[p - 1] == EMPTY {
		if n.Has_dec_sent[p] == false || n.Occupant[p] != EMPTY{
			n.Has_dec_sent[p] = true
			go n.sendDec(p, n.Occupant[p], request)
		}
		if n.Has_received_advance[p] == true {
			n.advance_one_position(p, request)			
		}
		p ++
	}
}

func (n *Node) sendReport(dst int, request Request) {
	request.MessageType = REPORT_TYPE
	request.RequesterNodeId = n.Philosopher.Id

	content, err := MarshalRequest(request)
	if err != nil {
		Logger.Fatal("sendReport", err)
	}			
	// Logger.Debug("Node #", n.Philosopher.Id, ", send REPORT #", request.RequestId, ":", content, " to Node #", dst, ", routine #", getGID())
	Logger.Debug("Node #", n.Philosopher.Id, ", send REPORT #", request.RequestId, " to Node #", dst, ", routine #", getGID())
	n.Messages[dst] <- content
}

func (n *Node) sendSelect(position int, dst int, request Request) {
	request.MessageType = SELECT_TYPE
	request.RequesterNodeId = n.Philosopher.Id
	request.Position = position

	content, err := MarshalRequest(request)
	if err != nil {
		Logger.Fatal("sendSelect", err)
	}			
	// Logger.Debug("Node #", n.Philosopher.Id, ", send SELECT #", request.RequestId, " with position=", position, ":", content, " to Node #", dst, ", routine #", getGID())
	Logger.Debug("Node #", n.Philosopher.Id, ", send SELECT #", request.RequestId, " with position=", position, " to Node #", dst, ", routine #", getGID())
	n.Messages[dst] <- content
}

func (n *Node) sendRelease(dst int, request Request) {
	request.MessageType = RELEASE_TYPE
	request.RequesterNodeId = n.Philosopher.Id

	content, err := MarshalRequest(request)
	if err != nil {
		Logger.Fatal("sendRelease", err)
	}			
	// Logger.Debug("Node #", n.Philosopher.Id, ", send RELEASE #", request.RequestId, ":", content, " to Node #", dst, ", routine #", getGID())
	Logger.Debug("Node #", n.Philosopher.Id, ", send RELEASE #", request.RequestId, " to Node #", dst, ", routine #", getGID())
	n.Messages[dst] <- content
}

func (n *Node) sendMarked(occupied []int, dst int, request Request) {
	request.MessageType = MARKED_TYPE
	request.RequesterNodeId = n.Philosopher.Id
	request.Occupied = occupied

	content, err := MarshalRequest(request)
	if err != nil {
		Logger.Fatal("sendMarked", err)
	}			
	// Logger.Debug("Node #", n.Philosopher.Id, ", send MARKED #", request.RequestId, ":", content, " to Node #", dst, " with occupied", occupied, ", routine #", getGID())	
	Logger.Debug("Node #", n.Philosopher.Id, ", send MARKED #", request.RequestId, " to Node #", dst, " with occupied", occupied, ", routine #", getGID())	
	n.Messages[dst] <- content
}

func (n *Node) sendGrant(dst int, request Request) {
	request.MessageType = GRANT_TYPE
	request.RequesterNodeId = n.Philosopher.Id

	content, err := MarshalRequest(request)
	if err != nil {
		Logger.Fatal("sendGrant", err)
	}			
	// Logger.Debug("Node #", n.Philosopher.Id, ", send GRANT #", request.RequestId, ":", content, " to Node #", dst, ", routine #", getGID())
	Logger.Debug("Node #", n.Philosopher.Id, ", send GRANT #", request.RequestId, " to Node #", dst, ", routine #", getGID())
	n.Messages[dst] <- content
}

func (n *Node) sendAdv(position int, dst int, request Request) {
	request.MessageType = ADV_TYPE
	request.RequesterNodeId = n.Philosopher.Id
	request.Position = position

	content, err := MarshalRequest(request)
	if err != nil {
		Logger.Fatal("sendAdv", err)
	}			
	// Logger.Debug("Node #", n.Philosopher.Id, ", send ADV #", request.RequestId, " with position=", position, ":", content, " to Node #", dst, ", routine #", getGID())
	Logger.Debug("Node #", n.Philosopher.Id, ", send ADV #", request.RequestId, " with position=", position, " to Node #", dst, ", routine #", getGID())
	n.Messages[dst] <- content
}

func (n *Node) sendDec(p int, dst int, request Request) {
	request.MessageType = DEC_TYPE
	request.RequesterNodeId = n.Philosopher.Id

	content, err := MarshalRequest(request)
	if err != nil {
		Logger.Fatal("sendDec", err)
	}			
	// Logger.Debug("Node #", n.Philosopher.Id, ", send DEC #", request.RequestId, ":", content, " to Node #", dst, ", routine #", getGID())
	Logger.Debug("Node #", n.Philosopher.Id, ", send DEC #", request.RequestId, " to Node #", dst, ", routine #", getGID())
	n.Messages[dst] <- content
}

func (n *Node) receiveReport(request Request) {
	var occupied []int
	Logger.Debug("Node #", n.Philosopher.Id,"** Node.receiveReport **, req=", request.String());
	if n.Rm_critical == true {
		Logger.Fatal("Node #", n.Philosopher.Id, " already rm_critical !")
	}
	
	if n.Rm_critical == false {		
		n.Rm_critical = true
		for i := 0; i < len(n.Occupant); i ++ {
			if n.Occupant[i] != EMPTY {
				if i > 0 {
					occupied = append(occupied, i - 1)
				}
				occupied = append(occupied, i)
			}
		}
		go n.sendMarked(occupied, request.RequesterNodeId, request)
	} else {
		Logger.Info("Node #", n.Philosopher.Id, " receiveReport n.Rm_critical = true")
	}
}

func (n *Node) receiveSelect(request Request) {
	Logger.Debug("Node #", n.Philosopher.Id,"** Node.receiveSelect **, req=", request.String());
	n.Rm_critical = false
	// Logger.Debug("len(n.Occupant)=", len(n.Occupant))
	n.Occupant[request.Position] = request.RequestId
	if request.Position == 0 {
		n.sendGrant(request.RequesterNodeId, request)		
	}
	n.adjust_queue(request.Position, request)
}

func (n *Node) receiveRelease(request Request) {
	Logger.Debug("Node #", n.Philosopher.Id, "** Node.receiveRelease **");
	if n.Rm_critical == true {
		n.Occupant[0] = EMPTY
		n.adjust_queue(1, request)
	} else {
		Logger.Fatal("receiveRelease but not rm_critical")
	}
}

func (n *Node) receiveMarked(request Request) {
	var position_selected int = 0;
	n.nbMarkedRcv ++
	if len(request.Occupied) > 0 {
		var maxPosition int = 0;
		for i := 0; i < len(request.Occupied); i ++ {
			if request.Occupied[i] > maxPosition {
				maxPosition = i
			}		
		}
		position_selected = maxPosition + 1
	}
	if position_selected > n.PositionSelected[request.RequestId] {
		n.PositionSelected[request.RequestId] = position_selected
	}
	if n.nbMarkedRcv == len(request.ResourceId) {
		Logger.Debug("Node #", n.Philosopher.Id, " ALL MARKED RECEIVED")
		for k := 0; k < len(request.ResourceId); k++ {
			go n.sendSelect(n.PositionSelected[request.RequestId], request.ResourceId[k], request)
		}
		n.Req_report = false
		n.ReleaseCMCS(request)
	} else {
		Logger.Debug("Node #", n.Philosopher.Id, " is still expecting ", len(request.ResourceId) - n.nbMarkedRcv, " MARKED")
	}
}

func (n *Node) receiveGrant(request Request) {
	Logger.Debug("Node #", n.Philosopher.Id,"** Node.receiveGrant **");
	n.nbGrantRcv ++
	if n.nbGrantRcv == len(request.ResourceId) {
		Logger.Debug("Node #", n.Philosopher.Id, " ALL GRANT RECEIVED")
		n.InRheeCS = true
		n.EnterCS(request)
		n.ExecuteCSCode(request)
		n.ReleaseCS(request)
		go n.requestCS()
	} else {
		Logger.Debug("Node #", n.Philosopher.Id, " is still expecting ", len(request.ResourceId) - n.nbGrantRcv, " GRANT")
	}
}

func (n *Node) receiveAdv(request Request) {
	Logger.Debug("Node #", n.Philosopher.Id,"** Node.receiveAdv **");
	if n.Rm_critical == true {
		n.Has_received_advance[request.Position] = true
		n.adjust_queue(request.Position, request)
	} else {
		Logger.Fatal("receiveAdv but not rm_critical")
	}
}

func (n *Node) receiveDec(request Request) {
	Logger.Debug("Node #", n.Philosopher.Id,"** Node.receiveDec **");
	for k := 0; k < len(request.ResourceId); k++ {
		go n.sendAdv(request.Position, request.ResourceId[k], request)
	}
}


func (n *Node) enterCMCSIfICan(request Request) {
	var hasSentReq bool = false
	log.Print("Node #", n.Philosopher.Id, ", checking if forks are missing")
	for j := 0; j < n.Philosopher.NbNodes - 1; j++ {
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
			
			for i := 0; i < n.Philosopher.NbNodes - 1; i ++ {
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
			var requester = request.RequesterNodeId
			if (request.MessageType == REQ_TYPE) {
				var res = request.ResourceId
				Logger.Debug("Node #", n.Philosopher.Id, "<-REQ#", request.RequestId, ", Requester #", requester, ", nb of res:", len(res), " res ", res)
				go n.handleRequest(request)
			} else
			if (request.MessageType == REP_TYPE) {
				// Logger.Info("Node #", n.Philosopher.Id, ", received REPLY from Node #", requester, ",", msg)
				Logger.Info("Node #", n.Philosopher.Id, ", received REPLY from Node #", requester)
			} else if (request.MessageType == REPORT_TYPE) {
				// Logger.Info("Node #", n.Philosopher.Id, ", received REPORT from Node #", requester, ",", msg)
				Logger.Info("Node #", n.Philosopher.Id, ", received REPORT from Node #", requester)
				go n.receiveReport(request)
			} else if (request.MessageType == SELECT_TYPE) {
				// Logger.Info("Node #", n.Philosopher.Id, ", received SELECT from Node #", requester, ",", msg)
				Logger.Info("Node #", n.Philosopher.Id, ", received SELECT from Node #", requester)
				go n.receiveSelect(request)
			} else if (request.MessageType == RELEASE_TYPE) {
				// Logger.Info("Node #", n.Philosopher.Id, ", received RELEASE from Node #", requester, ",", msg)
				Logger.Info("Node #", n.Philosopher.Id, ", received RELEASE from Node #", requester)
				go n.receiveRelease(request)
			} else if (request.MessageType == MARKED_TYPE) {
				// Logger.Info("Node #", n.Philosopher.Id, ", received MARKED from Node #", requester, ",", msg)
				Logger.Info("Node #", n.Philosopher.Id, ", received MARKED from Node #", requester)
				go n.receiveMarked(request)
			} else if (request.MessageType == GRANT_TYPE) {
				// Logger.Info("Node #", n.Philosopher.Id, ", received GRANT from Node #", requester, ",", msg)
				Logger.Info("Node #", n.Philosopher.Id, ", received GRANT from Node #", requester)
				go n.receiveGrant(request)
			} else if (request.MessageType == ADV_TYPE) {
				// Logger.Info("Node #", n.Philosopher.Id, ", received ADV from Node #", requester, ",", msg)
				Logger.Info("Node #", n.Philosopher.Id, ", received ADV from Node #", requester)
				go n.receiveAdv(request)
			} else if (request.MessageType == DEC_TYPE) {
				// Logger.Info("Node #", n.Philosopher.Id, ", received DEC from Node #", requester, ",", msg)
				Logger.Info("Node #", n.Philosopher.Id, ", received DEC from Node #", requester)
				go n.receiveDec(request)
			} else if (request.MessageType == REQUEST_FORK) {
				// Logger.Info("Node #", n.Philosopher.Id, ", received REQUEST_FORK from Node #", requester, ",", msg)
				Logger.Info("Node #", n.Philosopher.Id, ", received REQUEST_FORK from Node #", requester)
				for i := 0; i < n.Philosopher.NbNodes - 1; i ++ {
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
			} else if (request.MessageType == SEND_FORK) {
				// Logger.Info("Node #", n.Philosopher.Id, ", received SEND_FORK from Node #", requester, ",", msg)
				Logger.Info("Node #", n.Philosopher.Id, ", received SEND_FORK from Node #", requester)
				log.Print(requester, ": ", n.Philosopher.Id, " <==== ", requester)	
				for i := 0; i < n.Philosopher.NbNodes - 1; i ++ {
					if (requester == n.Philosopher.ForkId[i]) {
						n.Philosopher.ForkStatus[i]    = true
						n.Philosopher.ForkClean[i]     = true
						break
					}
				}
				n.enterCMCSIfICan(request)
			} else {
				Logger.Fatal("Unknown message type=", request.MessageType)
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

		for i := 0; i < n.Philosopher.NbNodes - 1; i ++ {
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
		for j := 0; j < n.Philosopher.NbNodes - 1; j++ {
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
	request.MessageType = REQUEST_FORK
	request.RequesterNodeId = n.Philosopher.Id

	content, err := MarshalRequest(request)
	if err != nil {
		Logger.Fatal("RequestFort", err)
	}			
	// Logger.Debug("Node #", n.Philosopher.Id, ", send REQUEST_FORK #", request.RequestId, ":", content, " to Node #", dst, ", routine #", getGID())	
	Logger.Info(n.Philosopher.Id, " --", dst, "--> ", dst)	
	n.Messages[dst] <- content
	NB_MSG ++
}

func (n *Node) SendFork(dst int, request Request) {
	request.MessageType = SEND_FORK
	request.RequesterNodeId = n.Philosopher.Id

	content, err := MarshalRequest(request)
	if err != nil {
		Logger.Fatal("SendFork", err)
	}			
	// Logger.Debug("Node #", n.Philosopher.Id, ", send SEND_FORK #", request.RequestId, ":", content, " to Node #", dst, ", routine #", getGID())	
	Logger.Info(n.Philosopher.Id,": ", n.Philosopher.Id, " ====> ", dst)	
	n.Messages[dst] <- content
	NB_MSG ++
}

func (n *Node) requestCS() {
	Logger.Info("Node #", n.Philosopher.Id, " requestCS")

	var waitingTime int = rand.Intn(100)
	time.Sleep(time.Duration(waitingTime) * time.Millisecond)

	var request Request = n.buildRequest()
	
	var requester = request.RequesterNodeId
	var res = request.ResourceId
	Logger.Debug("Node #", n.Philosopher.Id, "<-REQ#", request.RequestId, ", Requester #", requester, ", nb of res:", len(res), " res ", res)
	n.handleRequest(request)

	Logger.Info("Node #", n.Philosopher.Id," END requestCS")	
}

func (n *Node) Rhee(wg *sync.WaitGroup) {
	Logger.Info("Node #", n.Philosopher.Id)

	go n.requestCS()
	go n.rcv()
	for {
		time.Sleep(100 * time.Millisecond)
		if CURRENT_ITERATION == n.Philosopher.NbIterations {
			break
		}
	}

	Logger.Info("Node #", n.Philosopher.Id," END after ", n.Philosopher.NbIterations," CS entries")	
	wg.Done()
}

func Init(nbNodes int, nbIterations int, requestSize int) {
	Logger.SetLevel(log.DebugLevel)
	// Logger.SetLevel(log.InfoLevel)
	Logger.Print("Rhee.Init")	
	ChandyMisra.Init(nbNodes, nbIterations)

	gob.Register(Node{})
	
	Nodes = make([]Node, nbNodes)
	var messages = make([]chan bytes.Buffer, nbNodes)
	var philosopherMessages = make([]chan string, nbNodes)
	var has_received_advance = make([]bool, nbNodes)
	var has_dec_sent = make([]bool, nbNodes)

	Logger.Info("nb_process #", nbNodes)
	
	for i := 0; i < nbNodes; i++ {		
		ChandyMisra.InitPhilosopher(&Nodes[i].Philosopher, i , nbNodes, nbIterations)
		philosopherMessages[i] = make(chan string)
		messages[i] = make(chan bytes.Buffer)
		Nodes[i].RequestIdCounter = i * 100
		Nodes[i].PositionSelected = make(map[int]int)
		Nodes[i].Occupant = make(map[int]int)
		Nodes[i].Has_received_advance = make([]bool, nbNodes)
		Nodes[i].Has_dec_sent = make([]bool, nbNodes)
		Nodes[i].Rm_critical = false
		Nodes[i].Req_report = false
		Nodes[i].requestSize = requestSize
		// occupants[i] = EMPTY
		has_received_advance[i] = false
		has_dec_sent[i] = false
	}

	for i := 0; i < nbNodes; i++ {
		Nodes[i].Philosopher.Messages = philosopherMessages
		Nodes[i].Messages = messages
		copy(Nodes[i].Has_received_advance, has_received_advance)
		copy(Nodes[i].Has_dec_sent, has_dec_sent)
	} 
}
