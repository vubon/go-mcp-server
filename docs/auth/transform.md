# Transform Authorization Strategy

The **transform** strategy allows you to modify the format of the incoming authorization header before sending it to the backend service. This is useful when your backend expects a different authorization format than what the client provides.

## Overview

The transform strategy extracts the incoming authorization header, modifies its prefix (e.g., `Bearer` → `ApiKey`), and sends the transformed value to the backend. You can also remove the prefix entirely or change the header name.

## Configuration

### Basic Transform

Transform a Bearer token to an ApiKey format:

```yaml
handlers:
  get_user:
    type: http
    method: GET
    path: /api/v1/users/{userId}
    authorization:
      strategy: transform
      transform:
        fromPrefix: Bearer
        toPrefix: ApiKey
```

**Example:**
- **Input**: `Bearer token-123`
- **Output**: `ApiKey token-123`

### Remove Prefix

Remove the prefix entirely, sending just the token:

```yaml
handlers:
  get_user:
    type: http
    method: GET
    path: /api/v1/users/{userId}
    authorization:
      strategy: transform
      transform:
        fromPrefix: Bearer
        toPrefix: ""  # Empty string removes prefix
```

**Example:**
- **Input**: `Bearer token-123`
- **Output**: `token-123`

### Change Header Name

Send the transformed value to a different header:

```yaml
handlers:
  get_user:
    type: http
    method: GET
    path: /api/v1/users/{userId}
    authorization:
      strategy: transform
      headerName: X-API-Key
      transform:
        fromPrefix: Bearer
        toPrefix: ApiKey
```

**Example:**
- **Input Header**: `Authorization: Bearer token-123`
- **Output Header**: `X-API-Key: ApiKey token-123`

## Configuration Options

### `fromPrefix` (required)
The prefix to match in the incoming authorization header. Only headers starting with this prefix will be transformed.

### `toPrefix` (optional)
The prefix to use in the output. If empty, the prefix is removed entirely.

### `headerName` (optional)
The header name to use in the backend request. Defaults to `Authorization`.

## How It Works

1. **Client Request**: Client sends `Authorization: Bearer token-123`
2. **Extraction**: HTTP transport extracts the header value
3. **Matching**: System checks if the header starts with `fromPrefix`
4. **Transformation**: If matched, replaces `fromPrefix` with `toPrefix` (or removes it if `toPrefix` is empty)
5. **Backend Request**: Transformed value is sent to the backend

## Use Cases

- **API Key Conversion**: Convert Bearer tokens to API key format for legacy backends
- **Header Name Changes**: Move authorization to a custom header (e.g., `X-API-Key`)
- **Prefix Removal**: Extract just the token value without any prefix
- **Format Standardization**: Normalize different client auth formats to a single backend format

## Examples

### Example 1: Bearer to ApiKey

```yaml
handlers:
  get_user:
    type: http
    method: GET
    path: /api/v1/users/{userId}
    authorization:
      strategy: transform
      transform:
        fromPrefix: Bearer
        toPrefix: ApiKey
```

**Transformation:**
- Input: `Authorization: Bearer abc123`
- Output: `Authorization: ApiKey abc123`

### Example 2: Bearer to Custom Header

```yaml
handlers:
  get_user:
    type: http
    method: GET
    path: /api/v1/users/{userId}
    authorization:
      strategy: transform
      headerName: X-API-Key
      transform:
        fromPrefix: Bearer
        toPrefix: ""
```

**Transformation:**
- Input: `Authorization: Bearer abc123`
- Output: `X-API-Key: abc123`

### Example 3: Service-Level Transform

```yaml
serviceConfig:
  legacy-api:
    baseURL: https://api.example.com
    authorization:
      strategy: transform
      headerName: X-API-Key
      transform:
        fromPrefix: Bearer
        toPrefix: ApiKey

handlers:
  get_user:
    type: http
    method: GET
    path: /api/v1/users/{userId}
    # Inherits service-level transform config
```

## Behavior When Prefix Doesn't Match

If the incoming authorization header doesn't start with the specified `fromPrefix`:
- The original header value is returned unchanged
- This allows fallback behavior for different auth formats

**Example:**
```yaml
transform:
  fromPrefix: Bearer
  toPrefix: ApiKey
```

- Input: `Bearer token` → Output: `ApiKey token` ✅
- Input: `Basic dXNlcjpwYXNz` → Output: `Basic dXNlcjpwYXNz` (unchanged)

## Multiple Transform Scenarios

### Scenario 1: Different Clients, Same Backend

Handle multiple client auth formats and normalize to one backend format:

```yaml
# Client sends: Authorization: Bearer token
# Backend expects: X-API-Key: token

handlers:
  get_user:
    authorization:
      strategy: transform
      headerName: X-API-Key
      transform:
        fromPrefix: Bearer
        toPrefix: ""
```

### Scenario 2: Legacy API Compatibility

Convert modern Bearer tokens to legacy API key format:

```yaml
handlers:
  legacy_endpoint:
    authorization:
      strategy: transform
      transform:
        fromPrefix: Bearer
        toPrefix: ApiKey
```

## Limitations

- **Single Prefix Match**: Only one `fromPrefix` can be specified per transform
- **Exact Prefix Match**: The prefix must match exactly (case-sensitive)
- **No Value Modification**: Only the prefix is transformed; the token value remains unchanged

## Security Considerations

- **Token Preservation**: The actual token value is never modified, only the prefix
- **Header Validation**: Ensure your backend validates the transformed authorization header
- **Transport Security**: Always use HTTPS to protect authorization headers in transit

## Related Strategies

- **[Pass-Through](./pass-through.md)**: Use when no transformation is needed
- **[Static](./static.md)**: Use when you need a fixed authorization value
- **[Basic](./basic.md)**: Use for Basic Authentication encoding
- **[None](./none.md)**: Use to explicitly disable authorization

