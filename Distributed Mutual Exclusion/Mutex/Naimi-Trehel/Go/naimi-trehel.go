/*
  Copyright "Guillaume Fraysse <gfraysse dot spam plus code at gmail dot com>"

How-to run: 
  go run naimi-trehel.go 2>&1 |tee /tmp/tmp.log

Parameters:
- Number of nodes is set with NB_NODES global variable
- Number of CS entries is set with NB_ITERATIONS global variable
*/ 

/*
    Go implementation of Naimi-Trehel mutual exclusion algorithm

References :
* M. Naimi, M. Tréhel, A. Arnold, "A Log(N) Distributed Mutual Exclusion Algorithm Based on the Path Reversal", Journal of Parallel and Distributed Computing, 34, 1-13 (1996).
* M.Tréhel, M.Naimi: "Un algorithme distribué d'exclusion mutuelle", TSI Vol 6, no 2, p. 141–150, (1987).
* M.Naimi, M. Tréhel : "How to detect a failure and regenerate the token in the Log(n) distributed algorithm for mutual exclusion" , 2nd International Workshop on Distributed Algorithms, Amsterdam, (Juill. 1987), paru dans Lecture Notes in Computer Science, no 312, p. 149-158, édité par J. Van Leeween.
* https://fr.wikipedia.org/wiki/Algorithme_de_Naimi-Trehel

Complexity is O(Log(n))
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
var NB_ITERATIONS int = 50
var CURRENT_ITERATION int = 0
/*
// Debug function
func displayNodes() {
	for i := 0; i < len(nodes); i++ {
		log.Print("Node #", nodes[i].id, ", last=", nodes[i].last, ", next=", nodes[i].next)
	}
}
*/

type Node struct {
	id         int
	has_token  bool
	requesting bool
	nbCS       int
	next       int // the dynamic distributed list
	last       int // called father in the original paper. Called last here as in Sopena et al. as it stores the last requester
	messages   [4]chan string
}

func (n *Node) enterCS() {
	log.Print("Node #", n.id, " ######################### enterCS")
	CURRENT_ITERATION ++
	n.nbCS ++
	time.Sleep(500 * time.Millisecond)
}

func (n *Node) releaseCS() {
	log.Print("Node #", n.id," releaseCS #######################")	
	n.requesting = false
	if n.next != -1 {
		var content = fmt.Sprintf("token%d", n.next)
		// log.Print("node #", n.id, " releaseCS, SENDING ", content, " to next #", n.next)				
		n.messages[n.next] <- content
		n.has_token = false
		n.next = -1
	}
}

func (n *Node) requestCS() {
	for {
		time.Sleep(100 * time.Millisecond)

		// Initialization of request
		n.has_token = false
		n.requesting = false
		n.next = -1
		n.last = 0

		if n.last == n.id {
			n.has_token = true
			n.last = -1
		} else {
			n.has_token = false
		}

		n.requesting = true
		if n.last != -1 {
			var content = fmt.Sprintf("REQ%d", n.id)
			log.Print("node #", n.id, " requestCS, SENDING ", content, " to last #", n.last)				
			n.messages[n.last] <- content
			n.last = -1		
		}

		for {
			time.Sleep(100 * time.Millisecond)
			if (n.requesting == true && n.has_token == true) {
				n.enterCS() // not explicit in paper
				n.releaseCS() // not explicit in paper
				break
			}
		} 
	}
}

func (n *Node) receiveRequestCS(j int) {
	if n.last == -1 {
		if n.requesting {
			n.next = j
		} else {
			n.has_token = false
			var content = fmt.Sprintf("token%d", j)
			// log.Print("node #", n.id, " receiveRequestCS SENDING ", content, " to j #", j)				
			n.messages[j] <- content
		}		
	} else {
		// Forwarding request to last
		var content = fmt.Sprintf("REQ%d", j)
		// log.Print("node #", n.id, " receiveRequestCS fwd SENDING ", content, " to last #", n.last)				
		n.messages[n.last] <- content
	}
	n.last = j
	// log.Print("node #", n.id, " receiveRequestCS, *update* n.last #", n.last)
}

func (n *Node) receiveToken() {
	log.Print("** Node #", n.id, " Got TOKEN **")
	n.has_token = true
}

func (n *Node) waitForReplies() {	
	for {
		select {
		case msg := <-n.messages[n.id]:
			if (strings.Contains(msg, "REQ")) {
				var requester, err = strconv.Atoi(msg[3:])
				if err != nil {
					log.Fatal(err)
				}
				n.receiveRequestCS(requester)
				
			} else if (strings.Contains(msg, "token")) {
				n.receiveToken()
				// if (n.requesting == true && n.has_token == true) {
				// 	n.enterCS() // not explicit in paper
				// 	n.releaseCS() // not explicit in paper
			} else {
				log.Fatal("WTF")	
			}
		}
	}	
}

func (n *Node) NaimiTrehel(wg *sync.WaitGroup) {
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
	var nodes [4]Node	
	var wg sync.WaitGroup
	var messages [len(nodes)]chan string
	
	log.Print("nb_process #", len(nodes))
	
	for i := 0; i < len(nodes); i++ {
		nodes[i].id = i
		nodes[i].nbCS = 0
		messages[i] = make(chan string)
	}
	for i := 0; i < len(nodes); i++ {
		nodes[i].messages = messages
	}
	
	for i := 0; i < len(nodes); i++ {
		wg.Add(1)
		go nodes[i].NaimiTrehel(&wg)
	}
	wg.Wait()
	for i := 0; i < NB_NODES; i++ {
		log.Print("Node #", nodes[i].id," entered CS ", nodes[i].nbCS," time")	
	}
}
