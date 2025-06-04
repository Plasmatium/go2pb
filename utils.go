package main

import (
	"regexp"
	"strings"
)

var re1 = regexp.MustCompile(`([a-z0-9])([A-Z])`)
var re2 = regexp.MustCompile(`([A-Z])([A-Z][a-z])`)

// ToSnakeCase 将驼峰或帕斯卡命名转换为 snake_case
func ToSnakeCase(str string) string {
	// 匹配小写后跟大写字母的情况，例如 "camelCase" -> "camel_Case"
	str = re1.ReplaceAllString(str, "${1}_${2}")

	// 匹配大写字母后跟小写字母的情况（用于处理全大写缩写词后的字符），例如 "HTTPClient" -> "HTTP_Client"
	str = re2.ReplaceAllString(str, "${1}_${2}")

	// 转换为小写
	return strings.ToLower(str)
}
