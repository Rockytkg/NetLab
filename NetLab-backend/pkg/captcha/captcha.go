package captcha

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"github.com/google/uuid"
	"golang.org/x/image/font/gofont/goregular"
)

// Manager 负责算术验证码的生成与校验。
//
// 图片展示一个随机的算术表达式（例如 "12 + 34 = ?"），
// 存储的答案是该表达式的数值结果。相比字符验证码，这对 OCR 机器人
// 更难破解，因为求解者不仅要识别字形，还要计算表达式的结果，
// 而对人类来说依然轻而易举。
type Manager struct {
	store      Store
	width      int
	height     int
	ttl        time.Duration
	maxRetries int64
	font       *truetype.Font
}

// Store 是验证码存储的接口。
type Store interface {
	Set(id, answer string, ttl time.Duration) error
	Get(id string) (string, error)
	Delete(id string) error
	// Incr 递增并返回某个验证码的失败尝试计数，
	// 首次使用时将该计数器绑定到给定的 ttl。
	Incr(id string, ttl time.Duration) (int64, error)
}

// CaptchaResult 包含生成的验证码信息。
type CaptchaResult struct {
	CaptchaID    string `json:"captchaId"`
	CaptchaImage string `json:"captchaImage"` // base64 编码的 PNG
}

const (
	// defaultTTL 是验证码保持有效的时长。
	defaultTTL = 5 * time.Minute
	// defaultMaxRetries 是验证码失效前允许的错误猜测次数，
	// 用于缓解对较小答案空间的暴力破解。
	defaultMaxRetries = 5
)

// parsedFont 是内嵌的 Go 字体，在包初始化时解析一次，
// 这样每次 Generate 调用都可复用它，而无需重新解析 TTF 字节。
var parsedFont *truetype.Font

func init() {
	f, err := truetype.Parse(goregular.TTF)
	if err != nil {
		// 该字体已编译进二进制文件；解析失败属于
		// 编程/构建错误，而非运行时状况。
		panic("captcha: parse embedded font: " + err.Error())
	}
	parsedFont = f
}

// NewManager 创建一个新的验证码管理器。maxRetries <= 0 时回退到
// 默认值。width/height 为渲染出的 PNG 尺寸（单位为像素）。
func NewManager(store Store, width, height int, maxRetries int) *Manager {
	retries := int64(maxRetries)
	if retries <= 0 {
		retries = defaultMaxRetries
	}
	return &Manager{
		store:      store,
		width:      width,
		height:     height,
		ttl:        defaultTTL,
		maxRetries: retries,
		font:       parsedFont,
	}
}

// Generate 创建一个新的算术验证码并返回结果。
func (m *Manager) Generate() (*CaptchaResult, error) {
	expr, answer := m.randomExpression()
	id := newCaptchaID()

	if err := m.store.Set(id, strconv.Itoa(answer), m.ttl); err != nil {
		return nil, fmt.Errorf("store captcha: %w", err)
	}

	png, err := m.renderPNG(expr)
	if err != nil {
		return nil, fmt.Errorf("render captcha: %w", err)
	}

	return &CaptchaResult{
		CaptchaID:    id,
		CaptchaImage: base64.StdEncoding.EncodeToString(png),
	}, nil
}

// newCaptchaID 将验证码标识符派生为一个随机 UUID v4 的
// 小写十六进制 MD5 摘要（32 个十六进制字符）。与此前的
// "captcha_<nano>_<rand>" 方案不同，它既不与时间相关也不可枚举，
// 因此攻击者无法猜测或预计算出有效的挑战 ID。该摘要用作存储
// 键、用于校验以及日志追踪 —— 在 5 分钟 TTL 窗口内的碰撞概率
// 可忽略不计。
func newCaptchaID() string {
	sum := md5.Sum([]byte(uuid.NewString()))
	return hex.EncodeToString(sum[:])
}

// Verify 检查提供的答案是否与存储的验证码结果匹配。
//
// 它对每个验证码强制执行重试限制：错误猜测达到 maxRetries 次后，
// 验证码将被删除，从而防止攻击者针对同一挑战穷举较小的数值答案
// 空间（0–999）。答案正确时会立即删除验证码（一次性使用）。
func (m *Manager) Verify(id, answer string) (bool, error) {
	stored, err := m.store.Get(id)
	if err != nil {
		return false, fmt.Errorf("get captcha: %w", err)
	}
	if stored == "" {
		return false, nil
	}

	// 按数值比较，这样前导零/首尾空白不会
	// 导致误判为不匹配（例如 "07" == 7）。
	guess, convErr := strconv.Atoi(strings.TrimSpace(answer))
	want, _ := strconv.Atoi(stored)

	if convErr != nil || guess != want {
		// 猜测错误（或非数字）—— 计数，并在重试预算
		// 用尽后使该挑战失效。
		attempts, incrErr := m.store.Incr(id, m.ttl)
		if incrErr == nil && attempts >= m.maxRetries {
			_ = m.store.Delete(id)
		}
		return false, nil
	}

	// 正确 —— 一次性使用。
	_ = m.store.Delete(id)
	return true, nil
}

