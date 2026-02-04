package handler

type Channel struct {
	ID       uint
	CustomID string
	Name     string
	URL      string
	M3U8     string
	Proxy    bool
}

type Config struct {
	Cmd          string
	Args         string
	Cookies      string
	CookiesContent string
	CookiesDomain string
	YtdlTimeout  string
	BaseURL      string
	M3UFilename  string
	ChannelParam string
	TSTimeout    string
}
