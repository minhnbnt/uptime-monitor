## Why

Hiện tại `ping-service` chỉ đánh giá tình trạng endpoint dựa trên HTTP status code (`ExpectedCode`). Nhiều service trả về 200 nhưng nội dung body lại lỗi (vd: trang báo lỗi, JSON rỗng, thiếu field). Người dùng cần một cách để validate nội dung response body mà không phải tự viết code.

## What Changes

- Thêm trường cấu hình `body_check_expr` (chuỗi biểu thức expr-lang) cho mỗi Endpoint.
- Khi một Endpoint có `body_check_expr`, `ping-service` đọc response body (giới hạn 1MB), chạy biểu thức expr với env `{ body: string, status: int }`, và coi kết quả boolean để quyết định status.
- Nếu `body_check_expr` rỗng → hành vi giữ nguyên (chỉ check status code).
- Nếu biểu thức lỗi cú pháp / lỗi runtime → **fail-safe**: coi như endpoint DOWN (ghi nhận sự cố thay vì bỏ qua).
- Biểu thức được compile một lần (cache) thay vì compile mỗi lần ping.

## Capabilities

### New Capabilities
- `response-body-check`: Khả năng định nghĩa và đánh giá biểu thức expr để validate nội dung response body của một Endpoint, bao gồm giới hạn kích thước, compile cache và fail-safe khi lỗi.

### Modified Capabilities
<!-- none -->

## Impact

- `common/proto/endpoint/v1/endpoint_service.proto`: thêm field `body_check_expr` (regenerate code).
- `ping-service/internal/domain/endpoint.go`: thêm field `BodyCheckExpr`.
- `ping-service/internal/infrastructure/grpcclient/endpoint_client.go`: map field mới từ proto.
- `ping-service/internal/infrastructure/ping.go`: đọc body, chạy expr, thay đổi signature trả về.
- `ping-service/internal/handler/zsetworker.go`: logic quyết định `StatusOn/Off` kết hợp body check.
- Thêm dependency `github.com/expr-lang/expr` vào `ping-service/go.mod`.
