## Purpose

Xác thực người dùng dựa trên GoTrue: người dùng đăng ký/đăng nhập bằng email qua GoTrue, và các service xác minh JWT của người dùng qua OIDC discovery, dùng UUID làm định danh người dùng (model Go `uuid.UUID`, wire gRPC `string`).

## ADDED Requirements

### Requirement: Đăng ký và đăng nhập bằng email
Hệ thống SHALL cho phép người dùng đăng ký và đăng nhập bằng email + password thông qua GoTrue. Sau khi đăng nhập thành công, hệ thống SHALL cấp cho client một access token và refresh token của GoTrue. Hệ thống SHALL NOT yêu cầu username hoặc tên hiển thị khi đăng ký.

#### Scenario: Đăng ký tài khoản mới
- **WHEN** client gửi email và password hợp lệ tới endpoint signup của GoTrue
- **THEN** GoTrue tạo tài khoản và trả về access token cùng refresh token

#### Scenario: Đăng nhập sai mật khẩu
- **WHEN** client đăng nhập với email tồn tại nhưng sai password
- **THEN** GoTrue trả về lỗi xác thực và không cấp token

#### Scenario: Làm mới access token hết hạn
- **WHEN** access token hết hạn và client gửi refresh token hợp lệ tới GoTrue
- **THEN** GoTrue trả về access token mới

### Requirement: Verify JWT ở mỗi service qua OIDC
Mọi service HTTP của hệ thống SHALL xác minh access token của request bằng OIDC discovery từ issuer GoTrue trước khi xử lý yêu cầu. Request thiếu token hoặc token không hợp lệ SHALL bị từ chối với lỗi 401.

#### Scenario: Request hợp lệ
- **WHEN** client gọi endpoint cần xác thực với access token hợp lệ của GoTrue
- **THEN** service xác minh token thành công và xử lý yêu cầu với user id từ token

#### Scenario: Thiếu access token
- **WHEN** client gọi endpoint cần xác thực mà không kèm access token
- **THEN** service trả về lỗi 401 và không xử lý yêu cầu

#### Scenario: Access token không hợp lệ hoặc hết hạn
- **WHEN** client gọi endpoint cần xác thực với access token sai, bị thu hồi, hoặc hết hạn
- **THEN** service trả về lỗi 401 và không xử lý yêu cầu

### Requirement: Định danh người dùng là UUID
Mỗi người dùng SHALL được định danh bằng một UUID (claim `sub` trong token của GoTrue). Các service SHALL dùng `uuid.UUID` (Go) làm `user_id` khi lưu và truy vấn dữ liệu; trường user id trong message gRPC SHALL là chuỗi UUID (wire format `string`). Dữ liệu cũ dùng số nguyên SHALL NOT được tiếp tục sử dụng.

#### Scenario: Lưu dữ liệu theo user id UUID
- **WHEN** service lưu hoặc truy vấn dữ liệu thuộc về người dùng
- **THEN** service dùng `uuid.UUID` của người dùng làm giá trị user id

#### Scenario: Truyền user id qua gRPC
- **WHEN** service gọi service khác qua gRPC với ngữ cảnh người dùng
- **THEN** trường user id trong message gRPC là chuỗi UUID

### Requirement: Lấy thông tin người dùng qua GoTrue Admin API
notification-service SHALL lấy email của người dùng bằng GoTrue Admin API (`GET /admin/users/{uuid}`) với service_role JWT khi cần gửi thông báo qua email. Không còn endpoint private nào của auth-service cho việc này.

#### Scenario: Gửi digest email cho người dùng
- **WHEN** notification-service cần gửi digest email cho một user id UUID
- **THEN** service gọi GoTrue Admin API bằng service_role JWT để lấy email của người dùng, và gửi email tới địa chỉ đó
