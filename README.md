# dummyjson

A Go clone of [DummyJSON](https://dummyjson.com). It serves placeholder data
for frontend work. Same paths, query parameters, envelopes and error wording.

## Run

```sh
go run ./cmd/server
```

The server listens on port 8888. Set `ADDR`, `DATA_DIR` and `JWT_SECRET` to
override the defaults.

## Test

```sh
go vet ./...
gofmt -l .
go test ./... -v
go test -race ./...
```

The suite in `test/e2e_test.go` starts a real server with `httptest` and calls
it over HTTP. It checks every resource, the list pipeline, relations, writes,
auth and the middleware.

## Curl cheat sheet

```sh
curl localhost:8888/test
curl 'localhost:8888/products?limit=5&skip=10'
curl 'localhost:8888/products?sortBy=title&order=asc&limit=3'
curl 'localhost:8888/products?select=title,price&limit=2'
curl localhost:8888/products/1
curl 'localhost:8888/products/search?q=phone'
curl localhost:8888/products/categories
curl 'localhost:8888/users/filter?key=hair.color&value=Brown'
curl localhost:8888/users/1/posts
curl localhost:8888/posts/1/comments
curl localhost:8888/todos/random/3
curl localhost:8888/quotes/random
curl localhost:8888/recipes/meal-type/breakfast
curl -X POST localhost:8888/products/add -H 'Content-Type: application/json' \
  -d '{"title":"Test","price":5}'
curl -X PATCH localhost:8888/products/1 -H 'Content-Type: application/json' \
  -d '{"price":1}'
curl -X DELETE localhost:8888/products/1
```

Login returns two tokens. Send the access token as a Bearer header.

```sh
curl -X POST localhost:8888/auth/login -H 'Content-Type: application/json' \
  -d '{"username":"emilys","password":"emilyspass"}'
curl localhost:8888/auth/me -H 'Authorization: Bearer <accessToken>'
curl -X POST localhost:8888/auth/refresh -H 'Content-Type: application/json' \
  -d '{"refreshToken":"<refreshToken>"}'
```

## Docker

```sh
docker build -t dummyjson .
docker run --rm -p 8888:8888 dummyjson
```

## Data

The eight files in `data/` come from
[github.com/Ovi/DummyJSON](https://github.com/Ovi/DummyJSON) (`database/`),
MIT licensed. Row counts: products 194, users 208, posts 251, comments 340,
carts 208, todos 254, quotes 1454, recipes 50.

## Deliberate differences from upstream

- **D3. Writes are simulated.** POST, PUT, PATCH and DELETE return a plausible
  row, but nothing is saved. The store is read-only after boot. Upstream does
  the same.
- **D5. Auth is a test double.** Passwords stay plaintext because upstream
  ships them that way. JWTs are signed with a dev secret by default. Never use
  this auth design in a real service.

We also cap `?delay=` at 10 seconds and request bodies at 300 KB, and set
server timeouts. Upstream has none of these.

## Docs

- `PLAN.md`: contract, architecture and decisions.
- `TEACHING.md`: the working agreement for the lessons.
