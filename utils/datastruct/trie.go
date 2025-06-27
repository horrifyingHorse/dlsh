package datastruct

import (
	"fmt"
	"slices"
)

type TrieNode struct {
	word     bool
	char     rune
	parent   *TrieNode
	children map[rune]*TrieNode
}

type Trie struct {
	root  *TrieNode
	pList []*TrieNode
}

func NewTrieNode(c rune) *TrieNode {
	trieNode := new(TrieNode)
	trieNode.char = c
	trieNode.parent = nil
	trieNode.children = make(map[rune]*TrieNode)
	return trieNode
}

func (trieNode *TrieNode) List(list *[]*TrieNode) {
	if trieNode.word {
		*list = append(*list, trieNode)
	}
	for _, child := range trieNode.children {
		child.List(list)
	}
}

func (trieNode *TrieNode) IsEmpty() bool {
	return len(trieNode.children) == 0
}

func (trieNode *TrieNode) InformChild(r rune) {
	trieNode.children[r].parent = trieNode
}

func (trieNode *TrieNode) GetString() string {
	if !trieNode.word {
		return "[Error]: TrieNode is not a word"
	}
	var s string
	for trieNode.parent != nil {
		s = string(trieNode.char) + s
		trieNode = trieNode.parent
	}
	return s
}

func (trie *Trie) Size() int {
	return len(trie.pList)
}

func (trie *Trie) Set(index uint, s string) {
	node := trie.insertHelper(trie.root, s)
	// if node != trie.Root {
	node.word = true
	trie.pList[index] = node
	// }
}

func (trie *Trie) At(index uint) (string, error) {
	if index < 0 {
		return "", fmt.Errorf("Invalid index: %d", index)
	} else if index > uint(len(trie.pList)) {
		return "", fmt.Errorf("Cannot access list index %d of size %d", index, len(trie.pList))
	} else if index == uint(len(trie.pList)) {
		return "", nil
	}
	return trie.pList[index].GetString(), nil
}

func (trie *Trie) NodeAt(index uint) *TrieNode {
	if index < 0 {
		return nil
	} else if index >= uint(len(trie.pList)) {
		return nil
	}
	return trie.pList[index]
}

func NewTrie() *Trie {
	trie := new(Trie)
	trie.root = NewTrieNode(' ')
	return trie
}

func (trie *Trie) Insert(s string) {
	node := trie.insertHelper(trie.root, s)
	node.word = true
	trie.pList = append(trie.pList, node)
}

func (trie *Trie) insertHelper(node *TrieNode, s string) *TrieNode {
	for _, c := range s {
		if child, exists := node.children[c]; exists {
			node = child
			continue
		}
		node.children[c] = NewTrieNode(c)
		node.InformChild(c)
		node = node.children[c]
	}
	return node
}

func (trie *Trie) IsEmpty() bool {
	return trie.root.IsEmpty()
}

func (trie *Trie) Delete(s string) {
	trieDelHelper(trie, trie.root, s, 0)
}

func trieDelHelper(trie *Trie, trieNode *TrieNode, s string, depth int) *TrieNode {
	if len(s) == 0 {
		trieNode.word = false
		trie.pList = slices.DeleteFunc(trie.pList, func(item *TrieNode) bool {
			return item == trieNode
		})
		if trieNode.IsEmpty() {
			trieNode = nil
		}
		return trieNode
	}

	if child, exists := trieNode.children[rune(s[0])]; exists {
		child = trieDelHelper(trie, child, s[1:], depth+1)
		if child == nil {
			delete(trieNode.children, rune(s[0]))
		}
		if trieNode.IsEmpty() && !trieNode.word {
			trieNode = nil
		}
	}
	return trieNode
}

func (trie *Trie) Search(s string) *Heap[*TrieNode] {
	if s == "" {
		return nil
	}
	node := trie.root
	nodes := []*TrieNode{}
	pq := new(Heap[*TrieNode])

	for _, c := range s {
		if child, exists := node.children[c]; exists {
			node = child
			continue
		} else {
			return pq
		}
	}

	node.List(&nodes)
	var prev *TrieNode
	for priority, ptr := range trie.pList {
		if prev != ptr && slices.Index(nodes, ptr) != -1 {
			prev = ptr
			pq.Insert(ptr, uint(priority))
		}
	}
	return pq
}

func (trie *Trie) List() *[]*TrieNode {
	nodes := new([]*TrieNode)
	for _, child := range trie.root.children {
		child.List(nodes)
	}
	return nodes
}
