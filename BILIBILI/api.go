package BILIBILI

import (
  "encoding/json"
  "fmt"
  "io/ioutil"
  "net/http"
  "net/url"
)

//https://api.bilibili.com/x/space/channel/video?mid=37877654&cid=89742&pn=1&ps=30&order=0&jsonp=jsonp&callback=__jp5
type API_space_channel_video_result_data_list_archive struct {
  AVID  uint64 `json:"aid"`
  BVID  string `json:"bvid"`
  Title string `json:"title"`
}

type API_space_channel_video_result_data_list struct {
  Archives []API_space_channel_video_result_data_list_archive `json:"archives"`
}

type API_space_channel_video_result_data_page struct {
  Count uint64 `json:"count"`
  Num   uint64 `json:"num"`
  Size  uint64 `json:"size"`
}

type API_space_channel_video_result_data struct {
  List API_space_channel_video_result_data_list `json:"list"`
  Page API_space_channel_video_result_data_page `json:"page"`
}

type API_space_channel_video_result struct {
  Code int                                 `json:"code"`
  TTL  int                                 `json:"ttl"`
  Msg  string                              `json:"message"`
  Data API_space_channel_video_result_data `json:"data"`
}

type API_web_interface_view_page struct {
  Id   uint64 `json:"cid"`
  Page int    `json:"page"`
  Part string `json:"part"`
}

type API_web_interface_view_data struct {
  AVID  uint64                        `json:"aid"`
  BVID  string                        `json:"bvid"`
  Title string                        `json:"title"`
  Pages []API_web_interface_view_page `json:"pages"`
}

type API_playurl_dash_audio_item struct {
  Quality uint   `json:"id"`
  BaseUrl string `json:"baseUrl"`
}

type API_playurl_dash_video_item struct {
  Quality uint   `json:"id"`
  BaseUrl string `json:"baseUrl"`
}

type API_play_playurl_dash struct {
  Duration uint64                        `json:"duration"`
  Video    []API_playurl_dash_video_item `json:"video"`
  Audio    []API_playurl_dash_audio_item `json:"audio"`
}

type API_playurl_durl_item struct {
  Url string `json:"url"`
}

//web-interface/view
type API_web_interface_view_result struct {
  Code int                         `json:"code"`
  TTL  int                         `json:"ttl"`
  Msg  string                      `json:"message"`
  Data API_web_interface_view_data `json:"data"`
}

type API_play_playurl_durl []API_playurl_durl_item

type API_play_playurl_data struct {
  Quality    int                    `json:"quality"`
  Format     string                 `json:"format"`
  Timelength int                    `json:"timelength"`
  Durl       *API_play_playurl_durl `json:"durl"`
  Dash       *API_play_playurl_dash `json:"dash"`
}

type API_play_playurl_result struct {
  Code int                   `json:"code"`
  TTL  int                   `json:"ttl"`
  Msg  string                `json:"message"`
  Data API_play_playurl_data `json:"data"`
}

func GetQualityName(quality int) string {
  switch quality {
  case 0:
    return "自动"
  case 15:
    return "流畅 360P"
  case 16:
    return "流畅 360P"
  case 32:
    return "清晰 480P"
  case 48:
    return "高清 720P"
  case 64:
    return "高清 720P"
  case 74:
    return "高清 720P60"
  case 80:
    return "高清 1080P"
  case 112:
    return "高清 1080P+"
  case 116:
    return "高清 1080P60"
  case 120:
    return "超清 4K"
  }
  return ""
}

func API_play_playurl(avid uint64, bvid string, cid uint64) *API_play_playurl_result {
  var result *API_play_playurl_result

  args := url.Values{}
  args.Add("avid", fmt.Sprintf("%v", avid))
  args.Add("bvid", fmt.Sprintf("%v", bvid))
  args.Add("cid", fmt.Sprintf("%v", cid))
  args.Add("qn", "120")
  args.Add("type", "")
  args.Add("otype", "json")
  args.Add("fnver", "0")
  args.Add("fnval", "16")
  //args.Add("session", "aa7e4f4b55270fec8d9bd99db95c94f6")
  if body := SimpleGET(fmt.Sprintf("https://api.bilibili.com/x/player/playurl?%v", args.Encode())); body != nil {
    result = &API_play_playurl_result{}
    json.Unmarshal([]byte(body), result)
  }
  return result
}

func API_web_interface_view(avid *uint64, bvid *string, cid *uint64) *API_web_interface_view_result {
  var result *API_web_interface_view_result

  args := url.Values{}
  if avid != nil {
    args.Add("aid", fmt.Sprintf("%v", *avid))
  }
  if bvid != nil {
    args.Add("bvid", fmt.Sprintf("%v", *bvid))
  }
  if cid != nil {
    args.Add("aid", fmt.Sprintf("%v", *cid))
  }

  if body := SimpleGET(fmt.Sprintf("https://api.bilibili.com/x/web-interface/view?%v", args.Encode())); body != nil {
    result = &API_web_interface_view_result{}
    json.Unmarshal([]byte(body), result)
  }
  return result
}

func API_space_channel_video(mid uint64, cid uint64) *API_space_channel_video_result {
  var result *API_space_channel_video_result
  args := url.Values{}
  args.Add("mid", fmt.Sprintf("%v", mid))
  args.Add("cid", fmt.Sprintf("%v", cid))
  args.Add("pn", "1")
  args.Add("ps", "30")
  args.Add("order", "0")

  if body := SimpleGET(fmt.Sprintf("http://api.bilibili.com/x/space/channel/video?%v", args.Encode())); body != nil {
    result = &API_space_channel_video_result{}
    json.Unmarshal([]byte(body), result)
  }
  return result
}

func SimpleGET(url string) []byte {
  req, _ := http.NewRequest("GET", url, nil)
  req.Header.Add("Cookie", Cookie)
  req.Header.Add("Referer", Referer)
  req.Header.Add("User-Agent", UserAgent)
  client := http.Client{}
  resp, err := client.Do(req)

  if err != nil {
    return nil
  }
  defer resp.Body.Close()

  body, _ := ioutil.ReadAll(resp.Body)
  return body
}
