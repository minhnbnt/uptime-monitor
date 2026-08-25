# Delta Spec: session-management

## Purpose

Cho phép người dùng liệt kê các phiên đăng nhập (session) của chính mình và thu hồi từng session qua REST API của auth-service, xác thực bằng access token scope `app` — phục vụ việc quản lý thiết bị đăng nhập và session agent do `CreatePingSession` sinh ra.

## ADDED Requirements

### Requirement: Liệt kê session của user

Hệ thống SHALL cung cấp `GET /api/v1/auth/sessions` trả về danh sách session thuộc sở hữu user trong token, sắp xếp theo created_at giảm dần. Endpoint SHALL hỗ trợ phân trang qua query `page` (mặc định 1, tối thiểu 1) và `per_page` (mặc định 20, tối thiểu 1, tối đa 100) theo cùng quy ước với route list servers. Mỗi mục SHALL gồm: `id` (UUID của JTI), `scopes` (mảng), `current` (boolean), `created_at`, `expires_at`. Response SHALL là `{ "data": [...], "meta": { "page", "per_page", "total" } }` với `total` là tổng số session chưa hết hạn của user bất kể trang hiện tại.

#### Scenario: User có nhiều session

- **WHEN** user gọi GET `/api/v1/auth/sessions` với token hợp lệ scope `app`
- **THEN** hệ thống trả về 200 với các session chưa hết hạn của user đó, mỗi mục đủ các trường trên

#### Scenario: Phân trang

- **WHEN** user có 45 session chưa hết hạn và gọi với `page=2&per_page=20`
- **THEN** hệ thống trả về đúng 20 mục của trang 2 theo thứ tự created_at giảm dần và `meta = {page: 2, per_page: 20, total: 45}`

#### Scenario: Tham số phân trang mặc định

- **WHEN** user gọi endpoint không kèm `page`/`per_page`
- **THEN** hệ thống áp dụng mặc định page=1, per_page=20

### Requirement: Đánh dấu session hiện tại

Mục session trùng với claim `sid` của token dùng để gọi SHALL có `current = true`; các mục khác SHALL có `current = false`.

#### Scenario: Nhận diện phiên đang dùng

- **WHEN** user gọi endpoint bằng token của một session đang hoạt động
- **THEN** đúng một mục trong danh sách có `current = true` và khớp với thiết bị đang gọi

### Requirement: Ẩn session đã hết hạn

Session có `expires_at` trước thời điểm hiện tại SHALL KHÔNG xuất hiện trong danh sách.

#### Scenario: Session cũ hết hạn

- **WHEN** danh sách chứa session có expires_at đã quá hạn
- **THEN** session đó bị loại khỏi response

### Requirement: Thu hồi một session

Hệ thống SHALL cung cấp `DELETE /api/v1/auth/sessions/{sessionId}` xóa đúng một session khi nó thuộc sở hữu user trong token (`user_id` khớp). Xóa thành công SHALL trả về 204. Sau thu hồi, refresh token gắn với session SHALL bị từ chối ngay lập tức; access token đang còn hạn được chấp nhận còn hiệu lực đến khi hết hạn (đánh đổi có chủ đích, không tra cứu DB mỗi request).

#### Scenario: Thu hồi thành công

- **WHEN** user xóa session của chính mình bằng id hợp lệ
- **THEN** hệ thống trả về 204, session biến mất khỏi danh sách lần sau, refresh token của session đó không thể refresh

#### Scenario: Session không tồn tại hoặc của người khác

- **WHEN** `sessionId` không tồn tại hoặc thuộc user khác
- **THEN** hệ thống trả về 404 với lý do chung, không tiết lộ sự tồn tại của session người khác; các session khác không bị ảnh hưởng

#### Scenario: Thu hồi session hiện tại

- **WHEN** user xóa session có `current = true`
- **THEN** hệ thống vẫn trả về 204 (tương đương đăng xuất phiên này)

### Requirement: Kiểm soát quyền theo scope

Cả hai endpoint SHALL yêu cầu Bearer token hợp lệ. Token thiếu scope `app` (ví dụ token chỉ có scope `ping`) SHALL bị từ chối với mã 403 và không thực thi thao tác nào.

#### Scenario: Token ping gọi quản lý session

- **WHEN** agent dùng token scope `ping` gọi GET hoặc DELETE sessions
- **THEN** hệ thống trả về 403 và không trả danh sách cũng như không xóa gì

#### Scenario: Token không hợp lệ

- **WHEN** request không kèm hoặc kèm token sai/hết hạn
- **THEN** hệ thống trả về 401
