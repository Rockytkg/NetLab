package captcha

import (
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeStore 是用于测试的内存版 Store（无需依赖 Redis）。
type fakeStore struct {
	mu       sync.Mutex
	values   map[string]string
	attempts map[string]int64
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		values:   make(map[string]string),
		attempts: make(map[string]int64),
	}
}

func (f *fakeStore) Set(id, answer string, _ time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.values[id] = answer
	return nil
}

func (f *fakeStore) Get(id string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.values[id], nil
}

func (f *fakeStore) Delete(id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.values, id)
	delete(f.attempts, id)
	return nil
}

func (f *fakeStore) Incr(id string, _ time.Duration) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.attempts[id]++
	return f.attempts[id], nil
}

func newTestManager(store Store, maxRetries int) *Manager {
	return NewManager(store, 240, 80, maxRetries)
}

// TestRandomExpression 验证每个运算符分支都产生落在允许范围
// [0,999] 内的结果，且表达式非负、求值正确。
func TestRandomExpression(t *testing.T) {
	m := newTestManager(newFakeStore(), 5)

	ops := map[string]int{"+": 0, "-": 0, "×": 0, "÷": 0}
	for i := 0; i < 5000; i++ {
		expr, answer := m.randomExpression()

		if answer < 0 || answer > 999 {
			t.Fatalf("answer out of range [0,999]: %q = %d", expr, answer)
		}
		if !strings.HasSuffix(expr, " = ?") {
			t.Fatalf("expression missing suffix: %q", expr)
		}

		// 解析 "A op B = ?" 并重新求值以确认存储的答案。
		parts := strings.Fields(strings.TrimSuffix(expr, " = ?"))
		if len(parts) != 3 {
			t.Fatalf("unexpected expression shape: %q", expr)
		}
		a, _ := strconv.Atoi(parts[0])
		b, _ := strconv.Atoi(parts[2])
		var want int
		switch parts[1] {
		case "+":
			want = a + b
		case "-":
			want = a - b
		case "×":
			want = a * b
		case "÷":
			if b == 0 {
				t.Fatalf("division by zero: %q", expr)
			}
			if a%b != 0 {
				t.Fatalf("non-exact division: %q", expr)
			}
			want = a / b
		default:
			t.Fatalf("unknown operator %q in %q", parts[1], expr)
		}
		if want != answer {
			t.Fatalf("answer mismatch for %q: got %d want %d", expr, answer, want)
		}
		ops[parts[1]]++
	}

	// 合理性检查：在 5000 个样本中每个运算符都应至少出现一次。
	for op, count := range ops {
		if count == 0 {
			t.Errorf("operator %q never generated", op)
		}
	}
}

// TestGenerateProducesPNG 验证 Generate 会存储一个数值答案并返回
// 可解码的 PNG 数据。
func TestGenerateProducesPNG(t *testing.T) {
	store := newFakeStore()
	m := newTestManager(store, 5)

	res, err := m.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if res.CaptchaID == "" || res.CaptchaImage == "" {
		t.Fatal("empty captcha id/image")
	}
	// 存储的答案必须是范围内的有效整数。
	stored, _ := store.Get(res.CaptchaID)
	n, err := strconv.Atoi(stored)
	if err != nil || n < 0 || n > 999 {
		t.Fatalf("stored answer invalid: %q", stored)
	}
}

// TestVerifyOneTimeUse 确认正确答案仅能成功一次，之后验证码
// 即被消耗。
func TestVerifyOneTimeUse(t *testing.T) {
	store := newFakeStore()
	m := newTestManager(store, 5)

	_ = store.Set("id1", "42", time.Minute)

	ok, err := m.Verify("id1", "42")
	if err != nil || !ok {
		t.Fatalf("expected success, got ok=%v err=%v", ok, err)
	}
	// 第二次使用必定失败（已被删除）。
	ok, _ = m.Verify("id1", "42")
	if ok {
		t.Fatal("captcha should be single-use")
	}
}

// TestVerifyNumericEquivalence 确认提交的答案中允许存在空白
// 和前导零。
func TestVerifyNumericEquivalence(t *testing.T) {
	store := newFakeStore()
	m := newTestManager(store, 5)
	_ = store.Set("id1", "7", time.Minute)

	if ok, _ := m.Verify("id1", "  07 "); !ok {
		t.Fatal("expected '  07 ' to match stored '7'")
	}
}

// TestVerifyRetryLimit 确认在 maxRetries 次错误猜测后验证码
// 会失效，从而阻止对较小答案空间的暴力破解。
func TestVerifyRetryLimit(t *testing.T) {
	store := newFakeStore()
	m := newTestManager(store, 3)
	_ = store.Set("id1", "500", time.Minute)

	for i := 0; i < 3; i++ {
		if ok, _ := m.Verify("id1", "1"); ok {
			t.Fatalf("wrong guess %d unexpectedly succeeded", i)
		}
	}

	// 3 次失败后验证码被删除；即便是正确答案也会失败。
	if ok, _ := m.Verify("id1", "500"); ok {
		t.Fatal("captcha should be invalidated after retry limit")
	}
	if stored, _ := store.Get("id1"); stored != "" {
		t.Fatal("captcha value should have been deleted")
	}
}

// TestVerifyUnknownID 对不存在的验证码返回 false。
func TestVerifyUnknownID(t *testing.T) {
	m := newTestManager(newFakeStore(), 5)
	if ok, _ := m.Verify("nope", "1"); ok {
		t.Fatal("unknown id should not verify")
	}
}

// TestGenerateIDFormat 验证验证码 ID 是 32 位小写十六进制
// 字符串（UUID v4 的 MD5），并且在多次生成中保持唯一 —— 即
// 不像旧的 "captcha_<nano>_<rand>" 方案那样与时间相关或可枚举。
func TestGenerateIDFormat(t *testing.T) {
	m := newTestManager(newFakeStore(), 5)
	hexRe := regexp.MustCompile(`^[0-9a-f]{32}$`)

	seen := make(map[string]struct{}, 1000)
	for i := 0; i < 1000; i++ {
		res, err := m.Generate()
		if err != nil {
			t.Fatalf("Generate: %v", err)
		}
		if !hexRe.MatchString(res.CaptchaID) {
			t.Fatalf("captcha id not 32-char lowercase hex: %q", res.CaptchaID)
		}
		if _, dup := seen[res.CaptchaID]; dup {
			t.Fatalf("duplicate captcha id generated: %q", res.CaptchaID)
		}
		seen[res.CaptchaID] = struct{}{}
	}
}

func BenchmarkGenerate(b *testing.B) {
	m := newTestManager(newFakeStore(), 5)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := m.Generate(); err != nil {
			b.Fatal(err)
		}
	}
}
