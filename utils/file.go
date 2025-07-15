package utils

import (
	"os"
	"path/filepath"
)

// 确保目录存在
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// 检查文件是否存在
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// 获取文件目录
func GetFileDir(path string) string {
	return filepath.Dir(path)
}

// 创建文件及其目录
func CreateFile(path string) (*os.File, error) {
	dir := GetFileDir(path)
	err := EnsureDir(dir)
	if err != nil {
		return nil, err
	}

	return os.Create(path)
}
