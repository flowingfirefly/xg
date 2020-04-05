package BILIBILI

import (
  "encoding/json"
  "fmt"
  "io/ioutil"
  "net/http"
  "os"
  "regexp"
  "strconv"
  "sync"
)

type CInfo struct {
  Id   uint64
  Name string
}

type CInfoLite struct {
  Id   uint64 `json:"cid"`
  Page int    `json:"page"`
  Part string `json:"part"`
}

type API_view_data struct {
  Id    uint64      `json:"aid"`
  Pages []CInfoLite `json:"pages"`
}

type API_playurl_dash_audio_item struct {
  Quality uint   `json:"id"`
  BaseUrl string `json:"baseUrl"`
}

type API_playurl_dash_video_item struct {
  Quality uint   `json:"id"`
  BaseUrl string `json:"baseUrl"`
}

type API_playurl_dash struct {
  Duration uint64                        `json:"duration"`
  Video    []API_playurl_dash_video_item `json:"video"`
  Audio    []API_playurl_dash_audio_item `json:"audio"`
}

type API_playurl_durl_item struct {
  Url string `json:"url"`
}

type API_playurl_durl []API_playurl_durl_item

type API_playurl_data struct {
  Quality    int               `json:"quality"`
  Format     string            `json:"format"`
  Timelength int               `json:"timelength"`
  Durl       *API_playurl_durl `json:"durl"`
  Dash       *API_playurl_dash `json:"dash"`
}

type API_view_result struct {
  Code int           `json:"code"`
  TTL  int           `json:"ttl"`
  Msg  string        `json:"message"`
  Data API_view_data `json:"data"`
}

//type API_playlist_result struct {
//  Code int           `json:"code"`
//  TTL  int           `json:"ttl"`
//  Msg  string        `json:"message"`
//  Data API_view_data `json:"data"`
//}

type API_playurl_result struct {
  Code int              `json:"code"`
  TTL  int              `json:"ttl"`
  Msg  string           `json:"message"`
  Data API_playurl_data `json:"data"`
}

var quality map[int]string

//   "0":"自动",
//   "15":"流畅 360P",
//   "16":"流畅 360P",
//   "32":"清晰 480P",
//   "48":"高清 720P",
//   "64":"高清 720P",
//   "74":"高清 720P60",
//   "80":"高清 1080P",
//   "112":"高清 1080P+",
//   "116":"高清 1080P60",
//   "120":"超清 4K"
func SimpleGET(url string) []byte {
  resp, err := http.Get(url)
  if err != nil {
    return nil

  }
  defer resp.Body.Close()

  body, _ := ioutil.ReadAll(resp.Body)
  return body
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

func QueryPlayurl(avid uint64, c CInfoLite, group *sync.WaitGroup) CInfo {
  var info CInfo

  header := make(http.Header)
  header.Add("Referer", "https://www.bilibili.com")
  header.Add("Cookie", "SESSDATA=576d39c8%2C1601646951%2Cd08bf*41")
  header.Add("Cache-Control", "no-cache")
  header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/73.0.3683.103 Safari/537.36")

  playurlReq, _ := http.NewRequest("GET", fmt.Sprintf("https://api.bilibili.com/x/player/playurl?avid=%v&cid=%v&bvid=&qn=120&type=&otype=json&fnver=0&fnval=16&session=aa7e4f4b55270fec8d9bd99db95c94f6", avid, c.Id), nil)
  copy_header(&playurlReq.Header, &header)

  client := http.Client{}
  resp, _ := client.Do(playurlReq)
  if resp == nil {
    fmt.Println("????")
  }
  defer resp.Body.Close()

  body, _ := ioutil.ReadAll(resp.Body)
  var result API_playurl_result
  json.Unmarshal(body, &result)

  if result.Data.Durl != nil {
    for i := 0; i < len(*result.Data.Durl); i++ {
      fmt.Println((*result.Data.Durl)[i].Url)
      group.Add(1)
      filename := get_filename_from_url((*result.Data.Durl)[i].Url)
      videoReq, _ := http.NewRequest("GET", (*result.Data.Durl)[i].Url, nil)
      copy_header(&videoReq.Header, &header)
      dl(*videoReq, filename, group)
    }
  }
  if result.Data.Dash != nil {
    fmt.Println(result.Data.Dash.Video[0].BaseUrl)
    fmt.Println(result.Data.Dash.Audio[0].BaseUrl)

    videoReq, _ := http.NewRequest("GET", result.Data.Dash.Video[0].BaseUrl, nil)
    audioReq, _ := http.NewRequest("GET", result.Data.Dash.Audio[0].BaseUrl, nil)
    copy_header(&videoReq.Header, &header)
    copy_header(&audioReq.Header, &header)
    videoName := get_filename_from_url(result.Data.Dash.Video[0].BaseUrl)
    audioName := get_filename_from_url(result.Data.Dash.Audio[0].BaseUrl)

    group.Add(2)
    dl(*videoReq, videoName, group)
    dl(*audioReq, audioName, group)
  }
  return info
}

func QueryAvInfo(avid string) API_view_result {
  body := SimpleGET("https://api.bilibili.com/x/web-interface/view?aid=" + avid)

  var result API_view_result
  //fmt.Println(string(body))
  json.Unmarshal([]byte(body), &result)
  return result
}

func QueryBvInfo(bvid string) API_view_result {
  body := SimpleGET("https://api.bilibili.com/x/web-interface/view?bvid=" + bvid)

  var result API_view_result
  //fmt.Println(string(body))
  json.Unmarshal([]byte(body), &result)
  return result
}

func AutoDownload(url string, group *sync.WaitGroup) bool {
  var result API_view_result

  regexAVID, _ := regexp.Compile(`video/av([0-9]+)`)
  regexBVID, _ := regexp.Compile(`video/(BV[a-zA-Z0-9]+)`)
  matchsAVID := regexAVID.FindStringSubmatch(url)
  matchsBVID := regexBVID.FindStringSubmatch(url)

  if matchsAVID != nil {
    result = QueryAvInfo(matchsAVID[1])
  } else if matchsBVID != nil {
    result = QueryBvInfo(matchsBVID[1])
  } else {
    return false
  }

  for i := 0; i < len(result.Data.Pages); i++ {
    fmt.Println(result.Data.Pages[i].Part)
    QueryPlayurl(result.Data.Id, result.Data.Pages[i], group)
  }

  return true
}
