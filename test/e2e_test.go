package test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"dummyjson/internal/api"
	"dummyjson/internal/auth"
	"dummyjson/internal/store"
)

type caseSpec struct {
	name   string
	method string
	path   string
	body   string
	token  bool
	status int
	check  func(t *testing.T, body []byte)
}

func newServer(t *testing.T) *httptest.Server {
	t.Helper()

	st, err := store.Load("../data")
	if err != nil {
		t.Fatalf("load store: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httptest.NewServer(api.NewServer(st, logger, auth.NewManager("test-secret")).Handler())
	t.Cleanup(server.Close)
	return server
}

func decode(t *testing.T, body []byte, target any) {
	t.Helper()
	if err := json.Unmarshal(body, target); err != nil {
		t.Fatalf("decode: %v (body = %s)", err, body)
	}
}

func call(t *testing.T, server *httptest.Server, method, path, body, token string) (int, []byte) {
	t.Helper()

	request, err := http.NewRequest(method, server.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return response.StatusCode, data
}

func run(t *testing.T, server *httptest.Server, token string, tests []caseSpec) {
	t.Helper()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			use := ""
			if test.token {
				use = token
			}
			status, body := call(t, server, test.method, test.path, test.body, use)
			if status != test.status {
				t.Fatalf("status = %d, want %d, body = %s", status, test.status, body)
			}
			if test.check != nil {
				test.check(t, body)
			}
		})
	}
}

func getTokens(t *testing.T, server *httptest.Server) (string, string) {
	t.Helper()

	_, body := call(t, server, "POST", "/auth/login", `{"username":"emilys","password":"emilyspass"}`, "")
	var response struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}
	decode(t, body, &response)
	return response.AccessToken, response.RefreshToken
}

func expectTotal(t *testing.T, body []byte, want int) {
	t.Helper()
	var response struct {
		Total int `json:"total"`
	}
	decode(t, body, &response)
	if response.Total != want {
		t.Fatalf("total = %d, want %d", response.Total, want)
	}
}

func expectMessage(t *testing.T, body []byte, want string) {
	t.Helper()
	var response struct {
		Message string `json:"message"`
	}
	decode(t, body, &response)
	if response.Message != want {
		t.Fatalf("message = %q, want %q", response.Message, want)
	}
}

func expectArrayLen(t *testing.T, body []byte, want int) {
	t.Helper()
	var rows []any
	decode(t, body, &rows)
	if len(rows) != want {
		t.Fatalf("len = %d, want %d", len(rows), want)
	}
}

func expectKeys(t *testing.T, body []byte, want ...string) {
	t.Helper()
	var object map[string]any
	decode(t, body, &object)

	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	sort.Strings(want)
	if !slices.Equal(keys, want) {
		t.Fatalf("keys = %v, want %v", keys, want)
	}
}

func TestLists(t *testing.T) {
	server := newServer(t)
	run(t, server, "", []caseSpec{
		{"products", "GET", "/products?limit=0", "", false, 200, func(t *testing.T, body []byte) { expectTotal(t, body, 194) }},
		{"users", "GET", "/users?limit=0", "", false, 200, func(t *testing.T, body []byte) { expectTotal(t, body, 208) }},
		{"posts", "GET", "/posts?limit=0", "", false, 200, func(t *testing.T, body []byte) { expectTotal(t, body, 251) }},
		{"comments", "GET", "/comments?limit=0", "", false, 200, func(t *testing.T, body []byte) { expectTotal(t, body, 340) }},
		{"carts", "GET", "/carts?limit=0", "", false, 200, func(t *testing.T, body []byte) { expectTotal(t, body, 208) }},
		{"todos", "GET", "/todos?limit=0", "", false, 200, func(t *testing.T, body []byte) { expectTotal(t, body, 254) }},
		{"quotes", "GET", "/quotes?limit=0", "", false, 200, func(t *testing.T, body []byte) { expectTotal(t, body, 1454) }},
		{"recipes", "GET", "/recipes?limit=0", "", false, 200, func(t *testing.T, body []byte) { expectTotal(t, body, 50) }},
	})
}

