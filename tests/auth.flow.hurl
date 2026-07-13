# ── Auth Flow ────────────────────────────────────────────────────────
# Runs through Kong on :8080 (routes /api/v1/auth/* → auth-service:8081)

# 1. Register a new user
POST {{base_url}}/api/v1/auth/register
{
  "email": "test@example.com",
    "username": "testuser",
    "password": "password123",
    "name": "Test User"
}
HTTP 201
[Captures]
access_token: jsonpath "$.access_token"
refresh_token: jsonpath "$.refresh_token"
user_id: jsonpath "$.user.id"
[Asserts]
jsonpath "$.user.email" == "test@example.com"
jsonpath "$.user.username" == "testuser"
jsonpath "$.user.name" == "Test User"
jsonpath "$.access_token" isString
jsonpath "$.refresh_token" isString

# 2. Login with username
POST {{base_url}}/api/v1/auth/login
{
    "login": "testuser",
    "password": "password123"
}
HTTP 200
[Captures]
login_access_token: jsonpath "$.access_token"
login_refresh_token: jsonpath "$.refresh_token"
[Asserts]
jsonpath "$.user.id" == {{user_id}}
jsonpath "$.access_token" isString

# 3. Login with email
POST {{base_url}}/api/v1/auth/login
{
    "login": "test@example.com",
    "password": "password123"
}
HTTP 200
[Asserts]
jsonpath "$.user.id" == {{user_id}}

# 4. Login with wrong password
POST {{base_url}}/api/v1/auth/login
{
    "login": "testuser",
    "password": "wrongpassword"
}
HTTP 401

# 5. Validate access token
GET {{base_url}}/api/v1/auth/validate
Authorization: Bearer {{access_token}}
HTTP 200
[Asserts]
jsonpath "$.user_id" == {{user_id}}

# 6. Refresh token
POST {{base_url}}/api/v1/auth/refresh
{
    "refresh_token": "{{refresh_token}}"
}
HTTP 200
[Captures]
refreshed_access_token: jsonpath "$.access_token"
refreshed_refresh_token: jsonpath "$.refresh_token"
[Asserts]
jsonpath "$.user.id" == {{user_id}}

# 7. Logout
POST {{base_url}}/api/v1/auth/logout
Authorization: Bearer {{refreshed_access_token}}
{
    "refresh_token": "{{refreshed_refresh_token}}"
}
HTTP 204

# 8. Use revoked refresh token
POST {{base_url}}/api/v1/auth/refresh
{
    "refresh_token": "{{refreshed_refresh_token}}"
}
HTTP 401