// randomExpression 使用 +、-、×、÷ 构建一个随机算术表达式，
// 返回展示字符串（末尾带 " = ?"）及其整数结果。
//
// 为保证可读性并避免溢出/负数展示而强制执行的约束：
//   - 结果始终在 [0, 999] 范围内
//   - 减法结果永不为负
//   - 除法始终为整除（无余数）且展示整数操作数
func (m *Manager) randomExpression() (string, int) {
	switch rand.Intn(4) {
	case 0: // 加法：a + b，两者均在 [1,99]，和 <= 999（此处恒成立）
		a := rand.Intn(99) + 1
		b := rand.Intn(99) + 1
		return fmt.Sprintf("%d + %d = ?", a, b), a + b

	case 1: // 减法：a - b >= 0
		a := rand.Intn(99) + 1
		b := rand.Intn(a) + 1 // 1..a，保证非负
		if b > a {
			b = a
		}
		return fmt.Sprintf("%d - %d = ?", a, b), a - b

	case 2: // 乘法：一位数 × 两位数，乘积 <= 999
		a := rand.Intn(9) + 1  // 1..9
		b := rand.Intn(99) + 1 // 1..99
		if a*b > 999 {
			b = 999 / a
		}
		return fmt.Sprintf("%d × %d = ?", a, b), a * b

	default: // 除法：整除，商×除数 = 被除数
		divisor := rand.Intn(9) + 1   // 1..9
		quotient := rand.Intn(20) + 1 // 1..20
		dividend := divisor * quotient
		return fmt.Sprintf("%d ÷ %d = ?", dividend, divisor), quotient
	}
}

// renderPNG 将表达式绘制到 PNG 上，并加入反 OCR 干扰：
// 对角渐变背景、弯曲干扰线、散布噪点，以及逐字形的旋转/
// 垂直抖动/颜色变化。
func (m *Manager) renderPNG(text string) ([]byte, error) {
	w, h := m.width, m.height
	dc := gg.NewContext(w, h)

	// ── 渐变背景（柔和、浅色，使深色字形仍清晰可读）──
	grad := gg.NewLinearGradient(0, 0, float64(w), float64(h))
	c1 := m.randomLight()
	c2 := m.randomLight()
	grad.AddColorStop(0, c1)
	grad.AddColorStop(1, c2)
	dc.SetFillStyle(grad)
	dc.DrawRectangle(0, 0, float64(w), float64(h))
	dc.Fill()

	// ── 干扰曲线（二次贝塞尔曲线）──
	for i := 0; i < 4; i++ {
		dc.SetColor(m.randomMid())
		dc.SetLineWidth(float64(rand.Intn(2) + 1))
		x0 := rand.Float64() * float64(w) * 0.3
		y0 := rand.Float64() * float64(h)
		cx := rand.Float64() * float64(w)
		cy := rand.Float64() * float64(h)
		x1 := float64(w)*0.7 + rand.Float64()*float64(w)*0.3
		y1 := rand.Float64() * float64(h)
		dc.MoveTo(x0, y0)
		dc.QuadraticTo(cx, cy, x1, y1)
		dc.Stroke()
	}

	// ── 噪点 ──
	for i := 0; i < 40; i++ {
		dc.SetColor(m.randomMid())
		dc.DrawCircle(rand.Float64()*float64(w), rand.Float64()*float64(h), rand.Float64()*1.6+0.4)
		dc.Fill()
	}

	// ── 字形：旋转、抖动、逐个着色 ──
	runes := []rune(text)
	fontSize := float64(h) * 0.5
	face := truetype.NewFace(m.font, &truetype.Options{Size: fontSize})
	dc.SetFontFace(face)

	// 将字形布局在宽度中间 90% 的区域内。
	margin := float64(w) * 0.05
	usable := float64(w) - 2*margin
	step := usable / float64(len(runes))

	for i, r := range runes {
		dc.Push()
		x := margin + step*(float64(i)+0.5)
		y := float64(h)/2 + (rand.Float64()*10 - 5)
		angle := (rand.Float64()*30 - 15) * math.Pi / 180 // ±15°
		dc.RotateAbout(angle, x, y)
		dc.SetColor(m.randomDark())
		dc.DrawStringAnchored(string(r), x, y, 0.5, 0.5)
		dc.Pop()
	}

	var buf bytes.Buffer
	if err := dc.EncodePNG(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (m *Manager) randomLight() color.Color {
	return color.RGBA{
		R: uint8(230 + rand.Intn(25)),
		G: uint8(230 + rand.Intn(25)),
		B: uint8(230 + rand.Intn(25)),
		A: 255,
	}
}

func (m *Manager) randomMid() color.Color {
	return color.RGBA{
		R: uint8(120 + rand.Intn(80)),
		G: uint8(120 + rand.Intn(80)),
		B: uint8(120 + rand.Intn(80)),
		A: 255,
	}
}

func (m *Manager) randomDark() color.Color {
	return color.RGBA{
		R: uint8(rand.Intn(90)),
		G: uint8(rand.Intn(90)),
		B: uint8(rand.Intn(90)),
		A: 255,
	}
}
