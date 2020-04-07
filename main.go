package main

import (
  "fmt"
  "os"
  "regexp"
  "xg/BILIBILI"
  "xg/dl"
)

const version string = "v0.1"

type XgOptions struct {
  Upgrade bool
  Url     string
  Verbose bool
  Output  string
}

func bilibili(downloader *dl.Dl, options XgOptions) {
  infos := BILIBILI.AutoParse(options.Url)
  for _, v := range infos {
    g := dl.TaskGroup{}
    if v.M4S != nil {
      t1 := dl.TaskInfo{}
      t1.Filename = v.M4S.Video.Name
      t1.Url = v.M4S.Video.Url
      t1.TotalSize = v.M4S.Video.Size
      g.Tasks = append(g.Tasks, t1)

      t2 := dl.TaskInfo{}
      t2.Filename = v.M4S.Audio.Name
      t2.Url = v.M4S.Audio.Url
      t2.TotalSize = v.M4S.Audio.Size
      g.Tasks = append(g.Tasks, t2)
    }

    if v.FLV != nil {
      for _, v := range *v.FLV {
        t1 := dl.TaskInfo{}
        t1.Filename = v.Name
        t1.Url = v.Url
        t1.TotalSize = v.Size
        g.Tasks = append(g.Tasks, t1)
      }
    }

    handler := v // copy
    g.Handler = &handler

    downloader.AddTaskGroup(g)
  }

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

  downloader := dl.Dl{MaxParallel: 32}
  switch website[1] {
  case "www.bilibili.com":
    bilibili(&downloader, options)
  case "space.bilibili.com":
    bilibili(&downloader, options)
  case "www.youtube.com":
    fmt.Println(website)
  default:
    fmt.Println("website unsupported")
  }
  downloader.SyncRun()

}
