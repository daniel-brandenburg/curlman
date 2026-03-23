// testserver is a local HTTP server for running curlman integration tests.
//
// Usage:
//
//	go run ./testserver
//	./curlman.exe --env .env.test
package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type User struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
}

var (
	mu    sync.RWMutex
	store = map[string]User{
		// Pre-seeded so tests can rely on a known ID without depending on POST /users.
		"00000000-0000-0000-0000-000000000001": {
			ID:        "00000000-0000-0000-0000-000000000001",
			Name:      "Alice",
			Email:     "alice@example.com",
			CreatedAt: "2024-01-01T00:00:00Z",
		},
		// Pre-seeded for delete tests — tests should DELETE this ID, not Alice's.
		"00000000-0000-0000-0000-000000000002": {
			ID:        "00000000-0000-0000-0000-000000000002",
			Name:      "Bob",
			Email:     "bob@example.com",
			CreatedAt: "2024-01-01T00:00:00Z",
		},
	}
)

func newUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// GET /health
func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"version": "1.0.0",
	})
}

// GET /no-content
func handleNoContent(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

// GET /slow/{ms} — sleeps for ms milliseconds (capped at 5000) then responds.
func handleSlow(w http.ResponseWriter, r *http.Request) {
	msStr := strings.TrimPrefix(r.URL.Path, "/slow/")
	ms, err := strconv.Atoi(msStr)
	if err != nil || ms < 0 {
		writeError(w, http.StatusBadRequest, "invalid delay")
		return
	}
	if ms > 5000 {
		ms = 5000
	}
	time.Sleep(time.Duration(ms) * time.Millisecond)
	writeJSON(w, http.StatusOK, map[string]int{"delay_ms": ms})
}

// GET /redirect/{code} — redirects to /health with the given status code.
func handleRedirect(w http.ResponseWriter, r *http.Request) {
	codeStr := strings.TrimPrefix(r.URL.Path, "/redirect/")
	code, err := strconv.Atoi(codeStr)
	if err != nil || code < 300 || code > 308 {
		writeError(w, http.StatusBadRequest, "invalid redirect code")
		return
	}
	http.Redirect(w, r, "/health", code)
}

// GET /text — returns a plain text response.
func handleText(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "Hello, world!")
}

// GET /query-echo — returns all query parameters as JSON.
func handleQueryEcho(w http.ResponseWriter, r *http.Request) {
	params := map[string]string{}
	for k, vals := range r.URL.Query() {
		params[k] = strings.Join(vals, ", ")
	}
	writeJSON(w, http.StatusOK, map[string]any{"params": params})
}

// POST /auth/protected — requires a non-empty Bearer token.
func handleProtected(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") || strings.TrimPrefix(auth, "Bearer ") == "" {
		writeError(w, http.StatusUnauthorized, "authorization required")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "authorized"})
}

// GET /large — returns a JSON array of 500 items.
func handleLarge(w http.ResponseWriter, r *http.Request) {
	items := make([]map[string]int, 500)
	for i := range items {
		items[i] = map[string]int{"id": i + 1}
	}
	writeJSON(w, http.StatusOK, map[string]any{"count": 500, "items": items})
}

// GET|POST /users
func handleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		mu.RLock()
		list := make([]User, 0, len(store))
		for _, u := range store {
			list = append(list, u)
		}
		mu.RUnlock()
		sort.Slice(list, func(i, j int) bool {
			return list[i].ID < list[j].ID
		})
		writeJSON(w, http.StatusOK, map[string]any{"users": list, "count": len(list)})

	case http.MethodPost:
		var body struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
			writeError(w, http.StatusBadRequest, "name is required")
			return
		}
		u := User{
			ID:        newUUID(),
			Name:      body.Name,
			Email:     body.Email,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		}
		mu.Lock()
		store[u.ID] = u
		mu.Unlock()
		writeJSON(w, http.StatusCreated, u)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// GET|DELETE /users/{id}
func handleUser(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/users/")
	if id == "" {
		handleUsers(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		mu.RLock()
		u, ok := store[id]
		mu.RUnlock()
		if !ok {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeJSON(w, http.StatusOK, u)

	case http.MethodDelete:
		mu.Lock()
		_, ok := store[id]
		if ok {
			delete(store, id)
		}
		mu.Unlock()
		if !ok {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// GET /echo — reflects method, path, and request headers back as JSON.
func handleEcho(w http.ResponseWriter, r *http.Request) {
	headers := map[string]string{}
	for k, vals := range r.Header {
		headers[strings.ToLower(k)] = strings.Join(vals, ", ")
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"method":  r.Method,
		"path":    r.URL.Path,
		"query":   r.URL.RawQuery,
		"headers": headers,
	})
}

// GET /status/{code} — responds with exactly the given HTTP status code.
func handleStatus(w http.ResponseWriter, r *http.Request) {
	codeStr := strings.TrimPrefix(r.URL.Path, "/status/")
	code, err := strconv.Atoi(codeStr)
	if err != nil || code < 100 || code > 599 {
		writeError(w, http.StatusBadRequest, "invalid status code")
		return
	}
	if code != http.StatusNoContent {
		writeJSON(w, code, map[string]int{"status": code})
	} else {
		w.WriteHeader(code)
	}
}

// POST /auth/login — accepts {"username":"admin","password":"secret"}.
func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var creds struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if creds.Username == "admin" && creds.Password == "secret" {
		writeJSON(w, http.StatusOK, map[string]string{
			"token": "test-token-abc123",
			"type":  "Bearer",
		})
		return
	}
	writeError(w, http.StatusUnauthorized, "invalid credentials")
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/users", handleUsers)
	mux.HandleFunc("/users/", handleUser)
	mux.HandleFunc("/echo", handleEcho)
	mux.HandleFunc("/status/", handleStatus)
	mux.HandleFunc("/auth/login", handleLogin)
	mux.HandleFunc("/no-content", handleNoContent)
	mux.HandleFunc("/slow/", handleSlow)
	mux.HandleFunc("/redirect/", handleRedirect)
	mux.HandleFunc("/text", handleText)
	mux.HandleFunc("/query-echo", handleQueryEcho)
	mux.HandleFunc("/auth/protected", handleProtected)
	mux.HandleFunc("/large", handleLarge)

	addr := ":8080"
	fmt.Printf("curlman test server listening on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
