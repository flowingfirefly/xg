package BILIBILI

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const (
	Cookie    string = "SESSDATA=820b28c1%2C1601742439%2C12e8d*41" // 此cookie随时可能过期,过期后将会导致视频可下载清晰度下降.
	UserAgent string = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/73.0.3683.103 Safari/537.36"
	Referer   string = "https://www.bilibili.com"
)

type HttpFileInfo struct {
	Url  string
	Size uint64
	Name string
}

type M4S struct {
	Video HttpFileInfo
	Audio HttpFileInfo
}

type FLV []HttpFileInfo

type CInfo struct {
	AVID        uint64
	BVID        string
	CID         uint64
	Part        string
	Title       string
	M4S         *M4S
	FLV         *FLV
	QualityName string
}

func query_size(req *http.Request) uint64 {
	client := http.Client{}
	resp, _ := client.Do(req)
	if resp == nil {
		return 0
	}
	resp.Body.Close()
	lenstr := resp.Header.Get("Content-Length")
	lenstr = lenstr
	u64, _ := strconv.ParseUint(lenstr, 10, 64)
	return u64
}

func get_filename_from_url(url string) string {
	regex, _ := regexp.Compile(`/([-\.a-z0-9]+)\?`)
	matchs := regex.FindStringSubmatch(url)
	if len(matchs) != 2 {
		return ""
	}
	return matchs[1]
}

func QueryPlayurl(avid uint64, bvid string, cid uint64) CInfo {
	var info CInfo

	result := API_play_playurl(avid, bvid, cid)

	info.QualityName = GetQualityName(result.Data.Quality)

	if result.Data.Durl != nil {
		info.FLV = &FLV{}
		for i := 0; i < len(*result.Data.Durl); i++ {
			*info.FLV = append(*info.FLV, HttpFileInfo{
				Url:  (*result.Data.Durl)[i].Url,
				Size: (*result.Data.Durl)[i].Size,
				Name: get_filename_from_url((*result.Data.Durl)[i].Url)})
		}
	}
	if result.Data.Dash != nil {
		info.M4S = &M4S{}

		info.M4S.Audio = HttpFileInfo{
			Url: result.Data.Dash.Audio[0].BaseUrl,

			Name: get_filename_from_url(result.Data.Dash.Audio[0].BaseUrl)}
		info.M4S.Video = HttpFileInfo{
			Url:  result.Data.Dash.Video[0].BaseUrl,
			Name: get_filename_from_url(result.Data.Dash.Video[0].BaseUrl)}

		req1, _ := http.NewRequest("GET", info.M4S.Audio.Url, nil)
		req2, _ := http.NewRequest("GET", info.M4S.Video.Url, nil)
		req1.Header.Add("Referer", Referer)
		req1.Header.Add("User-Agent", UserAgent)
		req2.Header.Add("Referer", Referer)
		req2.Header.Add("User-Agent", UserAgent)

		info.M4S.Audio.Size = query_size(req1)
		info.M4S.Video.Size = query_size(req2)
		//info.Requests = append(info.Requests, req1, req2)
	}
	return info
}

func (this *CInfo) MakeHeader() http.Header {
	header := http.Header{}
	header.Add("Referer", Referer)
	header.Add("User-Agent", UserAgent)
	return header
}

func cmd_exec(cmd *exec.Cmd) error {
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println(stderr.String())
		return fmt.Errorf("%s\n%s", err, stderr.String())
	}

	return nil
}

