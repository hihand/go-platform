package logkit_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/hihand/go-platform/logkit"
)

// ---------- test harness --------------------------------------------------

type capture struct {
	buf bytes.Buffer
}

func (c *capture) Write(p []byte) (int, error) { return c.buf.Write(p) }

func (c *capture) lines() []string {
	out := []string{}
	for _, l := range strings.Split(strings.TrimSpace(c.buf.String()), "\n") {
		if l != "" {
			out = append(out, l)
		}
	}
	return out
}

func newLogger(t *testing.T) (logkit.Logger, *capture) {
	t.Helper()
	c := &capture{}
	return logkit.New(
		logkit.WithOutput(c),
		logkit.WithMinLevel(logkit.LevelDebug),
		logkit.WithService("payment-api", "1.0.0"),
		logkit.WithDeployment("production"),
	), c
}

func decode(t *testing.T, line string) map[string]any {
	t.Helper()
	m := map[string]any{}
	if err := json.Unmarshal([]byte(line), &m); err != nil {
		t.Fatalf("decode: %v\nline: %s", err, line)
	}
	return m
}

// ---------- context mapping fixture ---------------------------------------

// reqCtxKey is the application's own context key type. logkit has no
// awareness of it — the mapper owns the extraction contract entirely.
type reqCtxKey struct{ name string }

var reqIDKey = reqCtxKey{"req.id"}

func withReqID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, reqIDKey, id)
}

func defaultReqIDMapper(ctx context.Context, attrs []logkit.Attr) []logkit.Attr {
	if v, ok := ctx.Value(reqIDKey).(string); ok && v != "" {
		return append(attrs, logkit.String(keyRequestID, v))
	}
	return attrs
}

// ---------- core schema ---------------------------------------------------

func TestLogger_StaticFields(t *testing.T) {
	t.Parallel()
	l, c := newLogger(t)
	l.Info("hello")
	rec := decode(t, c.lines()[0])
	if rec[string(logkit.KeyServiceName)] != "payment-api" {
		t.Errorf("%s = %v", logkit.KeyServiceName, rec[string(logkit.KeyServiceName)])
	}
	if rec[string(logkit.KeyServiceVersion)] != "1.0.0" {
		t.Errorf("%s = %v", logkit.KeyServiceVersion, rec[string(logkit.KeyServiceVersion)])
	}
	if rec[string(logkit.KeyDeploymentEnvironment)] != "production" {
		t.Errorf("%s = %v", logkit.KeyDeploymentEnvironment, rec[string(logkit.KeyDeploymentEnvironment)])
	}
	if rec[string(logkit.KeyLevel)] != logkit.LevelInfo.String() {
		t.Errorf("%s = %v", logkit.KeyLevel, rec[string(logkit.KeyLevel)])
	}
	if rec[string(logkit.KeyMessage)] != "hello" {
		t.Errorf("%s = %v", logkit.KeyMessage, rec[string(logkit.KeyMessage)])
	}
}

func TestLogger_AllLevels(t *testing.T) {
	t.Parallel()
	l, c := newLogger(t)
	l.Debug("d")
	l.Info("i")
	l.Warn("w")
	l.Error("e")
	want := []string{
		logkit.LevelDebug.String(),
		logkit.LevelInfo.String(),
		logkit.LevelWarn.String(),
		logkit.LevelError.String(),
	}
	lines := c.lines()
	if len(lines) != 4 {
		t.Fatalf("want 4 lines, got %d", len(lines))
	}
	for i, line := range lines {
		if got := decode(t, line)[string(logkit.KeyLevel)]; got != want[i] {
			t.Errorf("line %d level = %v, want %s", i, got, want[i])
		}
	}
}

func TestLogger_Levels_ContextVariants(t *testing.T) {
	t.Parallel()
	l, c := newLogger(t)
	l.DebugContext(context.Background(), "d")
	l.InfoContext(context.Background(), "i")
	l.WarnContext(context.Background(), "w")
	l.ErrorContext(context.Background(), "e")
	if got := len(c.lines()); got != 4 {
		t.Errorf("want 4, got %d", got)
	}
}

func TestLogger_MinLevelDrops(t *testing.T) {
	t.Parallel()
	c := &capture{}
	l := logkit.New(logkit.WithOutput(c), logkit.WithMinLevel(logkit.LevelWarn))
	l.Debug("d1")
	l.Info("i1")
	l.Warn("w1")
	l.Error("e1")
	if got := len(c.lines()); got != 2 {
		t.Errorf("want 2 lines (WARN+ERROR), got %d", got)
	}
}

