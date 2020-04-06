package main

import (
  "fmt"
  "os"
  "regexp"
  "sync"
  "xg/BILIBILI"
)

var version string = "v0.1"

type XgOptions struct {
  Upgrade bool
  Url     string
  Verbose bool
  Output  string
}

func main() {
  var options XgOptions
  for i := 1; i < len(os.Args); i++ {
    arg := os.Args[i]
    //fmt.Printf("args[%v] = %v\n", i, os.Args[i])
    switch arg {
    case "--upgrade":
      options.Upgrade = true
    case "-v":
      options.Verbose = true
    case "-o":
      i++
      options.Output = os.Args[i]
    default:
      options.Url = arg
    }
  }

  r, _ := regexp.Compile(`^https?://([a-z0-9\.]+)`)
  website := r.FindStringSubmatch(options.Url)
  if website == nil || len(website) < 2 {
    fmt.Println("missing URL")
    return
  }

  w := sync.WaitGroup{}
  switch website[1] {
  case "www.bilibili.com":
    BILIBILI.AutoDownload(options.Url, &w)
  case "space.bilibili.com":
    BILIBILI.AutoDownload(options.Url, &w)
  case "www.youtube.com":
    fmt.Println(website)
  default:
    fmt.Println(website)
  }

  w.Wait()
}
