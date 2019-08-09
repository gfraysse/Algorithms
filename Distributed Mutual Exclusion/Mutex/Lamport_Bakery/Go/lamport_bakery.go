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
	"sort"
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

func sumVector(v [4]int) int {
	var r = 0
	for i := 0; i < len(v); i++ {
		r += v[i]
	}
	return r
}

// ByTimestamp implements sort.Interface for Requests
type ByTimestamp []Request

func (a ByTimestamp) Len() int           { return len(a) }
func (a ByTimestamp) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByTimestamp) Less(i, j int) bool { return a[i].timestamp < a[j].timestamp }


type Request struct {
	id        int
	timestamp int
}

type Node struct {
	id         int
	timestamp  int
	queue      []Request
	replies    [4]int
	channel    chan string
	messages   [4]chan string
}

func (n *Node) String() string {
	var val string
	val = fmt.Sprintf("Node #%d, timestamp=%d\n", n.id, n.timestamp)
	for i := 0; i < len(n.queue); i ++ {
		val = val + fmt.Sprintf("  req #%d, timestamp=%d\n", n.queue[i].id, n.queue[i].timestamp)
		
	}
	return val
}

func (n *Node) enterCS() {
	log.Print("node #", n.id, " enterCS ************************************")
	time.Sleep(500 * time.Millisecond)
}

func (n *Node) sendRequestToAllOtherNodes(r Request) {
	// log.Print("node #", n.id," sendRequestToAllOtherNodes")	
	for i := 0; i < len(n.messages); i++ {
		if n.id != i {
			var content = fmt.Sprintf("REQ%d%d", n.id, r.timestamp)
			log.Print("node #", n.id, " , SENDING request ", content, " to node #", i)	
			n.messages[i] <- content
		}
	}
}

func (n *Node) sendReleaseToAllOtherNodes() {
	log.Print("node #", n.id," sendReleaseToAllOtherNodes")	
	for i := 0; i < len(n.messages); i++ {
		if n.id != i {
			var content = fmt.Sprintf("REL%d%d", n.id, n.timestamp)
			log.Print("node #", n.id, " , SENDING release ", content, " to node #", i)	
			n.messages[i] <- content
		}
	}
}

func (n *Node) requestCS() {
	// log.Print("node #", n.id," requestCS")	
	for i := 0; i < len(n.queue); i++ {
		if (n.queue[i].id == n.id) {
			// log.Print("node #", n.id," already waiting for CS")	
			return
		}
	}
	var r Request 
	r.id = n.id
	
	n.timestamp++
	r.timestamp = n.timestamp
	
	n.queue = append(n.queue, r)
	sort.Sort(ByTimestamp(n.queue))
	log.Print(n)
	
	n.sendRequestToAllOtherNodes(r)
	// log.Print("node #", n.id," requestCS - end")
}

func (n *Node) releaseCS() {
	log.Print("node #", n.id," releaseCS #########################")	
	if n.queue[0].id == n.id {
		// remove own request from queue
		n.queue = n.queue[1:]
		n.sendReleaseToAllOtherNodes()
	} else {
		log.Fatal("WTF1")
	}
}

func Max(x, y int) int {
        if x < y {
                return y
        }
        return x
}

func (n *Node) waitForReplies() {	
	log.Print("node #", n.id," waitForReplies")	
	//for ;sumVector(n.replies) != len(n.messages) - 1; {
		select {
		case msg := <-n.messages[n.id]:
			if (strings.Contains(msg, "REP")) {
				var requester, err = strconv.Atoi(msg[3:4])
				if err != nil {
					log.Fatal(err)
				}
				var ts, err2 = strconv.Atoi(msg[4:])
				if err2 != nil {
					log.Fatal(err2)
				}
				n.timestamp = Max(ts, n.timestamp) + 1
				n.replies[requester] = 1
				log.Print("node #", n.id, " , RECEIVED reply from node #", requester, n.replies)	

			} else if (strings.Contains(msg, "REQ")) {
				var requester, err = strconv.Atoi(msg[3:4])
				if err != nil {
					log.Fatal(err)
				}
				var ts, err2 = strconv.Atoi(msg[4:])
				if err2 != nil {
					log.Fatal(err2)
				}
				n.timestamp = Max(ts, n.timestamp) + 1

				var content = fmt.Sprintf("REP%d%d", n.id, n.timestamp)
				log.Print("node #", n.id, " , SENDING reply ", content, " to node #", requester)	
				var r Request 
				r.id = requester
				//n.timestamp++
				r.timestamp = ts //n.timestamp
				n.queue = append(n.queue, r)
				sort.Sort(ByTimestamp(n.queue))
				log.Print(n)
				n.messages[requester] <- content
			} else if (strings.Contains(msg, "REL")) {
				var requester, err = strconv.Atoi(msg[3:4])
				if err != nil {
					log.Fatal(err)
				}
				var ts, err2 = strconv.Atoi(msg[4:])
				if err2 != nil {
					log.Fatal(err2)
				}
				n.timestamp = Max(ts, n.timestamp) + 1
				for i := 0; i < len(n.queue); i++ {
					if n.queue[i].id == requester {
						n.queue = append(n.queue[:i], n.queue[i+1:]...)
					}
				}
				log.Print("Node #", n.id, " received release from ", requester)
				// log.Print(n)

			} else {
				log.Fatal("WTF2")	
			}
		}
//}
	log.Print("Node #", n.id, " got all replies")
	log.Print(n)
	if sumVector(n.replies) == len(n.messages) - 1 && n.queue[0].id == n.id {
		n.enterCS()
		n.releaseCS()
	}
}

func (n *Node) LamportBakery(wg *sync.WaitGroup) {
	log.Print("node #", n.id)

	// Initialization
	n.queue = make([]Request, 0, 100)

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
		nodes[i].timestamp = i * 10
		for r := 0; r < len(nodes); r++ {
			nodes[i].replies[r] = 0
		}
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
