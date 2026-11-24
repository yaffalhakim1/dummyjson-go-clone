# Plan — DummyJSON clone in Go

Project 3 of the Go course. Learner writes all code. This file is the map, not
the code. Facts below were checked against github.com/Ovi/DummyJSON and the
live service on 2026-11-12.

## 1. Goal

Build an HTTP server in Go that serves the same paths, the same query
parameters, the same JSON envelopes, and the same error wording as
dummyjson.com, for the eight core resources plus auth. Port 8888.

**Non-goals, for now.** Image generator, identicon, QR and barcode, webhook,
2FA, custom-response, IP tool, docs site, rate limiter, database persistence.
These are optional later stages. The project is complete without them.

## 2. Contract to reproduce (verified)

Port 8888. Every resource supports `limit`, `skip`, `select`, `sortBy`,
`order`, and `delay`.

| Path | Method | Result |
| --- | --- | --- |
| `/test`, `/ping`, `/health` | GET | `{"status":"ok"}` |
| `/products` | GET | envelope |
| `/products/{id}` | GET | one row, 404 if missing |
| `/products/search?q=` | GET | envelope, matches title or description |
| `/products/categories` | GET | array of `{slug,name,url}` |
| `/products/category-list` | GET | array of slug strings |
| `/products/category/{slug}` | GET | envelope |
| `/products/add` | POST | new row, status 201, `id = len+1` |
| `/products/{id}` | PUT, PATCH | merged row |
| `/products/{id}` | DELETE | row plus `isDeleted`, `deletedOn` |
| `/users`, `/users/{id}` | GET | as products |
| `/users/search?q=` | GET | envelope |
| `/users/filter?key=hair.color&value=Brown` | GET | envelope, nested key path |
| `/users/{id}/posts`, `/users/{id}/carts`, `/users/{id}/todos` | GET | envelope |
| `/users/login` | POST | same body as `/auth/login` |
| `/users/{id}` | PUT, PATCH, DELETE | simulated |
| `/carts`, `/carts/{id}`, `/carts/user/{userId}` | GET | envelope |
| `/carts/{id}` | PUT, PATCH, DELETE | simulated |
| `/posts`, `/posts/{id}`, `/posts/search?q=` | GET | envelope |
| `/posts/tags`, `/posts/tag-list` | GET | tags |
| `/posts/tag/{tag}`, `/posts/user/{userId}` | GET | envelope |
| `/posts/{id}/comments` | GET | envelope |
| `/posts/{id}` | PUT, PATCH, DELETE | simulated |
| `/comments`, `/comments/{id}`, `/comments/post/{postId}` | GET | envelope |
| `/comments/{id}` | PUT, PATCH, DELETE | simulated |
| `/todos`, `/todos/{id}`, `/todos/user/{userId}` | GET | envelope |
| `/todos/random`, `/todos/random/{n}` | GET | one row, or N rows |
| `/quotes`, `/quotes/{id}` | GET | envelope |
| `/quotes/random`, `/quotes/random/{n}` | GET | one row, or N rows |
| `/recipes`, `/recipes/{id}`, `/recipes/search?q=` | GET | envelope |
| `/recipes/tags`, `/recipes/tag/{tag}`, `/recipes/meal-type/{type}` | GET | tags or envelope |
| `/auth/login` | POST | `{accessToken, refreshToken, id, username, email, firstName, lastName, gender, image}` |
| `/auth/me` | GET | user, needs `Authorization: Bearer <token>` |
| `/auth/refresh` | POST | new token pair |
| `/auth/logout` | POST | `{message: "..."}`, needs token |

**List envelope.** `{"<resource>": [...], "total": N, "skip": S, "limit": L}`
where `L` is the length of the returned page. Pipeline order is fixed:
sort, then skip, then limit, then select. Default `limit` is 30. `limit=0`
returns every row. `select=title,price` projects those keys only.

**Error shape.** `{"message": "Product with id '9999' not found"}`, status 404.
Wrong method or unknown path: 404 with the same shape.

**Singular aliases.** Upstream also accepts `/product`, `/user`, `/post`,
`/comment`, `/cart`, `/todo`, `/quote`, `/recipe`. Stage 10 adds them if time
allows.

## 3. Data

Copy `database/*.json` from the upstream repo into `data/`. This is already
done in the scaffold. Row counts, used by tests as `total`:

| File | Rows | Resource key |
| --- | --- | --- |
| products.json | 194 | products |
| users.json | 208 | users |
| posts.json | 251 | posts |
| comments.json | 340 | comments |
| carts.json | 208 | carts |
| todos.json | 254 | todos |
| quotes.json | 1454 | quotes |
| recipes.json | 50 | recipes |

Every row has an `id`. Users carry `username` and `password` for login.
Categories and tag lists are derived at boot from the rows, like upstream.

## 4. Architecture

