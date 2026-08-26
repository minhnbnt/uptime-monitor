# Delta Spec: ontime-unknown-stats

## Purpose

Định nghĩa cách hệ thống đo và báo cáo cửa sổ uptime chứa trạng thái unknown (agent im lặng): thời gian unknown được đo tách biệt khỏi thời gian online, cửa sổ mù trọn vẹn được coi là không có dữ liệu, số giây unknown được trả về qua REST API, và kết quả per [endpoint, day] được cache dưới dạng hash Redis kèm TTL.

## ADDED Requirements

### Requirement: Đo thời gian unknown tách khỏi thời gian online

Trong một cửa sổ quan sát, hệ thống SHALL đo tổng thời gian trạng thái unknown thành một chỉ số riêng (`unknown_seconds`), tách biệt với thời gian online (`online_seconds`). Khoảng unknown SHALL NOT được tính vào thời gian online, và SHALL NOT phá vỡ tính liên tục của các khoảng ON/OFF trước và sau nó.

#### Scenario: Unknown chia đôi cửa sổ

- **WHEN** một server ON từ đầu cửa sổ, chuyển thành unknown giữa giờ và ở lại unknown đến hết cửa sổ
- **THEN** `online_seconds` bằng đúng độ dài đoạn trước unknown và `unknown_seconds` bằng đúng độ dài đoạn còn lại; hai giá trị cộng lại bằng độ dài khoảng quan sát

#### Scenario: Ngày không có unknown

- **WHEN** cửa sổ chỉ chứa event ON/OFF
- **THEN** `unknown_seconds` bằng 0 và `online_seconds` được tính như trước đây

### Requirement: Cửa sổ mù trọn vẹn coi là không có dữ liệu

Khi thời gian unknown phủ trọn khoảng quan sát của cửa sổ (so khớp trong dung sai sai số float), cửa sổ đó SHALL được báo là không có dữ liệu (`has_data = false`) thay vì mang một trạng thái cũ vô căn cứ. Cửa sổ chỉ unknown một phần SHALL vẫn được báo là có dữ liệu (`has_data = true`).

#### Scenario: Trạng thái cũ không chạy băng qua khoảng mù

- **WHEN** sự kiện biết cuối cùng của server nằm trước cửa sổ, mọi dữ liệu trong cửa sổ là unknown
- **THEN** cửa sổ trả về `has_data = false` thay vì tính uptime từ trạng thái cũ

#### Scenario: Unknown một phần vẫn là có dữ liệu

- **WHEN** cửa sổ chứa cả khoảng ON/OFF lẫn khoảng unknown
- **THEN** cửa sổ trả về `has_data = true` cùng `online_seconds` và `unknown_seconds` tương ứng

### Requirement: API trả số giây unknown cho từng điểm thống kê

Response thống kê ontime theo ngày SHALL chứa thêm trường `unknown_seconds` — số giây unknown thô (số thực) của ngày đó. Ngày không có unknown SHALL trả `unknown_seconds` bằng 0. Trường này đi kèm `stats`, `has_data` hiện có mà không đổi ý nghĩa của chúng.

#### Scenario: Response chứa unknown_seconds

- **WHEN** client gọi endpoint thống kê ontime của một server
- **THEN** mỗi phần tử thống kê theo ngày chứa `unknown_seconds`; ngày có khoảng unknown phản ánh đúng số giây đã đo

### Requirement: Cache kết quả ngày dạng hash kèm TTL

Kết quả [endpoint, day] SHALL được cache trong Redis dưới dạng hash gồm các trường trạng thái có dữ liệu, tỷ lệ uptime và số giây unknown, với thời hạn sống: ngày đã qua được giữ lâu hơn ngày hiện tại. Giá trị cache SHALL giữ đủ độ chính xác của phép tính (không làm tròn trước khi lưu). Các entry cache tạo bởi định dạng cũ SHALL NOT gây lỗi: chúng bị coi là cache miss và được ghi đè bằng định dạng mới.

#### Scenario: Cache hit trả đủ bộ giá trị

- **WHEN** kết quả [endpoint, day] đã được cache rồi được đọc lại
- **THEN** cả ba giá trị has_data, uptime và unknown_seconds được trả về đúng như lúc ghi, giữ nguyên độ chính xác

#### Scenario: Entry định dạng cũ không phá hỏng luồng

- **WHEN** Redis còn giữ entry định dạng cũ cho một [endpoint, day]
- **THEN** hệ thống đọc không lỗi, coi như miss và ghi đè bằng hash định dạng mới