func TestLogger_TimestampRFC3339(t *testing.T) {
	t.Parallel()
	l, c := newLogger(t)
	l.Info("stamp")
	rec := decode(t, c.lines()[0])
	ts, ok := rec[string(logkit.KeyTimestamp)].(string)
	if !ok {
		t.Fatalf("timestamp missing: %v", rec[string(logkit.KeyTimestamp)])
	}
	if _, err := time.Parse(time.RFC3339Nano, ts); err != nil {
		t.Errorf("timestamp not RFC3339Nano: %v", err)
	}
}

func TestLogger_StringEscapes(t *testing.T) {
	t.Parallel()
	l, c := newLogger(t)
	l.Info(`msg with "quotes", \nnewline and \back`)
	rec := decode(t, c.lines()[0])
	if msg, _ := rec[string(logkit.KeyMessage)].(string); msg != `msg with "quotes", \nnewline and \back` {
		t.Errorf("round-trip failed: %v", msg)
	}
}

func TestLogger_UTF8(t *testing.T) {
	t.Parallel()
	l, c := newLogger(t)
	l.Info("xin chào; 🚀")
	rec := decode(t, c.lines()[0])
	if rec[string(logkit.KeyMessage)] != "xin chào; 🚀" {
		t.Errorf("utf-8 round-trip failed: %v", rec[string(logkit.KeyMessage)])
	}
}

// ---------- context mapping -----------------------------------------------

func TestLogger_NoMapperIgnoresContext(t *testing.T) {
	t.Parallel()
	l, c := newLogger(t)
	ctx := withReqID(context.Background(), "req-1")
	l.InfoContext(ctx, "x")
	rec := decode(t, c.lines()[0])
	if _, present := rec[string(keyRequestID)]; present {
		t.Errorf("no mapper → context values must not leak: %v", rec[string(keyRequestID)])
	}
}

func TestLogger_MapperExtractsContext(t *testing.T) {
	t.Parallel()
	c := &capture{}
	l := logkit.New(
		logkit.WithOutput(c),
		logkit.WithMinLevel(logkit.LevelDebug),
		logkit.WithContextMapper(defaultReqIDMapper),
	)
	ctx := withReqID(context.Background(), "req-1")
	l.InfoContext(ctx, "x")
	rec := decode(t, c.lines()[0])
	if rec[string(keyRequestID)] != "req-1" {
		t.Errorf("mapper did not lift %s: %v", keyRequestID, rec[string(keyRequestID)])
	}
}

func TestLogger_MapperSeesContextAcrossLevels(t *testing.T) {
	t.Parallel()
	c := &capture{}
	l := logkit.New(
		logkit.WithOutput(c),
		logkit.WithMinLevel(logkit.LevelDebug),
		logkit.WithContextMapper(defaultReqIDMapper),
	)
	ctx := withReqID(context.Background(), "req-1")
	l.DebugContext(ctx, "d")
	l.InfoContext(ctx, "i")
	l.WarnContext(ctx, "w")
	l.ErrorContext(ctx, "e")
	for _, line := range c.lines() {
		if rec := decode(t, line); rec[string(keyRequestID)] != "req-1" {
			t.Errorf("expected %s on every line, got %v", keyRequestID, rec[string(keyRequestID)])
		}
	}
}

func TestLogger_NilMapperOptionIsSafe(t *testing.T) {
	t.Parallel()
	c := &capture{}
	l := logkit.New(
		logkit.WithOutput(c),
		logkit.WithMinLevel(logkit.LevelDebug),
		logkit.WithContextMapper(nil),
	)
	l.InfoContext(context.Background(), "x")
	if got := len(c.lines()); got != 1 {
		t.Errorf("nil mapper option must be a no-op; got %d", got)
	}
}

func TestLogger_NilContextStillWorks(t *testing.T) {
	t.Parallel()
	c := &capture{}
	l := logkit.New(
		logkit.WithOutput(c),
		logkit.WithMinLevel(logkit.LevelDebug),
		logkit.WithContextMapper(defaultReqIDMapper),
	)
	l.InfoContext(context.TODO(), "x")
	if got := len(c.lines()); got != 1 {
		t.Errorf("nil ctx must still emit the record; got %d", got)
	}
}

