## Context

`ping-service` poll các Endpoint theo interval, gửi HTTP request qua `PingWorker.Ping` (`ping-service/internal/infrastructure/ping.go`), chỉ trả về `statusCode` và so sánh với `ep.ExpectedCode` ở `zsetworker.go:73`. Body hiện bị vứt vào `io.Discard`. Endpoint được load từ `endpoint-service` qua gRPC (`endpoint_client.go`), map proto → domain.

Muốn cho phép user viết biểu thức expr để kiểm tra nội dung body mà không deploy lại code.

## Goals / Non-Goals

**Goals:**
- User cấu hình được biểu thức expr kiểm tra body mỗi Endpoint.
- Body check chạy cùng với status code check, kết hợp để ra `StatusOn/Off`.
- Fail-safe: expr lỗi → DOWN.
- Compile expr mỗi lần ping (không cache).
- Giới hạn đọc body 1MB để tránh nghẽn bộ nhớ.

**Non-Goals:**
- Không hỗ trợ check header (chỉ body).
- Không parse JSON tự động (user dùng `contains`/`matches` trên string body).
- Không áp dụng cho `ontime-service` ở giai đoạn này.

## Decisions

1. **Thêm field `body_check_expr` vào proto `EndpointData`** (field 8, string).
   - Lý do: endpoint config đã nằm trong proto, cách sạch nhất để truyền xuống ping-service.

2. **Env truyền vào expr**: `map[string]any{ "body": string(bodyBytes), "status": response.StatusCode }`.
   - Chỉ string body + status. Đơn giản, expr-lang hỗ trợ `contains`/`matches` tốt trên string.
   - Không pre-parse JSON: body có thể không phải JSON, parse thất bại sẽ nhiễu.

3. **Signature `Ping` đổi thành** `(statusCode int, bodyOK bool, err error)`.
   - `bodyOK = true` khi không có expr (backward compatible).
   - `bodyOK = false` khi expr eval false HOẶC expr lỗi (fail-safe).

4. **Compile mỗi lần ping**: gọi `expr.Compile` rồi `expr.Run` trực tiếp bên trong `Ping` cho mỗi lần ping. KHÔNG cache compiled program.
   - Lý do: bỏ complexity cache/mutex; expr compile rất nhanh, endpoint interval thường ≥30s nên cost compile lại không đáng kể, đổi lấy code đơn giản, không state, không race condition.

5. **Giới hạn 1MB**: dùng `io.LimitReader(response.Body, 1<<20)` rồi `io.ReadAll`. Phần vượt discard tự động.

6. **Tách struct `BodyChecker` riêng biệt (single responsibility)**: logic expr nằm trong struct mới `BodyChecker`, không nhét vào `PingWorker`.
   - `PingWorker` chỉ lo HTTP request + đọc body (trách nhiệm hiện tại).
   - `BodyChecker` struct mới, method `Check(body string, status int) (ok bool, err error)`: compile+run expr mỗi lần gọi, fail-safe khi lỗi.
   - `PingWorker.Ping` khởi tạo/truyền `BodyChecker` (qua DI `samber/do` như các dependency khác), chỉ gọi `bodyChecker.Check(...)` khi `ep.BodyCheckExpr != ""`.
   - Lý do: tách biệt rõ trách nhiệm, dễ test `BodyChecker` độc lập, `PingWorker` không phình to.

## Risks / Trade-offs

- [Expr user viết infinity loop / slow] → Không có timeout native trong expr-lang. Mitigation: giới hạn size body nhỏ + doc khuyên viết biểu thức đơn giản; chấp nhận rủi ro low do nội bộ.
- [Body lớn 1MB vẫn tốn bộ nhớ] → 1MB là cap cố định, chấp nhận.
- [Expr lỗi do body không như预期] → fail-safe DOWN, an toàn hơn false-positive UP.

## Migration Plan

1. Thêm field proto, regenerate, deploy `common/proto` + `endpoint-service` (proto backward compatible: field mới optional).
2. Deploy `ping-service` với logic mới. Endpoint chưa set expr → hành vi không đổi.
3. Rollback: revert ping-service, expr field bị ignore.

## Open Questions

<!-- none -->
