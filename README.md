# curlman

A CLI tool for running HTTP tests defined as `.http` and `.assert` files.

---

## Installation

```sh
go build -o curlman .
```

---

## Usage

```
curlman [test-path] [flags]
curlman run [test-path] [flags]
```

`test-path` is an optional filter. Omit it to run all tests found under the search directory.

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--path` | current directory | Directory to search for `.http` files |
| `--env` | `.env` | Path to a `.env` file for variable substitution |
| `--update` | false | Write actual responses back to `.assert` files |
| `--fail-fast` | false | Stop after the first failing test |
| `-v, --verbose` | false | Print request and response for every test |

### Examples

```sh
# Run all tests in the current directory
curlman

# Run all tests in a specific directory
curlman --path ./tests

# Run only auth tests
curlman --path ./tests auth/

# Run with a specific env file
curlman --env .env.staging

# Bootstrap .assert files from real responses
curlman --update --env tests/.test.env

# Stop on first failure
curlman --fail-fast
```

---

## File formats

### `.http` — request definition

Standard HTTP wire format with `{{VAR}}` substitution. Multiple requests can be defined in one file, separated by `###` followed by a name.

```
### Create user
POST http://{{BASE_URL}}/users
Content-Type: application/json

{"name":"Alice","email":"alice@example.com"}
```

### `.assert` — assertions

Each block must have the same `### Name` as its corresponding request block. Supported assertion types:

| Directive | Example | Description |
|-----------|---------|-------------|
| `STATUS` | `STATUS 201` | Exact HTTP status code |
| `HEADER` | `HEADER content-type: application/json` | Response header contains value |
| `BODY` | `BODY $.name == "Alice"` | JSONPath field check |

**BODY operators:** `exists`, `not exists`, `==`, `!=`, `contains`

```
### Create user
STATUS 201
HEADER content-type: application/json
BODY $.id exists
BODY $.name == "Alice"
BODY $.created_at exists
```

### `.env`

```
BASE_URL=localhost:8080
TOKEN=secret123
```

---

## Test layout

Tests are discovered recursively from the search directory. Each `.http` file is paired with a `.assert` file of the same name.

```
tests/
├── auth/
│   ├── login.http
│   ├── login.assert
│   ├── login-fail.http
│   └── login-fail.assert
└── users/
    ├── create.http
    ├── create.assert
    ├── get.http
    └── get.assert
```

---

## Test server

A local HTTP server is included for integration testing. It runs entirely in-memory with no external dependencies.

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Returns `{"status":"ok","version":"1.0.0"}` |
| `GET` | `/users` | Lists all users |
| `POST` | `/users` | Creates a user (`name` required) |
| `GET` | `/users/{id}` | Fetches a user by ID |
| `DELETE` | `/users/{id}` | Deletes a user by ID |
| `GET` | `/echo` | Reflects method, path, and headers back as JSON |
| `GET` | `/status/{code}` | Responds with the given HTTP status code |
| `POST` | `/auth/login` | Returns a token for `admin`/`secret`, 401 otherwise |

The server starts pre-seeded with one user:

```json
{
  "id": "00000000-0000-0000-0000-000000000001",
  "name": "Alice",
  "email": "alice@example.com"
}
```

### Running the tests

**1. Start the test server**

```sh
go run ./testserver
# curlman test server listening on :8080
```

**2. Run all tests against it**

```sh
curlman --env tests/.test.env --path tests/
```

**3. Run a specific group**

```sh
curlman --env tests/.test.env --path tests/ auth/
curlman --env tests/.test.env --path tests/ users/
```

The `tests/.test.env` file sets `BASE_URL=localhost:8080`, which is substituted into every `.http` file.

### Bootstrapping new tests

If you add a new `.http` file and want to generate the initial `.assert` from the real response:

```sh
curlman --env tests/.test.env --update --path tests/
```

This runs each request and writes the actual status, headers, and a basic body assertion to the paired `.assert` file. Review and tighten the generated assertions before committing.
