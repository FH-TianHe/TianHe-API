package auth

import (
	"TianHe-API/config"
	"TianHe-API/utils"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

const (
	LoginURL = "https://passport.bilibili.com/x/passport-login/web/qrcode/generate"
	PollURL  = "https://passport.bilibili.com/x/passport-login/web/qrcode/poll"
	NavURL   = "https://api.bilibili.com/x/web-interface/nav"
)

var client = &http.Client{
	Timeout: 30 * time.Second,
}

// 检查是否已登录
func IsLoggedIn() bool {
	cookie, err := config.LoadCookie("config/cookie.json")
	if err != nil {
		return false
	}

	// 检查cookie是否过期
	if time.Now().Unix() > cookie.ExpireTime {
		return false
	}

	// 验证cookie有效性
	return validateCookie(cookie)
}

// 验证cookie有效性
func validateCookie(cookie *config.Cookie) bool {
	req, err := http.NewRequest("GET", NavURL, nil)
	if err != nil {
		return false
	}

	req.Header.Set("Cookie", formatCookie(cookie))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	result := gjson.ParseBytes(body)
	return result.Get("code").Int() == 0 && result.Get("data.isLogin").Bool()
}

// 格式化cookie
func formatCookie(cookie *config.Cookie) string {
	return fmt.Sprintf("SESSDATA=%s; bili_jct=%s; DedeUserID=%s; DedeUserID__ckMd5=%s; sid=%s",
		cookie.SESSDATA, cookie.BiliJct, cookie.DedeUserID, cookie.DedeUserID__ckMd5, cookie.Sid)
}

// 获取登录token
func getLoginToken() (string, string, error) {
	req, err := http.NewRequest("GET", LoginURL, nil)
	if err != nil {
		return "", "", err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	result := gjson.ParseBytes(body)
	if result.Get("code").Int() != 0 {
		return "", "", fmt.Errorf("获取登录token失败: %s", result.Get("message").String())
	}

	qrcodeKey := result.Get("data.qrcode_key").String()
	qrcodeURL := result.Get("data.url").String()

	return qrcodeKey, qrcodeURL, nil
}

// 轮询登录状态
func pollLogin(qrcodeKey string) (*config.Cookie, error) {
	data := url.Values{}
	data.Set("qrcode_key", qrcodeKey)

	for {
		req, err := http.NewRequest("POST", PollURL, strings.NewReader(data.Encode()))
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		result := gjson.ParseBytes(body)
		code := result.Get("data.code").Int()

		switch code {
		case 0: // 登录成功
			utils.Logger.Info("登录成功！")
			return parseCookieFromResponse(resp), nil
		case 86101: // 未扫描
			utils.Logger.Info("等待扫描...")
		case 86090: // 已扫描未确认
			utils.Logger.Info("已扫描，等待确认...")
		case 86038: // 二维码已过期
			return nil, fmt.Errorf("二维码已过期")
		default:
			return nil, fmt.Errorf("登录失败: %s", result.Get("message").String())
		}

		time.Sleep(2 * time.Second)
	}
}

// 从响应中解析cookie
func parseCookieFromResponse(resp *http.Response) *config.Cookie {
	cookie := &config.Cookie{
		ExpireTime: time.Now().Add(30 * 24 * time.Hour).Unix(), // 30天后过期
	}

	for _, c := range resp.Cookies() {
		switch c.Name {
		case "SESSDATA":
			cookie.SESSDATA = c.Value
		case "bili_jct":
			cookie.BiliJct = c.Value
		case "DedeUserID":
			cookie.DedeUserID = c.Value
		case "DedeUserID__ckMd5":
			cookie.DedeUserID__ckMd5 = c.Value
		case "sid":
			cookie.Sid = c.Value
		}
	}

	return cookie
}

// 获取当前用户信息
func GetUserInfo() (map[string]interface{}, error) {
	cookie, err := config.LoadCookie("config/cookie.json")
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", NavURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Cookie", formatCookie(cookie))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	result := gjson.ParseBytes(body)
	if result.Get("code").Int() != 0 {
		return nil, fmt.Errorf("获取用户信息失败: %s", result.Get("message").String())
	}

	userInfo := make(map[string]interface{})
	userInfo["uname"] = result.Get("data.uname").String()
	userInfo["uid"] = result.Get("data.mid").Int()
	userInfo["face"] = result.Get("data.face").String()
	userInfo["level"] = result.Get("data.level_info.current_level").Int()

	return userInfo, nil
}

// 获取Cookie字符串
func GetCookieString() string {
	cookie, err := config.LoadCookie("config/cookie.json")
	if err != nil {
		return ""
	}

	return formatCookie(cookie)
}

// 生成WS认证token
func GenerateToken(roomID int) string {
	cookie, err := config.LoadCookie("config/cookie.json")
	if err != nil {
		return ""
	}

	// 使用SESSDATA和roomID生成token
	h := md5.New()
	h.Write([]byte(fmt.Sprintf("%s%d", cookie.SESSDATA, roomID)))
	return hex.EncodeToString(h.Sum(nil))
}
