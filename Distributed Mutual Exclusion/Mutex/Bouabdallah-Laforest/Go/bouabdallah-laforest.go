/*
  Copyright "Guillaume Fraysse <gfraysse dot spam plus code at gmail dot com>"

TODO: 
- finalize implementation

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

import (
	"fmt"
	"log"
	"math/rand"
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

/*
// Debug function
func displayNodes() {
	for i := 0; i < len(nodes); i++ {
		log.Print("Node #", nodes[i].id, ", last=", nodes[i].last, ", next=", nodes[i].next)
	}
}
*/
type Token struct {
	id     int
	locked bool
}

// type TokensForNode struct {
// 	node   Node
// 	tokens []Token
// }

type ControlToken struct {	
	A []int
	B map[int][]int
}

var ControlTokenInstance ControlToken

type Request struct {
	requesterNodeId int
	requestId      int
	messageType    int
	resourceId     []int
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
	messageWaitingForRequest  Request
	next                      int // the dynamic distributed list
	last                      int // called father in the original paper. Called last here as in Sopena et al. as it stores the last requester
	inBLCS                    bool
	requestInCS               Request
	// Implementation specific
	nbCS           int // the number of time the node entered its Critical Section
	queue          []Request
	messages       []chan []byte
}

////////////////////////////////////////////////////////////
// Utility functions
////////////////////////////////////////////////////////////
func UnmarshalRequest(text []byte, request *Request) error {
	request.requesterNodeId = int(text[0])
	request.requestId      = int(text[1])
	request.messageType    = int(text[2])


	if request.messageType == REQ_CT_TYPE {
	} else if request.messageType == REQ_TYPE || request.messageType == INQUIRE_TYPE {
		request.resourceId = make ([]int, REQUEST_SIZE)
		
		for i := 0; i < REQUEST_SIZE; i++ {
			request.resourceId[i] = int(text[3 + i])
		}
	}
	
	return nil
}

func MarshalRequest(request Request) ([]byte, error) {
	var ret = make ([]byte, 3 + REQUEST_SIZE)

	ret[0] = byte(request.requesterNodeId)
	ret[1] = byte(request.requestId)
	ret[2] = byte(request.messageType)

	if request.messageType == REQ_CT_TYPE {
	} else if request.messageType == REQ_TYPE || request.messageType == INQUIRE_TYPE {
		for i := 0; i < REQUEST_SIZE; i++ {
			ret[3 + i] = byte(request.resourceId[i])
		}
	}
		
	return ret, nil
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

	val = fmt.Sprintf("ControlToken A={%s}, B={%s} \n",
		A,
		B)
	return val
}

func (ct *ControlToken) getTokensPossessedByNode(n *Node) []int{
    // var tokens []Tokens
    // if len(ct.B) > 0 {
    //     for i := 0; i < len(ct.B); i++ {
    //         if (ct.B[i].id == n.id) {
	return ct.B[n.id];
    //         }
    //     }
    // }
    // return tokens;
}

func (ct *ControlToken) addFreeToken(token int) {
	ct.A = append(ct.A, token)
}

// func (ct *ControlToken) removeNeededTokensFromFreeTokens(requestedTokens []Token, tokensOwned []Token) {
func (ct *ControlToken) removeNeededTokensFromFreeTokens(node *Node) {
	// log.Print("removeNeededTokensFromFreeTokens")
	
	// var tokensOwned []Token = make([]Token, len(node.tokens))
	// copy(tokensOwned, node.tokens)

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
}

func (ct *ControlToken) isTokenPossessedByNode(token int) bool {
	// log.Print("isTokenPossessedByNode")
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
	// log.Print("getTokenOwnerFromPossessedByNode token=", token)
	for key, _ := range ct.B {
		for i := 0; i < len(ct.B[key]); i ++ {
            if token == ct.B[key][i] {
				// log.Print(key)
                return key;
            }
        }
    }
    return -1;
}

