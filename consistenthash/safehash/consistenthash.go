package safehash

import (
	"hash/crc32"
	"sort"
	"sync"
)

const defaultNodeCount = 1000

type HashRing []uint32

func (c HashRing) Len() int {
	return len(c)
}

func (c HashRing) Less(i, j int) bool {
	return c[i] < c[j]
}

func (c HashRing) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

type Node struct {
	Id       string
	Position uint32
	Extra    interface{}
}

func NewNode(id string, position uint32, extra interface{}) *Node {
	return &Node{
		Id:       id,
		Position: position,
		Extra:    extra,
	}
}

type Consistent struct {
	Nodes        map[uint32]*Node
	maxNodeCount uint32
	Resources    map[string]bool
	ring         HashRing
	sync.RWMutex
}

func NewConsistent(maxNodeCount uint32) *Consistent {
	if maxNodeCount < 1 {
		maxNodeCount = defaultNodeCount
	}
	return &Consistent{
		Nodes:        make(map[uint32]*Node),
		Resources:    make(map[string]bool),
		ring:         HashRing{},
		maxNodeCount: maxNodeCount,
	}
}

func (c *Consistent) Add(node *Node) bool {
	c.Lock()
	defer c.Unlock()

	if _, ok := c.Resources[node.Id]; ok {
		return false
	}
	c.Nodes[node.Position] = node
	c.Resources[node.Id] = true
	c.sortHashRing()
	return true
}

func (c *Consistent) sortHashRing() {
	c.ring = HashRing{}
	for k := range c.Nodes {
		c.ring = append(c.ring, k)
	}
	sort.Sort(c.ring)
}

func (c *Consistent) hashStr(key string) uint32 {
	return crc32.ChecksumIEEE([]byte(key)) % c.maxNodeCount
}

func (c *Consistent) Get(key string) *Node {
	c.RLock()
	defer c.RUnlock()

	hash := c.hashStr(key)
	i := c.search(hash)

	return c.Nodes[c.ring[i]]
}

func (c *Consistent) search(hash uint32) int {
	n := len(c.ring)
	i := sort.Search(n, func(i int) bool { return c.ring[i] >= hash })
	if i < n {
		return i
	} else {
		return 0
	}
}

func (c *Consistent) Remove(node *Node) {
	c.Lock()
	defer c.Unlock()

	if _, ok := c.Resources[node.Id]; !ok {
		return
	}
	delete(c.Resources, node.Id)
	delete(c.Nodes, node.Position)
	c.sortHashRing()
}
