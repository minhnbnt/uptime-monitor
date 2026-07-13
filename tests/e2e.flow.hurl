# ── Full E2E Flow ─────────────────────────────────────────────────────
# Register → Login → Create Server → CRUD → Logout

# 1. Register
POST {{base_url}}/api/v1/auth/register
{
  "email": "e2e@example.com",
    "username": "e2e-user",
    "password": "Str0ng!Pass",
    "name": "E2E User"
}
HTTP 201
[Captures]
access_token: jsonpath "$.access_token"
refresh_token: jsonpath "$.refresh_token"
user_id: jsonpath "$.user.id"

# 2. Create server
POST {{base_url}}/api/v1/servers
Authorization: Bearer {{access_token}}
{
    "name": "e2e-server"
}
HTTP 201
[Captures]
server_id: jsonpath "$.data.id"

# 3. Get server by ID — verify it exists
GET {{base_url}}/api/v1/servers/{{server_id}}
Authorization: Bearer {{access_token}}
HTTP 200
[Asserts]
jsonpath "$.data.id" == {{server_id}}
jsonpath "$.data.name" == "e2e-server"

# 5. Refresh token
POST {{base_url}}/api/v1/auth/refresh
{
    "refresh_token": "{{refresh_token}}"
}
HTTP 200
[Captures]
new_access_token: jsonpath "$.access_token"

# 6. Delete server with refreshed token
DELETE {{base_url}}/api/v1/servers/{{server_id}}
Authorization: Bearer {{new_access_token}}
HTTP 204

# 7. Verify deletion
GET {{base_url}}/api/v1/servers/{{server_id}}
Authorization: Bearer {{new_access_token}}
HTTP 404

# 8. Logout
POST {{base_url}}/api/v1/auth/logout
Authorization: Bearer {{new_access_token}}
{
    "refresh_token": "{{refresh_token}}"
}
HTTP 204
