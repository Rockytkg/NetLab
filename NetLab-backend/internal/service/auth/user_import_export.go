package auth

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	"go.uber.org/zap"

	"netlab-backend/internal/model"
	"netlab-backend/internal/repository"
	"netlab-backend/internal/validation"
	"netlab-backend/pkg/apperrors"
	"netlab-backend/pkg/crypto"
	"netlab-backend/pkg/i18n"
)

// UserImportExportService 承载用户 JSON 批量导入与 Excel 导出业务。
type UserImportExportService struct {
	userRepo *repository.UserRepository
	logger   *zap.Logger
}

// NewUserImportExportService 创建用户导入导出服务。
func NewUserImportExportService(userRepo *repository.UserRepository, logger *zap.Logger) *UserImportExportService {
	return &UserImportExportService{userRepo: userRepo, logger: logger}
}

// UserImportRecord 是前端解析表格后提交的一条用户记录。
type UserImportRecord struct {
	Username string
	Nickname string
	Phone    string
	Email    string
	Role     string
	Password string
}

// ImportUsers 从 JSON 记录批量导入用户。表格解析不在后端执行。
func (s *UserImportExportService) ImportUsers(ctx context.Context, records []UserImportRecord) (*ImportSummary, *apperrors.AppError) {
	summary := &ImportSummary{Errors: []string{}}
	for rowNumber, record := range records {
		line := rowNumber + 1
		username, nameErr := validation.NormalizeUsername(record.Username)
		nickname, nicknameErr := validation.NormalizeNickname(record.Nickname)
		phone, phoneErr := validation.NormalizePhone(record.Phone)
		email, emailErr := validation.NormalizeEmail(record.Email)
		if nameErr != nil || nicknameErr != nil || phoneErr != nil || emailErr != nil {
			summary.Errors = append(summary.Errors, fmt.Sprintf("row %d: username, nickname, phone and email are required and valid", line))
			summary.Skipped++
			continue
		}
		if strings.EqualFold(username, "superadmin") {
			summary.Errors = append(summary.Errors, fmt.Sprintf("row %d: superadmin username is reserved", line))
			summary.Skipped++
			continue
		}

		role := strings.TrimSpace(record.Role)
		if role == "" {
			role = string(model.RoleViewer)
		}
		normalizedRole, roleErr := validation.NormalizeRole(role, false)
		if roleErr != nil {
			summary.Errors = append(summary.Errors, fmt.Sprintf("row %d: %s", line, roleErr.Message))
			summary.Skipped++
			continue
		}

		password := strings.TrimSpace(record.Password)
		if password == "" {
			password = username
		}
		if passwordErr := validation.ValidatePassword(password); passwordErr != nil {
			summary.Errors = append(summary.Errors, fmt.Sprintf("row %d: %s", line, passwordErr.Message))
			summary.Skipped++
			continue
		}

		exists, checkErr := s.userRepo.ExistsByUsername(ctx, username)
		if checkErr != nil {
			return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to check username", checkErr)
		}
		if exists {
			summary.Errors = append(summary.Errors, fmt.Sprintf("row %d: username already exists: %s", line, username))
			summary.Skipped++
			continue
		}
		exists, checkErr = s.userRepo.ExistsByPhone(ctx, phone)
		if checkErr != nil {
			return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to check phone", checkErr)
		}
		if exists {
			summary.Errors = append(summary.Errors, fmt.Sprintf("row %d: phone already exists: %s", line, phone))
			summary.Skipped++
			continue
		}
		exists, checkErr = s.userRepo.ExistsByEmail(ctx, email)
		if checkErr != nil {
			return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to check email", checkErr)
		}
		if exists {
			summary.Errors = append(summary.Errors, fmt.Sprintf("row %d: email already exists: %s", line, email))
			summary.Skipped++
			continue
		}

		hash, hashErr := crypto.HashPassword(password)
		if hashErr != nil {
			summary.Errors = append(summary.Errors, fmt.Sprintf("row %d: failed to hash password", line))
			summary.Skipped++
			continue
		}
		now := time.Now()
		user := &model.User{
			Username: username, Nickname: nickname, Phone: phone, Email: email, PasswordHash: hash,
			Role: normalizedRole, Status: model.StatusActive,
			ForcePasswordChange: true, ForceEmailChange: true, PasswordChangedAt: &now,
		}
		if createErr := s.userRepo.Create(ctx, user); createErr != nil {
			summary.Errors = append(summary.Errors, fmt.Sprintf("row %d: failed to create user", line))
			summary.Skipped++
			continue
		}
		summary.Created++
	}

	s.logger.Info("json user import finished", zap.Int("created", summary.Created), zap.Int("skipped", summary.Skipped))
	return summary, nil
}

// ─── 导出与模板 ─────────────────────────────────────────────────────────────

// exportHeaderKeys 定义导出文件的列顺序；表头文案经 pkg/i18n 按请求
// locale 解析（messages/*.json 中的 export.users.header.*）。
var exportHeaderKeys = []string{"username", "nickname", "phone", "email", "role", "status", "twoFactor", "createdAt"}

