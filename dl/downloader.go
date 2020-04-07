package dl

import (
  "fmt"
  "github.com/cheggaaa/pb/v3"
  "net/http"
  "os"
  "sync"
)

type Dl struct {
  MaxParallel uint16
  groups      []TaskGroup
}

type ITask interface {
  MakeHeader() http.Header
  Completed()
}

type TaskInfo struct {
  Url       string
  Filename  string
  TotalSize uint64
}

type TaskGroup struct {
  Tasks   []TaskInfo
  Handler ITask
}

const (
  SEGMENT_MIN_SIZE uint64 = (1024 * 1024 * 10)
)

func __min(a uint64, b uint64) uint64 {
  if a < b {
    return a
  }
  return b
}

func __max(a uint64, b uint64) uint64 {
  if a > b {
    return a
  }
  return b
}

func copy_header(dst *http.Header, src *http.Header) {

  for k, v := range *src {
    for _, v2 := range v {
      dst.Add(k, v2)
    }
  }
}

func dl_range(req http.Request, file *os.File, start uint64, end uint64, group *sync.WaitGroup, bar *pb.ProgressBar) {
  defer group.Done()

  rangeReq := req
  rangeReq.Header = make(http.Header) // must be deep copy .
  copy_header(&rangeReq.Header, &req.Header)
  //fmt.Printf("%p %p\n", &rangeReq, &req)
  //fmt.Printf("%p %p\n", &rangeReq.Header, &req.Header)
  rangeReq.Header.Add("DNT", "1")
  rangeReq.Header.Add("Range", fmt.Sprintf("bytes=%v-%v", start, end))
  rangeReq = rangeReq

  client := http.Client{}
  resp, _ := client.Do(&rangeReq)
  if resp == nil {
    return
  }
  defer resp.Body.Close()
  lenstr := resp.Header.Get("Content-Length")
  lenstr = lenstr

  buf := make([]byte, 1024*1024)

  for {
    nread, _ := resp.Body.Read(buf)
    if nread <= 0 {
      break
    }

    file.WriteAt(buf[0:nread], int64(start))
    start += uint64(nread)
    bar.Add(nread)
    file.Sync()
  }
}

func (this *Dl) download(group TaskGroup) {

  totalSize := uint64(0)
  for _, v := range group.Tasks {
    totalSize += v.TotalSize
  }
  pbar := pb.Start64(int64(totalSize))
  pbar.Start()
  pbar.Set(pb.Bytes, true)

  for _, v := range group.Tasks {
    tmpto := "tmp/_" + v.Filename
    saveto := "tmp/" + v.Filename
    fi, err := os.Stat(saveto)
    if err == nil {
      fi.IsDir()
      pbar.Add64(int64(v.TotalSize))
      continue
    }

    os.MkdirAll("tmp", os.ModePerm)
    file, e := os.OpenFile(tmpto, os.O_CREATE|os.O_WRONLY, 0600)
    if file == nil {
      e = e
      continue
    }

    request, _ := http.NewRequest("GET", v.Url, nil)
    request.Header = group.Handler.MakeHeader()

    rangeGroup := new(sync.WaitGroup)
    var segmentSize uint64 = v.TotalSize / 32
    segmentSize = __max(segmentSize, SEGMENT_MIN_SIZE)

    for range_start := uint64(0); range_start < v.TotalSize; range_start += segmentSize {
      rangeGroup.Add(1)
      range_end := __min(v.TotalSize-1, range_start+segmentSize-1)
      go dl_range(*request, file, range_start, range_end, rangeGroup, pbar)
    }
    rangeGroup.Wait()

    file.Close()
    os.Rename(tmpto, saveto)
  }
  pbar.Finish()
  group.Handler.Completed()

}

func (this *Dl) SyncRun() {
  for _, v := range this.groups {
    this.download(v)
  }
}

func (this *Dl) AddTaskGroup(group TaskGroup) {

  this.groups = append(this.groups, group)

}
