package common

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func CreateFile(fileName string, in interface{}) error {

	var file *os.File
	var err_f error

	b, err := DictToYAML(in)

	if err != nil {
		fmt.Println("file.go.CreateFile_任务数据解析失败：", err)
		return err
	}

	content := string(b)

	if Exists(fileName) {
		file, err_f = os.OpenFile(fileName, os.O_WRONLY|os.O_TRUNC, 0666)
		if err_f != nil {
			fmt.Println("打开文件错误：", err_f)
			return err_f
		}
	} else {
		file, err_f = os.Create(fileName)
		if err_f != nil {
			fmt.Println("创建失败", err_f)
			return err_f
		}
	}
	defer file.Close()
	//写入文件
	Writefile(file, content)

	return nil
}

func Writefile(file io.Writer, content string) error {
	_, err := io.WriteString(file, content)
	if err != nil {
		fmt.Println("写入错误：", err)
		return err
	}
	fmt.Println("写入成功")
	return nil
}

// 判断所给路径文件/文件夹是否存在
func Exists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) { //判断一个文件是否已经存在
			return os.IsExist(err) //文件存在返回true
		}
		return false
	}
	return true
}

// 判断文件是否存在
func FileExists(filePath string) (bool, error) {
	_, err := os.Stat(filePath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, err
	}
	return false, err
}

// 删除文件
func Delconfigfile(fileName string) error {
	if Exists(fileName) {
		err := os.Remove(fileName)
		if err != nil {
			fmt.Println("删除文件失败")
			return err
		}
		fmt.Printf("%v已删除", fileName)
		return nil
	} else {
		fmt.Println("文件不存在")
		return nil
	}
}

func DirPath(path string) string {

	var cpath string

	if !filepath.IsAbs(path) {
		_path, _err := filepath.Abs(path)
		if _err != nil {
			return cpath
		}
		cpath = _path
	} else {
		cpath = path
	}

	return filepath.Dir(cpath)
}
