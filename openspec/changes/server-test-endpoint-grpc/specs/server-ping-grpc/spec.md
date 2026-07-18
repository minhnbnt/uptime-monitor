## ADDED Requirements

### Requirement: PingService gRPC endpoint
`ping-service` SHALL expose một gRPC service `PingService` với RPC `Ping` nhận `PingRequest` và trả `PingResponse{ status_code, error }`.

#### Scenario: Successful ping
- **WHEN** request hợp lệ và endpoint trả status code khớp (và body expr pass nếu có)
- **THEN** `PingResponse` có `status_code` thực tế và `error` rỗng

#### Scenario: Transport failure
- **WHEN** không gửi được request (timeout/DNS/connection refused)
- **THEN** `PingResponse.error` bắt đầu bằng `ping error:` và `status_code = 0`

#### Scenario: Check failure
- **WHEN** request thành công nhưng status code hoặc body expr không khớp
- **THEN** `PingResponse.error` bắt đầu bằng `check failed:` và `status_code` là code thực tế

### Requirement: Server test uses shared ping engine
`server-service.TestEndpoint` SHALL gọi `ping-service.Ping` qua gRPC thay vì tự thực hiện HTTP, và áp dụng cùng logic check (status code + body expr).

#### Scenario: Test with body expr
- **WHEN** `TestEndpointRequest` chứa `body_check_expr` và body thực tế không khớp
- **THEN** response `Success = false` và `Error` mang thông tin check failure

#### Scenario: Test without body expr
- **WHEN** `TestEndpointRequest` không có `body_check_expr`
- **THEN** chỉ check status code, behaviour như cũ

### Requirement: Body check expr propagated
Trường `body_check_expr` SHALL được truyền từ `TestEndpointRequest`/`SetCheckMethodRequest` qua gRPC `PingRequest` tới engine check.

#### Scenario: Expr forwarded
- **WHEN** user gửi `body_check_expr` trong request test
- **THEN** giá trị được đưa vào `PingRequest.body_check_expr` và đánh giá bởi `ResponseChecker`

### Requirement: Endpoint model carries body check expr
`server-service` domain `Endpoint` SHALL có trường `body_check_expr` (nullable) tương ứng với cột DB.

#### Scenario: Model field present
- **WHEN** endpoint được lưu/đọc với `body_check_expr`
- **THEN** trường được map đúng vào domain model và DTO

### Requirement: Redundant ping client removed
Sau khi `TestEndpoint` chuyển sang gRPC, `server-service/internal/infrastructure/pingclient.go` SHALL bị xoá.

#### Scenario: No reference remains
- **WHEN** triển khai hoàn tất
- **THEN** không còn file hay reference nào tới `infrastructure.PingURL`
