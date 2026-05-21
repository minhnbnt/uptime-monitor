# ==========================================
# Validation error tests
# ==========================================

# Create server - empty name
POST {{base_url}}/api/v1/servers
Content-Type: application/json
{
    "name": ""
}
HTTP 400
[Asserts]
jsonpath "$.error.code" == "VALIDATION_ERROR"
jsonpath "$.error.message" contains "required"

# Create server - name too long (256 chars)
POST {{base_url}}/api/v1/servers
Content-Type: application/json
{
    "name": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
}
HTTP 400
[Asserts]
jsonpath "$.error.code" == "VALIDATION_ERROR"

# SetCheckMethod - invalid method
POST {{base_url}}/api/v1/servers
Content-Type: application/json
{
    "name": "validation-test"
}
HTTP 201
[Captures]
server_id: jsonpath "$.data.id"

# Invalid method
PUT {{base_url}}/api/v1/servers/{{server_id}}/check_method
Content-Type: application/json
{
    "method": "invalid",
    "endpoint": {
        "url": "https://example.com",
        "interval": 30,
        "timeout": 10,
        "method": "GET",
        "expected_code": 200
    }
}
HTTP 400
[Asserts]
jsonpath "$.error.code" == "VALIDATION_ERROR"

# Invalid URL
PUT {{base_url}}/api/v1/servers/{{server_id}}/check_method
Content-Type: application/json
{
    "method": "pull",
    "endpoint": {
        "url": "not-a-url",
        "interval": 30,
        "timeout": 10,
        "method": "GET",
        "expected_code": 200
    }
}
HTTP 400

# Interval = 0
PUT {{base_url}}/api/v1/servers/{{server_id}}/check_method
Content-Type: application/json
{
    "method": "pull",
    "endpoint": {
        "url": "https://example.com",
        "interval": 0,
        "timeout": 10,
        "method": "GET",
        "expected_code": 200
    }
}
HTTP 400

# Expected code < 100
PUT {{base_url}}/api/v1/servers/{{server_id}}/check_method
Content-Type: application/json
{
    "method": "pull",
    "endpoint": {
        "url": "https://example.com",
        "interval": 30,
        "timeout": 10,
        "method": "GET",
        "expected_code": 99
    }
}
HTTP 400

# Expected code > 599
PUT {{base_url}}/api/v1/servers/{{server_id}}/check_method
Content-Type: application/json
{
    "method": "pull",
    "endpoint": {
        "url": "https://example.com",
        "interval": 30,
        "timeout": 10,
        "method": "GET",
        "expected_code": 600
    }
}
HTTP 400
