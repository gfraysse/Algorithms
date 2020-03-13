/*
  Copyright "Guillaume Fraysse <gfraysse dot spam plus code at gmail dot com>"

TODO: 
- everything

How-to run: 
  go build ricart-agrawala.go 
  ./ricart-agrawala 2>&1 |tee /tmp/tmp.log

Terminology
* A site is any computing device which runs the Ricart-Agrawala Algorithm
* The requesting site is the site which is requesting to enter the critical section.
* The receiving site is every other site which is receiving a request from the requesting site.

Parameters:
- Number of nodes is set with NB_NODES global variable
- Number of CS entries is set with NB_ITERATIONS global variable
*/ 

/*
    Go implementation of Ricart-Agrawala mutual exclusion algorithm
    Algorithm by Glenn Ricart and Ashok Agrawala 1981

References : 
  - https://doi.org/10.1145%2F358527.358537
  - https://en.wikipedia.org/wiki/Ricart%E2%80%93Agrawala_algorithm
  - https://www.geeksforgeeks.org/ricart-agrawala-algorithm-in-mutual-exclusion-in-distributed-system/
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
var NB_NODES int = 4
var NB_ITERATIONS int = 10
var CURRENT_ITERATION int = 0

/*
// Debug function
func displayNodes() {
	for i := 0; i < len(nodes); i++ {
		log.Print("Node #", nodes[i].id, ", last=", nodes[i].last, ", next=", nodes[i].next)
	}
}
*/
type Request struct {
	nodeId         int
}

type Node struct {
	id                    int
	seqNumber             int // The sequence number chosen by a request originating at this node
	highestSeqNumber      int // The highest sequence number seen in any REQUEST message sent or received
	outstandingReplyCount int // The number of REPLY messages still expected
	nbCS                  int // the number of time the node entered its Critical Section
	isRequestingCS        bool // true when this node is requesting access to its critical section
	replyDeferred         []bool // Reply_Deferred [j] is TRUE when this node is deferring a REPLY to j's REQUEST message
	queue                 []Request
	channel               chan string
	messages              []chan string
}

func (n *Node) String() string {
	var val string
	val = fmt.Sprintf("Node #%d, highestSeqNumber=%d, outstandingReplyCount=%d \n",
		n.id,
		n.highestSeqNumber,
		n.outstandingReplyCount)
	return val
}

func (n *Node) enterCS() {
	log.Print("Node #", n.id, " ######################### enterCS")
	CURRENT_ITERATION ++
	n.nbCS ++
	// log.Print(n)
	time.Sleep(500 * time.Millisecond)
}

func (n *Node) releaseCS() {
	log.Print("Node #", n.id," releaseCS #########################")	
	n.isRequestingCS  = false
	for j := 0; j < NB_NODES; j++ {
		if (n.replyDeferred[j]) {
			n.replyDeferred[j] = false
			n.sendReply(j)
		}
	}
	// log.Print(n)
}

func (n *Node) sendRequest(seqNumber int, nodeId int, destNodeId int) {
	for i := 0; i < len(n.messages); i++ {
		if i == destNodeId {
			var content = fmt.Sprintf("REQ%d%d", n.id, seqNumber)
			log.Print("Node #", n.id, ", SENDING request ", content, " with seqNumber #", seqNumber, " to Node #", destNodeId)	
			n.messages[i] <- content
		}
	}
}

func (n *Node) sendReply(destNodeId int) {
	var content = fmt.Sprintf("REP%d", n.id)
	log.Print("Node #", n.id, ", SENDING reply ", content, " to Node #", destNodeId)	
	n.messages[destNodeId] <- content
}

func (n *Node) waitForReplies() {	
	// log.Print("Node #", n.id," waitForReplies")	
	for {
		select {
		case msg := <-n.messages[n.id]:
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

				if k > n.highestSeqNumber {
					n.highestSeqNumber = k
				}
				var defer_it bool = n.isRequestingCS && ((k > n.seqNumber) || (k == n.seqNumber && requester > n.id))
				if defer_it {
					n.replyDeferred[requester] = true
				} else {
					n.sendReply(requester)
				}
			}  else if (strings.Contains(msg, "REP")) {
				var sender, err = strconv.Atoi(msg[3:4])
				if err != nil {
					log.Fatal(err)
				}
				log.Print("Node #", n.id, ", RECEIVED reply from Node #", sender, ",", msg)
				n.outstandingReplyCount --
			} else {
				log.Fatal("WTF")
			}
		}
	}
	// log.Print(n)
	// log.Print("Node #", n.id, " end waitForReplies")
}