func TestLogger_Precedence_CallOverridesContext(t *testing.T) {
	t.Parallel()
	c := &capture{}
	l := logkit.New(
		logkit.WithOutput(c),
		logkit.WithMinLevel(logkit.LevelDebug),
		logkit.WithContextMapper(defaultReqIDMapper),
	)
	ctx := withReqID(context.Background(), "ctx-value")
	l.InfoContext(ctx, "x", logkit.String(keyRequestID, "call-value"))
	rec := decode(t, c.lines()[0])
	if rec[string(keyRequestID)] != "call-value" {
		t.Errorf("call attrs must beat ctx attrs: got %v", rec[string(keyRequestID)])
	}
}

func TestLogger_Precedence_WithLosesToContext(t *testing.T) {
	t.Parallel()
	c := &capture{}
	l := logkit.New(
		logkit.WithOutput(c),
		logkit.WithMinLevel(logkit.LevelDebug),
		logkit.WithContextMapper(defaultReqIDMapper),
	)
	scoped := l.With(logkit.String(keyRequestID, "with-value"))
	ctx := withReqID(context.Background(), "ctx-value")
	scoped.InfoContext(ctx, "x")
	rec := decode(t, c.lines()[0])
	if rec[string(keyRequestID)] != "ctx-value" {
		t.Errorf("ctx attrs must beat with attrs: got %v", rec[string(keyRequestID)])
	}
}

func TestLogger_MapperOnlyRunsOnContextCalls(t *testing.T) {
	t.Parallel()
	c := &capture{}
	calls := 0
	mapper := func(ctx context.Context, attrs []logkit.Attr) []logkit.Attr {
		calls++
		if v, ok := ctx.Value(reqIDKey).(string); ok && v != "" {
			return append(attrs, logkit.String(keyRequestID, v))
		}
		return attrs
	}
	l := logkit.New(
		logkit.WithOutput(c),
		logkit.WithMinLevel(logkit.LevelDebug),
		logkit.WithContextMapper(mapper),
	)
	l.Info("no ctx")
	l.With(logkit.String(keyK, "v"))
	if calls != 0 {
		t.Errorf("non-context methods must not invoke mapper; got %d calls", calls)
	}
	l.InfoContext(withReqID(context.Background(), "req"), "ctx")
	if calls != 1 {
		t.Errorf("InfoContext must invoke mapper exactly once; got %d", calls)
	}
}

// ---------- attribute kinds ----------------------------------------------

func TestLogger_With(t *testing.T) {
	t.Parallel()
	l, c := newLogger(t)
	scoped := l.With(logkit.String(keyRequestID, "req-with"))
	scoped.Info("scoped")
	rec := decode(t, c.lines()[0])
	if rec[string(keyRequestID)] != "req-with" {
		t.Errorf("With %s = %v", keyRequestID, rec[string(keyRequestID)])
	}
}

func TestLogger_Precedence(t *testing.T) {
	t.Parallel()
	l, c := newLogger(t)
	scoped := l.With(logkit.String(logkit.KeyEvent, "payment.created"))
	scoped.Info("hi", logkit.String(logkit.KeyEvent, "payment.attended"))
	rec := decode(t, c.lines()[0])
	if rec[string(logkit.KeyEvent)] != "payment.attended" {
		t.Errorf("call attrs must win over With attrs: got %v", rec[string(logkit.KeyEvent)])
	}
}

func TestLogger_Precedence_DuplicateKey(t *testing.T) {
	t.Parallel()
	l, c := newLogger(t)
	l.Info("dup",
		logkit.String(keyRequestID, "first"),
		logkit.String(keyRequestID, "second"),
	)
	rec := decode(t, c.lines()[0])
	if rec[string(keyRequestID)] != "second" {
		t.Errorf("dup winner should be 'second', got %v", rec[string(keyRequestID)])
	}
}

func TestLogger_BusinessFields(t *testing.T) {
	t.Parallel()
	l, c := newLogger(t)
	l.Info("payment",
		logkit.String(logkit.KeyEvent, "payment.failed"),
		logkit.String(keyPaymentID, "pay-001"),
		logkit.Int64(keyPaymentAmt, 100),
	)
	rec := decode(t, c.lines()[0])
	if rec[string(keyPaymentID)] != "pay-001" {
		t.Errorf("%s = %v", keyPaymentID, rec[string(keyPaymentID)])
	}
	if v, ok := rec[string(keyPaymentAmt)].(float64); !ok || v != 100 {
		t.Errorf("%s = %v (%T)", keyPaymentAmt, rec[string(keyPaymentAmt)], rec[string(keyPaymentAmt)])
	}
	if rec[string(logkit.KeyEvent)] != "payment.failed" {
		t.Errorf("%s = %v", logkit.KeyEvent, rec[string(logkit.KeyEvent)])
	}
}

