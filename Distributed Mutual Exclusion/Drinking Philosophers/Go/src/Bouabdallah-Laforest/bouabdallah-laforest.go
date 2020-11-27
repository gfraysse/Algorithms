/*
  Copyright "Guillaume Fraysse <gfraysse dot spam plus code at gmail dot com>"

How-to run: 
  go run bouabdallah-laforest.go 2>&1 |tee /tmp/tmp.log

Parameters:
- Number of nodes is set with NB_NODES global variable
- Number of CS entries is set with NB_ITERATIONS global variable
*/ 

/*
    Go implementation of Bouabdallah-Laforest mutual exclusion algorithm
    Algorithm by Abdelmadjid Bouabdallah and Christian Laforest 2000

References : 
 * https://doi.org/10.1145/506117.506125 : A. Bouabdallah and C. Laforest. 2000. A distributed token-based algorithm for the dynamic resource allocation problem. SIGOPS Oper. Syst. Rev. 34, 3 (July 2000), 60–68. 
*/
package main

/*
  logrus package is used for logs: https://github.com/sirupsen/logrus
*/

import (
	// "bufio"
	"bytes" // for gid
	"encoding/gob"
	"fmt"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"os"
	"runtime" // for debugging purpose
	"strconv"
	"sync"
	"time"
)

/* global variable declaration */
var NB_NODES          int = 4
var REQUEST_SIZE      int = 2
var NB_ITERATIONS     int = 10
var CURRENT_ITERATION int = 0

var BL_FREE    bool = false
var BL_LOCKED  bool = true

var REQ_TYPE      int = 0
var REP_TYPE      int = 1
var REQ_CT_TYPE   int = 2
var REP_CT_TYPE   int = 3
var INQUIRE_TYPE  int = 4
var ACK1_TYPE     int = 5
var ACK2_TYPE     int = 6

var logger = log.New()
/*
// Debug function
func displayNodes() {
	for i := 0; i < len(nodes); i++ {
		logger.Print("Node #", nodes[i].id, ", last=", nodes[i].last, ", next=", nodes[i].next)
	}
}
*/
type Token struct {
	id     int
	locked bool
}

type ControlToken struct {	
	A []int
	B map[int][]int
}

var ControlTokenInstance ControlToken

type Request struct {
	RequesterNodeId int
	RequestId      int
	MessageType    int
	ResourceId     []int
}

type Node struct {
	// From the algorithm
	id                        int
	has_CT                    bool
	tokens                    []Token
	tokensNeeded              []int
	requesting                bool
	currentlyRequestingTokens []int
	currentRequest            Request
	RequestIdCounter          int
	waitingSet                map[int][]int
	next                      int // the dynamic distributed list
	last                      int // called father in the original paper. Called last here as in Sopena et al. as it stores the last requester
	inBLCS                    bool
	requestInCS               Request
	// Implementation specific
	mutex          sync.Mutex
	nbCS           int // the number of time the node entered its Critical Section
	queue          []Request
	messages       []chan bytes.Buffer
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

	for i := 0; i < len(r.ResourceId); i ++ {
		res +=  strconv.Itoa(r.ResourceId[i]) + ", "
	}

	val = fmt.Sprintf("Request #=%d, requester node=%d, messageType=%d, resources={%s}",
		r.RequestId,
		r.RequesterNodeId,
		r.MessageType,
		res)
	return val
}

////////////////////////////////////////////////////////////
// ControlToken class
////////////////////////////////////////////////////////////
func (ct *ControlToken) String() string {
	var val string
	var A string = ""
	var B string = ""

	for i := 0; i < len(ct.A); i ++ {
		if ct.A[i] != -1 {
			A += strconv.Itoa(ct.A[i]) + ", "
		}
	}
	for key, _ := range ct.B {
		B += "{" + strconv.Itoa(key) + "("
		for i := 0; i < len(ct.B[key]); i ++ {
			B += strconv.Itoa(ct.B[key][i]) + ", "
		}
		B += ")}"
	}

	val = fmt.Sprintf("ControlToken A={%s}, B={%s}",
		A,
		B)
	return val
}

func (ct *ControlToken) getTokensPossessedByNode(n *Node) []int{
	return ct.B[n.id];
}

func (ct *ControlToken) addFreeToken(token int) {
	ct.A = append(ct.A, token)
}

