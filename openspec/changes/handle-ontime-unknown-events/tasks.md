# Tasks: handle-ontime-unknown-events

## 1. Uptime SQL — đo unknown + HasData suy bằng Go

- [x] 1.1 `known_events` thêm `'UNKNOWN'` vào filter; SELECT cuối bỏ `true AS has_data`, thêm `COALESCE(SUM(...) FILTER (WHERE status = 'UNKNOWN'), 0) AS unknown_seconds`
- [x] 1.2 `UptimeRow` thêm `UnknownSeconds float64`; sau scan trong `BatchGetUptime`, suy `HasData = UnknownSeconds < total − ε` (ε=1e-3); cập nhật doc comment semantics (carry-in unknown)
- [x] 1.3 Integration test: ON đầu cửa sổ → UNKNOWN giữa giờ → online/unknown chia đúng từng khoảng, HasData=true; cửa sổ toàn UNKNOWN (carry-in) → HasData=false + unknown trọn span; case cũ không đổi kết quả

## 2. Cache hash + EXPIRE

- [x] 2.1 `ontimecache.go`: key suffix bump (`:stats:v2`), MGet = pipeline HGETALL parse strconv, MSet = pipeline HSet+Expire giữ dual-TTL 1h/10s; fields `has_data`/`uptime`/`unknown` full precision; bỏ sentinel `__NULL__` và làm tròn `%.2f`
- [x] 2.2 `dto.DayResult` thêm `Unknown float64`
- [x] 2.3 Viết lại expectation `ontimecache_integration_test.go`: cache hit đủ 3 field đúng precision; entry string định dạng cũ → coi như miss và ghi đè không lỗi

## 3. Passthrough DTO/service

- [x] 3.1 batcher: `fillMisses` và `buildResponse` truyền `Unknown`; `ontime.go` `getServersOntime` map xuôi (:143) và chiều ngược (:158) truyền đủ
- [x] 3.2 `dto.OntimeStats` thêm `UnknownSeconds float64` tag `json:"unknown_seconds"`; unit test mock rows trả UnknownSeconds → response chứa đúng giá trị (case HasData true/false)

## 4. REST API

- [x] 4.1 `api/spec.yaml`: schema `OntimeStats` thêm `unknown_seconds` (type number, format double); regenerate ogen
- [x] 4.2 `toOntimeStats` (ontimehandler.go) map thêm `UnknownSeconds`; cập nhật handler test nếu có literal exhaustive

## 5. Verification end-to-end

- [x] 5.1 `go test ./...` ontime-service xanh (kể cả integration testcontainers)
- [x] 5.2 `golangci-lint run` sạch (binary ở GOPATH)
- [ ] 5.3 Smoke compose: agent im lặng → event UNKNOWN xuất hiện trong server_events → GET ontime stats trả `unknown_seconds > 0`, ngày mù trọn vẹn trả `has_data=false`
