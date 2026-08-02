// Package shared は収集コマンドが共通で使う小道具を置く。
package shared

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// 相手先に誰が叩いているか分かるようにしておく。RSS と公式 API しか叩かないが、
// 素性の分からない UA で巡回するのは避ける。
const UserAgent = "pechka-tech-feed/1.0 (personal home media; +https://github.com/k-wa-wa/pechka)"

const requestTimeout = 20 * time.Second

func NewHTTPClient() *http.Client {
	return &http.Client{Timeout: requestTimeout}
}

// GetJSON は out へ JSON をデコードする。
func GetJSON(client *http.Client, url string, out any) error {
	res, err := Get(client, url)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	return json.NewDecoder(res.Body).Decode(out)
}

// Get は UA を付けて GET し、2xx 以外はエラーにする。
func Get(client *http.Client, url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		res.Body.Close()
		return nil, fmt.Errorf("HTTP %d", res.StatusCode)
	}
	return res, nil
}

// StripTags は RSS の description に混ざる HTML を落とす。整形が目的ではなく、
// プロンプトへ素の文章だけ渡すための粗い処理である。
func StripTags(s string) string {
	var b strings.Builder
	depth := 0
	for _, r := range s {
		switch {
		case r == '<':
			depth++
		case r == '>':
			if depth > 0 {
				depth--
			}
		case depth == 0:
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// Truncate は rune 単位で詰める。バイト単位で切ると日本語が壊れる。
func Truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
