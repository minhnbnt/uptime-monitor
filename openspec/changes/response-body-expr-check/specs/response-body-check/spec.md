## ADDED Requirements

### Requirement: Endpoint configures body check expression
Mỗi Endpoint SHALL có thể mang một biểu thức expr-lang tùy chọn (`body_check_expr`) dùng để validate nội dung response body. Khi trường này rỗng, hệ thống SHALL bỏ qua body check.

#### Scenario: Endpoint without expression
- **WHEN** một Endpoint có `body_check_expr` rỗng được ping
- **THEN** hệ thống chỉ đánh giá dựa trên status code, body không được kiểm tra

#### Scenario: Endpoint with expression
- **WHEN** một Endpoint có `body_check_expr = 'body contains "OK"'` được ping và body chứa "OK"
- **THEN** body check trả về hợp lệ (bodyOK = true)

### Requirement: Body read is size-limited
Hệ thống SHALL đọc tối đa 1MB của response body để đưa vào biểu thức expr. Phần vượt quá 1MB SHALL bị bỏ qua.

#### Scenario: Body exceeds 1MB
- **WHEN** response body lớn hơn 1MB
- **THEN** chỉ 1MB đầu được đưa vào biểu thức expr, phần còn lại bị discard

### Requirement: Expression evaluates against body and status
Hệ thống SHALL chạy biểu thức expr với env chứa `body` (string) và `status` (int của status code).

#### Scenario: Expression true
- **WHEN** biểu thức eval trả về true
- **THEN** bodyOK = true và endpoint không bị coi là down vì lý do body

#### Scenario: Expression false
- **WHEN** biểu thức eval trả về false
- **THEN** bodyOK = false và endpoint được ghi nhận DOWN

### Requirement: Expression error is fail-safe
Khi biểu thức lỗi cú pháp hoặc lỗi runtime trong quá trình eval, hệ thống SHALL coi body check là thất bại (bodyOK = false) và ghi nhận endpoint DOWN.

#### Scenario: Invalid expression
- **WHEN** `body_check_expr` không compile được hoặc gây lỗi runtime
- **THEN** bodyOK = false và endpoint được ghi nhận DOWN (fail-safe)

### Requirement: Expression is compiled per ping
Hệ thống SHALL compile biểu thức expr mỗi lần thực hiện ping (không cache compiled program giữa các lần ping).

#### Scenario: Repeated pings compile each time
- **WHEN** cùng một Endpoint được ping nhiều lần với cùng `body_check_expr`
- **THEN** biểu thức được compile lại ở mỗi lần ping (không tái sử dụng cached program)