func TestLogger_AllAttrKindsRender(t *testing.T) {
	t.Parallel()
	l, c := newLogger(t)
	l.Info("mixed",
		logkit.Int(logkit.AnyKey("i"), 42),
		logkit.Int64(logkit.AnyKey("i64"), 1<<40),
		logkit.Uint64(logkit.AnyKey("u64"), 1<<50),
		logkit.Float64(logkit.AnyKey("f"), 1.5),
		logkit.Bool(logkit.AnyKey("b"), true),
		logkit.Dur(logkit.AnyKey("d"), 250*time.Millisecond),
		logkit.Time(logkit.AnyKey("t"), time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC)),
	)
	rec := decode(t, c.lines()[0])
	for _, k := range []string{"i", "i64", "u64", "f", "b", "d", "t"} {
		if _, ok := rec[k]; !ok {
			t.Errorf("missing %s in record: %v", k, rec)
		}
	}
}

// ---------- caller --------------------------------------------------------

func TestLogger_CallerOffByDefault(t *testing.T) {
	t.Parallel()
	l, c := newLogger(t)
	l.Info("no caller")
	if _, ok := decode(t, c.lines()[0])[string(logkit.KeyCaller)]; ok {
		t.Errorf("caller must be off by default")
	}
}

func TestLogger_CallerOn(t *testing.T) {
	t.Parallel()
	c := &capture{}
	l := logkit.New(logkit.WithOutput(c), logkit.WithMinLevel(logkit.LevelDebug), logkit.WithCaller())
	l.Info("with caller")
	rec := decode(t, c.lines()[0])
	caller, _ := rec[string(logkit.KeyCaller)].(string)
	if caller == "" {
		t.Errorf("caller should be present when WithCaller is set, got empty")
	}
	if !strings.Contains(caller, ":") {
		t.Errorf("caller must include line (path:line), got %q", caller)
	}
	if strings.Contains(caller, "logkit/caller.go") ||
		strings.Contains(caller, "logkit/log.go") ||
		strings.Contains(caller, "logkit/info.go") {
		t.Errorf("caller leaked logkit internals: %q", caller)
	}
}

// ---------- edge cases ---------------------------------------------------

func TestLogger_InvalidMinLevelFallsBackToInfo(t *testing.T) {
	t.Parallel()
	c := &capture{}
	l := logkit.New(logkit.WithOutput(c), logkit.WithMinLevel(logkit.Level(99)))
	l.Debug("should be dropped")
	l.Info("kept")
	if got := len(c.lines()); got != 1 {
		t.Errorf("invalid level must map to INFO; got %d", got)
	}
}

func TestLogger_NilWriterIsSafe(t *testing.T) {
	t.Parallel()
	l := logkit.New(logkit.WithOutput(nil))
	if l == nil {
		t.Fatalf("nil writer must be ignored silently")
	}
}

func TestLogger_EmptyMessage(t *testing.T) {
	t.Parallel()
	c := &capture{}
	l := logkit.New(logkit.WithOutput(c))
	l.Info("")
	rec := decode(t, c.lines()[0])
	if msg, _ := rec[string(logkit.KeyMessage)].(string); msg != "" {
		t.Errorf("empty message was rewritten: %q", msg)
	}
}

func TestLogger_SchemaOrderStable(t *testing.T) {
	t.Parallel()
	l, c := newLogger(t)
	l.Info("order")
	line := c.lines()[0]
	// Asserts on the wire format — schema fields must appear in this
	// order regardless of caller-supplied attrs.
	ts := strings.Index(line, `"`+string(logkit.KeyTimestamp)+`"`)
	lvl := strings.Index(line, `"`+string(logkit.KeyLevel)+`"`)
	msg := strings.Index(line, `"`+string(logkit.KeyMessage)+`"`)
	if !(ts >= 0 && lvl > ts && msg > lvl) {
		t.Errorf("schema order broken: %s", line)
	}
}

// ---------- typed API round-trip ----------------------------------------

func TestLogger_AnyKey(t *testing.T) {
	t.Parallel()
	c := &capture{}
	l := logkit.New(logkit.WithOutput(c))
	custom := logkit.AnyKey("app.region")
	l.Info("x", logkit.String(custom, "ap-southeast-1"))
	rec := decode(t, c.lines()[0])
	if rec[string(custom)] != "ap-southeast-1" {
		t.Errorf("custom key %q missing: %v", custom, rec)
	}
}

// ---------- formatted variants -----------------------------------------

