package model

import "strings"

// macSeparators 是 MAC 列表输入支持的分隔符（换行/逗号/分号/空格/制表符）。
func macSeparators(r rune) bool {
	return r == '\n' || r == '\r' || r == ',' || r == ';' || r == ' ' || r == '\t'
}

// normalizeMac 归一化单个 MAC：去空白、'-' 转为 ':'、转小写。
func normalizeMac(mac string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(mac), "-", ":"))
}

// NormalizeMacList 把用户输入的 MAC 列表（支持换行/逗号/分号/空格分隔，
// 兼容 '-' 分隔符与大小写混杂）归一化为单行逗号分隔、小写冒号格式的
// 存储形态；空项被丢弃，重复项按首次出现顺序去重。
func NormalizeMacList(raw string) string {
	seen := make(map[string]struct{})
	out := make([]string, 0, 4)
	for _, token := range strings.FieldsFunc(raw, macSeparators) {
		mac := normalizeMac(token)
		if mac == "" {
			continue
		}
		if _, ok := seen[mac]; ok {
			continue
		}
		seen[mac] = struct{}{}
		out = append(out, mac)
	}
	return strings.Join(out, ",")
}

// MacListContains 判断存储形态的 MAC 列表（逗号/换行分隔）是否包含指定
// MAC；两侧均按 '-'→':'、小写、去空白归一后比较。mac 或 list 为空时返回 false。
func MacListContains(list, mac string) bool {
	mac = normalizeMac(mac)
	if mac == "" || strings.TrimSpace(list) == "" {
		return false
	}
	for _, item := range strings.FieldsFunc(list, func(r rune) bool { return r == ',' || r == '\n' || r == '\r' }) {
		if normalizeMac(item) == mac {
			return true
		}
	}
	return false
}
