# Basic Authentication Strategy

The **basic** strategy implements HTTP Basic Authentication by encoding credentials as `Basic base64(username:password)`. It supports extracting credentials from incoming request headers or using static values from configuration.

## Overview

The basic strategy can operate in two modes:

1. **Header Extraction Mode**: Extracts username and password from incoming request headers and encodes them as Basic Auth
2. **Static Mode**: Uses username/password from configuration or environment variables and encodes them as Basic Auth

## Configuration

### Header Extraction Mode (Recommended)

Extract credentials from incoming request headers and encode as Basic Auth:

```yaml
handlers:
  get_user:
    type: http
    method: GET
    path: /api/v1/users/{userId}
    authorization:
      strategy: basic
      basicAuth:
        usernameHeader: SECREAT_KEY
        passwordHeader: PASSWORD
```

**How it works:**
1. Client sends request with headers: `SECREAT_KEY: myuser` and `PASSWORD: mypass`
2. System extracts these header values
3. Encodes as Basic Auth: `Basic base64(myuser:mypass)`
4. Sends to backend: `Authorization: Basic bXl1c2VyOm15cGFzcw==`

### Password Optional

If password is not required, set `passwordHeader` to `"None"`:

```yaml
handlers:
  get_user:
    type: http
    method: GET
    path: /api/v1/users/{userId}
    authorization:
      strategy: basic
      basicAuth:
        usernameHeader: SECREAT_KEY
        passwordHeader: None
```

**Result:** Only username is encoded: `Basic base64(myuser)`

### Static Mode (Legacy)

Use static username and password from configuration:

```yaml
handlers:
  get_user:
    type: http
    method: GET
    path: /api/v1/users/{userId}
    authorization:
      strategy: basic
      basicAuth:
        username: myuser
        password: mypass
```

### Static Mode with Environment Variables

```yaml
handlers:
  get_user:
    type: http
    method: GET
    path: /api/v1/users/{userId}
    authorization:
      strategy: basic
      basicAuth:
        usernameEnv: API_USERNAME
        passwordEnv: API_PASSWORD
```

**Environment Variables:**
```bash
export API_USERNAME="myuser"
export API_PASSWORD="mypass"
```

## Configuration Options

### Header Extraction Mode

- **`usernameHeader`** (required): Header name in incoming request containing the username
- **`passwordHeader`** (optional): Header name in incoming request containing the password. Set to `"None"` if password is not required

### Static Mode (Legacy)

- **`username`**: Direct username value
- **`password`**: Direct password value
- **`usernameEnv`**: Environment variable name for username
- **`passwordEnv`**: Environment variable name for password

### Pre-encoded Values

- **`encodedValue`**: Pre-encoded Basic Auth value (e.g., `"Basic base64(...)"`)
- **`encodedValueEnv`**: Environment variable containing pre-encoded Basic Auth value

## How It Works

### Header Extraction Mode

1. **Client Request**: Client sends request with specified headers (e.g., `SECREAT_KEY: user`, `PASSWORD: pass`)
2. **Extraction**: System extracts header values from incoming request
3. **Encoding**: Values are combined as `username:password` and base64 encoded
4. **Backend Request**: Sends `Authorization: Basic base64(username:password)`

### Static Mode

1. **Configuration**: Username and password are read from config or environment variables
2. **Encoding**: Values are combined as `username:password` and base64 encoded
3. **Backend Request**: Sends `Authorization: Basic base64(username:password)`

## Use Cases

- **API Gateway**: Extract credentials from custom headers and convert to Basic Auth for backend
- **Legacy Backend Integration**: Backend services that require Basic Authentication
- **Service-to-Service Auth**: Using Basic Auth for internal service communication
- **Header-Based Credentials**: When clients send credentials in custom headers instead of Authorization header

## Examples

### Example 1: Header Extraction with Both Username and Password

```yaml
handlers:
  get_user:
    type: http
    method: GET
    path: /api/v1/users/{userId}
    authorization:
      strategy: basic
      basicAuth:
        usernameHeader: SECREAT_KEY
        passwordHeader: PASSWORD
```

**Client Request:**
```
SECREAT_KEY: myuser
PASSWORD: mypass
```

**Backend Request:**
```
Authorization: Basic bXl1c2VyOm15cGFzcw==
```

