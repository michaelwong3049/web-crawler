# web-crawler

A concurrent web crawler written in Go that performs a BFS traversal starting from Hunter College's website (`hunter.cuny.edu`) and stores each crawled page's URL and title into MongoDB.

## How it works

- Starts at `https://hunter.cuny.edu/` and discovers links via HTML parsing
- Runs 5 concurrent worker goroutines that pull URLs from a shared channel
- Tracks visited URLs using a hashed set to avoid duplicates
- Stops after crawling 250 pages
- Saves each page's URL and title as a document in MongoDB

## Prerequisites

- [Go](https://go.dev/doc/install) 1.21+
- A running [MongoDB](https://www.mongodb.com/docs/manual/installation/) instance (local or Atlas)

## Installation

```bash
git clone https://github.com/michaelwong3049/web-crawler.git
cd web-crawler
go mod download
```

## Configuration

Create a `.env` file in the project root:

For MongoDB Atlas, use your connection string:

```env
MONGODB_URI=mongodb+srv://<user>:<password>@<cluster>.mongodb.net/
```

## Running

```bash
go run main.go
```

Crawled pages are saved to the `web-crawler` database under the `websites` collection, each document containing:

```json
{
  "url": "https://hunter.cuny.edu/some/page",
  "title": "Page Title"
}
```