```
Y:\Learn\Go\Backend\dummyjson
  go.mod                     module dummyjson, go 1.27.3
  cmd/server/main.go         flags, store load, router, listen, graceful shutdown
  internal/api/
    server.go                Server struct: store, mux, config
    router.go                all route registration, one place
    respond.go               writeJSON, writeError, status helpers
    query.go                 options parsing, list engine, sort, select
    middleware.go            recover, logger, CORS, JSON, body cap, delay
    products.go              product handlers
    users.go                 user handlers
    carts.go                 cart handlers
    posts.go                 post handlers
    comments.go              comment handlers
    todos.go                 todo handlers
    quotes.go                quote handlers
    recipes.go               recipe handlers
    auth.go                  login, me, refresh, logout
  internal/store/
    store.go                 Store type, lookup, derived indexes
    load.go                  read data/*.json, decode, derive categories
  internal/auth/
    tokens.go                JWT sign and verify, claims
  data/                      the eight JSON files
  test/e2e_test.go           httptest suite over the real router
  PLAN.md, TEACHING.md
```

Budgets: no file over 400 lines, no function over 60 lines. One resource per
file. No `utils` folder. Shared helpers live in `respond.go` or `query.go`
because those names say what they own.

Handler shape, every resource:

```go
func (s *Server) listProducts(w http.ResponseWriter, r *http.Request)
```

One `Server` value carries the store and config, so there are no globals and no
`init`. This is the dependency injection pattern, the Go version of passing
`app` into a router.

## 5. Decisions

**D1. Rows are `map[string]any`, not typed structs.**
The upstream data is JSON with unknown depth (`user.bank.cardNumber`,
`product.meta.qrCode`). Search, filter, sort, and `select` must work on any
key. One engine then serves all eight resources. Cost: no compile-time field
safety, and type assertions are needed. Mitigation: two helpers,
`str(m, key)` and `num(m, key)`, and nothing else reads maps directly. Typed
structs are used where input is parsed: request bodies, auth response, error
body. The tradeoff is real. Stage 11 includes an optional exercise that
rewrites Products as structs to feel the cost.

**D2. Standard library router, no framework.**
Go 1.22 and later `ServeMux` supports `GET /products/{id}` and
`r.PathValue("id")`. That covers every route in the table. A framework would
add a dependency and hide the thing we are learning.

**D3. Writes are simulated, the store is read-only.**
Upstream does the same. This means no locks, no races, no disk writes, and
`go test -race` stays honest. If persistence is added later, the store needs
`sync.RWMutex`; that is a documented extension, not part of this plan.

**D4. Limits that upstream does not have.**
`delay` is capped at 10 seconds, request bodies at 300 KB
(`http.MaxBytesReader`), server read header timeout at 5 seconds, idle timeout
at 60 seconds. A mock API must not become a way to hold thousands of
connections. This is the one place where we knowingly differ from upstream.

**D5. Auth is JWT HS256 with `golang-jwt/jwt/v5`.**
Access token lives 60 minutes, refresh token 7 days, same as upstream. The
signing key comes from `JWT_SECRET`, defaulting to a fixed dev value with a
loud log line. Passwords stay plaintext in the data file because upstream
ships them that way. This is a test double, never a real auth design, and the
README must say so.

**D6. Tests are E2E only, over HTTP.**
`httptest.NewServer` around the real router, then plain HTTP calls. No test
reaches into `internal/`. That is the same promise a client gets.

## 6. Stage roadmap

One stage is one lesson, in the format of TEACHING.md. Each stage ends with a
working server and evidence the learner pastes back.

**Stage 0. Boot.** Problem: nothing listens. Learn: `package main`, `main`,
`http.Server`, `http.Handler`, `http.HandlerFunc`, `json.NewEncoder`,
`log/slog`, graceful shutdown with `os/signal` and `context`. Done: `GET
/test` returns `{"status":"ok"}` and Ctrl+C shuts down cleanly. Pattern: the
handler interface.

**Stage 1. First resource.** Problem: `/products` is 404. Learn: `os.ReadFile`,
`json.Unmarshal` into `[]map[string]any`, `any` and `interface{}`, type
assertion with comma-ok, envelope as a map, `Total` from `len`. Done: `GET
/products` returns 30 rows and `total: 194`. Pattern: load once, serve from
memory.

**Stage 2. Query options.** Problem: every request returns the same 30 rows.
Learn: `r.URL.Query()`, `strconv.Atoi`, `select` on presence, defaults, the
sort, skip, limit, select pipeline, `sort.SliceStable`, comparing numbers as
numbers and strings as strings, projection by key list. Done: `limit`, `skip`,
`sortBy=title&order=asc`, `select=title,price` all match upstream output.
Pattern: pipeline.

