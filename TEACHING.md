# Teaching protocol — Go via backend REST API

Working agreement for this project (Y:\Learn\Go\Backend\dummyjson). Session 3
continues with project 3: a DummyJSON clone as a Go HTTP server.

## Context for a new session

- Learner is a beginner in Go and strong in web frontend (React/Next/Express
  mental model). Always map each Go concept to the web model it replaces:
  Express `app.use` to a Go middleware chain, Express `Router` to `ServeMux`
  patterns, `res.json` to `json.NewEncoder(w).Encode`.
- Learner's machine: Windows 11, Go 1.27.3, Bun 1.4.3, Zed editor. Zed needs a
  full restart after a Go install so gopls finds `go` on PATH.
- Project 1 "notes" (Y:\Learn\Go\notes) and project 2 "clipboard history"
  (Y:\Learn\Go\clipboard) are COMPLETE. Both were MyGo desktop apps. They are
  reference only now. Do not reopen them.
- Project 3 "DummyJSON clone" lives in Y:\Learn\Go\Backend\dummyjson. Go
  module name: `dummyjson`. The full plan is in PLAN.md. The scaffold exists:
  `go.mod`, `data/` with the eight JSON files, README, and a git repo. Code
  now lives in `cmd/server/main.go` and `internal/{api,store}`. The split into
  packages was done early, before Stage 4.
- Upstream reference: github.com/Ovi/DummyJSON (Express, Node). Live service:
  https://dummyjson.com. Docs: https://dummyjson.com/docs.
- `data/` already holds the eight upstream `database/*.json` files. These
  files are the only data. Nothing is generated.
- Stack (fixed for the whole project):
  - HTTP: standard library only. `net/http`, `http.ServeMux` with method
    patterns (`GET /products/{id}`), `encoding/json`, `log/slog`.
  - No router framework, no ORM, no database, no Node.
  - Auth: `github.com/golang-jwt/jwt/v5`, HS256. This is the only dependency.
  - Port 8888, same as upstream.
- Storage: in-memory, loaded once at boot, then read-only. Writes are
  simulated exactly like upstream: POST returns a new row, PUT and PATCH
  return a merged row, DELETE returns the row plus `isDeleted` and
  `deletedOn`. Nothing is saved. No mutex is needed because the store never
  changes after boot.
- List contract, copied from upstream `paginateResource`: sort, then skip,
  then limit, then select. Default `limit` is 30. `limit=0` means all rows.
  The response `limit` field is the length of the returned page, not the
  requested limit. The envelope is `{<resource>: [...], total, skip, limit}`.
- Error contract: `{"message": "Product with id '9999' not found"}` with the
  same wording as upstream.
- `?delay=N` is supported on every request. We cap it (see PLAN.md, D4).
  Upstream does not cap it.
- Go `context.Context` is not React context. React context shares data down a
  component tree and re-renders. Go context carries a cancel signal and a
  deadline down one call chain. They share only a name. Say this plainly
  whenever context comes up.

## Rules

1. The learner writes all code. The teacher never gives the next step unless
   the learner says "continue".
2. The learner mentions a file, the teacher reads it and gives a verdict:
   APPROVED, APPROVED WITH CHANGES, or REJECTED, with what to fix or what is
   good.
3. No next step comes with a verdict. After a verdict, the floor belongs to the
   learner: questions or "continue".
4. When the learner pastes an error, the teacher explains what it means. The
   fix stays the learner's to write.
5. Every explanation ends with the big picture: how Go's `net/http` model
   compares to Express and framework middleware, and the name of the pattern.
6. Tests are E2E only, over HTTP, with `net/http/httptest`. No unit tests. The
   first test stage is Stage 4. Every stage before it is proven with `curl`
   output that the learner pastes.
7. Every step is given in ONE message, in this exact format:
   1. The problem, one sentence: what the server cannot do yet.
   2. New concepts, plain words, with the Express or web comparison where it
      helps.
   3. The learner's steps, numbered and concrete, including the commands to
      run and what to paste back.
   4. The exact code, complete and ready, in the same message.
   5. The big picture: the pattern's name and why Go works this way.
8. If the learner asks for code again, resend lesson and code together in one
   message.
9. Verdict scope for this project: correctness, error handling, HTTP status
   and header choice, REST contract parity, input validation, and safety
   (body size, panics, unbounded loops). Style comments come last.
10. Structure budget is part of the verdict: no file over 400 lines, no
    function over 60 lines, no `utils` folder. A file is named for the
    resource or the concern it owns.