### Example 2: Header Extraction with Username Only

```yaml
handlers:
  get_user:
    type: http
    method: GET
    path: /api/v1/users/{userId}
    authorization:
      strategy: basic
      basicAuth:
        usernameHeader: API_KEY
        passwordHeader: None
```

**Client Request:**
```
API_KEY: my-api-key-123
```

**Backend Request:**
```
Authorization: Basic bXktYXBpLWtleS0xMjM=
```

### Example 3: Static Mode with Environment Variables

```yaml
handlers:
  get_user:
    type: http
    method: GET
    path: /api/v1/users/{userId}
    authorization:
      strategy: basic
      basicAuth:
        usernameEnv: BACKEND_USERNAME
        passwordEnv: BACKEND_PASSWORD
```

**Environment Variables:**
```bash
export BACKEND_USERNAME="service-account"
export BACKEND_PASSWORD="secret-password"
```

**Backend Request:**
```
Authorization: Basic c2VydmljZS1hY2NvdW50OnNlY3JldC1wYXNzd29yZA==
```

### Example 4: Service-Level Basic Auth

```yaml
serviceConfig:
  legacy-api:
    baseURL: https://legacy-api.example.com
    authorization:
      strategy: basic
      basicAuth:
        usernameHeader: SECREAT_KEY
        passwordHeader: PASSWORD

handlers:
  get_user:
    type: http
    method: GET
    path: /api/v1/users/{userId}
    # Inherits service-level basic auth config
```

### Example 5: Case-Insensitive Header Matching

Header names are matched case-insensitively:

```yaml
authorization:
  strategy: basic
  basicAuth:
    usernameHeader: SECREAT_KEY
    passwordHeader: PASSWORD
```

**Client Request (any case works):**
```
Secreat-Key: myuser
Password: mypass
```

**Result:** Headers are matched and encoded correctly.

## Encoding Details

The Basic Auth encoding follows RFC 7617:

1. Combine username and password: `username:password`
2. Encode to base64: `base64(username:password)`
3. Add "Basic " prefix: `Basic base64(username:password)`

**Example:**
- Username: `myuser`
- Password: `mypass`
- Combined: `myuser:mypass`
- Base64: `bXl1c2VyOm15cGFzcw==`
- Final: `Basic bXl1c2VyOm15cGFzcw==`

## Priority Resolution

When multiple configuration options are provided, the priority is:

1. **Header Extraction** (`usernameHeader` specified) - Highest priority
2. **Pre-encoded Value** (`encodedValue` or `encodedValueEnv`)
3. **Static Values** (`username`/`password` or `usernameEnv`/`passwordEnv`)

## Security Considerations

1. **HTTPS Required**: Basic Auth sends credentials in base64 encoding (not encryption). Always use HTTPS.
2. **Header Security**: When using header extraction, ensure the transport layer is secure
3. **Credential Storage**: Never store credentials in configuration files. Use environment variables or secret management systems.
4. **Token Rotation**: Implement a process for rotating credentials regularly
5. **Least Privilege**: Use credentials with minimal required permissions

## Common Patterns

### Pattern 1: API Gateway with Header Extraction

```yaml
handlers:
  # Extract credentials from custom headers and convert to Basic Auth
  backend_call:
    authorization:
      strategy: basic
      basicAuth:
        usernameHeader: X-API-Key
        passwordHeader: None
```

### Pattern 2: Legacy Backend Integration

```yaml
serviceConfig:
  legacy-backend:
    baseURL: https://legacy.example.com
    authorization:
      strategy: basic
      basicAuth:
        usernameEnv: LEGACY_USERNAME
        passwordEnv: LEGACY_PASSWORD
```

## Behavior When Headers Are Missing

If `usernameHeader` is specified but the header is not found in the incoming request:
- An empty string is returned
- No authorization header is sent to the backend
- This may cause authentication failures

**Best Practice**: Always validate that required headers are present in the request.

## Related Strategies

- **[Pass-Through](./pass-through.md)**: Use when you want to forward client authorization as-is
- **[Transform](./transform.md)**: Use when you need to modify authorization header format
- **[Static](./static.md)**: Use when you need a fixed authorization value
- **[None](./none.md)**: Use to explicitly disable authorization

