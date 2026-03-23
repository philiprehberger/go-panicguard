# Changelog

## 0.2.0

- Add `GoNamed` for named goroutines with panic recovery
- Add `GoCtx` for context-aware panic-safe goroutines
- Add `Stats` and `ResetStats` for global panic statistics tracking
- Add `RecoverAs` generic function for typed panic recovery
- Add `PanicStats` type with `TotalPanics`, `LastPanic`, and `LastValue` fields
- Track panic statistics across all recovery paths (`Go`, `GoErr`, `GoNamed`, `GoCtx`, middleware)

## 0.1.2

- Consolidate README badges onto single line

## 0.1.1

- Add badges and Development section to README

## 0.1.0

- Initial release
- `Go` and `GoErr` for panic-safe goroutines
- `Recover` for deferred panic-to-error conversion
- HTTP middleware for panic recovery