func (n *Node) requestCS() {
	// log.Print("Node #", n.id, " requestCS")

	for {
		time.Sleep(100 * time.Millisecond)
		// Mutex on shared variable
		n.isRequestingCS = true
		n.seqNumber = n.highestSeqNumber + 1
		// end mutex on shared variable
		n.outstandingReplyCount = NB_NODES - 1

		for j := 0; j < NB_NODES; j ++ {
			if (j != n.id) {
				n.sendRequest(n.seqNumber, n.id, j)
			}
		}
		for {
			time.Sleep(100 * time.Millisecond)
			if (n.outstandingReplyCount == 0) {
				n.enterCS()
				n.releaseCS()
				break
			}
		} 
	}	
	// log.Print("Node #", n.id," END")	
}

func (n *Node) RicartAgrawala(wg *sync.WaitGroup) {
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
	var messages = make([]chan string, NB_NODES)
	
	log.Print("nb_process #", NB_NODES)

	// Initialization
	for i := 0; i < NB_NODES; i++ {
		nodes[i].id = i
		nodes[i].nbCS = 0 
		nodes[i].seqNumber = 0
		nodes[i].highestSeqNumber = 0
		nodes[i].outstandingReplyCount = 0
		nodes[i].isRequestingCS = false
		nodes[i].replyDeferred = make([]bool, NB_NODES)
		for j := 0; j < NB_NODES; j++ {
			nodes[i].replyDeferred[j] = false
		}

		nodes[i].channel = messages[i]
		messages[i] = make(chan string)
	}
	for i := 0; i < NB_NODES; i++ {
		nodes[i].messages = messages
	}

	// start
	for i := 0; i < NB_NODES; i++ {
		wg.Add(1)
		go nodes[i].RicartAgrawala(&wg)
	}
	wg.Wait()
	for i := 0; i < NB_NODES; i++ {
		log.Print("Node #", nodes[i].id," entered CS ", nodes[i].nbCS," time")	
	}
}
/*
Pseudo-code in Algol-like language from original paper

SHARED DATABASE
CONSTANT
       me, ! This node's unique number
       N; ! The number of nodes in the network
INTEGER Our_SequenceNumber,
           ! The sequence number chosen by a request
           ! originating at this node
        HighestSequenceNumber initial (0),
           ! The highest sequence number seen in any
           ! REQUEST message sent or received
        Outstanding_Reply_Count;
           ! The number of REPLY messages still
           ! expected
BOOLEAN Requesting Critical_Section initial (FALSE),
           ! TRUE when this node is requesting access
           ! to its critical section
        Reply_Deferred [I:N] initial (FALSE);
           ! Reply_Deferred [j] is TRUE when this node
           ! is deferring a REPLY to j's REQUEST message
BINARY SEMAPHORE
        Shared vars initial (1);
           ! Interlock access to the above shared
           ! variables when necessary
PROCESS WHICH INVOKES MUTUAL EXCLUSION FOR
THIS NODE
Comment Request Entry to our Critical Section;
  P (Shared_vats)
    Comment Choose a sequence number;
    RequestingCritical_Section := TRUE;
    Our_Sequence_Number := Highest_Sequence_Number + l;
  V (Shared_vars);
  Outstanding_ReplyCount := N - 1;
  FOR j := I STEP 1 UNTIL N DO IF j != me THEN
      Send_Message(REQUEST(Our_Sequence_Number, me), j);
    Comment sent a REQUEST message containing our sequence number
    and our node number to all other nodes;
    Comment Now wait for a REPLY from each of the other nodes;
  WAITFOR (Outstanding_Reply_Count = 0);
    Comment Critical Section Processing can be performed at this point;
    Comment Release the Critical Section;
  RequestingCritical_Section := FALSE;
  FOR j := 1 STEP 1 UNTIL N DO
    IF Reply_Deferred[j] THEN
      BEGIN 
        Reply_Deferred[j] := FALSE;
        Send_Message (REPLY, j);
          Comment send a REPLY to node j;
      END;

PROCESS WHICH RECEIVES REQUEST (k, j) MESSAGES
Comment k is the sequence number begin requested,
        j is the node number making the request;
BOOLEAN Defer it ;
! TRUE when we cannot reply immediately
Highest_Sequence_Number := Maximum (Highest_Sequence_Number, k);
P (Shared_vars);
  Defer it :=
    Requesting_Critical_Section
    AND ((k > Our_sequence_Number)
          OR (k = Our_Sequence_Number AND j > me));
V (Shared_vars);
  Comment Defer_it will be TRUE if we have priority over
     node j's request;
IF Defer_it THEN Reply_Deferred[j] := TRUE ELSE
  Send_Message (REPLY, j);

PROCESS WHICH RECEIVES REPLY MESSAGES
Outstanding_Reply_Count := Outstanding_Reply_Count - 1;
*/
