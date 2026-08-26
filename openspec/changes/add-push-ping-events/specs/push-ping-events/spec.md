# Delta Spec: push-ping-events

## Purpose

Cung cấp kênh push để agent chủ động gửi event trạng thái on/off của server tới ping-service qua REST, xác thực bằng JWT scope `ping`, với rate-limit theo session dựa trên băm `session_id` — dùng chung pipeline ghi nhận event với cơ chế pull hiện có.

## ADDED Requirements

### Requirement: Endpoint nhận batch push event

Hệ thống SHALL cung cấp endpoint `POST /api/v1/ping/events` nhận mảng event dạng `[{"id": number, "status": "ON"|"OFF"}]` trong đó `id` là ID của server. Mỗi event KHÔNG chứa timestamp — hệ thống SHALL gắn thời điểm nhận tại thời gian xử lý. Response chấp nhận SHALL kèm `stale_at` (unix milliseconds): mốc mà nếu agent không push lại trước thời điểm đó, các server vừa được chấp nhận sẽ bị coi là im lặng (phát UNKNOWN). Request với body rỗng hoặc JSON không hợp lệ SHALL bị từ chối với mã 400.

#### Scenario: Gửi batch event hợp lệ
- **WHEN** agent gửi POST `/api/v1/ping/events` với body `[{"id":1,"status":"ON"}]` kèm token hợp lệ scope `ping`
- **THEN** hệ thống trả về 200 với `next_time` (thời điểm được phép gửi tiếp theo, đơn vị unix milliseconds), `accepted` chứa danh sách ID đã ghi nhận, và `stale_at` xấp xỉ thời điểm nhận cộng khoảng stale

#### Scenario: Body không hợp lệ
- **WHEN** agent gửi body rỗng, mảng rỗng, hoặc JSON sai cấu trúc
- **THEN** hệ thống trả về 400 và không ghi nhận event nào

### Requirement: Xác thực JWT và kiểm tra scope ping

Endpoint push SHALL yêu cầu xác thực qua cơ chế forward-auth hiện có: JWT hợp lệ do auth-service phát. Token MUST chứa scope `ping`; request thiếu scope SHALL bị từ chối với mã 403, token không hợp lệ SHALL bị từ chối với mã 401. Hệ thống SHALL sử dụng `session_id` (claim `sid`) từ JWT cho việc tính toán lịch push; request từ token không chứa `sid` SHALL bị từ chối.

#### Scenario: Token đúng scope
- **WHEN** agent dùng token lấy từ API tạo ping-session của auth-service (scope `ping`)
- **THEN** request được chấp nhận xử lý

#### Scenario: Token thiếu scope ping
- **WHEN** agent dùng token chỉ có scope `app`
- **THEN** hệ thống trả về 403 và không ghi nhận event nào

### Requirement: Kiểm tra quyền sở hữu server qua ánh xạ ID

Mỗi event tham chiếu server theo ID. Hệ thống SHALL ánh xạ ID → endpoint trong phạm vi các server thuộc sở hữu của user trong token. Event tham chiếu ID không tồn tại hoặc không thuộc owner SHALL không được ghi nhận và được báo lỗi trong response (báo chung một lý do, không lộ sự tồn tại của server người khác).

#### Scenario: ID server không tồn tại hoặc không thuộc owner
- **WHEN** batch chứa ID không khớp bất kỳ server nào của user
- **THEN** item đó nằm trong mảng `errors` với lý do phù hợp, các item hợp lệ khác vẫn được ghi nhận

### Requirement: Response một phần khi batch có lỗi

Khi batch chứa ít nhất một item lỗi nhưng cũng có item hợp lệ, hệ thống SHALL trả về 207 với danh sách item đã chấp nhận (`accepted`) và mảng `errors` mô tả từng item lỗi (`id`, `error`). Item lỗi phổ biến: ID không tìm thấy/không thuộc owner, status sai định dạng.

#### Scenario: Batch một phần thành công
- **WHEN** batch gồm 2 ID hợp lệ và 1 ID lạ
- **THEN** response 207 chứa `accepted` với 2 ID, `errors` với 1 entry, và 2 event hợp lệ đã được ghi nhận

### Requirement: Rate limit theo session dựa trên băm session_id