func (ct *ControlToken) removeTokenFromPossessedByNode(token int) int {
	// log.Print("removeTokenFromPossessedByNode")
	for key, _ := range ct.B {
		for i := 0; i < len(ct.B[key]); i ++ {
            if token == ct.B[key][i] {
                ct.B[key][i] = ct.B[key][len(ct.B[key]) - 1]
                ct.B[key] = ct.B[key][:len(ct.B[key]) - 1]
                return key
            }
        }
    }
    return -1
}

func (ct *ControlToken) updateForRequest(
	n *Node,
	request Request,
	// requestedTokens []Token,
	// tokensOwned []Token,
	missingTokens []int,
	requestedResourcesForNode map[int][]int) {

	log.Print("Node #", n.id, ", updateForRequest")
	// log.Print(ct.String())
    // First move the tokens already owned by nodeName in B to A
	var tokensPossessedByNode []int = ct.getTokensPossessedByNode(n)
	for i := 0; i < len(tokensPossessedByNode); i ++ {
		ct.addFreeToken(tokensPossessedByNode[i])
	}
	// log.Print(ct.String())
	
    // Remove needed tokens from A
    // ct.removeNeededTokensFromFreeTokens(requestedTokens, tokensOwned);
	n.tokensNeeded = request.resourceId
	// log.Print("n.tokensNeeded", n.tokensNeeded)
    ct.removeNeededTokensFromFreeTokens(n);
	// log.Print("n.tokensNeeded 2", n.tokensNeeded)

	// log.Print(ct.String())
	
    // First remove the tokens from B if already present
    // Identify owner in B of requestedTokens to build
    // requestedResourcesForNode dictionnary
    // for i := 0; i < len(requestedTokens); i ++ {
    for i := 0; i < len(n.tokensNeeded); i ++ {
        var owner int = ct.getTokenOwnerFromPossessedByNode(n.tokensNeeded[i]);
        if owner != n.id {
            requestedResourcesForNode[owner] = append(requestedResourcesForNode[owner], n.tokensNeeded[i]);
        }
    }

	// log.Print("missingTokens", missingTokens)
    for i := 0; i < len(n.tokensNeeded); i++ {
        if ct.isTokenPossessedByNode(n.tokensNeeded[i]) {
			// log.Print("isTokenPossessedByNode true")
            var owner int = ct.getTokenOwnerFromPossessedByNode(n.tokensNeeded[i]);
            if owner != n.id{
                owner = ct.removeTokenFromPossessedByNode(n.tokensNeeded[i]);
                missingTokens = append(missingTokens, n.tokensNeeded[i]);
            }
        }
    }
	// log.Print("BEFORE ", ct.String())
	// log.Print("BEFORE missingTokens ", missingTokens)
	// log.Print("BEFORE n.tokensNeeded ", n.tokensNeeded)
    ct.B[n.id] = n.tokensNeeded
	// log.Print("AFTER ", ct.String())

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
	// log.Print(ct.String())	
}
////////////////////////////////////////////////////////////
// Node class
////////////////////////////////////////////////////////////
func (n *Node) String() string {
	var val string
	val = fmt.Sprintf("Node #%d next=%d, last=%d \n",
		n.id,
		n.next,
		n.last)
	return val
}

func (n *Node) enterCS() {
	log.Print("Node #", n.id, " ######################### enterCS")
	log.Print(ControlTokenInstance.String())
	CURRENT_ITERATION ++
	n.nbCS ++
	// log.Print(n)
	time.Sleep(500 * time.Millisecond)
}

func (n *Node) releaseCS() {
	log.Print("Node #", n.id," releaseCS #########################")	
	// log.Print(n)
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
	request.requesterNodeId = n.id
	request.requestId = 0
	request.messageType = REP_CT_TYPE
	
	content, err := MarshalRequest(request)
	if err != nil {
		log.Fatal(err)
	}			
	log.Print("Node #", n.id, ",  SEND CT #", request.requestId, ":", content, " to Node #", dst)	
	n.messages[dst] <- content
}

