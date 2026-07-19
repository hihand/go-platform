# Go Platform — File & Package Conventions

---

## File Naming Cheat Sheet

| Filename | What goes in it |
|----------|-----------------|
| `spec.go` | Interface canonical |
| `new.go` | `impl` struct (unexported) + `New()` constructor |
| `<func>.go` | Implementation của func đó. One func per file. |
| `<func>_models.go` | Models/structs riêng cho func đó (nếu cần, không bắt buộc) |
| `common.go` | Shared helpers dùng chung cho nhiều func |
| `<name>_test.go` | Gom chung, không tách `<func>_test.go` lẻ |
| `<package>_mock.go` | Gom chung mocks, không tách theo func |
| `example_test.go` | Runnable godoc examples |

**Rule:** one exported func per file. Filename = func name (lowercase, snake_case nếu cần).

---

## Package Layout

```
mypkg/
├── spec.go            # interface definition
├── new.go             # impl struct + New()
├── func1.go           # implementation of Func1()
├── func1_models.go   # models cho Func1 (optional)
├── func2.go           # implementation of Func2()
├── func2_models.go   # models cho Func2 (optional)
└── common.go          # shared helpers
└── example_test.go
```

### Test & Mock

- **Không tách test theo từng func.** Gom hết vào `<package>_test.go`.
- **Mocks gom chung** vào `<package>_mock.go`.

```
mypkg/
├── spec.go
├── new.go
├── func1.go
├── func1_models.go
├── func2.go
├── common.go
├── mypkg_test.go        # tất cả test, không tách func1_test.go
├── mypkg_mock.go        # tất cả mocks
└── example_test.go
```

### Adapter Package (maps core → protocol)

Thin translation table. Single file:

```
errkit/<proto>err/
├── <proto>err.go
└── <proto>err_test.go
```

---

## Naming Conventions Inside Files

### `spec.go`
- Interface: singular noun. Method names là verb hoặc noun phrase.
- Optional capabilities: separate small interface (`MetadataAccessor`, `Closer`).

### `new.go`
- `type impl struct` — unexported, lowercase.
- Fields: short, single-word, unexported: `code`, `message`, `cause`, `metadata`.
- `New(opts ...Option) InterfaceName` — returns interface, not concrete type.
- `Wrap(existing, opts ...Option) InterfaceName` — `Wrap(nil) = nil`.

### `options.go` (optional, đặt trong `new.go` hoặc tách riêng)
- `type Option func(*impl)`
- Each option: `func WithXxx(value XxxType) Option`

### `<func>.go`
- One exported func per file.
- Func type signature defined in `spec.go`, body implemented here.
- Nếu cần internal helpers: đặt trong file đó (unexported).

### `<func>_models.go`
- Chỉ tạo khi func cần structs/types riêng biệt mà không dùng chung.
- Nếu dùng chung → bỏ vào `common.go`.

### `common.go`
- Helpers dùng chung cho nhiều func.
- Tất cả unexported (lowercase) trừ khi muốn export.

---

## Test & Mock Conventions

- **Tests gom chung** trong `<package>_test.go`. Không tách `func1_test.go`, `func2_test.go`.
- **Mocks gom chung** trong `<package>_mock.go`.
- Lý do: tìm nhanh, đọc nhanh, không nhảy file nhiều.

Every test:
- lives in external package `<pkg>_test`
- starts with `t.Parallel()`
- uses `tc := tc` in `t.Run` loops

---

## Decision Flow

```
Does it translate core codes to a protocol?
  YES → single-file adapter: <proto>err.go
  NO  → Core package (implements interface + funcs)?
          YES → spec.go + new.go + <func>.go (+ <func>_models.go + common.go as needed)
          NO  → Utility package (stateless helpers)?
                  YES → <pkg>.go + common.go
```

---

## Cross-cutting Rules (from implementation)

- **Core package: stdlib only.** No external imports.
- **Adapter may import protocol package only.**
- **`Wrap(nil) = nil`** — safe to return directly.
- **Defensive copy** on map input/output.
- **All `Of(err)` helpers tolerate nil & non-matching errors** — return zero value, never panic.
- **`Error()` format: `CODE: message` hoặc `CODE: message: cause.Error()`.**
- **Every package: top-level doc comment.**
- **`New()` returns interface, not concrete type.**
- **Sugar constructors** for common cases — one-liners in `new.go`.