func (this *CInfo) Completed() {

	dir, _ := os.Getwd()

	get_dirname := func() string {
		dir := this.Title
		// windows 平台下文件名不允许出现这些字符
		var chars []string = []string{"\\", "/", ":", "*", "?", "<", ">", "|"}
		for _, v := range chars {
			dir = strings.ReplaceAll(dir, v, "_")
		}
		return dir
	}
	get_filename := func() string {
		var name string
		var suffix string
		if this.FLV != nil {
			suffix = ".flv"
		}
		if this.M4S != nil {
			suffix = ".mp4"
		}

		if this.Part != "" {
			name = this.Part
		} else {
			name = fmt.Sprintf("%v", this.CID)
		}

		// windows 平台下文件名不允许出现这些字符
		var chars []string = []string{"\\", "/", ":", "*", "?", "<", ">", "|"}
		for _, v := range chars {
			name = strings.ReplaceAll(name, v, "_")
		}
		return name + suffix
	}

	outputDir := get_dirname()
	outputName := get_filename()
	outputPath := outputDir + "/" + outputName
	os.Mkdir(outputDir, 0700)

	_, err := os.Stat(outputPath)
	if err == nil {
		return
	}
	println(this.Title + " 开始合并")

	println("输出路径为: " + outputPath)

	if this.FLV != nil {
		concatPath := ".concat"
		concatFile, _ := os.OpenFile(concatPath, os.O_CREATE|os.O_WRONLY, 0600)
		for _, v := range *this.FLV {
			concatFile.Write([]byte(fmt.Sprintf("file '%s'\n", "tmp/"+v.Name)))
		}
		concatFile.Close()

		cmd := exec.Command(
			dir+"/ffmpeg",
			"-y",
			"-f", "concat",
			"-safe",
			"-1",
			"-i", concatPath,
			"-c",
			"copy",
			"-bsf:a", "aac_adtstoasc",
			outputPath,
		)
		cmd_exec(cmd)
		os.Remove(concatPath)
	}

	if this.M4S != nil {
		cmd := exec.Command(
			dir+"/ffmpeg",
			"-y",
			"-i", "tmp/"+this.M4S.Video.Name,
			"-i", "tmp/"+this.M4S.Audio.Name,
			"-c:v", "copy",
			"-c:a", "copy",
			outputPath)
		cmd_exec(cmd)
	}
	println(this.Title + " 合并完成")
}

func AutoParse(url string) []CInfo {

	regexAVID, _ := regexp.Compile(`video/av([0-9]+)`)
	regexBVID, _ := regexp.Compile(`video/(BV[a-zA-Z0-9]+)`)
	regexChannel, _ := regexp.Compile(`space.bilibili.com/(\d+)/channel/detail\?cid=(\d+)$`)

	var infos []CInfo
	if matchs := regexAVID.FindStringSubmatch(url); matchs != nil && len(matchs) == 2 {
		if avid, err := strconv.ParseUint(matchs[1], 10, 64); err == nil {
			result := API_web_interface_view(&avid, nil, nil)
			fmt.Println(result.Data.Title)
			fmt.Println(result.Data.Owner.Name)
			for i := 0; i < len(result.Data.Pages); i++ {
				info := QueryPlayurl(result.Data.AVID, result.Data.BVID, result.Data.Pages[i].Id)
				info.Title = result.Data.Title
				info.Part = result.Data.Pages[i].Part
				info.AVID = result.Data.AVID
				info.BVID = result.Data.BVID
				info.CID = result.Data.Pages[i].Id
				fmt.Println(info.AVID, info.BVID, info.CID, info.QualityName, info.Part)
				infos = append(infos, info)
			}
		} else {
			return nil
		}
	} else if matchs := regexBVID.FindStringSubmatch(url); matchs != nil && len(matchs) == 2 {
		bvid := matchs[1]
		result := API_web_interface_view(nil, &bvid, nil)
		fmt.Println(result.Data.Title)
		fmt.Println(result.Data.Owner.Name)
		for i := 0; i < len(result.Data.Pages); i++ {
			info := QueryPlayurl(result.Data.AVID, result.Data.BVID, result.Data.Pages[i].Id)
			info.Title = result.Data.Title
			info.Part = result.Data.Pages[i].Part
			info.AVID = result.Data.AVID
			info.BVID = result.Data.BVID
			info.CID = result.Data.Pages[i].Id
			fmt.Println(info.AVID, info.BVID, info.CID, info.QualityName, info.Part)
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
				info.Title = result.Data.Title
				info.Part = result.Data.Pages[i].Part
				info.AVID = result.Data.AVID
				info.BVID = result.Data.BVID
				info.CID = result.Data.Pages[i].Id
				fmt.Println(info.AVID, info.BVID, info.CID, info.QualityName, info.Part)
				infos = append(infos, info)
			}
		}
	} else {
		return nil
	}

	return infos
}
