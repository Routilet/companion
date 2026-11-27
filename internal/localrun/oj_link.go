package localrun

import (
	"errors"
	"net/url"
)

// OpenOJLink opens a validated canonical Accoding URL in the default browser.
func OpenOJLink(link string) error {
	u, e := url.Parse(link)
	if e != nil || u.Scheme != "https" || u.User != nil || u.RawQuery != "" || (u.Host != "accoding.buaa.edu.cn" && u.Host != "accoding.buaa.edu.cn:4000") {
		return errors.New("无效原题链接")
	}
	return launchDefault(link, true)
}
