package main

import (
	"bytes"
	"encoding/binary"
	_ "fmt"
	"log"
	_ "os"
)

const HEADER = 4

const BTREE_PAGE_SIZE = 4096
const BTREE_MAX_KEY_SIZE = 1000
const BTREE_MAX_VAL_SIZE = 3000

func assert(condition bool) {
	if !condition {
		log.Fatal("Assertion failed!")
	}
}

func init() {
	node1max := HEADER + 8 + 2 + 4 + BTREE_MAX_KEY_SIZE + BTREE_MAX_VAL_SIZE
	assert(node1max <= BTREE_PAGE_SIZE) // maximum KV
}

type BNode []byte

type BTree struct {
	root uint64
	get  func(uint64) []byte
	new  func([]byte) uint64
	del  func(uint64)
}

const (
	BNODE_NODE = 1
	BNODE_LEAF = 2
)

func (node BNode) btype() uint16 {
	return binary.LittleEndian.Uint16(node[0:2])
}

func (node BNode) nkeys() uint16 {
	return binary.LittleEndian.Uint16(node[2:4])
}

func (node BNode) setHeader(btype uint16, nkeys uint16) {
	binary.LittleEndian.PutUint16(node[0:2], btype)
	binary.LittleEndian.PutUint16(node[2:4], nkeys)
}

func (node BNode) getPtr(idx uint16) uint64 {
	assert(idx < node.nkeys())
	pos := HEADER + 8*idx
	return binary.LittleEndian.Uint64(node[pos:])
}
func (node BNode) setPtr(idx uint16, val uint64)

// find offset in offset list
func offsetPos(node BNode, idx uint16) uint16 {
	assert(1 <= idx && idx <= node.nkeys())
	return HEADER + 8*node.nkeys() + 2*(idx-1)
}
func (node BNode) getOffset(idx uint16) uint16 {
	if idx == 0 {
		return 0
	}
	return binary.LittleEndian.Uint16(node[offsetPos(node, idx):])
}
func (node BNode) setOffset(idx uint16, offset uint16)

// find key length and keys
func (node BNode) kvPos(idx uint16) uint16 {
	assert(idx <= node.nkeys())
	return HEADER + 8*node.nkeys() + 2*node.nkeys() + node.getOffset(idx)
}
func (node BNode) getKey(idx uint16) []byte {
	assert(idx < node.nkeys())
	pos := node.kvPos(idx)
	klen := binary.LittleEndian.Uint16(node[pos:])
	return node[pos+4:][:klen]
}
func (node BNode) getVal(idx uint16) []byte

func (node BNode) nbytes() uint16 {
	return node.kvPos(node.nkeys())
}

// returns the first kid node whose range intersects the key
// (kid[i] <= key)
func nodeLookupLE(node BNode, key []byte) uint16 {
	nkeys := node.nkeys()
	found := uint16(0)

	for i := uint16(1); i < nkeys; i++ {
		cmp := bytes.Compare(node.getKey(i), key)
		if cmp <= 0 {
			found = i
		}
		if cmp >= 0 {
			break
		}
	}
	return found
}

// add a new key to a leaf node
func leafInsert(new BNode, old BNode, idx uint16, key []byte, val []byte) {
	new.setHeader(BNODE_LEAF, old.nkeys()+1)
	nodeAppendRange(new, old, 0, 0, idx)
	nodeAppendKV(new, idx, 0, key, val)
	nodeAppendRange(new, old, idx+1, idx, old.nkeys()-idx)
}

// copy a KV into the position
func nodeAppendKV(new BNode, idx uint16, ptr uint64, key []byte, val []byte) {
	// pointers
	new.setPtr(idx, ptr)
	// kvs
	pos := new.kvPos(idx)
	binary.LittleEndian.PutUint16(new[pos+0:], uint16(len(key)))
	binary.LittleEndian.PutUint16(new[pos+2:], uint16(len(val)))
	copy(new[pos+4:], key)
	copy(new[pos+4+uint16(len(key)):], val)
	// the offset of the next key
	new.setOffset(idx+1, new.getOffset(idx)+4+uint16((len(key)+len(val))))
}

// copy multiple KVs into the position from the old node
func nodeAppendRange(new BNode, old BNode, dstNew uint16, srcOld uint16, n uint16)

