package main

import (
	"sort"
)

type Node interface {
	Hash(i int) uint32
}

type vNodeSlice []uint32

func (p vNodeSlice) Len() int           { return len(p) }
func (p vNodeSlice) Less(i, j int) bool { return p[i] < p[j] }
func (p vNodeSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

type ConstHash struct {
	nodes  map[uint32]Node
	vnodes vNodeSlice
	vcount int
}

func NewConstHash(vcount int, nodes []Node) *ConstHash {
	ch := &ConstHash{
		vcount: vcount,
		nodes:  make(map[uint32]Node)}

	for _, n := range nodes {
		for i := 0; i < ch.vcount; i++ {
			k := n.Hash(i)
			ch.nodes[k] = n
			ch.vnodes = append(ch.vnodes, k)
		}
	}

	sort.Sort(ch.vnodes)

	return ch
}

func (ch *ConstHash) Get(h uint32) Node {
	i := sort.Search(ch.vnodes.Len(),
		func(i int) bool { return ch.vnodes[i] >= h })
	return ch.nodes[ch.vnodes[i%ch.vnodes.Len()]]
}