func (ct *ControlToken) checkCT() bool {
	for n := 0; n < NB_NODES; n++ {
		var found int = 0
		for j := 0; j < len(ct.A); j ++ {
			if (ct.A[j] == n) {
				found ++
			}
		}
		for key, _ := range ct.B {
			for i := 0; i < len(ct.B[key]); i ++ {
				if ct.B[key][i] == n {
					found ++;
				}
			}
		}
		if found != 1 {
			logger.Debug("Found ", found, " instance of token #", n, " in ControlToken:", ct.String())
			return false
		}
	}
	return true
}

func (ct *ControlToken) removeNeededTokensFromFreeTokens(node *Node) {
	logger.Debug("Node #", node.id, ", BEGIN ControlToken.removeNeededTokensFromFreeTokens ct.A=", ct.A, " node.tokensNeeded=", node.tokensNeeded)
	
	var tokens []int = make ([]int, len(node.tokensNeeded))
	copy(tokens, node.tokensNeeded)
	
	for i := 0; i < len(tokens); {
		var found bool = false;
		for j := 0; j < len(ct.A); j ++ {
			if (ct.A[j] == tokens[i]) {
				found = true;
				var token Token
				token.id = tokens[i]
				token.locked = BL_LOCKED
				node.tokens = append(node.tokens, token)
				ct.A[j] = ct.A[len(ct.A) - 1]
				ct.A = ct.A[:len(ct.A) - 1]

				tokens[i] = tokens[len(tokens) - 1]
				tokens = tokens[:len(tokens) - 1]
				break
			}
		}
		if (!found) {
			i++
		}
	}
	
	logger.Debug("Node #", node.id, ", END ControlToken.removeNeededTokensFromFreeTokens ct.A=", ct.A, " node.tokensNeeded=", node.tokensNeeded)
}

func (ct *ControlToken) isTokenPossessedByNode(token int) bool {
	logger.Debug("ControlToken.isTokenPossessedByNode")
    for key, _ := range ct.B {
		for i := 0; i < len(ct.B[key]); i ++ {
			if ct.B[key][i] == token {
				return true;
			}
		}
	}
    return false;
}

func (ct *ControlToken) getTokenOwnerFromPossessedByNode(token int) int{
	logger.Debug("ControlToken.getTokenOwnerFromPossessedByNode token=", token)
	for key, _ := range ct.B {
		for i := 0; i < len(ct.B[key]); i ++ {
			if token == ct.B[key][i] {
				logger.Debug("token ", token, ", found for key ", key)
				return key;
			}
		}
	}
	return -1;
}

func (ct *ControlToken) removeTokenFromPossessedByNode(token int) int {
	logger.Debug("BEGIN ControlToken.removeTokenFromPossessedByNode", ct.String())
	for key, _ := range ct.B {
		for i := 0; i < len(ct.B[key]); i ++ {
			if token == ct.B[key][i] {
				ct.B[key][i] = ct.B[key][len(ct.B[key]) - 1]
				ct.B[key] = ct.B[key][:len(ct.B[key]) - 1]
				logger.Debug("END1 ControlToken.removeTokenFromPossessedByNode", ct.String())
				return key
			}
		}
	}
	logger.Debug("END2 ControlToken.removeTokenFromPossessedByNode", ct.String())
	return -1
}

