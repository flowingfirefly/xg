package BILIBILI

import (
  "fmt"
  "net/http"
  "os"
  "regexp"
  "strconv"
  "sync"
)

const (
  Cookie    string = "SESSDATA=7e12429d%2C1601717808%2C35fc0*41" // 此cookie随时可能过期,过期后将会导致视频可下载清晰度下降.
  UserAgent string = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/73.0.3683.103 Safari/537.36"
  Referer   string = "https://www.bilibili.com"
)

type M4S struct {
  VideoUrl string
  AudioUrl string
}

type FLV []string

type CInfo struct {
  AVID        uint64
  BVID        string
  CID         uint64
  Title       string
  M4S         *M4S
  FLV         *FLV
  QualityName string
}

func query_size(req http.Request) uint64 {
  client := http.Client{}
  resp, _ := client.Do(&req)
  if resp == nil {
    return 0
  }
  resp.Body.Close()
  lenstr := resp.Header.Get("Content-Length")
  lenstr = lenstr
  u64, _ := strconv.ParseUint(lenstr, 10, 64)
  return u64
}

const (
  SEGMENT_MIN_SIZE uint64 = (1024 * 1024)
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

func dl_range(req http.Request, file *os.File, start uint64, end uint64, group *sync.WaitGroup) {
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
    file.Sync()
  }
}

func dl(req http.Request, filename string, group *sync.WaitGroup) {

  defer group.Done()

  fi, err := os.Stat(filename)
  if err == nil {
    fi.IsDir()
    return
  }

  totalSize := query_size(req)
  if totalSize <= 0 {
    return
  }

  tmpfilename := "_" + filename
  file, e := os.OpenFile(tmpfilename, os.O_CREATE|os.O_WRONLY, 600)
  if file == nil {
    e = e
    return
  }

  rangeGroup := new(sync.WaitGroup)
  var segmentSize uint64 = totalSize / 32
  segmentSize = __max(segmentSize, SEGMENT_MIN_SIZE)

  for range_start := uint64(0); range_start < totalSize; range_start += segmentSize {
    rangeGroup.Add(1)
    range_end := __min(totalSize-1, range_start+segmentSize-1)
    go dl_range(req, file, range_start, range_end, rangeGroup)
  }
  rangeGroup.Wait()

  file.Close()
  os.Rename(tmpfilename, filename)
}

func get_filename_from_url(url string) string {
  regex, _ := regexp.Compile(`/([-\.a-z0-9]+)\?`)
  matchs := regex.FindStringSubmatch(url)
  if len(matchs) != 2 {
    return ""
  }
  return matchs[1]
}

func copy_header(dst *http.Header, src *http.Header) {

  for k, v := range *src {
    for _, v2 := range v {
      dst.Add(k, v2)
    }
  }
}

func QueryPlayurl(avid uint64, bvid string, cid uint64) CInfo {
  var info CInfo

  result := API_play_playurl(avid, bvid, cid)

  info.QualityName = GetQualityName(result.Data.Quality)

  if result.Data.Durl != nil {
    info.FLV = &FLV{}
    for i := 0; i < len(*result.Data.Durl); i++ {
      *info.FLV = append(*info.FLV, (*result.Data.Durl)[i].Url)
    }
  }
  if result.Data.Dash != nil {
    info.M4S = &M4S{}

    info.M4S.AudioUrl = result.Data.Dash.Audio[0].BaseUrl
    info.M4S.VideoUrl = result.Data.Dash.Video[0].BaseUrl
  }
  return info
}

func AutoDownload(url string) bool {

  regexAVID, _ := regexp.Compile(`video/av([0-9]+)`)
  regexBVID, _ := regexp.Compile(`video/(BV[a-zA-Z0-9]+)`)
  regexChannel, _ := regexp.Compile(`space.bilibili.com/(\d+)/channel/detail\?cid=(\d+)$`)

  var infos []CInfo
  if matchs := regexAVID.FindStringSubmatch(url); matchs != nil && len(matchs) == 2 {
    if avid, err := strconv.ParseUint(matchs[1], 10, 64); err == nil {
      result := API_web_interface_view(&avid, nil, nil)
      for i := 0; i < len(result.Data.Pages); i++ {
        info := QueryPlayurl(result.Data.AVID, result.Data.BVID, result.Data.Pages[i].Id)
        info.Title = result.Data.Pages[i].Part
        info.AVID = result.Data.AVID
        info.BVID = result.Data.BVID
        info.CID = result.Data.Pages[i].Id
        fmt.Println(info.AVID, info.BVID, info.CID, info.QualityName, info.Title)
        infos = append(infos, info)
      }
    } else {
      return false
    }
  } else if matchs := regexBVID.FindStringSubmatch(url); matchs != nil && len(matchs) == 2 {
    bvid := matchs[1]
    result := API_web_interface_view(nil, &bvid, nil)
    for i := 0; i < len(result.Data.Pages); i++ {
      info := QueryPlayurl(result.Data.AVID, result.Data.BVID, result.Data.Pages[i].Id)
      info.Title = result.Data.Pages[i].Part
      info.AVID = result.Data.AVID
      info.BVID = result.Data.BVID
      info.CID = result.Data.Pages[i].Id
      fmt.Println(info.AVID, info.BVID, info.CID, info.QualityName, info.Title)
      infos = append(infos, info)
    }
  } else if matchs := regexChannel.FindStringSubmatch(url); matchs != nil && len(matchs) == 3 {
    mid, _ := strconv.ParseUint(matchs[1], 10, 64)
    cid, _ := strconv.ParseUint(matchs[2], 10, 64)

    result := API_space_channel_video(mid, cid)
    for _, v := range result.Data.List.Archives {
      result := API_web_interface_view(&v.AVID, &v.BVID, nil)

      for i := 0; i < len(result.Data.Pages); i++ {
        info := QueryPlayurl(result.Data.AVID, result.Data.BVID, result.Data.Pages[i].Id)
        info.Title = result.Data.Pages[i].Part
        info.AVID = result.Data.AVID
        info.BVID = result.Data.BVID
        info.CID = result.Data.Pages[i].Id
        fmt.Println(info.AVID, info.BVID, info.CID, info.QualityName, info.Title)
        infos = append(infos, info)
      }
    }
  } else {
    return false
  }

  return true
}
