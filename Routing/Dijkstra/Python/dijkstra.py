#!/usr/bin/env python3
# -*- coding: utf-8 -*-
#
#   Copyright "Guillaume Fraysse <gfraysse dot spam plus code at gmail dot com>"
#
# How-to run: 
#   python3 dijkstra.py

# Python implementation of the Dijkstra Shortest Path algorithm (1956)

# References :
#   * https://doi.org/10.1007%2FBF01386390
#   * http://www-m3.ma.tum.de/twiki/pub/MN0506/WebHome/dijkstra.pdf
#   * https://en.wikipedia.org/wiki/Dijkstra%27s_algorithm

# Complexity is O(|E| + |V|^2)

import sys
import string

import networkx as nx

##############################
# Local modules
##############################
def neighbours(vertex, adjacency_matrix):
    Q = []
    for v in adjacency_matrix[vertex]:
        w = adjacency_matrix[vertex][v]['weight']
        if  w != 0:
            Q.append(v)
    return Q

def dijkstra(adjacency_matrix, entry):
    INFINITY = 100000

    current_vertex = entry
    num_nodes = len(adjacency_matrix)
    vertices = []
    for i in range(num_nodes):        
        vertices.append(string.ascii_uppercase[i])
        
    Q = [i for i in range(num_nodes)]

    dist = [0 for _ in range(num_nodes)] # distance to src    
    prev = [0 for _ in range(num_nodes)] # list of nodes on the path
    
    for i in range(num_nodes):
        dist[i] = INFINITY
        prev[i] = []

    dist[current_vertex] = 0        
    
    last_found_resource = -1
        
    while len(Q) != 0:        
        min_dist = INFINITY
        for q in Q:
            if (dist[q] < min_dist):
                v = q

        N = neighbours(v, adjacency_matrix)
        Q.remove(v)
        for n in N:            
            alt = dist[v] + adjacency_matrix[v][n]['weight']
            if alt < dist[n] or dist[n] == INFINITY:
                dist[n] = alt
                prev[n] = prev[v][:]
                prev[n].append(v)

    return dist, prev

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

    node = 0
    dist, prev = dijkstra(adjacency_matrix, node)
    print ("Distance vector from node #" + vertices[node] + " is", dist)
    print ("Nodes on the path from node #" + vertices[node] + " are", prev)
    
    sys.exit(0)
  
#=============================================================================#
if __name__ == '__main__':
    if sys.version_info[0] < 3:
        print ("Untested with  Python < 3")

    main(sys.argv)
else:
  sys.exit(1)
