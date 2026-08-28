## Purpose

Giữ cho các CDC consumer (ontime ownership từ bảng `servers`, ping endpoint từ bảng `endpoints`) không mất update khi worker chết và không áp lại các sự kiện CDC lỗi thời do Redis Stream gửi lại.

## ADDED Requirements

### Requirement: Reclaim pending CDC messages

Consumer CDC SHALL định kỳ nhận lại (reclaim) các message đã được deliver cho consumer group nhưng chưa được acknowledge quá một ngưỡng idle (mặc định 1 phút), dùng Redis `XAutoClaim`, và SHALL xử lý rồi acknowledge chúng. Bước reclaim SHALL chạy sau mỗi batch đọc bình thường.

#### Scenario: Worker chết để lại message pending

- **WHEN** một message đã được deliver cho một worker đã chết và nằm pending quá ngưỡng idle
- **THEN** consumer sống thực hiện reclaim, xử lý message đó và acknowledge nó (không mất update)

#### Scenario: Không có pending message nào quá hạn

- **WHEN** không có message pending nào vượt quá ngưỡng idle
- **THEN** bước reclaim không reclaim message nào, không báo lỗi, và không xử lý trùng

### Requirement: Deduplicate out-of-order CDC messages

Consumer CDC SHALL lưu trữ last-applied stream message id (định dạng `ms-seq`) cho mỗi entity (server cho ontime, endpoint cho ping) trong Redis với TTL, và SHALL bỏ qua (acknowledge mà không áp dụng) mọi message có id cũ hơn offset đã lưu. Sau khi áp thành công, SHALL cập nhật offset mới.

#### Scenario: Message cũ đến sau message mới đã áp dụng

- **WHEN** offset của entity E là message id `M2` và một message có id `M1` (`M1` < `M2`) cho entity E đến
- **THEN** consumer acknowledge message đó mà không gọi handler (không áp lại trạng thái cũ)

#### Scenario: Message mới hoặc chưa có offset

- **WHEN** chưa có offset cho entity E, hoặc message có id mới hơn offset đã lưu, và áp thành công
- **THEN** consumer gọi handler và cập nhật offset thành id của message vừa áp dụng

### Requirement: Permanent malformed messages go to dead-letter

Consumer CDC SHALL coi một message không thể parse đủ để xác định entity id (không có `before` cũng không có `after`) là lỗi permanent và SHALL route nó vào dead-letter stream, acknowledge nó (không retry vô hạn).

#### Scenario: Message thiếu before và after

- **WHEN** payload của một message không có `before` cũng không có `after`
- **THEN** message được ghi vào dead-letter stream và được acknowledge (không quay lại vòng lặp retry)
