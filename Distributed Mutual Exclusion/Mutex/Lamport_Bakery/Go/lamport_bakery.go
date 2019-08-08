/*
  Copyright "Guillaume Fraysse <gfraysse dot spam plus code at gmail dot com>"

How-to run: 
  go build lamport_bakery.go
  ./lamport_bakery 2>&1 |tee /tmp/tmp.log

Parameters:
- Number of nodes is hardcoded in the main function
- Number of iterations is hardcoded in NaimiTrehel function
*/ 

/*
    Go implementation of the Leslie Lamport Distributed mutual exclusion algorithm, aka the Bakery algorithm (1974)

References :
* Leslie Lamport, Communications of the ACM 17, 8   (August 1974), 453-455.: http://lamport.azurewebsites.net/pubs/bakery.pdf
* http://research.microsoft.com/en-us/um/people/lamport/pubs/pubs.html#bakery
* https://en.wikipedia.org/wiki/Lamport%27s_bakery_algorithm
* https://en.wikipedia.org/wiki/Lamport%27s_distributed_mutual_exclusion_algorithm

Number of messages if 3 * (N - 1) where N is thenumber of processes
- (N − 1) total number of requests
- (N − 1) total number of replies
- (N − 1) total number of releases
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

/*
// Debug function
func displayNodes() {
	for i := 0; i < len(nodes); i++ {
		log.Print("Node #", nodes[i].id, ", last=", nodes[i].last, ", next=", nodes[i].next)
	}
}
*/

func sumVector(v [10]int) int {
	var r = 0
	for i := 0; i < len(v); i++ {
		r += v[i]
	}
	return r
}

type Request struct {
	id        int
	timestamp int
}

type Node struct {
	id         int
	timestamp  int
	queue      []Request
	channel    chan string
	messages   [4]chan string
}

func (n *Node) enterCS() {
	log.Print("node #", n.id, " enterCS ************************************")
	time.Sleep(500 * time.Millisecond)
}

func (n *Node) sendRequestToAllOtherNodes() {
	log.Print("node #", n.id," sendRequestToAllOtherNodes")	
	for i := 0; i < len(n.messages); i++ {
		log.Print("node #", n.id," sendRequestToAllOtherNodes i=", i)

		if n.id != i {
			var content = fmt.Sprintf("REQ%d", n.id)
			log.Print("node #", n.id, " , SENDING request ", content, " to node #", i)	
			n.messages[i] <- content
		}
	}
}

func (n *Node) sendReleaseToAllOtherNodes() {
	log.Print("node #", n.id," sendReleaseToAllOtherNodes")	
	for i := 0; i < len(n.messages); i++ {
		if n.id != i {
			var content = fmt.Sprintf("REL%d", n.id)
			log.Print("node #", n.id, " , SENDING release ", content, " to node #", i)	
			n.messages[i] <- content
		}
	}
}

func (n *Node) requestCS() {
	log.Print("node #", n.id," requestCS")	
	var r Request 
	r.id = n.id
	
	n.timestamp++
	r.timestamp = n.timestamp
	
	n.queue = append(n.queue, r)
	n.sendRequestToAllOtherNodes()
	log.Print("node #", n.id," requestCS - end")	
}

func (n *Node) releaseCS() {
	log.Print("node #", n.id," releaseCS")	
	if n.queue[0].id == n.id {
		// remove own request from queue
		n.queue = n.queue[1:]
		n.sendReleaseToAllOtherNodes()
	} else {
		log.Fatal("WTF1")
	}
}

func (n *Node) waitForReplies() {	
	log.Print("node #", n.id," waitForReplies")	
	var replies [10]int
	for i := 0; i < len(n.messages); i++ {
		replies[i] = 0
	}
	for ;sumVector(replies) != len(n.messages) - 1; {
		select {
		case msg := <-n.messages[n.id]:
			if (strings.Contains(msg, "REP")) {
				var requester, err = strconv.Atoi(msg[3:4])
				if err != nil {
					log.Fatal(err)
				}
				// var ts, err2 = strconv.Atoi(msg[4:])
				// if err2 != nil {
				// 	log.Fatal(err2)
				// }
				replies[requester] = 1
			} else if (strings.Contains(msg, "REQ")) {
				var requester, err = strconv.Atoi(msg[3:4])
				if err != nil {
					log.Fatal(err)
				}
				var content = fmt.Sprintf("REP%d%d", n.id, n.timestamp)
				log.Print("node #", n.id, " , SENDING reply ", content, " to node #", requester)	
				n.messages[requester] <- content
				var r Request 
				r.id = requester
				r.timestamp = n.timestamp
				n.queue = append(n.queue, r)
				n.timestamp++
			} else if (strings.Contains(msg, "REL")) {
				var requester, err = strconv.Atoi(msg[3:4])
				if err != nil {
					log.Fatal(err)
				}
				for i := 0; i < len(n.queue); i++ {
					if n.queue[i].id == requester {
						n.queue = append(n.queue[:i], n.queue[i+1:]...)
					}
				}

			} else {
				log.Fatal("WTF2")	
			}
		}
	}
	if n.queue[0].id == n.id {
		n.enterCS()
		n.releaseCS()
	}
}

func (n *Node) LamportBakery(wg *sync.WaitGroup) {
	log.Print("node #", n.id)

	// Initialization
	n.queue = make([]Request, 1, 100)

	for i := 1; i < 10000000; i ++ {
		time.Sleep(100 * time.Millisecond)
		go n.requestCS()
		go n.waitForReplies()
	}

	log.Print("node #", n.id," END")	
	wg.Done()
}

func main() {
	// Will only work with a numer of nodes N < 10 due to messages structure
	// To increase max of N, messages create/parse need to be modified
	var nodes [4]Node	
	var wg sync.WaitGroup
	var messages [len(nodes)]chan string
	
	log.Print("nb_process #", len(nodes))
	
	for i := 0; i < len(nodes); i++ {
		nodes[i].id = i
		nodes[i].channel = messages[i]
		messages[i] = make(chan string)
	}
	for i := 0; i < len(nodes); i++ {
		nodes[i].messages = messages
	}
	
	for i := 0; i < len(nodes); i++ {
		wg.Add(1)
		go nodes[i].LamportBakery(&wg)
	}
	wg.Wait()
}
