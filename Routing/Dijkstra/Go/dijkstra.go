/*
Copyright "Guillaume Fraysse <gfraysse dot spam plus code at gmail dot com>"

How-to run: 
  go run dijkstra.go

Go implementation of the Dijkstra Shortest Path algorithm (1956)

References :
  * https://doi.org/10.1007%2FBF01386390
  * http://www-m3.ma.tum.de/twiki/pub/MN0506/WebHome/dijkstra.pdf
  * https://en.wikipedia.org/wiki/Dijkstra%27s_algorithm

Complexity is O(|E| + |V|^2)
*/
package main

import (
	"log"
)

func neighbours(vertex int, adjacency_matrix [][]int) []int {
	var Q []int
	for i := 0; i < len(adjacency_matrix[vertex]); i++ {
		var w = adjacency_matrix[vertex][i]
		if  w != 0 {
			Q = append (Q, i)
		}
	}
	return Q
}

func dijkstraAlgo(adjacency_matrix [][]int, entry int) ([]int, [][]int){
	var INFINITY int = 100000

	var current_vertex = entry
	var num_nodes = len(adjacency_matrix)

        var Q []int = make([]int, num_nodes)
	// distance to entry
        var dist []int = make([]int, num_nodes)
	// list of nodes on the path
        var prev [][]int = make([][]int, num_nodes)
	for i := 0; i < num_nodes; i ++ {
		Q[i] = i
		dist[i] = INFINITY		
	}

	dist[current_vertex] = 0        
            
	for ; len(Q) != 0; {        
		var min_dist = INFINITY
		var v int
		for i := 0; i < len(Q); i ++ {
			var q = Q[i]
			if (dist[q] < min_dist) {
				v = q
			}
		}
		var N = neighbours(v, adjacency_matrix)
		for i := 0; i < len(Q); i ++ {
			if Q[i] == v {
				Q[i] = Q[len(Q) - 1]
				Q = Q[:len(Q) - 1]
				break
			}
		}
		for i := 0; i < len(N); i ++ {
			var n = N[i]
			var alt = dist[v] + adjacency_matrix[v][n]
			if alt < dist[n] || dist[n] == INFINITY {
				dist[n] = alt
				prev[n] = make([]int, len(prev[v]))
				copy (prev[n], prev[v])
				prev[n] = append(prev[n], v)
			}
		}
	}

    return dist, prev
}

func main() {
	var adjacency_matrix [][]int
	adjacency_matrix = [][]int {
		//A, B, C, D, E, F, G, H, I
		{0, 1, 0, 0, 1, 0, 0, 0, 0}, // A
		{1, 0, 9, 0, 1, 0, 0, 0, 0}, // B
		{0, 9, 0, 1, 0, 1, 0, 0, 0}, // C
		{0, 0, 1, 0, 0, 1, 0, 0, 0}, // D
		{1, 1, 0, 0, 0, 0, 7, 0, 0}, // E
		{0, 0, 1, 1, 0, 0, 0, 0, 0}, // F
		{0, 0, 0, 0, 7, 0, 0, 1, 1}, // G
		{0, 0, 0, 0, 0, 0, 1, 0, 1}, // H
		{0, 0, 0, 0, 0, 0, 1, 1, 0}}  // I
		//A, B, C, D, E, F, G, H, I

	var dist, prev = dijkstraAlgo(adjacency_matrix, 0)
	log.Print(dist)
	log.Print(prev)
}