// exportColumnWidths 与 exportHeaderKeys 一一对应的列宽（字符宽度）。
var exportColumnWidths = []float64{20, 20, 16, 32, 14, 12, 14, 24}

// templateHeaderKeys 定义导入模板的列顺序（与导入解析字段一致）。
var templateHeaderKeys = []string{"username", "nickname", "phone", "email", "role", "password"}

// templateColumnWidths 与 templateHeaderKeys 一一对应的列宽。
var templateColumnWidths = []float64{20, 20, 16, 32, 14, 24}

// ExportUsersExcel 将勾选的用户导出为 Excel 并返回文件字节。
// 表头按 locale 本地化；role/status 等数据值保持原始枚举串，保证导出
// 文件可直接回导；无效的 ID 会被跳过。
func (s *UserImportExportService) ExportUsersExcel(ctx context.Context, userIDs []string, locale string) ([]byte, *apperrors.AppError) {
	ids, idErr := normalizeUserIDs(userIDs)
	if idErr != nil {
		return nil, idErr
	}
	users, err := s.userRepo.FindByIDs(ctx, ids)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to load users for export", err)
	}

	filtered := users

	rows := make([][]interface{}, 0, len(filtered))
	for i := range filtered {
		user := &filtered[i]
		rows = append(rows, []interface{}{
			user.Username, user.Nickname, user.Phone, user.Email, string(user.Role), string(user.Status),
			user.TwoFactorEnabled, user.CreatedAt.Format(time.RFC3339),
		})
	}

	data, buildErr := buildExcel(localizedHeaders(locale, exportHeaderKeys), exportColumnWidths, rows)
	if buildErr != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to build export", buildErr)
	}
	s.logger.Info("excel user export finished", zap.Int("requested", len(ids)), zap.Int("exported", len(filtered)))
	return data, nil
}

// BuildImportTemplate 生成本地化的用户导入模板：表头 + 两行示例。
// 示例 role 为原始枚举值；示例密码满足强度策略，模板可直接回导验证。
func (s *UserImportExportService) BuildImportTemplate(locale string) ([]byte, *apperrors.AppError) {
	rows := [][]interface{}{
		{"alice", "Alice", "13800000001", "alice@example.com", string(model.RoleViewer), "Vermilion-Otter-42"},
		{"bob", "Bob", "13800000002", "bob@example.com", string(model.RoleEditor), "Harbor-Piano-Sunset-9"},
	}
	data, err := buildExcel(localizedHeaders(locale, templateHeaderKeys), templateColumnWidths, rows)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to build import template", err)
	}
	return data, nil
}

// localizedHeaders 将表头 key 解析为指定 locale 的文案。
func localizedHeaders(locale string, keys []string) []string {
	out := make([]string, len(keys))
	for i, key := range keys {
		out[i] = i18n.MustT(locale, "export.users.header."+key)
	}
	return out
}

// normalizeUserIDs 校验（十进制 uint64）并按首次出现顺序去重用户 ID。
func normalizeUserIDs(raw []string) ([]string, *apperrors.AppError) {
	seen := make(map[string]bool, len(raw))
	out := make([]string, 0, len(raw))
	for _, id := range raw {
		id = strings.TrimSpace(id)
		if _, err := strconv.ParseUint(id, 10, 64); err != nil {
			return nil, apperrors.New(apperrors.ErrCodeInvalidCode, "invalid user id: "+id)
		}
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out, nil
}

// buildExcel 在内存中构建单工作表 xlsx：加粗表头 + 列宽 + 数据行。
// 完整构建成功后才返回字节，调用方由此保证仅在成功时写下载响应头。
func buildExcel(headers []string, widths []float64, rows [][]interface{}) ([]byte, error) {
	file := excelize.NewFile()
	defer file.Close()
	sheet := file.GetSheetName(0)

	for column, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(column+1, 1)
		if err := file.SetCellValue(sheet, cell, header); err != nil {
			return nil, err
		}
	}
	styleID, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}, Fill: excelize.Fill{Type: "pattern", Color: []string{"D9E1F2"}, Pattern: 1}})
	if err != nil {
		return nil, err
	}
	if err := file.SetRowStyle(sheet, 1, 1, styleID); err != nil {
		return nil, err
	}
	for i, width := range widths {
		column, colErr := excelize.ColumnNumberToName(i + 1)
		if colErr != nil {
			return nil, colErr
		}
		if err := file.SetColWidth(sheet, column, column, width); err != nil {
			return nil, err
		}
	}

	for rowIndex, row := range rows {
		for column, value := range row {
			cell, _ := excelize.CoordinatesToCellName(column+1, rowIndex+2)
			if err := file.SetCellValue(sheet, cell, value); err != nil {
				return nil, err
			}
		}
	}

	var buf bytes.Buffer
	if err := file.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
