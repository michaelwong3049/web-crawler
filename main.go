package main

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"strings"
	"sync"
	"golang.org/x/net/html"
)

type CrawledSet struct {
  Links map[uint64]struct{}
  mu sync.Mutex
}

func NewCrawledSet() *CrawledSet {
  return &CrawledSet{
    Links: make(map[uint64]struct{}),
  }
}

const SITE_URL = "https://www.cc.gatech.edu/"

func main() {
  fmt.Println("--- runnning main ---")
  websites := make(chan string) 
  crawledSet := NewCrawledSet()
  var wg sync.WaitGroup
  for w := 0; w <= 4; w++ {
    wg.Add(1)
    go func() {
      err := worker(w, websites, crawledSet, &wg)
      if err != nil {
        fmt.Println("BIG ERROR FOUND: ", err)
        return
      }
    }()
  }

  var url uint64 = hashUrl("https://www.cc.gatech.edu/")
  crawledSet.mu.Lock() 
  crawledSet.Links[url] = struct{}{}
  crawledSet.mu.Unlock() 
  websites <- "https://www.cc.gatech.edu/"

  wg.Wait()
}

func worker(id int, websites chan string, cs *CrawledSet, wg *sync.WaitGroup) (err error) {
  defer wg.Done()

  fmt.Println("worker: ", id)

  for site := range websites {

    parseHTML(site, websites, cs)
  }

  return nil
}

func hashUrl(url string) uint64 {
  h := fnv.New64a()
  h.Write([]byte(url))
  return h.Sum64()
}

func maxWebsitesCrawled(cs *CrawledSet) (bool) {
  cs.mu.Lock()
  if len(cs.Links) >= 5000 {
    cs.mu.Unlock()
    return true
  } else {
    cs.mu.Unlock()
    return false
  }
}

func parseHTML(site string, websites chan string, cs *CrawledSet) (err error) {
  fmt.Println("parsing site: ", site)

  res, err := http.Get(site)
  if err != nil {
    fmt.Println("Error: failed to fetch/get site: ", site)
    return err
  }

  body, err := io.ReadAll(res.Body)
  reader := bytes.NewReader(body)
  z := html.NewTokenizer(reader)

  CRAWL:
  for {
    tt := z.Next()
    switch tt {
    case html.ErrorToken:
      if z.Err() == io.EOF {
        break CRAWL // if end of file, we stop crawling
      }
      if z.Err() != nil {
        return z.Err()
      }

    case html.StartTagToken:
      token := z.Token()
      if len(token.Data) == 1 && token.Data == "a" {
        url := ""
        for i := range token.Attr {
          key := token.Attr[i].Key
          val := token.Attr[i].Val
          if key == "href" {
            if strings.HasPrefix(val, "https") {
              url = val
            } else if len(val) > 1 && strings.HasPrefix(val, "/") {
              url = SITE_URL + val[1:]
            }
          }
        }

        if len(url) == 0 {
          continue
        }

        // now lets put this back in the channel to crawl more
        cs.mu.Lock()
        hashedUrl := hashUrl(url)
        if _, ok := cs.Links[hashedUrl]; !ok {
          cs.Links[hashedUrl] = struct{}{}
          cs.mu.Unlock()
          if !maxWebsitesCrawled(cs) {
            go func() {
              websites <- url
            }()
          }
        } else {
          cs.mu.Unlock()
        }
      }
    }
  }

  defer res.Body.Close()

  return nil
}

