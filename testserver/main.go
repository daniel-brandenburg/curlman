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
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"version": "1.0.0",
	})
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

	addr := ":8080"
	fmt.Printf("curlman test server listening on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
