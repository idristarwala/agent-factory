package tools

import (
	"net/url"
	"strings"
)

// AppendQuery adds parameters that are not already used in the path template as query values.
// Completes the request building for issue #12.
func AppendQuery(urlStr, template string, params map[string]string) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return urlStr
	}
	q := u.Query()
	for k, v := range params {
		if !strings.Contains(template, "{"+k+"}") {
			q.Set(k, v)
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}
