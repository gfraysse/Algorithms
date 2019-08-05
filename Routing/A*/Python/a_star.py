#!/usr/bin/env python3
# -*- coding: utf-8 -*-
#
#   Copyright "Guillaume Fraysse <gfraysse dot spam plus code at gmail dot com>"
#
# How-to run: 
#   python3 a_star.py

# Python implementation of the A* pathfinding algorithm (Hart, Nilsson and Raphael, 1968)

# References :
#   * https://en.wikipedia.org/wiki/A*_search_algorithm
#   * https://doi.org/10.1109%2FTSSC.1968.300136: Hart, P. E.; Nilsson, N. J.; Raphael, B. (1968). "A Formal Basis for the Heuristic Determination of Minimum Cost Paths
# Complexity is O(|V|)

import sys
import string

import networkx as nx

def neighbours(vertex, adjacency_matrix):
    Q = []
    for v in adjacency_matrix[vertex]:
        w = adjacency_matrix[vertex][v]['weight']
        if  w != 0:
            Q.append(v)
    return Q

def reconstruct_path(cameFrom, current):
    total_path = [current]
    for i in set(cameFrom):
        if i != -1:
            total_path.append(i)
    return total_path

def heuristic_function(adjacency_matrix, dest, node):
    INFINITY = 100000
    val = INFINITY
    try :
        val = adjacency_matrix[node][dest]['weight']
    except KeyError:
        pass
    
    return val    
    
def a_star(adjacency_matrix, start, goal):
    INFINITY = 100000
    num_vertices = len(adjacency_matrix)
    
    # The set of discovered nodes that need to be (re-)expanded. Initially, only the start node is known.
    openSet = [start]
    closedSet = []
    # For node n, cameFrom[n] is the node immediately preceding it on the cheapest path from start to n currently known.
    cameFrom = [-1 for _ in range(num_vertices)]

    # For node n, gScore[n] is the cost of the cheapest path from start to n currently known.
    gScore = [INFINITY for _ in range(num_vertices)]
    gScore[start] = 0

    # For node n, fScore[n] := gScore[n] + h(n).
    fScore = [INFINITY for _ in range(num_vertices)]
    fScore[start] = heuristic_function(adjacency_matrix, goal, start)

    while len(openSet) != 0:
        low = INFINITY + 1
        for i in openSet:
            if fScore[i] < low:
                current = i
                
        if current == goal:
            return reconstruct_path(cameFrom, current)

        openSet.remove(current)
        closedSet.append(current)
        N = neighbours(current, adjacency_matrix)
        for neighbor in N:
            if neighbor in closedSet:
                continue
            # d(current,neighbor) is the weight of the edge from current to neighbor
            # tentative_gScore is the distance from start to the neighbor through current
            tentative_gScore = gScore[current] + adjacency_matrix[current][neighbor]['weight']
            if neighbor not in openSet:
                openSet.append(neighbor)                
                cameFrom[neighbor] = current
            elif tentative_gScore < gScore[neighbor]:
                # This path to neighbor is better than any previous one. Record it!
                cameFrom[neighbor] = current
                print(neighbor, current)
                gScore[neighbor] = tentative_gScore
                fScore[neighbor] = gScore[neighbor] + heuristic_function(adjacency_matrix, goal, neighbor)

    # Open set is empty but goal was never reached
    return failure

#=============================================================================#
def main(arguments):
    num_nodes = 9
    vertices = []
    for i in range(num_nodes):        
        vertices.append(string.ascii_uppercase[i])
        
    G = nx.Graph()
    adjacency_matrix = [
        #A, B, C, D, E, F, G, H, I
        [0, 1, 0, 0, 1, 0, 0, 0, 0], # A
        [1, 0, 9, 0, 1, 0, 0, 0, 0], # B
        [0, 9, 0, 1, 0, 1, 0, 0, 0], # C
        [0, 0, 1, 0, 0, 1, 0, 0, 0], # D
        [1, 1, 0, 0, 0, 0, 7, 0, 0], # E
        [0, 0, 1, 1, 0, 0, 0, 0, 0], # F
        [0, 0, 0, 0, 7, 0, 0, 1, 1], # G
        [0, 0, 0, 0, 0, 0, 1, 0, 1], # H
        [0, 0, 0, 0, 0, 0, 1, 1, 0]  # I
        #A, B, C, D, E, F, G, H, I
    ]
    distance_matrix = adjacency_matrix
    G = nx.Graph()
    for i in range(len(adjacency_matrix)):
        for j in range(len(adjacency_matrix[i])):
            if adjacency_matrix[i][j] > 0:
                G.add_edge(i, j, weight = adjacency_matrix[i][j])
                if adjacency_matrix[i][j] != adjacency_matrix[j][i]:
                    print ("incoherent! in i=" + str(i) + ", j=" + str(j))
    adjacency_matrix = nx.convert.to_dict_of_dicts (G)

    src = 0
    dst = 7
    path = a_star(adjacency_matrix, src, dst)
    print("Nodes on the path from ", vertices[src], "to", vertices[dst], "are :")
    for i in path:
        print(vertices[i])
    
    sys.exit(0)
    
#=============================================================================#
if __name__ == '__main__':
    if sys.version_info[0] < 3:
        print ("Untested with  Python < 3")

    main(sys.argv)
else:
  sys.exit(1)
