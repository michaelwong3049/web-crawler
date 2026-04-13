package main

import (
	"fmt"
  "io"
  "os"
	"hash/fnv"
	"net/http"
	"sync"
  "bytes"
  "strings"
  "context"

	"golang.org/x/net/html"
  "github.com/joho/godotenv"
  "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type CrawledSet struct {
  Links map[uint64]struct{}
  mu sync.Mutex
}

type Website struct {
  Url string
  Title string
}

type DatabaseConnection struct {
  url string
  client *mongo.Client
  collection *mongo.Collection
}

func NewCrawledSet() *CrawledSet {
  return &CrawledSet{
    Links: make(map[uint64]struct{}),
  }
}

func NewDatabaseConnection(connectionUrl string) *DatabaseConnection {
  return &DatabaseConnection{
    url: connectionUrl,
    client: nil,
    collection: nil,
  }
}

func initializeDatabase(connectionUrl string) (db *DatabaseConnection, err error) {
  db = NewDatabaseConnection(connectionUrl)

  client, err := mongo.Connect(options.Client().ApplyURI(connectionUrl))

  if err != nil {
    return nil, err
  }

  db.client = client
  db.collection = client.Database("web-crawler").Collection("websites")

  return db, nil
}

const SITE_URL = "https://hunter.cuny.edu/"

func main() {
  fmt.Println("--- runnning main ---")

  err := godotenv.Load()
  if err != nil {
    panic(err)
  }

  MONGODB_URI := os.Getenv("MONGODB_URI")

  database, err := initializeDatabase(MONGODB_URI)
  if err != nil || database.client == nil {
    fmt.Println("Error connecting to database")
    return 
  }

  defer func() {
    if err := database.client.Disconnect(context.TODO()); err != nil {
      panic(err)
    }
  }()

  fmt.Println(database.client)

  websites := make(chan string) 
  crawledSet := NewCrawledSet()
  client := &http.Client{}

  var wg sync.WaitGroup
  for w := 0; w <= 4; w++ {
    wg.Add(1)
    go func() {
      err := worker(w, websites, crawledSet, database, client, &wg)
      if err != nil {
        fmt.Println("BIG ERROR FOUND: ", err)
        return
      }
    }()
  }

  var url uint64 = hashUrl(SITE_URL)
  crawledSet.mu.Lock() 
  crawledSet.Links[url] = struct{}{}
  crawledSet.mu.Unlock() 
  websites <- SITE_URL

  wg.Wait()
}

func worker(id int, websites chan string, cs *CrawledSet, db *DatabaseConnection, client *http.Client, wg *sync.WaitGroup) (err error) {
  defer wg.Done()

  fmt.Println("worker: ", id)

  for site := range websites {
    cs.mu.Lock()
    fmt.Print("length: ", len(cs.Links))
    cs.mu.Unlock()

    parseHTML(site, websites, db, cs, client)
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
  if len(cs.Links) >= 250 {
    cs.mu.Unlock()
    return true
  } else { 
    cs.mu.Unlock()
    return false
  }
}

func parseHTML(site string, websites chan string, db *DatabaseConnection, cs *CrawledSet, client *http.Client) (err error) {
  fmt.Println("parsing site: ", site)

  req, err := http.NewRequest("GET", site, nil)
  if err != nil {
    fmt.Println("Error: failed to fetch/get site: ", site)
    return err
  }

  req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36")
  res, err := client.Do(req)

  if err != nil {
    fmt.Println("Error from client.Do", err) 
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
      tt = z.Next()
      token := z.Token()
      if len(token.Data) == 5 && token.Data == "title" {
        z.Next()
        token := z.Token()
        website := Website{
          Url: site,
          Title: token.Data,
        }

        res, err := db.collection.InsertOne(context.TODO(), website)
        if err != nil {
          panic(err)
        }
        fmt.Println("res: ", res)
      }

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
          cs.mu.Unlock()
          if !maxWebsitesCrawled(cs) {
            fmt.Println("maxWebsitesCrawled: ", maxWebsitesCrawled(cs))
            cs.mu.Lock()
            cs.Links[hashedUrl] = struct{}{}
            cs.mu.Unlock()
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

