## 1. Proto & Domain

- [ ] 1.1 Thêm field `body_check_expr` (string, field 8) vào `EndpointData` trong `common/proto/endpoint/v1/endpoint_service.proto`
- [ ] 1.2 Regenerate proto code (`make gen-proto` hoặc `buf generate`)
- [ ] 1.3 Thêm field `BodyCheckExpr string` vào `ping-service/internal/domain/endpoint.go`
- [ ] 1.4 Map field mới từ proto sang domain trong `ping-service/internal/infrastructure/grpcclient/endpoint_client.go`

## 2. Body check implementation

- [ ] 2.1 Thêm dependency `github.com/expr-lang/expr` vào `ping-service/go.mod` (`go get`)
- [ ] 2.2 Đổi `PingWorker.Ping` thành `(statusCode int, bodyOK bool, err error)`: đọc body qua `io.LimitReader` 1MB
- [ ] 2.3 Tạo struct mới `BodyChecker` (file riêng `internal/infrastructure/bodychecker.go`), method `Check(body string, status int) (bool, error)`: compile+run expr mỗi lần gọi, env `{body, status}`, fail-safe khi lỗi; register qua `samber/do`
- [ ] 2.4 `PingWorker` inject `BodyChecker`, gọi `Check` khi `ep.BodyCheckExpr != ""`, ngược lại `bodyOK = true` (backward compatible)

## 3. Wiring & decision logic

- [ ] 3.1 Cập nhật `PingService.Ping` signature và `handler.PingService` interface
- [ ] 3.2 Cập nhật logic `zsetworker.go:73`: down nếu `pingErr != nil || statusCode != ExpectedCode || !bodyOK`

## 4. Tests & verification

- [ ] 4.1 Unit test `PingWorker.Ping`: expr đúng, expr sai, không có expr, expr lỗi cú pháp, body > 1MB
- [ ] 4.2 Build ping-service (`go build ./...`) và chạy test (`go test ./...`)
