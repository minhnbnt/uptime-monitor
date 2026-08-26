# Proposal: handle-ontime-unknown-events

## Why

Ping-service vừa thêm kênh phát hiện agent im lặng phát event status `UNKNOWN`, nhưng ontime-service chưa có quan điểm gì với trạng thái này ngoài việc lọc bỏ im lặng ở câu SQL tính uptime. Cần semantics rõ ràng cho khoảng "không biết": đo tách biệt khỏi thời gian ON/OFF, báo lên API để UI trình bày được cửa sổ "partial", và không mang trạng thái cũ chạy băng qua khoảng mù vô hạn.

## What Changes

- Uptime SQL đưa `UNKNOWN` vào stream sự kiện bên cạnh ON/OFF: mỗi khoảng unknown được chia segment riêng nhờ LEAD, đo thành `unknown_seconds` song song với `online_seconds`.
- Bỏ cột hằng số `true AS has_data` khỏi SQL; `HasData` được suy bằng Go sau khi scan: chỉ khi `unknown_seconds` phủ trọn khoảng quan sát (tolerance float 1ms) cửa sổ mới coi là không có dữ liệu.
- `UptimeRow`/`dto.DayResult`/`dto.OntimeStats` thêm trường `UnknownSeconds`; passthrough qua batcher và service layer.
- **BREAKING** (định dạng nội bộ Redis): cache ontime đổi từ string per key (`"__NULL__"` / `"%.2f"`) sang hash (`HSET` fields `has_data`/`uptime`/`unknown`) kèm EXPIRE giữ nguyên dual-TTL 1h/10s; key suffix bump để né WRONGTYPE với entry cũ, entry cũ tự chết theo TTL.
- Cache lưu đủ precision float64 (bỏ làm tròn `%.2f`), sentinel `__NULL__` bị thay bởi field `has_data`.
- REST API `OntimeStats` thêm field `unknown_seconds` (giây thô double); regenerate ogen.
- Ngoài scope: gRPC `OntimeDayStat` (chưa có cả `has_data`), hiển thị MonitorStatus `UNKNOWN` phía server-service/API enum.

## Capabilities

### New Capabilities

- `ontime-unknown-stats`: Cách hệ thống đo và báo cáo cửa sổ uptime chứa trạng thái unknown — chia segment ON/OFF/UNKNOWN, suy HasData từ độ phủ unknown, trả `unknown_seconds` qua REST API, cache hash per [endpoint, day].

### Modified Capabilities

<!-- Không có spec hiện hữu nào trong openspec/specs/ -->

## Impact

- **ontime-service/internal/infrastructure/repository**: `ontime_uptime.go` (SQL + `UptimeRow` + derive HasData), `ontimecache.go` (hash + TTL + key suffix), integration tests tương ứng.
- **ontime-service/internal/dto**: `DayResult.Unknown`, `OntimeStats.UnknownSeconds`.
- **ontime-service/internal/service**: batcher (`fillMisses`, `buildResponse`), `ontime.go` (`getServersOntime` cả chiều map ngược).
- **ontime-service/api/spec.yaml** + regenerate ogen (`generated/api`), handler mapping `toOntimeStats`.
- Không đổi DB schema, không đổi proto/gRPC, không đổi ping-service/server-service.