**Stage 3. One row and errors.** Problem: `/products/1` is 404. Learn: path
wildcards, `r.PathValue`, `http.Error` versus your own JSON error writer,
custom error values, `errors.Is`, status codes, and the upstream message
wording. Done: `/products/1` matches upstream, `/products/9999` returns the
exact 404 body. Pattern: error as a value, not an exception.

**Stage 4. Split the router and add the first test.** Problem: `main.go` is
growing and nothing is tested. Learn: splitting into `internal/api`,
registering routes in one place, table-driven `httptest` with `t.Run`.
Done: `go test ./...` passes with at least six cases. Pattern: composition
root.

**Stage 5. Search and filter.** Problem: no `q`, no nested filter. Learn:
`strings.Contains` and `strings.ToLower`, splitting `hair.color` on `.` and
walking the map, comparing a filter value as string, why search is O(n) here
and when that stops being acceptable. Done: `/products/search?q=phone` and
`/users/filter?key=hair.color&value=Brown` match upstream totals. Pattern:
inverted index deferred, scan now.

**Stage 6. Relations.** Problem: nested paths are 404. Learn: filtering by a
foreign key, small join helpers, why a naive per-row lookup is an N+1 and why
it is fine in memory but not across a network. Done: `/users/1/posts`,
`/posts/1/comments`, `/carts/user/1` match upstream. Pattern: derived view.

**Stage 7. Simulated writes.** Problem: POST, PUT, PATCH, DELETE return 405.
Learn: `json.NewDecoder`, `http.MaxBytesReader`, reading a body once, merging a
patch over an existing row, `id = len+1`, `time.Now().UTC().Format(time.RFC3339)`,
status 201 versus 200, and why we never mutate the store. Done: `/products/add`
returns a row with `id: 195`; DELETE returns `isDeleted: true` and the row is
still there on the next GET. Pattern: copy on write, request-scoped.

**Stage 8. Middleware.** Problem: no CORS, no logging, no delay, and a panic
kills the connection. Learn: `func(http.Handler) http.Handler`, closure
capturing config, `defer recover`, a `ResponseWriter` wrapper that records the
status, order of the chain, `OPTIONS` preflight, and the safe delay cap. Done:
`?delay=500` waits half a second, a forced panic returns 500 JSON and the
server survives, every response carries the right headers. Pattern: middleware
chain, the Go form of `app.use`.

**Stage 9. Auth.** Problem: `/auth/login` does not exist. Learn: decoding a
typed body, lookup by username, JWT signing and verification, claims,
`Authorization: Bearer`, putting a value on `r.Context()` and reading it back,
401 versus 403. Done: login returns two tokens, `/auth/me` works with the
token and returns 401 without it, `/auth/refresh` rotates. Pattern: decorator
around a handler.

**Stage 10. The rest of the resources.** Problem: five resources still 404.
Learn: route tables as data, reusing one engine, random selection with
`math/rand/v2`, derived tag lists, singular aliases. Done: every path in the
table answers. Pattern: table-driven registration.

**Stage 11. Harden and finish.** Problem: parity is unproven. Learn: a
table-driven E2E suite covering every endpoint, `go vet`, `go test -race`,
`gofmt`, a Dockerfile, config via environment, and a README with a curl cheat
sheet. Done: the checklist in section 8 is fully ticked. Pattern: the seams
are the tests.

## 7. Verification

- `go vet ./...` clean at every stage.
- `go test ./...` and `go test -race ./...` clean from Stage 4.
- Every stage proven by `curl` output pasted by the learner, compared against
  `https://dummyjson.com` for the same request.
- Parity checks compare only stable fields: status, envelope keys, `total`,
  `skip`, `limit`, and the shape of one row. Upstream data can change.
- The final artifact is `test/e2e_test.go`: one run proves the whole contract.

## 8. Definition of done

1. All paths in section 2 answer with the right status and shape.
2. `total` equals the row counts in section 3.
3. The list pipeline order is sort, skip, limit, select.
4. `limit=0` returns all rows.
5. 404 and 405 return the JSON error shape.
6. Panics return 500 JSON and the server keeps running.
7. `delay` is capped, bodies are capped, servers have timeouts.
8. `/auth/login`, `/auth/me`, `/auth/refresh` work, protected routes reject
   missing tokens with 401.
9. `go vet` and `go test -race` are clean.
10. README documents the difference from upstream: D3 and D5.

## 9. Risks

- **Scope creep.** Eight resources plus auth is already large. Utility tools
  stay out unless Stage 11 is finished early.
- **Map rows invite typos.** `row["titel"]` compiles. The two helper functions
  and the E2E suite are the guard.
- **JWT drift.** `golang-jwt/jwt/v5` is the only dependency. If the API moves,
  pin the version in `go.mod` and read the release notes before upgrading.
- **Upstream data changes.** Tests must never assert on a specific title or
  price from the seed files beyond row counts, or they will break on a data
  refresh.