func TestOptions(t *testing.T) {
	server := newServer(t)
	run(t, server, "", []caseSpec{
		{"skip and limit", "GET", "/products?limit=5&skip=10", "", false, 200, func(t *testing.T, body []byte) {
			var response struct {
				Skip  int `json:"skip"`
				Limit int `json:"limit"`
			}
			decode(t, body, &response)
			if response.Skip != 10 || response.Limit != 5 {
				t.Fatalf("skip = %d, limit = %d", response.Skip, response.Limit)
			}
		}},
		{"sort", "GET", "/products?limit=3&sortBy=title&order=asc", "", false, 200, func(t *testing.T, body []byte) {
			var response struct {
				Products []struct {
					Title string `json:"title"`
				} `json:"products"`
			}
			decode(t, body, &response)
			if response.Products[0].Title != "300 Touring" {
				t.Fatalf("first title = %q", response.Products[0].Title)
			}
		}},
		{"select", "GET", "/products?limit=1&select=title,price", "", false, 200, func(t *testing.T, body []byte) {
			var response struct {
				Products []map[string]any `json:"products"`
			}
			decode(t, body, &response)
			keys := make([]string, 0, len(response.Products[0]))
			for key := range response.Products[0] {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			if !slices.Equal(keys, []string{"id", "price", "title"}) {
				t.Fatalf("keys = %v", keys)
			}
		}},
	})
}

func TestRelations(t *testing.T) {
	server := newServer(t)
	run(t, server, "", []caseSpec{
		{"user posts", "GET", "/users/1/posts", "", false, 200, func(t *testing.T, body []byte) { expectTotal(t, body, 1) }},
		{"post comments", "GET", "/posts/1/comments", "", false, 200, func(t *testing.T, body []byte) { expectTotal(t, body, 3) }},
		{"cart user", "GET", "/carts/user/1", "", false, 200, func(t *testing.T, body []byte) { expectTotal(t, body, 1) }},
		{"product category", "GET", "/products/category/beauty", "", false, 200, func(t *testing.T, body []byte) { expectTotal(t, body, 5) }},
		{"post tag", "GET", "/posts/tag/history", "", false, 200, func(t *testing.T, body []byte) { expectTotal(t, body, 56) }},
		{"meal type", "GET", "/recipes/meal-type/breakfast", "", false, 200, func(t *testing.T, body []byte) { expectTotal(t, body, 4) }},
	})
}

func TestSearch(t *testing.T) {
	server := newServer(t)
	run(t, server, "", []caseSpec{
		{"product search", "GET", "/products/search?q=phone", "", false, 200, func(t *testing.T, body []byte) { expectTotal(t, body, 23) }},
		{"user filter", "GET", "/users/filter?key=hair.color&value=Brown", "", false, 200, func(t *testing.T, body []byte) { expectTotal(t, body, 23) }},
		{"recipe search", "GET", "/recipes/search?q=chicken", "", false, 200, func(t *testing.T, body []byte) { expectTotal(t, body, 8) }},
	})
}

func TestDerivedAndRandom(t *testing.T) {
	server := newServer(t)
	run(t, server, "", []caseSpec{
		{"categories", "GET", "/products/categories", "", false, 200, func(t *testing.T, body []byte) { expectArrayLen(t, body, 24) }},
		{"category list", "GET", "/products/category-list", "", false, 200, func(t *testing.T, body []byte) { expectArrayLen(t, body, 24) }},
		{"post tags", "GET", "/posts/tags", "", false, 200, func(t *testing.T, body []byte) { expectArrayLen(t, body, 170) }},
		{"post tag list", "GET", "/posts/tag-list", "", false, 200, func(t *testing.T, body []byte) { expectArrayLen(t, body, 170) }},
		{"recipe tags", "GET", "/recipes/tags", "", false, 200, func(t *testing.T, body []byte) { expectArrayLen(t, body, 87) }},
		{"todo random", "GET", "/todos/random", "", false, 200, func(t *testing.T, body []byte) {
			var row map[string]any
			decode(t, body, &row)
			if row["id"] == nil {
				t.Fatal("no id in random todo")
			}
		}},
		{"todo random n", "GET", "/todos/random/3", "", false, 200, func(t *testing.T, body []byte) { expectArrayLen(t, body, 3) }},
		{"quote random", "GET", "/quotes/random", "", false, 200, func(t *testing.T, body []byte) {
			var row map[string]any
			decode(t, body, &row)
			if row["quote"] == nil {
				t.Fatal("no quote in random quote")
			}
		}},
	})
}

func TestWrites(t *testing.T) {
	server := newServer(t)
	run(t, server, "", []caseSpec{
		{"add", "POST", "/products/add", `{"title":"Test","price":5}`, false, 201, func(t *testing.T, body []byte) {
			var row map[string]any
			decode(t, body, &row)
			if row["id"] != float64(195) {
				t.Fatalf("id = %v", row["id"])
			}
		}},
		{"patch", "PATCH", "/products/1", `{"title":"Patched"}`, false, 200, func(t *testing.T, body []byte) {
			var row map[string]any
			decode(t, body, &row)
			if row["title"] != "Patched" {
				t.Fatalf("title = %v", row["title"])
			}
		}},
		{"delete", "DELETE", "/products/1", "", false, 200, func(t *testing.T, body []byte) {
			var row map[string]any
			decode(t, body, &row)
			if row["isDeleted"] != true {
				t.Fatal("isDeleted is not true")
			}
		}},
	})
}

func TestAuth(t *testing.T) {
	server := newServer(t)
	access, refresh := getTokens(t, server)
	run(t, server, access, []caseSpec{
		{"login keys", "POST", "/auth/login", `{"username":"emilys","password":"emilyspass"}`, false, 200, func(t *testing.T, body []byte) {
			expectKeys(t, body, "accessToken", "email", "firstName", "gender", "id", "image", "lastName", "refreshToken", "username")
		}},
		{"me", "GET", "/auth/me", "", true, 200, func(t *testing.T, body []byte) {
			var row map[string]any
			decode(t, body, &row)
			if row["username"] != "emilys" {
				t.Fatalf("username = %v", row["username"])
			}
		}},
		{"me no token", "GET", "/auth/me", "", false, 401, func(t *testing.T, body []byte) {
			expectMessage(t, body, "Access Token is required")
		}},
		{"bad credentials", "POST", "/auth/login", `{"username":"nope","password":"nope"}`, false, 400, func(t *testing.T, body []byte) {
			expectMessage(t, body, "Invalid credentials")
		}},
		{"refresh", "POST", "/auth/refresh", `{"refreshToken":"` + refresh + `"}`, false, 200, func(t *testing.T, body []byte) {
			expectKeys(t, body, "accessToken", "refreshToken")
		}},
		{"logout", "POST", "/auth/logout", "", true, 200, func(t *testing.T, body []byte) {
			expectMessage(t, body, "Logged out successfully")
		}},
	})
}

func TestErrors(t *testing.T) {
	server := newServer(t)
	run(t, server, "", []caseSpec{
		{"missing product", "GET", "/products/9999", "", false, 404, func(t *testing.T, body []byte) {
			expectMessage(t, body, "Product with id '9999' not found")
		}},
		{"bad id", "GET", "/products/abc", "", false, 404, func(t *testing.T, body []byte) {
			expectMessage(t, body, "Product with id 'abc' not found")
		}},
		{"unknown path", "GET", "/nothing", "", false, 404, func(t *testing.T, body []byte) {
			expectMessage(t, body, "Not Found")
		}},
		{"wrong method", "POST", "/products/1", "", false, 405, func(t *testing.T, body []byte) {
			expectMessage(t, body, "Method Not Allowed")
		}},
		{"panic", "GET", "/panic", "", false, 500, func(t *testing.T, body []byte) {
			expectMessage(t, body, "internal server error")
		}},
		{"health after panic", "GET", "/test", "", false, 200, nil},
	})
}

func TestMiddleware(t *testing.T) {
	server := newServer(t)

	response, err := http.Get(server.URL + "/test")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer response.Body.Close()
	if response.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal("missing CORS header")
	}

	start := time.Now()
	call(t, server, "GET", "/test?delay=100", "", "")
	if time.Since(start) < 100*time.Millisecond {
		t.Fatal("delay was not applied")
	}
}
