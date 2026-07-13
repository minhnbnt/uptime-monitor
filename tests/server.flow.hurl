# ── Server CRUD Flow ─────────────────────────────────────────────────
# Goes through Kong on :8080 — provides JWT Bearer token,
# Kong validates it and injects X-User-ID header.

# 1. Register user + get token (prerequisite)
POST {{base_url}}/api/v1/auth/register
{
  "email": "server-test@example.com",
    "username": "server-test",
    "password": "password123",
    "name": "Server Tester"
}
HTTP 201
[Captures]
access_token: jsonpath "$.access_token"

# 2. Create server
POST {{base_url}}/api/v1/servers
Authorization: Bearer {{access_token}}
{
    "name": "my-test-server"
}
HTTP 201
[Captures]
server_id: jsonpath "$.data.id"
[Asserts]
jsonpath "$.data.name" == "my-test-server"
jsonpath "$.data.monitor_status" == "offline"

# 3. List servers
GET {{base_url}}/api/v1/servers
Authorization: Bearer {{access_token}}
HTTP 200
[Asserts]
jsonpath "$.data" count >= 1
jsonpath "$.meta.total" exists

# 4. Get server by ID
GET {{base_url}}/api/v1/servers/{{server_id}}
Authorization: Bearer {{access_token}}
HTTP 200
[Asserts]
jsonpath "$.data.id" == {{server_id}}
jsonpath "$.data.name" == "my-test-server"

# 5. Update server name
PUT {{base_url}}/api/v1/servers/{{server_id}}
Authorization: Bearer {{access_token}}
{
    "name": "updated-server-name"
}
HTTP 200
[Asserts]
jsonpath "$.data.name" == "updated-server-name"

# 6. Search servers
GET {{base_url}}/api/v1/servers/search?q=updated
Authorization: Bearer {{access_token}}
HTTP 200
[Asserts]
jsonpath "$.data[0].id" == {{server_id}}

# 7. Delete server
DELETE {{base_url}}/api/v1/servers/{{server_id}}
Authorization: Bearer {{access_token}}
HTTP 204

# 8. Get deleted server
GET {{base_url}}/api/v1/servers/{{server_id}}
Authorization: Bearer {{access_token}}
HTTP 404

# 9. Unauthorized — no Bearer token
GET {{base_url}}/api/v1/servers
HTTP 401

# 10. Unauthorized — invalid Bearer token
GET {{base_url}}/api/v1/servers
Authorization: Bearer invalid-jwt-token
HTTP 401
