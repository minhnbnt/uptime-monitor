# ==========================================
# SetCheckMethod tests
# ==========================================

# First create a server
POST {{base_url}}/api/v1/servers
Content-Type: application/json
{
    "name": "ping-server"
}
HTTP 201
[Captures]
server_id: jsonpath "$.data.id"

# Set pull check method
PUT {{base_url}}/api/v1/servers/{{server_id}}/check_method
Content-Type: application/json
{
    "method": "pull",
    "endpoint": {
        "url": "https://example.com",
        "interval": 30,
        "timeout": 10,
        "method": "GET",
        "expected_code": 200
    }
}
HTTP 200

# Update with different values
PUT {{base_url}}/api/v1/servers/{{server_id}}/check_method
Content-Type: application/json
{
    "method": "pull",
    "endpoint": {
        "url": "https://other.com/api",
        "interval": 60,
        "timeout": 5,
        "method": "POST",
        "expected_code": 201
    }
}
HTTP 200

# Set push method
PUT {{base_url}}/api/v1/servers/{{server_id}}/check_method
Content-Type: application/json
{
    "method": "push",
    "endpoint": {
        "url": "https://push.example.com",
        "interval": 15,
        "timeout": 3,
        "method": "HEAD",
        "expected_code": 204
    }
}
HTTP 200
