package GiteeAPI

type Asset struct {
  BrowserDownloadUrl string `json:"browser_download_url"`
}

type Author struct {
  Id   int    `json:"id"`
  Name string `json:"name"`
}

type Latest struct {
  Id         int     `json:"id"`
  TagName    string  `json:"tag_name"`
  Prerelease bool    `json:"prerelease"`
  Author     Author  `json:"author"`
  Assets     []Asset `json:"assets"`
}