func (n *Node) handleCTRequest(request Request) {
	if n.last == -1 {
		if n.requesting {
			n.next = request.requesterNodeId
		} else {
			n.has_CT = false
			n.sendCT(request.requesterNodeId)
		}		
	} else {
		// Code duplication to remove
		var fwdRequest Request
		fwdRequest.messageType = REQ_CT_TYPE
		fwdRequest.requesterNodeId = request.requesterNodeId
		
		content, err := MarshalRequest(fwdRequest)
		if err != nil {
			log.Fatal(err)
		}			
		log.Print("Node #", n.id, ",  FWD REQUEST CT #", request.requestId, ":", content, " to Node #", n.last)	
		n.messages[n.last] <- content
	}
	n.last = request.requesterNodeId
	log.Print("node #", n.id, " receiveRequestCT, *update* n.last #", n.last)
}

func (n *Node) enterBLCS(request Request) bool {
	n.enterCS()
	n.releaseCS()
	return false
}

func (n *Node) hasAllTokensForRequest(request Request) bool {
    var resourcesRequested []int = request.resourceId

    for i := 0; i < len(resourcesRequested); i++ {
        var hasToken bool = false
        for j := 0; j < len(n.tokens); j++ {
            if n.tokens[j].id == resourcesRequested[i] {
                hasToken = true
                break
            }
        }
        if (hasToken == false) {
			log.Print("Node #", n.id, " is missing token ", resourcesRequested[i])
            return false
        }
    }
	return true
}

func (n *Node) enterCSIfCan(request Request) bool {
	log.Print("Node #", n.id, " enterCSIfCan")
    if n.hasAllTokensForRequest(request) {
		log.Print("Node #", n.id, " ENTERS BLCS")
        n.inBLCS = true
        n.requestInCS = request

        n.enterBLCS(request)

        // Send the Control Token
		n.requesting = false
		var r Request
		r.requestId = -1
		n.messageWaitingForRequest = r
		
        // Finished using the Control Token, keep it going if there is a Next
        if n.next != -1 {
            n.sendCT(n.next)
            n.last = n.next
            n.next = -1
        }
        return true
    }
	log.Print("CANNOT enterCSIfCan")
    return false
}

func (n *Node) requestTokens(request Request, dst int, tokens[]int) {
	log.Print("Node #", n.id, ", requestTokens")
	n.currentlyRequestingTokens = tokens
	var fwdRequest Request
	fwdRequest.messageType = INQUIRE_TYPE
	fwdRequest.requesterNodeId = request.requesterNodeId
		
	fwdRequest.resourceId = make([]int, REQUEST_SIZE)
	for i := 0; i < len(tokens); i++ {
		fwdRequest.resourceId[i] = tokens[i]
	}
	if len(tokens) != REQUEST_SIZE {
		for i := len(tokens); i < REQUEST_SIZE; i++ {
			fwdRequest.resourceId[i] = -1
		}
	}
	content, err := MarshalRequest(fwdRequest)
	if err != nil {
		log.Fatal(err)
	}			
	log.Print("Node #", n.id, ", send INQUIRE #", request.requestId, ":", content, " to Node #", n.last)	
	n.messages[dst] <- content
	// fmt.Println("Key:", key, "Value:", value)
}

func (n *Node) addTokenToSet(token Token, status bool) {
	token.locked = status
	n.tokens = append(n.tokens, token)
}

func (n *Node) updateCTForRequest(request Request) {
	var missingTokens             []int
	var requestedResourcesForNode map[int][]int
	requestedResourcesForNode = make(map[int][]int)

	log.Print("Node #", n.id, ", updateCTForRequest")
	ControlTokenInstance.updateForRequest(n, request, missingTokens, requestedResourcesForNode)

	//GF to fix
	if len(missingTokens) == 0 {
		log.Print("Node #", n.id, ", updateCTForRequest 1 ")
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
			log.Print("Node #", n.id, ", updateCTForRequest 2")
			n.enterCSIfCan(request);
		} else {
			log.Print("Node #", n.id, ", updateCTForRequest 3")
			for i := 0; i < len(requestedResourcesForNode); i ++ {
				for key, _ := range requestedResourcesForNode {
					var tokens []int = requestedResourcesForNode[key]
					n.requestTokens(request, key, tokens);
					break
				}
			}
		}
	}
}

