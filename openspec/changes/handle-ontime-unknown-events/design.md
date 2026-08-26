# Design: handle-ontime-unknown-events

## Context

Ping-service (change `add-push-ping-events`, mục 6) phát event `UNKNOWN` khi agent im lặng, forward qua gRPC vào `server_events` của ontime-service (write path lưu raw varchar, không validate). Câu SQL tính uptime (`ontime_uptime.go`) hiện lọc chặt `status IN ('ON','OFF')` ở CTE `known_events` — UNKNOWN bị bỏ qua mà không advance boundary, tức trạng thái ON/OFF cuối cùng được carry-in chạy băng qua khoảng mù. Cache (`ontimecache.go`) đang là string per key: `"__NULL__"` cho no-data hoặc `"%.2f"` uptime, TTL 1h cho ngày đã qua / 10s cho hôm nay. REST API expose qua ogen từ `api/spec.yaml` (schema `OntimeStats`: date/stats/has_data); handler map tại `toOntimeStats`.

## Goals / Non-Goals

**Goals:**

- Đo unknown_seconds tách biệt trong cùng pipeline segment hiện có
- HasData do Go suy ra: unknown trọn cửa sổ ⇒ không có dữ liệu
- unknown_seconds lộ lên REST API
- Cache hash + EXPIRE, bỏ sentinel và bug làm tròn

**Non-Goals:**

- gRPC `OntimeDayStat` (chưa có cả has_data; kéo theo server-service)
- Hiển thị MonitorStatus "UNKNOWN" phía server-service/API enum docs
- Trừ khoảng unknown khỏi mẫu số phần trăm ở tầng server (UI tự quyết cách trình bày)

## Decisions

### D1 — Đưa UNKNOWN vào stream thay vì cộng thêm phép trừ gap ngoài

`known_events` thành `WHERE status IN ('ON','OFF','UNKNOWN')`. LEAD hiện có tự chia segment per status; `unknown_seconds` chỉ là một cột `SUM(...) FILTER (WHERE status = 'UNKNOWN')` song song với online_seconds.
- *Alternative bị loại*: giữ stream ON/OFF rồi trừ các khoảng unknown riêng — phải làm phép trừ gap giữa hai tập interval, phức tạp và dễ lệch.

### D2 — Hệ quả carry-in được chấp nhận có chủ đích

Với UNKNOWN trong stream, carry-in của cửa sổ mở đầu sau khoảng im lặng sẽ là UNKNOWN: cửa sổ bắt đầu "không biết" tới event thật đầu tiên, thay vì mang trạng thái cũ xuyên suốt. Đây chính là hành vi trung thực mong muốn; các test hiện có đều chỉ seed ON/OFF nên không case nào vỡ.

### D3 — Bỏ cột `true AS has_data`, suy HasData bằng Go sau scan

Cột SQL là hằng số (cửa sổ không data không sinh row) — fetch vô ích. Sau `Find`, loop rows: `HasData = UnknownSeconds < total − ε` với `ε = 1e-3` giây (EXTRACT(EPOCH) là float, so khớp tuyệt đối ngẫu nhiên sai). Field `HasData` giữ nguyên trên struct vì batcher đang đọc; semantics thuộc về repo nên consumer không đổi chữ nào.

### D4 — Cache hash: HSET + EXPIRE, suffix bump né WRONGTYPE

Key mới `ontime:{id}:{date}:stats:v2`; fields `has_data`/`uptime`/`unknown` parse bằng strconv hai chiều. MGet = pipeline HGETALL; MSet = pipeline `HSet + Expire` từng cặp. Dual-TTL 1h/10s giữ nguyên. Dùng HSET chứ không HMSET (deprecated). Lưu `strconv.FormatFloat(v,'f',-1,64)` — hết làm tròn `%.2f`, cache khớp số liệu tươi. Sentinel `__NULL__` bị field `has_data` thay thế.
- Key bump là bắt buộc: HGETALL/HSET lên string key cũ → WRONGTYPE; entry cũ tự chết theo TTL tối đa 1h, không cần code dọn.
- *Alternative bị loại*: DEL trước HSET vĩnh viễn (thao tác thừa mãi mãi); JSON string như cũ (vẫn phải marshal/unmarshal + sentinel logic).

### D5 — Passthrough UnknownSeconds dọc đường DTO

`UptimeRow.UnknownSeconds` → `dto.DayResult.Unknown` (cache tự chịu entry cũ thiếu field ⇒ zero) → `dto.OntimeStats.UnknownSeconds` (tag `json:"unknown_seconds"`) → schema `OntimeStats.unknown_seconds` (number/double) → regenerate ogen → `toOntimeStats` mapping. Ba điểm passthrough: `batcher.fillMisses`, `batcher.buildResponse`, `ontime.getServersOntime` (gồm chiều map ngược DayResult).

## Risks / Trade-offs

- [Ý nghĩa uptime đổi với cửa sổ sau khoảng mù] Trước đây hiển thị uptime mang trạng thái cũ; giờ báo no-data cho tới event thật. → Đúng mục tiêu change; UI cần biết để hiển thị "no data" thay vì 100%/0%.
- [Cache flush tự nhiên] Đổi key khiến toàn bộ entry cũ miss trong ~1h đầu deploy. → Chấp nhận, DB là nguồn sự thật, self-healing.
- [`unknown_seconds` đơn vị giây thô] Không phải %, UI tự quy đổi. → Nhất quán với `online_seconds` nội bộ, tránh mất precision sớm.

## Migration Plan

Deploy thuần ontime-service: repo → dto/service → spec.yaml + ogen → cache key v2. Không data migration; entry Redis định dạng chết tự hết hạn ≤1h; rollback an toàn vì DB bất biến.

## Open Questions

Không có — các điểm mở đã chốt: ε=1ms, HasData suy ở repo, gRPC để ngoài scope.