func (ct *ControlToken) updateForRequest(
	n *Node,
	request Request,
	// requestedTokens []Token,
	// tokensOwned []Token,
	missingTokens *([]int),
	requestedResourcesForNode *(map[int][]int)) {
	
	logger.Debug("Node #", n.id, ", BEGIN ControlToken.updateForRequest, ", ct.String(), ", routine #", getGID())
	// First move the tokens already owned by nodeName in B to A
	var tokensPossessedByNode []int = ct.getTokensPossessedByNode(n)
	for i := 0; i < len(tokensPossessedByNode); i ++ {
		ct.addFreeToken(tokensPossessedByNode[i])
	}
	logger.Debug(ct.String())
	
	// Remove needed tokens from A
	// ct.removeNeededTokensFromFreeTokens(requestedTokens, tokensOwned);
	n.tokensNeeded = request.ResourceId
	logger.Debug("Node #", n.id, ", n.tokensNeeded", n.tokensNeeded, ", routine #", getGID())
	ct.removeNeededTokensFromFreeTokens(n);
	logger.Debug("Node #", n.id, ", n.tokensNeeded 2", n.tokensNeeded, ", routine #", getGID())

	logger.Debug("Node #", n.id, ", updateForRequest 1 ", ct.String(), ", routine #", getGID())
	
	// First remove the tokens from B if already present
	// Identify owner in B of requestedTokens to build
	// requestedResourcesForNode dictionnary
	// for i := 0; i < len(requestedTokens); i ++ {
	for i := 0; i < len(n.tokensNeeded); i ++ {
		var owner int = ct.getTokenOwnerFromPossessedByNode(n.tokensNeeded[i]);
		if owner != n.id && owner != -1 {
			(*requestedResourcesForNode)[owner] = append((*requestedResourcesForNode)[owner], n.tokensNeeded[i]);
		}
	}
	
	logger.Debug("Node #", n.id, ", ControlToken.updateForRequest requestedResourcesForNode ", requestedResourcesForNode, ", routine #", getGID())
	logger.Debug("Node #", n.id, ", missingTokens", missingTokens)
	for i := 0; i < len(request.ResourceId); i++ {
		var token int = request.ResourceId[i]
		if ct.isTokenPossessedByNode(token) {
			logger.Debug("Node #", n.id, ", isTokenPossessedByNode true")
			var owner int = ct.getTokenOwnerFromPossessedByNode(token);
			if owner != n.id{
				owner = ct.removeTokenFromPossessedByNode(token);
				*missingTokens = append(*missingTokens, token);
			}
		}
	}
	logger.Debug("Node #", n.id, ", BEFORE ", ct.String())
	logger.Debug("Node #", n.id, ", missingTokens ", missingTokens)
	logger.Debug("Node #", n.id, ", BEFORE n.tokensNeeded ", n.tokensNeeded)
	ct.B[n.id] = n.tokensNeeded
	logger.Debug("Node #", n.id, ", AFTER ", ct.String())

	// Put unneeded tokens in Control Token
	// for i := 0; i < len(tokensOwned);  {
	for i := 0; i < len(n.tokens);  {
		if n.tokens[i].locked == BL_FREE {
			n.tokens[i] = n.tokens[len(n.tokens) - 1]
			n.tokens = n.tokens[:len(n.tokens) - 1]
		} else {
			i++
		}
	}
	logger.Debug(ct.String())
	var found bool = ct.checkCT()
	if found == false {
		logger.Debug("Node #", n.id, " !!!! ControlToken.updateForRequest inconsistent ControlToken !!!!", ", routine #", getGID())
		os.Exit(1)
	}
	logger.Debug("Node #", n.id, ", END ControlToken.updateForRequest", ct.String(), ", routine #", getGID())
}

////////////////////////////////////////////////////////////
// Node class
////////////////////////////////////////////////////////////
func (n *Node) String() string {
	var val string
	val = fmt.Sprintf("Node #%d next=%d, last=%d",
		n.id,
		n.next,
		n.last)
	return val
}

func (n *Node) enterCS() {
	logger.Debug("Node #", n.id, " ######################### enterCS")
	logger.Debug("Node #", n.id, ", ", ControlTokenInstance.String())
	CURRENT_ITERATION ++
	n.nbCS ++
	logger.Debug(n)
}

func (node *Node) executeCSCode() {
	logger.Debug("Node #", node.id, " ######################### executeCSCode")
	logger.Debug(node)
	time.Sleep(500 * time.Millisecond)
}

func (n *Node) releaseCS() {
	logger.Debug("Node #", n.id," releaseCS #########################")	
	n.leaveBLCS()
	logger.Debug(n)
}

func (n *Node) leaveBLCS() {
	logger.Debug("Node #", n.id, ", BEGIN leaveBLCS", ", routine #", getGID())
	for i := 0; i < len(n.tokens); i ++ {
		n.tokens[i].locked = BL_FREE
	}
	n.tokensNeeded = make([]int, 0)
	// n.tokens = make([]Token, 0)
	for key, _ := range n.waitingSet {
		if len(n.waitingSet[key]) > 0 {
			var tokens []int = n.waitingSet[key]
			go n.sendACK2(&tokens, key)
		}
	}
	n.waitingSet = make(map[int][]int)
	
	if n.has_CT == true {
		var tokensPossessedByNode []int = ControlTokenInstance.getTokensPossessedByNode(n)
		for i := 0; i < len(tokensPossessedByNode); i++ {
			ControlTokenInstance.addFreeToken(tokensPossessedByNode[i])
			ControlTokenInstance.removeTokenFromPossessedByNode(tokensPossessedByNode[i])
			var found bool = ControlTokenInstance.checkCT()
			if found == false {
				logger.Debug("Node #", n.id, "!!!! leaveBLCS inconsistent ControlToken !!!!", ", routine #", getGID())
				os.Exit(1)
			}
		}
		if n.next != -1 {
			n.last = n.next
			go n.sendCT(n.next)
			n.next = -1
		}
	}	
	logger.Debug("Node #", n.id, ", END leaveBLCS", ", routine #", getGID())
	go n.requestCS()
}

