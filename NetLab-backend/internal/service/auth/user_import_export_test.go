package auth

import (
	"bytes"
	"testing"

	"github.com/xuri/excelize/v2"
	"go.uber.org/zap"

	"netlab-backend/pkg/i18n"
)

// initTestI18N 加载真实消息文件，使表头本地化断言针对实际文案。
// i18n.Init 内部为 sync.Once，多次调用安全。
func initTestI18N(t *testing.T) {
	t.Helper()
	if err := i18n.Init("../../../messages/en-US.json", "../../../messages/zh-CN.json"); err != nil {
		t.Fatalf("init i18n: %v", err)
	}
}

func TestNormalizeUserIDs(t *testing.T) {
	ids, appErr := normalizeUserIDs([]string{" 1 ", "2", "1"})
	if appErr != nil {
		t.Fatalf("unexpected error: %v", appErr)
	}
	if len(ids) != 2 || ids[0] != "1" || ids[1] != "2" {
		t.Fatalf("unexpected ids: %v", ids)
	}
	if _, appErr := normalizeUserIDs([]string{"abc"}); appErr == nil {
		t.Fatal("expected error for non-numeric id")
	}
}

func TestLocalizedExportHeaders(t *testing.T) {
	initTestI18N(t)
	en := localizedHeaders("en-US", exportHeaderKeys)
	if en[0] != "Username" || en[4] != "2FA Enabled" {
		t.Fatalf("unexpected en headers: %v", en)
	}
	zh := localizedHeaders("zh-CN", exportHeaderKeys)
	if zh[0] != "用户名" || zh[6] != "创建时间" {
		t.Fatalf("unexpected zh headers: %v", zh)
	}
}

// TestBuildImportTemplateRoundTrip 验证两种 locale 生成的模板都能被导入端
// 的表头识别逻辑解析（即"导出/模板可直接回导"约定）。
func TestBuildImportTemplateRoundTrip(t *testing.T) {
	initTestI18N(t)
	svc := NewUserImportExportService(nil, zap.NewNop())

	for _, locale := range []string{"zh-CN", "en-US"} {
		t.Run(locale, func(t *testing.T) {
			data, appErr := svc.BuildImportTemplate(locale)
			if appErr != nil {
				t.Fatalf("build template: %v", appErr)
			}
			file, err := excelize.OpenReader(bytes.NewReader(data))
			if err != nil {
				t.Fatalf("open generated template: %v", err)
			}
			defer file.Close()

			rows, err := file.GetRows(file.GetSheetName(0))
			if err != nil {
				t.Fatalf("read rows: %v", err)
			}
			if len(rows) != 3 {
				t.Fatalf("expected header + 2 example rows, got %d rows", len(rows))
			}
			if len(rows[0]) != len(templateHeaderKeys) {
				t.Fatalf("expected %d template headers, got %v", len(templateHeaderKeys), rows[0])
			}
		})
	}

	// 表头确实按 locale 本地化。
	zhData, appErr := svc.BuildImportTemplate("zh-CN")
	if appErr != nil {
		t.Fatalf("build zh template: %v", appErr)
	}
	zhFile, err := excelize.OpenReader(bytes.NewReader(zhData))
	if err != nil {
		t.Fatalf("open zh template: %v", err)
	}
	defer zhFile.Close()
	zhRows, err := zhFile.GetRows(zhFile.GetSheetName(0))
	if err != nil {
		t.Fatalf("read zh rows: %v", err)
	}
	if zhRows[0][0] != "用户名" {
		t.Fatalf("expected zh header 用户名, got %q", zhRows[0][0])
	}
}
