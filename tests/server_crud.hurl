# ==========================================
# Server CRUD tests
# ==========================================

# Create server
POST {{base_url}}/api/v1/servers
Content-Type: application/json
{
    "name": "hurl-test-server"
}
HTTP 201
[Asserts]
jsonpath "$.data.name" == "hurl-test-server"
jsonpath "$.data.status" == "active"

# Capture server ID
[Captures]
server_id: jsonpath "$.data.id"

# Get server by ID
GET {{base_url}}/api/v1/servers/{{server_id}}
HTTP 200
[Asserts]
jsonpath "$.data.id" == {{server_id}}
jsonpath "$.data.name" == "hurl-test-server"

# List servers
GET {{base_url}}/api/v1/servers?page=1&per_page=20
HTTP 200
[Asserts]
jsonpath "$.data[0].name" == "hurl-test-server"

# Update server name
PUT {{base_url}}/api/v1/servers/{{server_id}}
Content-Type: application/json
{
    "name": "updated-server"
}
HTTP 200
[Asserts]
jsonpath "$.data.name" == "updated-server"

# Delete server
DELETE {{base_url}}/api/v1/servers/{{server_id}}
HTTP 204

# Verify deleted
GET {{base_url}}/api/v1/servers/{{server_id}}
HTTP 404