func (n *Node) ownsToken(id int) bool {
	for i := 0; i < len(n.tokens); i++ {
		if n.tokens[i].id == id {
			return true
		}
	}
	return false
}

func (n *Node) lockResource(id int) {
	for i := 0; i < len(n.tokens); i++ {
		if n.tokens[i].id == id {
			n.tokens[i].locked = true
			return
		}
	}	
}

func (n *Node) sendCT(dst int) {
	var request Request
	request.RequesterNodeId = n.id
	request.RequestId = 0
	request.MessageType = REP_CT_TYPE
	
	content, err := MarshalRequest(request)
	if err != nil {
		logger.Fatal(err)
	}			
	// logger.Debug("Node #", n.id, ",  SEND CT #", request.RequestId, ":", content, " to Node #", dst, ", routine #", getGID())	
	logger.Debug("Node #", n.id, ",  SEND CT #", request.RequestId, " to Node #", dst, ", routine #", getGID())	
	n.has_CT = false
	n.messages[dst] <- content
}

func (n *Node) handleCTRequest(request Request) {
	logger.Debug("Node #", n.id, ",  handleCTRequest request:", request.String(), ", routine #", getGID())
	if n.last == -1 {
		if n.requesting {
			n.next = request.RequesterNodeId
		} else {
			go n.sendCT(request.RequesterNodeId)
		}		
	} else {
		// Code duplication to remove
		var fwdRequest Request
		fwdRequest.MessageType = REQ_CT_TYPE
		fwdRequest.RequesterNodeId = request.RequesterNodeId
		
		content, err := MarshalRequest(fwdRequest)
		if err != nil {
			logger.Fatal(err)
		}			
		// logger.Debug("Node #", n.id, ",  FWD REQUEST CT #", request.RequestId, ":", content, " to Node #", n.last)	
		logger.Debug("Node #", n.id, ",  FWD REQUEST CT #", request.RequestId, " to Node #", n.last)	
		n.messages[n.last] <- content
	}
	n.last = request.RequesterNodeId
	logger.Debug("Node #", n.id, " handleCTRequest, *update* n.last #", n.last)
}

func (n *Node) enterBLCS(request Request) bool {
	logger.Debug("Node #", n.id, " enterBLCS, request:", request.String())
	n.enterCS()
	n.executeCSCode()
	n.releaseCS()
	return false
}

func (n *Node) hasAllTokensForRequest(request Request) bool {
	var resourcesRequested []int = request.ResourceId

	for i := 0; i < len(resourcesRequested); i++ {
		var hasToken bool = false
		for j := 0; j < len(n.tokens); j++ {
			if n.tokens[j].id == resourcesRequested[i] {
				hasToken = true
				break
			}
		}
		if (hasToken == false) {
			logger.Debug("Node #", n.id, " is missing token ", resourcesRequested[i])
			return false
		}
	}
	return true
}

func (n *Node) enterCSIfCan(request Request) bool {
	logger.Debug("Node #", n.id, ", enterCSIfCan")
	if n.hasAllTokensForRequest(request) {
		logger.Debug("Node #", n.id, ", ENTERS BLCS")
		n.inBLCS = true
		n.requestInCS = request

		n.enterBLCS(request)

		// Send the Control Token
		n.requesting = false
		
		// Finished using the Control Token, keep it going if there is a Next
		if n.next != -1 {
			go n.sendCT(n.next)
			n.last = n.next
			n.next = -1
		}
		return true
	}
	logger.Debug("Node #", n.id, ", CANNOT enterCSIfCan")
	return false
}

func (n *Node) requestTokens(request Request, dst int, tokens[]int) {
	logger.Debug("Node #", n.id, ", requestTokens")
	n.currentlyRequestingTokens = tokens
	var inquireRequest Request
	inquireRequest.MessageType = INQUIRE_TYPE
	inquireRequest.RequestId = request.RequestId
	inquireRequest.RequesterNodeId = n.id
		
	inquireRequest.ResourceId = make([]int, REQUEST_SIZE)
	for i := 0; i < len(tokens); i++ {
		inquireRequest.ResourceId[i] = tokens[i]
	}
	if len(tokens) != REQUEST_SIZE {
		for i := len(tokens); i < REQUEST_SIZE; i++ {
			inquireRequest.ResourceId[i] = -1
		}
	}
	content, err := MarshalRequest(inquireRequest)
	if err != nil {
		logger.Fatal(err)
	}			
	// logger.Debug("Node #", n.id, ", send INQUIRE #", inquireRequest.RequestId, ":", content, " to Node #", dst, " for res ", tokens)	
	logger.Debug("Node #", n.id, ", send INQUIRE #", inquireRequest.RequestId, " to Node #", dst, " for res ", tokens)	
	n.messages[dst] <- content
}

