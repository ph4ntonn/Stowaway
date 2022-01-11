package util

import "golang.org/x/text/encoding/simplifiedchinese"

func ConvertStr2GBK(str string) string {
	//将utf-8编码的字符串转换为GBK编码
	ret, err := simplifiedchinese.GBK.NewEncoder().String(str)
	if err != nil {
		ret = str
	}
	return ret //如果转换失败返回原始字符串
}

func ConvertGBK2Str(gbkStr string) string {
	//将GBK编码的字符串转换为utf-8编码
	ret, err := simplifiedchinese.GBK.NewDecoder().String(gbkStr)
	if err != nil {
		ret = gbkStr
	}
	return ret //如果转换失败返回原始字符串

}
