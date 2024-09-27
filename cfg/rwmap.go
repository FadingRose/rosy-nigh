package cfg

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/holiman/uint256"
)

type after struct {
	node *node
	cnt  int
}

type kvs [][2]uint256.Int

func (kvs *kvs) append(k, v uint256.Int) {
	val := kvs.find(k)
	if val == nil || val.Cmp(&v) != 0 {
		*kvs = append(*kvs, [2]uint256.Int{k, v})
	}
}

func (kvs *kvs) find(k uint256.Int) *uint256.Int {
	for _, kv := range *kvs {
		if kv[0].Cmp(&k) == 0 {
			return &kv[1]
		}
	}
	return nil
}

func (kvs *kvs) exist(key, value *uint256.Int) bool {
	for _, kv := range *kvs {
		if kv[0].Cmp(key) == 0 && kv[1].Cmp(value) == 0 {
			return true
		}
	}
	return false
}

type node struct {
	name string // `name` is the contract function name
	r    *kvs
	w    *kvs

	// if A read SLOT 0x0, and B write 0x0, meas A deps B, then A should be after B
	// in this case, B.after = append(B.after, A)
	afters []*after
}

func newNode(name string, r, w *kvs) *node {
	return &node{
		name:   name,
		r:      r,
		w:      w,
		afters: make([]*after, 0),
	}
}

func (n *node) string() string {
	var ret strings.Builder
	ret.WriteString(fmt.Sprintf("Node: %s [", n.name))
	for _, kv := range n.afters {
		ret.WriteString(fmt.Sprintf("%s(%d) ", kv.node.name, kv.cnt))
	}
	ret.WriteString("]\n")
	return ret.String()
}

func (n *node) appendAfter(as ...*node) {
	for _, a := range as {
		af := n.findAfter(a)
		if af == nil {
			n.afters = append(n.afters, &after{node: a, cnt: 1})
		} else {
			af.cnt++
		}
	}
}

// findAfter returns the after of a node
func (n *node) findAfter(a *node) *after {
	for _, after := range n.afters {
		if after.node.name == a.name {
			return after
		}
	}
	return nil
}

type RWMap struct {
	accessList map[string][]SlotAccess
	nodes      []*node
}

func NewRWMap(accessList map[string][]SlotAccess) *RWMap {
	var nodes []*node
	for s, sa := range accessList {
		var (
			// r = new(kvs)
			// w = new(kvs)
			r = make(kvs, 0)
			w = make(kvs, 0)
		)
		for _, s := range sa {
			switch s.AccessType {
			case Read:
				r.append(s.Key, s.Value)
			case Write:
				w.append(s.Key, s.Value)
			default:
				panic("unknown access type" + s.String())
			}
		}
		nodes = append(nodes, newNode(s, &r, &w))
	}

	rwmap := &RWMap{
		accessList: accessList,
		nodes:      nodes,
	}

	// construct all afters
	for _, n := range nodes {
		for _, kv := range *n.w {
			if reads, ok := rwmap.check(&kv[0], &kv[1], Read); ok {
				n.appendAfter(reads...)
			}
		}
	}

	return rwmap
}

// check returns all nodes whose read list contains the key-value pair
func (rwMap *RWMap) check(key, value *uint256.Int, acc AccessType) ([]*node, bool) {
	var (
		deps = make([]*node, 0)
		ok   = false
	)

	for _, n := range rwMap.nodes {
		if acc == Read && n.r.exist(key, value) {
			deps = append(deps, n)
			ok = true
		}

		if acc == Write && n.w.exist(key, value) {
			deps = append(deps, n)
			ok = true
		}
	}
	return deps, ok
}

// Filter returns all function name that access the key with the given access type
func (rwMap *RWMap) Filter(key *uint256.Int, acc AccessType) ([]string, bool) {
	var (
		ret  []string
		flag = false
	)

	for _, n := range rwMap.nodes {
		var kvs *kvs
		if acc == Read {
			kvs = n.r
		} else {
			kvs = n.w
		}

		if kvs.find(*key) != nil {
			ret = append(ret, n.name)
			flag = true
		}
	}

	return ret, flag
}

func (rwMap *RWMap) String() string {
	var ret strings.Builder
	for _, n := range rwMap.nodes {
		ret.WriteString(n.string())
	}
	return ret.String()
}

// Visit return a sequence of nodes
func (rwMap *RWMap) Visit(depth int) []string {
	entires := rwMap.entries()
	var (
		entry = entires[rand.Intn(len(entires))]
		ret   []string
		cur   = entry
	)

	// try to pick from afters
	picker := func(afts []*after) *node {
		totalCnt := 0
		for _, a := range afts {
			totalCnt += a.cnt
		}

		if totalCnt == 0 {
			return nil
		}

		randomNum := rand.Intn(totalCnt)
		thisCnt := 0
		for _, a := range afts {
			if randomNum < thisCnt+a.cnt {
				return a.node
			}
		}
		return nil
	}

	// pick from all nodes
	pickerNodes := func(nodes []*node) *node {
		// node with more write access should be picked first
		maxCnt := 0
		for _, n := range nodes {
			maxCnt += len(*n.w)
		}

		if maxCnt == 0 {
			return nodes[rand.Intn(len(nodes))]
		}

		randomNum := rand.Intn(maxCnt)
		thisCnt := 0
		for _, n := range nodes {
			if randomNum < thisCnt+len(*n.w) {
				return n
			}
			thisCnt += len(*n.w)
		}
		return nil
	}

	for i := 0; i < depth; i++ {
		// pick a node, weight is cnt
		cur = picker(cur.afters)
		if cur == nil {
			cur = pickerNodes(rwMap.nodes)
			if cur == nil {
				panic("no node found")
			}
		}
		ret = append(ret, cur.name)
	}

	return ret
}

// Nodes returns all nodes in the graph by name
func (rwMap *RWMap) Nodes() []string {
	var ret []string
	for _, n := range rwMap.nodes {
		ret = append(ret, n.name)
	}
	return ret
}

// entries returns all nodes which are the entry of the graph, means they are not deps of any other nodes
func (rwMap *RWMap) entries() []*node {
	entries := make([]*node, 0)
	for _, n := range rwMap.nodes {
		flag := false
		for _, nn := range rwMap.nodes {
			if nn.findAfter(n) != nil {
				flag = true
				break
			}
		}

		if !flag {
			entries = append(entries, n)
		}
	}

	if len(entries) == 0 {
		panic("no entry found")
	}
	return entries
}
