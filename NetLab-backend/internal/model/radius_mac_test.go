package model

import "testing"

// TestNormalizeMacList 验证 MAC 列表归一化：分隔符兼容、大小写与 '-' 归一、
// 空项丢弃、按首次出现顺序去重。
func TestNormalizeMacList(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{"空输入", "", ""},
		{"仅分隔符", " ,;\n\t ", ""},
		{"单个 MAC 原样归一", "AA-BB-CC-DD-EE-FF", "aa:bb:cc:dd:ee:ff"},
		{"逗号分隔多个", "aa:bb:cc:dd:ee:ff,11:22:33:44:55:66", "aa:bb:cc:dd:ee:ff,11:22:33:44:55:66"},
		{"换行分隔", "aa:bb:cc:dd:ee:ff\n11:22:33:44:55:66", "aa:bb:cc:dd:ee:ff,11:22:33:44:55:66"},
		{"分号与空格分隔", "aa:bb:cc:dd:ee:ff; 11:22:33:44:55:66", "aa:bb:cc:dd:ee:ff,11:22:33:44:55:66"},
		{"制表符与 CRLF 分隔", "aa:bb:cc:dd:ee:ff\r\n\t11:22:33:44:55:66", "aa:bb:cc:dd:ee:ff,11:22:33:44:55:66"},
		{"混合分隔与大小写", " AA-BB-CC-DD-EE-FF\naa:bb:cc:dd:ee:ff;11-22-33-44-55-66 ", "aa:bb:cc:dd:ee:ff,11:22:33:44:55:66"},
		{"重复项去重保序", "11:22:33:44:55:66,aa:bb:cc:dd:ee:ff,11:22:33:44:55:66", "11:22:33:44:55:66,aa:bb:cc:dd:ee:ff"},
		{"大小写不同视为重复", "AA:BB:CC:DD:EE:FF,aa:bb:cc:dd:ee:ff", "aa:bb:cc:dd:ee:ff"},
		{"空项丢弃", ",aa:bb:cc:dd:ee:ff,,", "aa:bb:cc:dd:ee:ff"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := NormalizeMacList(tc.raw); got != tc.want {
				t.Errorf("NormalizeMacList(%q) = %q，期望 %q", tc.raw, got, tc.want)
			}
		})
	}
}

// TestMacListContains 验证列表包含判断：归一化比较、空值边界。
func TestMacListContains(t *testing.T) {
	cases := []struct {
		name string
		list string
		mac  string
		want bool
	}{
		{"空列表", "", "aa:bb:cc:dd:ee:ff", false},
		{"空 MAC", "aa:bb:cc:dd:ee:ff", "", false},
		{"单元素命中", "aa:bb:cc:dd:ee:ff", "aa:bb:cc:dd:ee:ff", true},
		{"列表首元素命中", "aa:bb:cc:dd:ee:ff,11:22:33:44:55:66", "aa:bb:cc:dd:ee:ff", true},
		{"列表尾元素命中", "aa:bb:cc:dd:ee:ff,11:22:33:44:55:66", "11:22:33:44:55:66", true},
		{"未命中", "aa:bb:cc:dd:ee:ff,11:22:33:44:55:66", "22:33:44:55:66:77", false},
		{"大小写不敏感", "AA:BB:CC:DD:EE:FF", "aa:bb:cc:dd:ee:ff", true},
		{"'-' 分隔符归一", "aa:bb:cc:dd:ee:ff", "AA-BB-CC-DD-EE-FF", true},
		{"查询带空白", "aa:bb:cc:dd:ee:ff", " aa:bb:cc:dd:ee:ff ", true},
		{"列表项带空白", "aa:bb:cc:dd:ee:ff, 11:22:33:44:55:66", "11:22:33:44:55:66", true},
		{"换行分隔列表", "aa:bb:cc:dd:ee:ff\n11:22:33:44:55:66", "11:22:33:44:55:66", true},
		{"部分前缀不匹配", "aa:bb:cc:dd:ee:ff", "aa:bb:cc:dd:ee", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := MacListContains(tc.list, tc.mac); got != tc.want {
				t.Errorf("MacListContains(%q, %q) = %v，期望 %v", tc.list, tc.mac, got, tc.want)
			}
		})
	}
}
