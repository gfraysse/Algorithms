// https://fr.wikipedia.org/wiki/Algorithme_de_Naimi-Trehel
//    M. Naimi, M. Tréhel, A. Arnold, "A Log(N) Distributed Mutual Exclusion Algorithm Based on the Path Reversal", Journal of Parallel and Distributed Computing, 34, 1-13 (1996).
//    M.Tréhel, M.Naimi: "Un algorithme distribué d'exclusion mutuelle", TSI Vol 6, no 2, p. 141–150, (1987).
//    M.Naimi, M. Tréhel : "How to detect a failure and regenerate the token in the Log(n) distributed algorithm for mutual exclusion" , 2nd International Workshop on Distributed Algorithms, Amsterdam, (Juill. 1987), paru dans Lecture Notes in Computer Science, no 312, p. 149-158, édité par J. Van Leeween.
// https://www-master.ufr-info-p6.jussieu.fr/2018/spip.php?action=acceder_document&arg=23891&cle=5597164cc7d5a16ea0ce06e8b68c2c226bf8de89&file=pdf%2FNaimi_Trehel.pdf

// Complexity O(Log(n))
package main

import (
	"fmt"
	"log"
	"sync"
	"strings"
	"strconv"
)

type Node struct {
	id         int
	has_token  bool
	requesting bool
	next       int
	father     int
	channel    chan string
	messages   [10]chan string
}

func (n *Node) enterCS() {
	log.Print("goroutine #", n.id, " enterCS")
}

func (n *Node) requestCS() {
	// log.Print("goroutine #", n.id, " requestCS")
	log.Print("goroutine #", n.id, " requestCS, n.father=", n.father)
	n.requesting = true
	if n.father != -1 {
		var content = fmt.Sprintf("REQ%d", n.id)
		log.Print("goroutine #", n.id, " requestCS, SENDING ", content, " to father #", n.father)				
		n.messages[n.father] <- content
		n.father = -1		
	} else {
		log.Print("goroutine #", n.id, " father == -1")				
	}
}

func (n *Node) releaseCS() {
	log.Print("goroutine #", n.id, " releaseCS")
	n.requesting = false
	if n.next != -1 {
		var content = fmt.Sprintf("token%d", n.next)
		log.Print("goroutine #", n.id, " releaseCS, SENDING ", content, " to next #", n.next)				
		n.messages[n.next] <- content
		n.has_token = false
		n.next = -1
	}
}

func (n *Node) receiveRequestCS(j int) {
	log.Print("goroutine #", n.id, " receiveRequestCS")
	if n.father == -1 {
		if n.requesting {
			n.next = j
		} else {
			n.has_token = false
			var content = fmt.Sprintf("token%d", j)
			log.Print("goroutine #", n.id, " receiveRequestCS SENDING ", content, " to j #", j)				
			n.messages[j] <- content
		}		
	} else {
		var content = fmt.Sprintf("REQ%d", n.id)
		log.Print("goroutine #", n.id, " receiveRequestCS SENDING ", content, " to father #", n.father)				
		n.messages[n.father] <- content
	}
	n.father = j
	log.Print("goroutine #", n.id, " receiveRequestCS, n.father=", n.father)

}

func (n *Node) receiveToken() {
	log.Print("goroutine #", n.id, " receiveToken")
	log.Print("****** Got TOKEN #", n.id)
	n.has_token = true
}

func (n *Node) NaimiTrehel(wg *sync.WaitGroup) {
	log.Print("goroutine #", n.id)

	// Initialization
	n.has_token = false
	n.requesting = false
	n.next = -1
	n.father = 1

	if n.father == n.id {
		n.has_token = true
		n.father = -1
	} else {
		n.has_token = false
	}

	for i := 1; i < 100; i ++ {		
		if n.father == -1 {
			if n.has_token {
				n.enterCS()
				//if n.next != -1 {
				n.releaseCS()
				// } 				
			} else {
				log.Print("Is father but does not have the token, WTF ?")
			}
		} else {
			n.requestCS()
		}
		msg := <-n.messages[n.id]
		// log.Print("msg received=", msg)	
		if (strings.Contains(msg, "REQ")) {
			var requester, err = strconv.Atoi(msg[3:])
			if err != nil {
				log.Fatal(err)
			}
			// log.Print("received REQ=", requester, "by goroutine#", n.id)
			n.receiveRequestCS(requester)
			
		} else if (strings.Contains(msg, "token")) {
			n.receiveToken()
		} else {
			log.Fatal("WTF")	
		}
	}

	log.Print("goroutine #", n.id," has_token=", n.has_token)	
	wg.Done()
}

func main() {
	//var nb_process = 10
	var wg sync.WaitGroup
	var nodes [10]Node
	var messages [10]chan string
	
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
		go nodes[i].NaimiTrehel(&wg)
	}
	wg.Wait()
}