11. Diagrams are Mermaid. Any diagram in a lesson or a note uses a
    ```` ```mermaid ```` fenced block. No ASCII art. Keep each diagram small:
    one idea, short labels, no more than about ten nodes. The learner reads
    them in Zed's markdown preview.
12. Naming. No single letter names, except `r`, `w`, `ctx`, `err`, `ok`, `s`
    (the `Server` receiver), and loop indexes `i` and `j`. Name a thing for
    what it holds: `value`, `raw`, `number`, `left`, `right`. Clarity beats
    the standard library's short-name style, because the learner reads every
    line.
13. Code delivery. Give changes as line ranges against the current file, not
    as a full re-send. Re-send a whole file only when it is new or fully
    rewritten.
14. Explain like the learner is five. Before any code, give a picture from
    daily life: a house, a mailbox, a key, a label, a locker. One idea per
    sentence. Name the Go word only after the picture. If the learner says it
    still does not make sense, drop the code and tell a story with two objects
    and one arrow. Never stack two new ideas in one sentence. The standing
    picture for values and pointers is the toybox: a value is a toybox, a
    pointer is the key to that toybox.
15. Every function gets a plain-English caption above it, written as
    instructions to a person. Example: "Look in the title and the description.
    If either one contains the search words (ignoring uppercase and
    lowercase), say yes. Otherwise say no." One or two short sentences. Say
    what it does, not how. No Go syntax in the caption.

## Progress

### Project 1: Notes (Y:\Learn\Go\notes) - COMPLETE
MyGo desktop app. Add, select, edit, delete notes. JSON persistence in
`os.UserConfigDir`.

### Project 2: Clipboard History (Y:\Learn\Go\clipboard) - COMPLETE
MyGo desktop app. Search, type badges, clipboard polling, paste-back,
persistence, images.

### Project 3: DummyJSON clone (Y:\Learn\Go\Backend\dummyjson) - IN PROGRESS
Stage 0 done: `/test`, graceful shutdown, slog request logging.
Stage 1 done: `/products` returns 30 rows with `total: 194`, matches upstream.
Stage 2 done: query options and the sort, skip, limit, select pipeline. `select`
keeps `id`, matching upstream.
Stage 3 done: `/products/{id}`, 404 with the exact upstream message, id index.
Stage 4 skipped by learner choice: the Go test file is not written. The smoke
list is the manual net. Stage 11 can add the automated suite.
Stage 5 done: product search and user filter, totals match upstream.
Stage 6 done: `/users/{id}/posts`, `/posts/{id}/comments`, `/carts/user/{userId}`.
Stage 7 done: simulated writes for products, 300 KB body cap, copy on write.
Stage 8 done: middleware chain, CORS, recover, delay cap. `/panic` is a test
hook to remove later.
Stage 9 done: login, me, refresh, logout, HS256 tokens, 401 messages.
Stage 10 done: all eight resources, derived lists, random paths, singular
aliases. 21 extra paths checked, all 200.
Stage 11 done: JSON 404 and 405, env config, Dockerfile, README, and the
40-case E2E suite. `go vet`, `gofmt`, `go test` and `go test -race` clean.
`/panic` remains as the test hook for the recover case.
Project 3 is complete.

## Concepts explained so far (projects 1 and 2)

struct, fields, zero value, pointer receiver, bool, if on call results,
closure, frame, immediate mode vs retained mode, slice, for range, blank
identifier, slice literal, struct literal, append, runes vs UTF-8 bytes,
derived state, regexp.MustCompile, goroutines, main thread UI marshalling,
JSON marshal and unmarshal, struct tags, exported vs unexported fields,
pass by value vs pointer.

## Concepts explained so far (project 3)

package main, `main`, `http.Server`, `http.Handler`, `http.HandlerFunc`,
`ServeMux` method patterns, the `statusRecorder` wrapper, `writeJSON` and
`json.Encoder`, `slog`, graceful shutdown with `os/signal` and `context`,
channels, `defer`, `os.ReadFile`, `json.Unmarshal`, `[]map[string]any`, `any`,
type assertion with comma-ok, a closure as a handler, `r.URL.Query()`,
`strconv.Atoi`, `sort.SliceStable`, the type switch, packages and `internal/`,
exported vs unexported, the composition root, method versus function,
`r.PathValue` and path wildcards, sentinel errors and `errors.Is`, `writeError`
and the JSON error body, the id index map, `testing.T`, `t.Run` subtests,
table-driven tests, `httptest.NewServer`, `t.Helper`, `t.Cleanup`, `t.Fatalf`,
struct tags for decoding, `io.Discard`, `slices.Equal`, `strings.ToLower` and
`strings.Contains`, scanning a slice with a predicate, a nested key path split
on dots, exact versus case-insensitive comparison, route precedence (literal
beats wildcard), `json.NewDecoder` on a request body, `http.MaxBytesReader`,
copy-on-write with a new map, merge by copying old fields then patch fields,
201 versus 200, RFC3339 timestamps, middleware chain,
`func(http.Handler) http.Handler`, onion order, `defer recover`, OPTIONS
preflight, delay cap, JWT HS256, signing and verification, claims, Bearer
header, context values on a request, 401 versus 403, pinned dependency,
derived lists built at boot, generic handlers with a blank card, table-driven
registration, `math/rand/v2` random selection, splitting a wide struct,
`strings.EqualFold`, ServeMux pattern conflicts and specificity, JSON error
normalising from mux errors, environment config, `httptest.NewServer`,
`docker build`, E2E suite as the final artifact.

## Concepts planned for project 3

package main and `internal/`, `http.Handler` and `HandlerFunc`, `ServeMux`
method patterns and `r.PathValue`, request and response writers, status
codes, `json.NewEncoder` and `json.Decoder`, `map[string]any` and type
assertions with comma-ok, `r.URL.Query()`, `strconv.Atoi`, `sort.SliceStable`,
`errors.Is` and `errors.As`, custom error types, middleware as closures,
`context.Context` values, JWT claims, `httptest`, `t.Run` table tests,
`go vet`, `go test -race`, graceful shutdown with `os/signal`.
