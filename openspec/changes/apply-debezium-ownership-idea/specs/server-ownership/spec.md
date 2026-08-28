## ADDED Requirements

### Requirement: Ánh xạ sự kiện tạo/snapshot sang quyền sở hữu
Consumer Debezium MUST duy trì bản ghi `server_owners` (server_id → user_id) khi nhận sự kiện CDC `c` (create) hoặc `r` (snapshot/read) từ bảng `servers` có payload `after` hợp lệ. Mỗi server chỉ thuộc về đúng một user (upsert).

#### Scenario: Tạo server mới
- Given một server chưa có bản ghi `server_owners`
- When consumer nhận sự kiện `c` với `after = {id: 5, created_by_id: 10}`
- Then bản ghi `server_owners` (server_id=5, user_id=10) được tạo

#### Scenario: Snapshot khi khởi động
- Given Debezium gửi sự kiện snapshot `r` cho server đã tồn tại
- When consumer nhận sự kiện `r` với `after = {id: 5, created_by_id: 10}`
- Then bản ghi `server_owners` được upsert (giữ nguyên hoặc cập nhật user_id)

### Requirement: Route soft-delete vào xoá quyền sở hữu
Khi nhận sự kiện UPDATE (`u`) mà payload `after.deleted_at` khác `NULL` (server bị soft-delete ở nguồn), consumer MUST xoá bản ghi `server_owners` tương ứng thay vì cập nhật. Đường xử lý này MUST giống với sự kiện DELETE thật.

#### Scenario: Soft-delete server
- Given bản ghi `server_owners` (server_id=5) tồn tại
- When consumer nhận sự kiện `u` với `after = {id: 5, deleted_at: <timestamp>}`
- Then bản ghi `server_owners` (server_id=5) bị xoá (soft-delete)

### Requirement: Xoá quyền sở hữu khi server bị xoá
Khi nhận sự kiện DELETE (`d`) từ bảng `servers`, consumer MUST xoá bản ghi `server_owners` của server đó.

#### Scenario: Hard-delete server
- Given bản ghi `server_owners` (server_id=5) tồn tại
- When consumer nhận sự kiện `d` với `before = {id: 5}`
- Then bản ghi `server_owners` (server_id=5) bị xoá

### Requirement: Xoá quyền sở hữu phải idempotent
Thao tác xoá quyền sở hữu MUST là no-op (không báo lỗi) khi bản ghi `server_owners` tương ứng không tồn tại hoặc đã bị xoá trước đó. Điều này bao phủ các trường hợp Redis Stream gửi lại (redelivery) hoặc duplicate event, để consumer luôn có thể ack message.

#### Scenario: Xoá trùng lặp
- Given bản ghi `server_owners` (server_id=5) đã bị xoá trước đó
- When consumer nhận thêm một sự kiện xoá cho server_id=5 (redelivery/duplicate)
- Then thao tác trả về thành công (no-op), consumer ack message bình thường, không báo lỗi

#### Scenario: Xoá server không sở hữu
- Given không có bản ghi `server_owners` nào cho server_id=99
- When consumer nhận sự kiện xoá cho server_id=99
- Then thao tác trả về thành công (no-op)

### Requirement: Cập nhật thông thường vẫn upsert
Khi nhận sự kiện UPDATE (`u`) mà `after.deleted_at` là `NULL`, consumer MUST upsert `server_owners` (cập nhật `user_id` chủ sở hữu) như với sự kiện tạo.

#### Scenario: Đổi chủ sở hữu
- Given bản ghi `server_owners` (server_id=5, user_id=10)
- When consumer nhận sự kiện `u` với `after = {id: 5, created_by_id: 20, deleted_at: null}`
- Then bản ghi `server_owners` được cập nhật thành (server_id=5, user_id=20)
