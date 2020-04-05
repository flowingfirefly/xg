package GithubAPI

type Author struct {
  AvatarUrl string `json:"avatar_url"`
  Id        int    `json:"id"`
  Name      string `json:"login"`
}

type Asset struct {
  BrowserDownloadUrl string `json:"browser_download_url"`
  ContentType        string `json:"content_type"`
  DownloadCount      uint64 `json:"download_count"`
  Name               string `json:"name"`
  Size               uint64 `json:"size"`
  Url                string `json:"url"`
}

type Latest struct {
  Assets     []Asset `json:"assets"`
  Author     Author  `json:"author"`
  Draft      bool    `json:"draft"`
  Prerelease bool    `json:"prerelease"`
  TagName    string  `json:"tag_name"`
  TarballUrl string  `json:"tarball_url"`
  ZipballUrl string  `json:"zipball_url"`
}
