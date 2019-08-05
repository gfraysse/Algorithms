#!/usr/bin/env python3
# -*- coding: utf-8 -*-
#
#   Copyright "Guillaume Fraysse <gfraysse dot spam plus code at gmail dot com>"
#
# How-to run: 
#   python3 floyd-marshall.py

# Python implementation of the Bernard Roy (1959) or Floyd–Warshall Shortest Path algorithm (1962)

# References :
#   * https://en.wikipedia.org/wiki/Floyd%E2%80%93Warshall_algorithm
#   * https://gallica.bnf.fr/ark:/12148/bpt6k3201c/f222.image (Bernard Roy 1959)
#   * https://doi.org/10.1145%2F367766.368168 (Floyd, Robert W., June 1962)
#   * https://doi.org/10.1145%2F321105.321107 (Warshall, Stephen, January 1962)

# Complexity is O(|V|^3)

import sys
import string

import networkx as nx

def floyd_marshall(adjacency_matrix, entry):
    INFINITY = 100000
    num_vertices = len(adjacency_matrix)
    dist = [ [INFINITY for _ in range(num_vertices)] for _ in range(num_vertices)]
    prev = [ [0 for _ in range(num_vertices)] for _ in range(num_vertices)]

    print(adjacency_matrix)
    for i in range(num_vertices):
        for j in range(num_vertices):
            if i == j :
                dist[i][j] = 0
            else:
                try :
                    dist[i][j] = adjacency_matrix[i][j]['weight']
                except KeyError:
                    pass
                    
    for k in range(num_vertices):
        for i in range(num_vertices):
            for j in range(num_vertices):
                if dist[i][j] > dist[i][k] + dist[k][j]:
                    dist[i][j] = dist[i][k] + dist[k][j]

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
    dist, prev = floyd_marshall(adjacency_matrix, node)
    print ("Distance vector is", dist)
    print ("Nodes on the path are", prev)
    
    sys.exit(0)
    
#=============================================================================#
if __name__ == '__main__':
    if sys.version_info[0] < 3:
        print ("Untested with  Python < 3")

    main(sys.argv)
else:
  sys.exit(1)