func (n *Node) addTokenToSet(token Token, status bool) {
	token.locked = status
	n.tokens = append(n.tokens, token)
}

func (n *Node) updateCTForRequest(request Request) {
	var missingTokens             []int
	var requestedResourcesForNode map[int][]int
	requestedResourcesForNode = make(map[int][]int)

	n.mutex.Lock()
	logger.Debug("Node #", n.id, ", updateCTForRequest")
	ControlTokenInstance.updateForRequest(n, request, &missingTokens, &requestedResourcesForNode)
	n.mutex.Unlock()

	if len(missingTokens) == 0 {
		logger.Debug("Node #", n.id, ", updateCTForRequest 1 ")
		n.enterCSIfCan(request);
	} else {
		for i := 0; i < len(missingTokens); {
			if n.ownsToken(missingTokens[i]) == true {
				missingTokens[i] = missingTokens[len(missingTokens) - 1]
				missingTokens = missingTokens[:len(missingTokens) - 1]
			} else {
				i++
			}
		}
		if len(missingTokens) == 0 {
			logger.Debug("Node #", n.id, ", updateCTForRequest 2")
			n.enterCSIfCan(request);
		} else {
			logger.Debug("Node #", n.id, ", updateCTForRequest 3 ", requestedResourcesForNode)
			logger.Debug("Node #", n.id, ", updateCTForRequest ct=", ControlTokenInstance.String())
			for i := 0; i < len(requestedResourcesForNode); i ++ {
				logger.Debug("Node #", n.id, ", updateCTForRequest requestedResourcesForNode=", requestedResourcesForNode)
				for key, _ := range requestedResourcesForNode {
					logger.Debug("Node #", n.id, ", updateCTForRequest requestedResourcesForNode[", key, "]=", requestedResourcesForNode[key])
					var tokens []int = requestedResourcesForNode[key]
					logger.Debug("Node #", n.id, ", updateCTForRequest node ", key, " holds tokens", tokens)
					
					go n.requestTokens(request, key, tokens);
					break
				}
			}
		}
	}
}

func (n *Node) handleRequest(request Request) {
	logger.Debug("Node #", n.id," handleRequest")	
	var hasAllTokens bool = true
	for i := 0; i < REQUEST_SIZE; i++ {
		if ! n.ownsToken(request.ResourceId[i]) {
			hasAllTokens = false
		}
	}
	if hasAllTokens {
		logger.Debug("Node #", n.id," handleRequest hasAllTokens")	
		for i := 0; i < REQUEST_SIZE; i++ {
			n.lockResource(request.ResourceId[i])
		}
		n.enterCSIfCan(request)
	} else {
		logger.Debug("Node #", n.id," handleRequest NOT hasAllTokens")
		n.currentRequest = request
		if n.has_CT == true {
			logger.Debug("Node #", n.id," handleRequest NOT hasAllTokens 1")	
			n.updateCTForRequest(request)
		} else {
			logger.Debug("Node #", n.id," handleRequest NOT hasAllTokens 2")
			if n.requesting == false {
				go n.requestCT()
			}
		}
	}
}

func (n *Node) isTokenInSet(token int) bool{
    for i:= 0; i < len(n.tokens); i ++ {
        if n.tokens[i].id == token {
            return true
        }
    }
    return false
}

func (n *Node) isTokenLocked(token int) bool{
    for i:= 0; i < len(n.tokens); i ++ {
	    if n.tokens[i].id == token {
		    if n.tokens[i].locked == BL_LOCKED {
			    return true
		    } else {
			    return false
		    }
        }
    }
    return false
}

func (n *Node) removeTokenFromSet(token int) bool{
    for i:= 0; i < len(n.tokens); i ++ {
	    if n.tokens[i].id == token {
		    n.tokens[i] = n.tokens[len(n.tokens) - 1]
		    n.tokens = n.tokens[:len(n.tokens) - 1]
		    return true
	    }
    }
    return false
}

func (n *Node) isTokenNeeded(token int) bool{
    for i:= 0; i < len(n.tokensNeeded); i ++ {
        if n.tokensNeeded[i] == token {
            return true
        }
    }
    return false
}

