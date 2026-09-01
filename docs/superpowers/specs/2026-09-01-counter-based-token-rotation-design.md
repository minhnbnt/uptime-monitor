# Counter-based Token Rotation

## Problem

Hiện tại, mỗi lần refresh token, hệ thống tạo session mới với UUID mới. Điều này导致:
- Session ID (`sid` claim trong access token, `X-Session-ID` header) thay đổi sau mỗi lần rotate
- Downstream services看到 session ID khác nhau → phá vỡ rate-limiting và session tracking
- Session list có nhiều entries cho cùng một "logical session"

## Solution

Tách biệt identity của session khỏi identity của refresh token:
- **Session ID (JTI)** = UUID, tạo một lần, **không thay đổi** khi rotate
- **Counter** = int64, bắt đầu từ 0, **tăng lên** mỗi lần rotate
- Refresh token chứa claim `counter` để phát hiện replay

## Schema Changes

### domain.Session

```go
type Session struct {
    gorm.Model
    UserID    uint      `gorm:"not null;index"`
    JTI       uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
    Counter   int64     `gorm:"not null;default:0"`
    Scopes    string    `gorm:"type:text;not null"`
    ExpiresAt time.Time `gorm:"not null;index"`
}
```

Thêm cột `Counter` với default `0`.

### Migration

```sql
ALTER TABLE sessions ADD COLUMN counter BIGINT NOT NULL DEFAULT 0;
```

Sessions hiện tại đã có `counter = 0` (default), không cần update data.

## Token Format

### Access Token

```json
{
  "sub": "123",
  "email": "user@example.com",
  "username": "user",
  "scope": "app",
  "sid": "<JTI UUID>",  // Session ID, ổn định
  "exp": 1234567890
}
```

`sid` = JTI của session, **không thay đổi** khi rotate.

### Refresh Token

```json
{
  "sub": "123",
  "jti": "<JTI UUID>",
  "counter": 5,
  "exp": 1234567890
}
```

`counter` = giá trị counter tại thời điểm tạo token.

## Flows

### 1. Create Session (Login/Register)

```
1. Tạo JTI = UUID()
2. Tạo session: { jti, counter: 0, user_id, scopes, expires_at }
3. Tạo refresh token: { sub, jti, counter: 0 }
4. Tạo access token: { sub, sid: jti, scope, ... }
5. Lưu session vào DB
```

JTI được tạo một lần và giữ nguyên suốt đời session.

### 2. Rotate Token (Refresh)

```
1. Parse refresh token → lấy jti, counter
2. Lookup session by jti
3. Nếu session không tồn tại hoặc hết hạn → TỪ CHỐI
4. Nếu token.counter != session.counter → TỪ CHỐI (replay hoặc invalid)
5. Nếu token.counter == session.counter:
   a. Update session: counter++, expires_at = now + TTL
   b. JTI giữ nguyên (session_id = jti, ổn định)
   c. Tạo refresh token mới: same jti, counter = session.counter + 1
   d. Tạo access token mới: same sid = jti
   e. Trả về token pair mới
```

**JTI giữ nguyên** — đây là key point: session ID ổn định qua mỗi lần rotate.

### 3. Validate Access Token

```
1. Parse JWT → lấy sid claim
2. sid = session ID (ổn định)
3. Không cần check DB (access token là stateless)
```

### 4. Forward Auth

```
1. Validate access token
2. Set header X-Session-ID = info.SID
3. Session ID ổn định qua mỗi lần rotate
```

## Concurrency Protection

**Vấn đề:** Hai request cùng lúc với cùng refresh token đều đọc counter = N, cả 2 đều pass check.

**Giải pháp:** Atomic CAS (compare-and-swap) trong một dòng SQL — không cần transaction, không cần lock:

```sql
UPDATE sessions 
SET counter = counter + 1, expires_at = ? 
WHERE jti = ? AND counter = ?
```

- `rows_affected = 1` → thành công, rotate OK
- `rows_affected = 0` → counter đã bị tăng bởi request khác → reject

## Verification Logic

```go
func (s *SessionService) RotateSession(ctx context.Context, user *domain.User, newScope []string, session *domain.Session) (*dto.TokenPair, error) {
    // Atomic CAS: increment counter only if it matches
    updated, err := s.sessionRepository.IncrementCounter(ctx, session)
    if err != nil {
        s.logger.Error("failed to increment session counter", slog.Any("error", err))
        return nil, apperrors.ErrInternal
    }
    if !updated {
        // Counter mismatch — replay hoặc concurrent write
        return nil, apperrors.ErrInvalidRefreshToken
    }

    // Generate new token pair with same JTI (session ID stays stable)
    newCounter := session.Counter + 1
    tokenPair, err := s.tokenService.GenerateTokenPairWithCounter(user, newScope, session.JTI.String(), newCounter)
    if err != nil {
        return nil, err
    }

    return tokenPair, nil
}
```

## Changes Required

### 1. Domain Layer
- `domain.Session`: thêm `Counter int64`

### 2. Repository Layer
- `SessionRepository.IncrementCounter(ctx, session *domain.Session) (bool, error)`: atomic CAS increment
- `SessionRepository.GetByJTI`: trả về session với counter
- Giữ nguyên `SessionRepository.Rotate` cho backward compatibility (có thể deprecated sau)

### 3. Token Layer
- `tokenGenerator.GenerateRefreshToken`: thêm parameter `counter`, thêm claim `counter` vào JWT
- `tokenGenerator.GenerateAccessToken`: giữ nguyên (sid = JTI, không thay đổi)
- `tokenValidator.ValidateRefreshToken`: parse `counter` claim từ JWT
- `RefreshTokenInfo`: thêm field `Counter int64`

### 4. Service Layer
- `SessionService.RotateSession`: nhận `*domain.Session`, gọi `IncrementCounter`, kiểm tra result
- `TokenService.GenerateTokenPairWithCounter`: tạo token pair với JTI và counter cụ thể
- `TokenService.GenerateTokenPair`: giữ nguyên cho create session (counter = 0)

### 5. Handler Layer
- `AuthService.Refresh`: lookup session, truyền session object vào `RotateSession`

### 6. Migration
- Thêm cột `counter` vào bảng `sessions`

## Testing

1. **Unit test cho counter validation:**
   - Token counter == session counter → accept
   - Token counter != session counter → reject

2. **Integration test:**
   - Login → tạo session (counter = 0)
   - Refresh → rotate (counter = 1)
   - Refresh again → rotate (counter = 2)
   - Kiểm tra X-Session-ID giữ nguyên qua mỗi lần rotate

3. **Replay test:**
   - Lưu refresh token cũ
   - Rotate thành công
   - Thử dùng refresh token cũ → phải bị reject

## Backward Compatibility

- Sessions hiện tại đã có `counter = 0` (default)
- Không cần migration data
- API không thay đổi
- Token format thay đổi (thêm claim `counter`) → cần update client
