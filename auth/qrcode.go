package auth

import (
	"TianHe-API/utils"
	"fmt"
	"os"

	"github.com/skip2/go-qrcode"
)

// 二维码登录
func QRCodeLogin() error {
	// 获取二维码
	qrcodeKey, qrcodeURL, err := getLoginToken()
	if err != nil {
		return err
	}

	// 生成二维码
	err = generateQRCode(qrcodeURL)
	if err != nil {
		return err
	}

	utils.Logger.Info("请使用手机B站扫描二维码登录")
	utils.Logger.Info("二维码URL: " + qrcodeURL)

	// 轮询登录状态
	cookie, err := pollLogin(qrcodeKey)
	if err != nil {
		return err
	}

	// 保存cookie
	err = os.MkdirAll("config", 0755)
	if err != nil {
		return err
	}

	err = cookie.Save("config/cookie.json")
	if err != nil {
		return err
	}

	// 显示用户信息
	userInfo, err := GetUserInfo()
	if err == nil {
		utils.Logger.Infof("登录用户: %s (UID: %v)", userInfo["uname"], userInfo["uid"])
	}

	return nil
}

// 生成二维码文件
func generateQRCode(url string) error {
	qr, err := qrcode.New(url, qrcode.Medium)
	if err != nil {
		return err
	}

	// 保存为PNG文件
	err = qr.WriteFile(256, "qrcode.png")
	if err != nil {
		return err
	}

	// 在终端显示二维码
	fmt.Println(qr.ToSmallString(false))

	return nil
}
