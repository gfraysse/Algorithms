// https://fr.wikipedia.org/wiki/Algorithme_de_Naimi-Trehel
//    M. Naimi, M. Tréhel, A. Arnold, "A Log(N) Distributed Mutual Exclusion Algorithm Based on the Path Reversal", Journal of Parallel and Distributed Computing, 34, 1-13 (1996).
//    M.Tréhel, M.Naimi: "Un algorithme distribué d'exclusion mutuelle", TSI Vol 6, no 2, p. 141–150, (1987).
//    M.Naimi, M. Tréhel : "How to detect a failure and regenerate the token in the Log(n) distributed algorithm for mutual exclusion" , 2nd International Workshop on Distributed Algorithms, Amsterdam, (Juill. 1987), paru dans Lecture Notes in Computer Science, no 312, p. 149-158, édité par J. Van Leeween.

// Complexity O(Log(n))
package main

import (
	"fmt"
	"sync"
	"strings"
	"strconv"
)

func enterCS(id int) {
	fmt.Println("inCS : goroutine #", id)	
}

func Algo(id int, messages [10]chan string, wg *sync.WaitGroup) {
	// var waiting_list []int 
	fmt.Println("goroutine #", id)
	// var str strings.Builder

	// Initialization
	var has_token = false
	// var requested = false
	var child = -1
	var father = 1

	if father == id {
		has_token = true
		father = -1
	} else {
		has_token = false
	}

	for i := 1; i < 100; i ++ {
		if father == -1 {
			// requested = true
			enterCS(id)
			// requested = false
			if child != -1 {
				var content = fmt.Sprintf("token%d", child)
				fmt.Println("SENDING", content)				
				messages[child] <- content
				has_token = false
				child = -1
			}
		} else {
			var content = fmt.Sprintf("REQ%d", id)
			fmt.Println("SENDING ", content, "to father", father)				
			messages[father] <- content
		}
		msg := <-messages[id]
		fmt.Println("msg received=", msg)	
		if (strings.Contains(msg, "REQ")) {
			var requester, err = strconv.Atoi(msg[3:])
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println("received REQ=", requester, "by goroutine#", id)	
		} else if (strings.Contains(msg, "token")) {

		} else {
			fmt.Println("WTF")	
		}
	}

	fmt.Println("has_token", has_token)	
	wg.Done()
}

func main() {
	var nb_process = 10
	var wg sync.WaitGroup
	var messages [10]chan string
	
	fmt.Println("nb_process #", nb_process)
	
	for i := 0; i < nb_process; i++ {
		messages[i] = make(chan string)
	}
	
	for i := 0; i < nb_process; i++ {
		wg.Add(1)
		go Algo(i, messages, &wg)
	}
	wg.Wait()
}
