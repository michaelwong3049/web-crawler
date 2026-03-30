package main

import (
  "fmt"
  "hash/fnv"
  "net/http"
)

type CrawledSet struct {
  Links map[uint64]struct{}
}

func NewCrawledSet() *CrawledSet {
  return &CrawledSet{
    Links: make(map[uint64]struct{}),
  }
}

type Node struct {
  Url string
  Next *Node
}

func NewNode(url string) *Node {
  return &Node{Url: url}
}

type Queue struct {
  Head *Node 
  Tail *Node
  Size int
}

func NewQueue() *Queue {
  return &Queue{}
}

func (q *Queue) push(url string) {
  q.Tail.Next = NewNode(url)
  q.Tail = q.Tail.Next
  q.Size++
}

func main() {
  queue := NewQueue()
  crawledSet := NewCrawledSet()

  queue.push("https://www.hunter.cuny.edu/")
  var url uint64 = hashUrl("https://www.hunter.cuny.edu/")
  crawledSet.Links[url] = struct{}{}
}

func hashUrl(url string) uint64 {
  h := fnv.New64a()
  h.Write([]byte(url))
  return h.Sum64()
}