func (n *Node) receiveInquire(request Request) {
	logger.Debug("** Node #", n.id, "  receiveInquire ******************")

	var requester int = request.RequesterNodeId
	var sentTokens []int
	var notSentTokens []int
	for i := 0; i < len(request.ResourceId); i++ {
		var token int = request.ResourceId[i]
		logger.Debug("** Node #", n.id, "  receiveInquire i=", i, " token=", token)
		if token != -1 {
			if n.isTokenInSet(token) && !n.isTokenLocked(token) {
				logger.Debug("removeFromSet", token, "n ",n)
				n.removeTokenFromSet(token)			
				sentTokens = append(sentTokens, token)
				logger.Debug("removeFromSet", token, "n ", n, " end")
			} else {
				notSentTokens = append(notSentTokens, token)
				logger.Debug("notSentTokens", notSentTokens)
			}
		}
	}

	n.waitingSet[requester] = make([]int, len(notSentTokens))
	copy (n.waitingSet[requester], notSentTokens)
	
	if len(sentTokens) > 0 {
		go n.sendACK1(&sentTokens, requester)
	} else {
		logger.Debug("** Node #", n.id, ", no ACK1 sent")
	}
	logger.Debug("n=", n)
}

func (n *Node) sendACK1(sentTokens *([]int), dst int) {
	var ack1Request Request
	ack1Request.MessageType = ACK1_TYPE
	ack1Request.RequesterNodeId = n.id
		
	ack1Request.ResourceId = make([]int, REQUEST_SIZE)
	for i := 0; i < len(*sentTokens); i++ {
		ack1Request.ResourceId[i] = (*sentTokens)[i]
	}
	if len(*sentTokens) != REQUEST_SIZE {
		for i := len(*sentTokens); i < REQUEST_SIZE; i++ {
			ack1Request.ResourceId[i] = -1
		}
	}
	content, err := MarshalRequest(ack1Request)
	if err != nil {
		logger.Fatal(err)
	}			
	// logger.Debug("Node #", n.id, ", send ACK1 #", ack1Request.RequestId, ":", content, " to Node #", dst, " with tokens", ack1Request.ResourceId, ", routine #", getGID())	
	logger.Debug("Node #", n.id, ", send ACK1 #", ack1Request.RequestId, " to Node #", dst, " with tokens", ack1Request.ResourceId, ", routine #", getGID())	
	n.messages[dst] <- content
}

func (n *Node) sendACK2(tokens *([]int), dst int) {
	var ack2Request Request
	ack2Request.MessageType = ACK2_TYPE
	ack2Request.RequesterNodeId = n.id
		
	ack2Request.ResourceId = make([]int, REQUEST_SIZE)
	for i := 0; i < len(*tokens); i++ {
		ack2Request.ResourceId[i] = (*tokens)[i]
	}
	if len(*tokens) != REQUEST_SIZE {
		for i := len(*tokens); i < REQUEST_SIZE; i++ {
			ack2Request.ResourceId[i] = -1
		}
	}
	content, err := MarshalRequest(ack2Request)
	if err != nil {
		logger.Fatal(err)
	}			
	// logger.Debug("Node #", n.id, ", send ACK2 #", ack2Request.RequestId, ":", content, " to Node #", dst, " with tokens", ack2Request.ResourceId, ", routine #", getGID())	
	logger.Debug("Node #", n.id, ", send ACK2 #", ack2Request.RequestId, " to Node #", dst, " with tokens", ack2Request.ResourceId, ", routine #", getGID())	
	n.messages[dst] <- content
}

func (n *Node) receiveACK1(request Request) {
	logger.Debug("** Node #", n.id, "  receiveACK1 *******", ", routine #", getGID())
	var requestTokens []int = request.ResourceId
	for i := 0; i < len(requestTokens); i ++ {
		var token Token
		token.id = requestTokens[i]
		logger.Debug("** Node #", n.id, "  receiveACK1, token ", token.id, " received")
		token.locked = BL_LOCKED		
		n.addTokenToSet(token, BL_LOCKED)
		
		n.removeTokenFromCurrentlyRequestingTokens(requestTokens[i])
	}
	logger.Debug("Node #", n.id, " is still waiting for ", len(n.currentlyRequestingTokens), " tokens:", n.currentlyRequestingTokens)
	n.enterCSIfCan(request)
}