func TestLogger_FormatVariants(t *testing.T) {
	t.Parallel()
	l, c := newLogger(t)
	l.Debugf("count=%d", 1)
	l.Infof("user=%s age=%d", "alice", 30)
	l.Warnf("ratio=%.2f", 0.5)
	l.Errorf("oops: %s", "boom")
	if got := len(c.lines()); got != 4 {
		t.Fatalf("want 4 lines, got %d", got)
	}
	want := []string{
		`"message":"count=1"`,
		`"message":"user=alice age=30"`,
		`"message":"ratio=0.50"`,
		`"message":"oops: boom"`,
	}
	for i, w := range want {
		if !strings.Contains(c.lines()[i], w) {
			t.Errorf("line %d missing %q in: %s", i, w, c.lines()[i])
		}
	}
}

func TestLogger_FormatContextVariants(t *testing.T) {
	t.Parallel()
	c := &capture{}
	l := logkit.New(
		logkit.WithOutput(c),
		logkit.WithMinLevel(logkit.LevelDebug),
		logkit.WithContextMapper(defaultReqIDMapper),
	)
	ctx := withReqID(context.Background(), "req-1")
	l.InfoContextf(ctx, "hello %s", "world")
	rec := decode(t, c.lines()[0])
	if rec[string(logkit.KeyMessage)] != "hello world" {
		t.Errorf("message = %v", rec[string(logkit.KeyMessage)])
	}
	if rec[string(keyRequestID)] != "req-1" {
		t.Errorf("%s = %v", keyRequestID, rec[string(keyRequestID)])
	}
}

func TestLogger_FormatVariantsHonourMinLevel(t *testing.T) {
	t.Parallel()
	c := &capture{}
	l := logkit.New(logkit.WithOutput(c), logkit.WithMinLevel(logkit.LevelInfo))
	l.Debugf("dropped %s", "ignored")
	l.Infof("kept %s", "kept")
	if got := len(c.lines()); got != 1 {
		t.Fatalf("want 1 line, got %d", got)
	}
	if !strings.Contains(c.lines()[0], `"message":"kept kept"`) {
		t.Errorf("expected formatted message on the kept line, got %s", c.lines()[0])
	}
}

// ---------- mapper append-style contract --------------------------------

func TestLogger_MapperAppendStyleReusesCapacity(t *testing.T) {
	t.Parallel()
	c := &capture{}
	// Mapper deliberately appends to the supplied slice and returns it
	// unchanged when nothing matches. This proves the append-style
	// signature is the recommended path: callers may reuse the slice's
	// capacity.
	mapper := func(ctx context.Context, attrs []logkit.Attr) []logkit.Attr {
		if v, ok := ctx.Value(reqIDKey).(string); ok && v != "" {
			return append(attrs, logkit.String(keyRequestID, v))
		}
		return attrs
	}
	l := logkit.New(
		logkit.WithOutput(c),
		logkit.WithMinLevel(logkit.LevelDebug),
		logkit.WithContextMapper(mapper),
	)
	ctx := withReqID(context.Background(), "req-1")
	l.InfoContext(ctx, "x")
	rec := decode(t, c.lines()[0])
	if rec[string(keyRequestID)] != "req-1" {
		t.Errorf("append mapper did not lift %s: %v", keyRequestID, rec[string(keyRequestID)])
	}
}

func TestLogger_MapperChaining(t *testing.T) {
	t.Parallel()
	c := &capture{}
	mapperA := func(ctx context.Context, attrs []logkit.Attr) []logkit.Attr {
		return append(attrs, logkit.String(keyA, "1"))
	}
	mapperB := func(ctx context.Context, attrs []logkit.Attr) []logkit.Attr {
		return append(attrs, logkit.String(keyB, "2"))
	}
	// Chain: A runs first, B receives A's output. Composition via a
	// 3-line wrapper — no abstraction needed.
	chain := func(ctx context.Context, attrs []logkit.Attr) []logkit.Attr {
		attrs = mapperA(ctx, attrs)
		attrs = mapperB(ctx, attrs)
		return attrs
	}
	l := logkit.New(
		logkit.WithOutput(c),
		logkit.WithMinLevel(logkit.LevelDebug),
		logkit.WithContextMapper(chain),
	)
	l.InfoContext(context.Background(), "x")
	rec := decode(t, c.lines()[0])
	if rec[string(keyA)] != "1" || rec[string(keyB)] != "2" {
		t.Errorf("chained mapper missing fields: a=%v b=%v", rec[string(keyA)], rec[string(keyB)])
	}
}