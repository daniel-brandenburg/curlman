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
curlman tui [flags]
```

`test-path` is an optional filter. Omit it to run all tests found under the search directory.

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--path` | current directory | Directory to search for `.http` files |
| `--env` | `.env` | Path to a `.env` file for variable substitution |
| `--update` | false | Write actual responses back to `.assert` files |
| `--fail-fast` | false | Stop after the first failing test |
| `--plain` | false | Plain `PASS`/`FAIL` output without grouping or icons |
| `-v, --verbose` | false | Plain output with full request/response details |

### Output modes

| Mode | Flag | Description |
|------|------|-------------|
| Pretty | *(default)* | Grouped by directory, `✓`/`✗` icons, colored summary |
| Plain | `--plain` | `PASS`/`FAIL`/`SKIP` text, no grouping |
| Verbose | `--verbose` | Like plain, plus request method/URL and response body for every test |

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

# Plain output (e.g. for CI logs)
curlman --plain

# Verbose output with request/response details
curlman --verbose
```

---

## TUI

```sh
curlman tui --env tests/.test.env --path tests/
```

An interactive terminal UI with a split-panel layout. Tests do not run automatically — use the keyboard to trigger them.

### Keys

| Key | Action |
|-----|--------|
| `↑` / `↓` | Navigate the test list |
| `←` / `→` | Collapse / expand a folder |
| `enter` | Toggle folder or run selected test |
| `r` | Run selected test or folder |
| `R` | Run all tests |
| `o` | Expand all folders under cursor |
| `c` | Collapse all folders under cursor |
| `tab` | Switch between list and detail panels |
| `q` / `ctrl+c` | Quit |

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

Tests are discovered recursively from the search directory. Each `.http` file is paired with a `.assert` file of the same name. Hidden directories, `vendor/`, and `node_modules/` are skipped.

```
tests/
├── auth/
│   ├── login.http
│   ├── login.assert
│   ├── login-fail.http
│   ├── login-fail.assert
│   ├── protected-ok.http
│   ├── protected-ok.assert
│   ├── protected-fail.http
│   └── protected-fail.assert
├── edge/
│   ├── large.http / large.assert
│   ├── method-not-allowed.http / method-not-allowed.assert
│   └── unicode.http / unicode.assert
├── no-content/
│   └── get.http / get.assert
├── query/
│   └── params.http / params.assert
├── redirect/
│   ├── 301.http / 301.assert
│   └── follow.http / follow.assert
├── slow/
│   └── timeout.http / timeout.assert
├── text/
│   └── plain.http / plain.assert
└── users/
    ├── create.http / create.assert
    ├── create-and-check.http / create-and-check.assert
    ├── delete.http / delete.assert
    ├── get.http / get.assert
    └── list.http / list.assert
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
| `GET` | `/echo` | Reflects method, path, query, and headers back as JSON |
| `GET` | `/status/{code}` | Responds with the given HTTP status code |
| `POST` | `/auth/login` | Returns a token for `admin`/`secret`, 401 otherwise |
| `POST` | `/auth/protected` | Requires `Authorization: Bearer <token>`, 401 otherwise |
| `GET` | `/no-content` | Returns 204 with no body |
| `GET` | `/slow/{ms}` | Sleeps for `ms` milliseconds (max 5000) then responds |
| `GET` | `/redirect/{code}` | Redirects to `/health` with the given status code |
| `GET` | `/text` | Returns a `text/plain` response |
| `GET` | `/query-echo` | Returns all query parameters as JSON |
| `GET` | `/large` | Returns a JSON array of 500 items |

The server starts pre-seeded with two users:

```json
{"id": "00000000-0000-0000-0000-000000000001", "name": "Alice", "email": "alice@example.com"}
{"id": "00000000-0000-0000-0000-000000000002", "name": "Bob",   "email": "bob@example.com"}
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

---

## Running unit tests

```sh
go test ./internal/...
```