func (n *Node) handleRequest(request Request) {
	log.Print("Node #", n.id," handleRequest")	
	var hasAllTokens bool = true
	for i := 0; i < REQUEST_SIZE; i++ {
		if ! n.ownsToken(request.resourceId[i]) {
			hasAllTokens = false
		}
	}
	if hasAllTokens {
		log.Print("Node #", n.id," handleRequest hasAllTokens")	
		for i := 0; i < REQUEST_SIZE; i++ {
			n.lockResource(request.resourceId[i])
		}
		n.enterCSIfCan(request)
	} else {
		log.Print("Node #", n.id," handleRequest NOT hasAllTokens")
		n.currentRequest = request
		if n.has_CT == true {
			log.Print("Node #", n.id," handleRequest NOT hasAllTokens 1")	
			n.updateCTForRequest(request)
		} else {
			log.Print("Node #", n.id," handleRequest NOT hasAllTokens 2")
			if n.requesting == false {
				n.requestCT()
			}
		}
	}
}

// func (n *Node) updateA() {
// 	for i := 0; i < NB_NODES; i++ {
// 		var nodeIsInA bool = false
// 		for a := 0; a < len(ControlTokenInstance.A); a++ {
// 			if ControlTokenInstance.A[a] == i {
// 				nodeIsInA = true
// 				break
// 			}
// 		}
// 		if nodeIsInA {
// 		}
		
// 	}
// }

// func (n *Node) updateB() {
	
// }