func (n *Node) removeTokenFromCurrentlyRequestingTokens(token int)  {
	for i := 0; i < len(n.currentlyRequestingTokens); i ++ {
		if token == n.currentlyRequestingTokens[i] {
			n.currentlyRequestingTokens[i] = n.currentlyRequestingTokens[len(n.currentlyRequestingTokens) - 1]
			n.currentlyRequestingTokens = n.currentlyRequestingTokens[:len(n.currentlyRequestingTokens) - 1]				
			break
		}
	}
}
func (n *Node) receiveACK2(request Request) bool {
	logger.Debug("** Node #", n.id, "  receiveACK2 *******", ", routine #", getGID())
	var requestTokens []int = request.ResourceId

	for i := 0; i < len(requestTokens); i ++ {
		var token Token
		token.id = requestTokens[i]
		logger.Debug("** Node #", n.id, "  receiveACK2, token ", token.id, "received")
		token.locked = BL_LOCKED		
		n.addTokenToSet(token, BL_LOCKED)

		n.removeTokenFromCurrentlyRequestingTokens(requestTokens[i])
	}
	logger.Debug("Node #", n.id, " is still waiting for ", len(n.currentlyRequestingTokens), " tokens:", n.currentlyRequestingTokens)
	n.enterCSIfCan(request)
	return true
}

func (n *Node) receiveCT() {
	logger.Debug("** Node #", n.id, " Got TOKEN **", ", routine #", getGID())
	logger.Debug("** Node #", n.id, ", CT=", ControlTokenInstance.String())
 	logger.Debug("** Node #", n.id, " needs **", n.currentRequest.ResourceId)
	n.has_CT = true
 	logger.Debug("** Node #", n.id, " tokens before=", n.tokens)
	var tokens []Token = make([]Token, len(ControlTokenInstance.B[n.id]) + len (n.tokens))
	for i := 0; i < len (n.tokens); i++ {
		tokens[i] = n.tokens[i]
	}
	for i := len (n.tokens); i < len (n.tokens) + len(ControlTokenInstance.B[n.id]); i++ {
		tokens[i].id = ControlTokenInstance.B[n.id][i - len (n.tokens)]
	}
	n.tokens = tokens
 	logger.Debug("** Node #", n.id, " tokens after=", n.tokens)
	
	for i := 0; i < len(n.currentRequest.ResourceId); i++ {
		var token int = n.currentRequest.ResourceId[i]
		if !n.isTokenInSet(token) && !n.isTokenNeeded(token) {
			n.tokensNeeded = append (n.tokensNeeded, token)
		} else {
			n.lockResource(token)
		}
	}

	n.updateCTForRequest(n.currentRequest)
	logger.Debug("** Node #", n.id, "**", ControlTokenInstance.String())
	logger.Debug(n)
	logger.Debug("** Node #", n.id, ", END receiveCT")
}

func (n *Node) rcv() {	
	logger.Debug("Node #", n.id," rcv", ", routine #", getGID())	
	for {
		select {
		case msg := <-n.messages[n.id]:
			var request Request
			err := UnmarshalRequest(msg, &request)
			if err != nil {
				logger.Fatal(err)
			}			
			var requester = request.RequesterNodeId
			if (request.MessageType == REP_TYPE) {
				// logger.Info("Node #", n.id, ", received REPLY from Node #", requester, ",", msg)
				logger.Info("Node #", n.id, ", received REPLY from Node #", requester)
			} else if (request.MessageType == REQ_CT_TYPE) {
				// logger.Info("Node #", n.id, ", received REQUEST Control Token from Node #", requester, ",", msg)
				logger.Info("Node #", n.id, ", received REQUEST Control Token from Node #", requester)
				n.mutex.Lock()
				n.handleCTRequest(request)
				n.mutex.Unlock()
			} else if (request.MessageType == REP_CT_TYPE) {
				// logger.Info("Node #", n.id, ", received REPLY Control Token from Node #", requester, ",", msg)
				logger.Info("Node #", n.id, ", received REPLY Control Token from Node #", requester)
				go n.receiveCT()
			} else if (request.MessageType == INQUIRE_TYPE) {
				// logger.Info("Node #", n.id, ", received INQUIRE from Node #", requester, ",", msg)
				logger.Info("Node #", n.id, ", received INQUIRE from Node #", requester)
				go n.receiveInquire(request)
			} else if (request.MessageType == ACK1_TYPE) {
				// logger.Info("Node #", n.id, ", received ACK1 from Node #", requester, ",", msg)
				logger.Info("Node #", n.id, ", received ACK1 from Node #", requester)
				go n.receiveACK1(request)
			} else if (request.MessageType == ACK2_TYPE) {
				// logger.Info("Node #", n.id, ", received ACK2 from Node #", requester, ",", msg)
				logger.Info("Node #", n.id, ", received ACK2 from Node #", requester)
				go n.receiveACK2(request)
			} else {
				logger.Fatal("Fatal Error")
			}
		}
	}
	logger.Debug(n)
	logger.Debug("Node #", n.id, " end rcv")
}

