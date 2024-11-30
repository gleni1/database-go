package main 

import (
  "fmt"
  "encoding/binary"
)

const HEADER = 4

const BTREE_PAGE_SIZE = 4096
const BTREE_MAX_KEY_SIZE = 1000
const BTREE_MAX_VAL_SIZE = 3000

func init(){
  node1max := HEADER + 8 + 2 + 4 + BTREE_MAX_KEY_SIZE + BTREE_MAX_VAL_SIZE
  assert(node1max <= BTREE_PAGE_SIZE) // maximum KV
}

type BNode []byte 

type BTree struct {
  root uint64
  get func(uint64) []byte 
  new func([]byte) uint64
  del func(uint64)
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
func (node BNode) setPtr(idx uint64, val uint64)


























func SaveData(path string, data []byte) error {
  tmp := fmt.Sprintf("%s.tmp.%d", path, randomInt())
  fp, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0664)
  if err != nil {
    return err 
  }
  defer func() {
    fp.Close()
    if err != nil {
      os.Remove(tmp)
    }
  }()

  if _, err = fp.Write(data); err != nil {
    return err 
  }
  if err = fp.Sync(); err != nil {
    return 
    // Ensures that the data is immediately written in disk 
  }
  err = os.Rename(tmp, path)
  return err 
}