// replace a linke with one or multiple links
func nodeReplaceKidN(tree *BTree, new BNode, old BNode, idx uint16, kids ...BNode) {
	inc := uint16(len(kids))
	new.setHeader(BNODE_NODE, old.nkeys()+inc-1)
	nodeAppendRange(new, old, 0, 0, idx)
	for i, node := range kids {
		nodeAppendKV(new, idx+uint16(i), tree.new(node), node.getKey(0), nil)
	}
	nodeAppendRange(new, old, idx+inc, idx+1, old.nkeys()-(idx+1))
}

// splt an oversized node into 2 sot that the 2nd node always fits on a page
func nodeSplit12(left BNode, right BNode, old BNode) {

}

// splt a node if its too big, the results are 1-3 nodes
func nodeSplt3(old BNode) (uint16, [3]BNode) {
	if old.nbytes() <= BTREE_PAGE_SIZE {
		old = old[:BTREE_PAGE_SIZE]
		return 1, [3]BNode{old}
	}
	left := BNode(make([]byte, 2*BTREE_PAGE_SIZE))
	right := BNode(make([]byte, BTREE_PAGE_SIZE))
	nodeSplit2(left, right, old)
	if left.nbytes() <= BTREE_PAGE_SIZE {
		left = left[:BTREE_PAGE_SIZE]
		return 2, [3]BNode{left, right} // 2 nodes
	}
	leftleft := BNode(make([]byte, BTREE_PAGE_SIZE))
	middle := BNode(make([]byte, BTREE_PAGE_SIZE))
	nodeSplit2(leftleft, middle, left)
	assert(leftleft.nbytes() <= BTREE_PAGE_SIZE)
	return 3, [3]BNode{leftleft, middle, right} // 3 nodes
}

/*
	 insert a KV into a node, the result might be split
		the caller is responsible for deallocating the input node
		and splitting and allocating result nodes.
*/
func treeInsert(tree *BTree, node BNode, key []byte, val []byte) BNode {
	// the result node
	// it's allowed to be bigger than 1 page and will be split if so
	new := BNode{data: make([]byte, 2*BTREE_PAGE_SIZE)}

	// where to insert the key
	idx := nodeLookupLE(node, key)
	// act depending on the node type
	switch node.btype() {
	case BNODE_LEAF:
		if bytes.Equal(key, node.getKey(idx)) {
			// found the key, update it
			leafUpdate(new, node, idx, key, val)
		} else {
			// insert it after the position
			leafInsert(new, node, idx+1, key, val)
		}
	case BNODE_NODE:
		// internal node, insert it to a kid node.
		nodeInsert(tree, new, node, idx, key, val)
	default:
		panic("bad node!")
	}
	return new
}

// part of the treeInsert;  KV insertion to an internal node
func nodeInsert(tree *BTree, new BNode, node BNode, idx uint16, key []byte, val []byte) {
	kptr := node.getPtr(idx)
	// recursive insertion to the kid node
	knode := treeInsert(tree, tree.get(kptr), key, val)
	//split the result
	nsplit, split := nodeSplit3(knode)
	// deallocate the kid node
	tree.del(kptr)
	// update the kid links
	nodeReplaceKidN(tree, new, node, idx, split[:nsplit]...)
}


//insert a new key or update an existing key 
func (tree *BTree) Insert(key []byte, val []byte) error 

//delete a key and returns whether the key was there 
func (tree *BTree) Delete(key []byte) (bool, error)

func (tree *BTree) Insert(key []byte, val []byte) error {
  // 1. check the length limit imposed by the node format
  if err := checkLimit(key, val); err != nil {
    retun err 
  }
  // 2. create the first node 
  if tree.root == 0 {
    root := BNode(make([]byte, BTREE_PAGE_SIZE))
    
    tree.root = tree.new(root)
    return nil 
  }
  // 3. inset the key 
  node := treeInsert(tree, tree.get(tree.root), key, val)
  // 4. grow the tree if the root is split 
  nsplit, split := nodeSplit3(node)
  tree.del(tree.root)
  if nsplit > 1 {
    root := BNode(make([]byte, BTREE_PAGE_SIZE))
    tree.root = tree.new(root)
  } else {
    tree.root = tree.new(split[0])
  }
  return nil 
}