Sau khi chấp nhận ít nhất một event, hệ thống SHALL tính `next_time = NextExecutionTime(session_id, 30s)` bằng hàm băm FNV-64a lấy modulo 30 giây (cùng lưới băm với cơ chế cập nhật score của poll) và lưu làm mốc chặn. Request đến trước mốc `next_time` của session SHALL bị từ chối với mã 429 kèm `next_time` trong body. Batch mà tất cả item đều lỗi SHALL không thiết lập mốc chặn mới (agent sửa được lỗi thì gửi lại ngay). Agent gửi muộn hơn mốc SHALL được chấp nhận bình thường.

#### Scenario: Gửi lại quá sớm
- **WHEN** agent gửi request thứ hai trước `next_time` vừa nhận
- **THEN** hệ thống trả về 429 kèm `next_time` và không ghi nhận event nào

#### Scenario: Gửi sau next_time
- **WHEN** agent gửi request tại hoặc sau `next_time`
- **THEN** request được xử lý bình thường và nhận mốc `next_time` mới

#### Scenario: Batch toàn lỗi không khóa cửa sổ
- **WHEN** agent gửi batch mà mọi item đều lỗi ngay lần đầu
- **THEN** response trả về lỗi nhưng không thiết lập `next_time`; request sửa lỗi gửi lại ngay lập tức được chấp nhận

### Requirement: Ghi nhận event dùng chung pipeline với pull

Event push hợp lệ SHALL đi vào cùng đường ghi nhận với event pull: gắn thời điểm nhận, so sánh với trạng thái gần nhất, chỉ tạo event thay đổi trạng thái, và forward thay đổi sang ontime-service. Cơ chế pull SHALL tiếp tục hoạt động độc lập song song trên cùng server.

#### Scenario: Hai kiểu ping đồng thời
- **WHEN** một server đang được pull định kỳ đồng thời nhận push từ agent
- **THEN** cả hai nguồn đều cập nhật trạng thái qua cùng pipeline; event chỉ phát ra khi trạng thái thực sự thay đổi

#### Scenario: Push lặp cùng trạng thái
- **WHEN** agent push status ON cho server vốn đã ở trạng thái ON
- **THEN** không có event mới phát sinh downstream nhưng request vẫn tính là accepted và nhận `next_time`

### Requirement: Ghi nhận độ tươi của server qua zset

Trước khi ghi nhận bất kỳ event trạng thái nào cho một server (từ kênh push lẫn kênh pull), hệ thống SHALL cập nhật score của server đó trong một zset Redis thành `now + khoảng thời gian tươi`, trong đó khoảng tươi của event push là hằng số (~90s) và của event pull là interval của endpoint cộng thêm cùng hằng số đó. State SHALL nằm trong Redis để tồn tại qua lần restart ping-service.

#### Scenario: Push làm mới hạn stale
- **WHEN** agent push event hợp lệ được ghi nhận
- **THEN** score của server trong zset freshness được đặt lại thành `now + ~90s` trước khi event đi vào pipeline dedupe

#### Scenario: Pull giữ server sống khi agent tắt app
- **WHEN** agent ngừng push nhưng server vẫn được pull probe định kỳ và ghi event
- **THEN** score freshness tiếp tục được làm mới theo từng lần poll record, server không bị đánh unknown oan

### Requirement: Worker phát hiện và đánh dấu stale

Hệ thống SHALL chạy worker quét zset freshness: claim nguyên tử tối đa 10 entry có score ≤ now đồng thời bump score các entry đó thành `now + 10s` (chống worker khác lấy trùng), sau đó ghi 1 event UNKNOWN cho từng server. Với từng entry ghi UNKNOWN thành công, hệ thống SHALL xóa entry khỏi zset; nếu ghi lỗi SHALL giữ nguyên entry để tự thử lại sau ít nhất 10s. Nếu lượt claim trả đủ 10 entry, worker SHALL lặp ngay không sleep; ngược lại sleep bằng khoảng cách đến entry gần nhất nhưng không quá 30s. Status `UNKNOWN` đi qua cùng pipeline ghi nhận như ON/OFF.

#### Scenario: Agent im lặng quá hạn
- **WHEN** một server không được ghi nhận event nào trong khoảng thời gian tươi
- **THEN** worker phát 1 event UNKNOWN cho server đó rồi xóa entry; khi agent push lại, entry được tái tạo và trạng thái thật tiếp tục được ghi nhận

#### Scenario: Ghi UNKNOWN thất bại
- **WHEN** việc record event UNKNOWN bị lỗi (ví dụ mất kết nối downstream)
- **THEN** entry không bị xóa, score đã được bump thành `now + 10s` nên worker khác không lấy trùng và hệ thống tự thử lại ở vòng sau
