package utils

import (
	"os"
	"path/filepath"
)

// 转换为绝对路径
func ParseAbsolutePath(path string) (string, error) {
	str := []rune(path)
	firstKey := string(str[:1])
	if firstKey == "~" {
		home, err := os.UserHomeDir()
		if nil != err {
			return "", err
		}
		return filepath.Join(home, string(str[1:])), nil
	} else if firstKey == "/" {
		return path, nil
	} else {
		p, _ := filepath.Abs(filepath.Dir(os.Args[0]))
		return p + "/" + path, nil
	}
}

// 查找文件是否存在
func FindHomeFile(path string) (bool, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return false, err
	}
	filePath := filepath.Join(home, path)
	return Exists(filePath), nil
}

// 判断所给路径文件/文件夹是否存在
func Exists(path string) bool {
	_, err := os.Stat(path)    //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

// 判断所给路径是否为文件夹
func IsDir(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return s.IsDir()
}

// 判断所给路径是否为文件
func IsFile(path string) bool {
	return !IsDir(path)
}