func (n *Node) isTokenInSet(token int) bool{
    for i:= 0; i < len(n.tokens); i ++ {
        if n.tokens[i].id == token {
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
	log.Print("** Node #", n.id, "  receiveInquire ****************** TODO")
}

func (n *Node) receiveACK1(request Request) {
	log.Print("** Node #", n.id, "  receiveACK1 ****************** TODO")
}

func (n *Node) receiveACK2(request Request) {
	log.Print("** Node #", n.id, "  receiveACK2 ****************** TODO")
}

func (n *Node) receiveCT() {
	log.Print("** Node #", n.id, " Got TOKEN **")
	n.has_CT = true
    n.tokens = make([]Token, len(ControlTokenInstance.B[n.id]))
	for i := 0; i < len(ControlTokenInstance.B[n.id]); i++ {
		n.tokens[i].id = ControlTokenInstance.B[n.id][i]
	}

	
	for i := 0; i < len(n.currentRequest.resourceId); i++ {
		var token int = n.currentRequest.resourceId[i]
        if !n.isTokenInSet(token) && !n.isTokenNeeded(token) {
            n.tokensNeeded = append (n.tokensNeeded, token)
        } else {
			n.lockResource(token)
        }
    }

	n.updateCTForRequest(n.currentRequest)
	// n.updateA()
	// n.updateB()
}

func (n *Node) waitForReplies() {	
	// log.Print("Node #", n.id," waitForReplies")	
	for {
		select {
		case msg := <-n.messages[n.id]:
			var request Request
			err := UnmarshalRequest(msg, &request)
			if err != nil {
				log.Fatal(err)
			}			
			if (request.messageType == REQ_TYPE) {
				var requester = request.requesterNodeId
				var res = request.resourceId
				log.Print("Node #", n.id, "<-REQ#", request.requestId, ", Requester #", requester, ", nb of res:", len(res))
				n.handleRequest(request)
			} else if (request.messageType == REP_TYPE) {
				var requester = request.requesterNodeId
				log.Print("Node #", n.id, ", received REPLY from Node #", requester, ",", msg)
			} else if (request.messageType == REQ_CT_TYPE) {
				var requester = request.requesterNodeId
				log.Print("Node #", n.id, ", received REQUEST Control Token from Node #", requester, ",", msg)
				n.handleCTRequest(request)
			} else if (request.messageType == REP_CT_TYPE) {
				var requester = request.requesterNodeId
				log.Print("Node #", n.id, ", received REPLY Control Token from Node #", requester, ",", msg)
				n.receiveCT()
			} else if (request.messageType == INQUIRE_TYPE) {
				var requester = request.requesterNodeId
				log.Print("Node #", n.id, ", received INQUIRE from Node #", requester, ",", msg)
				n.receiveInquire(request)
			} else if (request.messageType == ACK1_TYPE) {
				var requester = request.requesterNodeId
				log.Print("Node #", n.id, ", received ACK1 from Node #", requester, ",", msg)
				n.receiveACK1(request)
			} else if (request.messageType == ACK2_TYPE) {
				var requester = request.requesterNodeId
				log.Print("Node #", n.id, ", received ACK2 from Node #", requester, ",", msg)
				n.receiveACK2(request)
			} else {
				log.Fatal("Fatal Error")
			}
		}
	}
	// log.Print(n)
	// log.Print("Node #", n.id, " end waitForReplies")
}

func (n *Node) requestCT() {
	log.Print(n)
	n.requesting = true
	
	var request Request
	request.requesterNodeId = n.id
	request.requestId = 0
	request.messageType = REQ_CT_TYPE
	
	content, err := MarshalRequest(request)
	if err != nil {
		log.Fatal(err)
	}			
	log.Print("Node #", n.id, ",  REQUEST CT #", request.requestId, ":", content, " to Node #", n.last)	
	n.messages[n.last] <- content
	n.last = -1		
}

func (n *Node) sendRequest(request Request) {
	content, err := MarshalRequest(request)
	if err != nil {
		log.Fatal(err)
	}			
	for i := 0; i < len(request.resourceId); i++ {
		for j := 0; j < len(n.messages); j++ {
			if i == j {
				log.Print("Node #", n.id, ",  REQUEST #", request.requestId, ":", content, " for resources #", request.resourceId[0], ", ", request.resourceId[1], " to Node #", j)	
				n.messages[j] <- content
			}
		}
	}
}

func (n *Node) requestCS() {
	// log.Print("Node #", n.id, " requestCS")
	
	for {
		time.Sleep(100 * time.Millisecond)

		// for i := 0; i < NB_NODES; i ++ {
		// 	if (i != n.id) {
				var request Request
				request.messageType = REQ_TYPE
				var resources = make([]int, NB_NODES)
				for j := 0; j < NB_NODES; j++ {
					resources[j] = j
				}
				request.requesterNodeId = n.id
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
				n.sendRequest(request)
		// 	}
		// }
		for {
			time.Sleep(100 * time.Millisecond)
		} 
	}	
	// log.Print("Node #", n.id," END")	
}

func (n *Node) BouabdallahLaforest(wg *sync.WaitGroup) {
	log.Print("Node #", n.id)

	go n.requestCS()
	go n.waitForReplies()
	for {
		time.Sleep(100 * time.Millisecond)
		if CURRENT_ITERATION > NB_ITERATIONS {
			break
		}
	}

	log.Print("Node #", n.id," END after ", NB_ITERATIONS," CS entries")	
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
		nodes[i].nbCS = 0
		
		// nodes[i].has_CT = false
		nodes[i].requesting = false
		nodes[i].next = -1
		nodes[i].last = 0
		
		// Initially the first node holds the Control Token
		if nodes[i].last == nodes[i].id {
			nodes[i].has_CT = true
			nodes[i].last = -1
		} else {
			nodes[i].has_CT = false
		}

		// nodes[i].currentRequest = nil

		messages[i] = make(chan []byte)

		// Initialize the Control Token such that A contains all free tokens
		var t Token
		t.id = i
		t.locked = false
		ControlTokenInstance.A = append(ControlTokenInstance.A, i)
		// Initially each node owns its token
		nodes[i].tokens = append(nodes[i].tokens, t)
	}
	ControlTokenInstance.B = make(map[int][]int)
	// log.Print(ControlTokenInstance.String())
	
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
	for i := 0; i < NB_NODES; i++ {
		log.Print("Node #", nodes[i].id," entered CS ", nodes[i].nbCS, " time")	
	}
}