func (n *Node) requestCT() {
	logger.Debug(n)
	var request Request
	request.RequesterNodeId = n.id
	request.RequestId = 0
	request.MessageType = REQ_CT_TYPE
	
	content, err := MarshalRequest(request)
	if err != nil {
		logger.Fatal(err)
	}			
	// logger.Debug("Node #", n.id, ",  REQUEST CT #", request.RequestId, ":", content, " to Node #", n.last)
	logger.Debug("Node #", n.id, ",  REQUEST CT #", request.RequestId, " to Node #", n.last)
	n.mutex.Lock()
	n.requesting = true	
	n.mutex.Unlock()

	n.messages[n.last] <- content

	n.mutex.Lock()
	n.last = -1		
	n.mutex.Unlock()
}

func (n *Node) buildRequest() Request {
	var request Request
	request.MessageType = REQ_TYPE
	var resources = make([]int, NB_NODES)
	for j := 0; j < NB_NODES; j++ {
		resources[j] = j
	}
	request.RequesterNodeId = n.id
	request.RequestId = n.RequestIdCounter
	n.RequestIdCounter ++
	request.ResourceId = make([]int, REQUEST_SIZE)
	for k := 0; k < REQUEST_SIZE; k++ {
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

func (n *Node) requestCS() {
	logger.Debug("Node #", n.id, " requestCS", ", routine #", getGID())
	
	var waitingTime int = rand.Intn(100)
	time.Sleep(time.Duration(waitingTime) * time.Millisecond)

	var request Request = n.buildRequest()
	
	var requester = request.RequesterNodeId
	var res = request.ResourceId
	logger.Debug("Node #", n.id, "<-REQ#", request.RequestId, ", Requester #", requester, ", nb of res:", len(res), " res ", res)
	n.handleRequest(request)
	
	logger.Debug("Node #", n.id," END requestCS", ", routine #", getGID())	
}

func (n *Node) BouabdallahLaforest(wg *sync.WaitGroup) {
	logger.Debug("Node #", n.id)

	go n.requestCS()
	go n.rcv()
	for {
		time.Sleep(500 * time.Millisecond)
		if CURRENT_ITERATION > NB_ITERATIONS {
			break
		}
	}

	// logger.Debug("Node #", n.id," END after ", NB_ITERATIONS," CS entries")	
	wg.Done()
}

func main() {
	var nodes = make([]Node, NB_NODES)
	var wg sync.WaitGroup
	var messages = make([]chan bytes.Buffer, NB_NODES)

	// logger.SetLevel(log.DebugLevel)
	logger.SetLevel(log.InfoLevel)
	logger.Info("nb_process #", NB_NODES)

	// Initialization
	for i := 0; i < NB_NODES; i++ {
		nodes[i].id = i
		nodes[i].nbCS = 0
		
		// nodes[i].has_CT = false
		nodes[i].requesting = false
		nodes[i].next = -1
		nodes[i].last = 0
		nodes[i].RequestIdCounter = i * 10
		
		// Initially the first node holds the Control Token
		if nodes[i].last == nodes[i].id {
			nodes[i].has_CT = true
			nodes[i].last = -1
		} else {
			nodes[i].has_CT = false
		}

		messages[i] = make(chan bytes.Buffer)
		
		nodes[i].waitingSet = make(map[int][]int)

		// Initialize the Control Token such that A contains all free tokens
		var t Token
		t.id = i
		t.locked = false
		ControlTokenInstance.A = append(ControlTokenInstance.A, i)
		// Initially each node owns its token
		nodes[i].tokens = append(nodes[i].tokens, t)
		nodes[i].mutex = sync.Mutex{}

	}
	ControlTokenInstance.B = make(map[int][]int)
	logger.Debug(ControlTokenInstance.String())
	
	for i := 0; i < NB_NODES; i++ {
		nodes[i].messages = messages
	}

	// start
	for i := 0; i < NB_NODES; i++ {
		wg.Add(1)
		go nodes[i].BouabdallahLaforest(&wg)
	}

	// end
	wg.Wait()
	logger.Info("************** END ****************")
	for i := 0; i < NB_NODES; i++ {
		logger.Info("Node #", nodes[i].id," entered CS ", nodes[i].nbCS, " time")	
	}
}
