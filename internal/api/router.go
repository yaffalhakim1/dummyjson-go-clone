package api

import "net/http"

func (s *Server) routes(mux *http.ServeMux) {
	s.registerHealth(mux)
	s.registerProducts(mux)
	s.registerUsers(mux)
	s.registerPosts(mux)
	s.registerComments(mux)
	s.registerCarts(mux)
	s.registerTodos(mux)
	s.registerQuotes(mux)
	s.registerRecipes(mux)
	s.registerAuth(mux)
	s.registerAliases(mux)
	mux.HandleFunc("GET /panic", s.handlePanic)
}

// Register the three health paths. They all answer the same way.
func (s *Server) registerHealth(mux *http.ServeMux) {
	mux.HandleFunc("GET /test", s.handleTest)
	mux.HandleFunc("GET /ping", s.handleTest)
	mux.HandleFunc("GET /health", s.handleTest)
}

// Register every product path, including writes and derived category lists.
func (s *Server) registerProducts(mux *http.ServeMux) {
	rows := s.store.Rows.Products
	mux.HandleFunc("GET /products", s.handleListProducts)
	mux.HandleFunc("GET /products/search", s.handleSearchProducts)
	mux.HandleFunc("GET /products/categories", s.handleProductCategories)
	mux.HandleFunc("GET /products/category-list", s.handleProductCategoryList)
	mux.Handle("GET /products/category/{slug}", s.listByField("products", "category", "slug", rows))
	mux.Handle("GET /products/{id}", s.getResource("products", "Product", rows))
	mux.HandleFunc("POST /products/add", s.handleAddProduct)
	mux.Handle("PUT /products/{id}", http.HandlerFunc(s.handleUpdateProduct))
	mux.Handle("PATCH /products/{id}", http.HandlerFunc(s.handleUpdateProduct))
	mux.Handle("DELETE /products/{id}", http.HandlerFunc(s.handleDeleteProduct))
}

// Register the user list, filter, detail, relations and login alias.
func (s *Server) registerUsers(mux *http.ServeMux) {
	rows := s.store.Rows.Users
	mux.HandleFunc("GET /users", s.handleListUsers)
	mux.HandleFunc("GET /users/filter", s.handleFilterUsers)
	mux.Handle("GET /users/{id}", s.getResource("users", "User", rows))
	mux.Handle("GET /users/{id}/posts", http.HandlerFunc(s.handleUserPosts))
	mux.Handle("GET /users/{id}/carts", http.HandlerFunc(s.handleUserCartsByID))
	mux.Handle("GET /users/{id}/todos", http.HandlerFunc(s.handleUserTodos))
	mux.HandleFunc("POST /users/login", s.handleLogin)
}

// Register the post list, search, tags, tag views and comments view.
func (s *Server) registerPosts(mux *http.ServeMux) {
	rows := s.store.Rows.Posts
	mux.Handle("GET /posts", s.listResource("posts", rows))
	mux.HandleFunc("GET /posts/search", s.handleSearchPosts)
	mux.HandleFunc("GET /posts/tags", s.handlePostTags)
	mux.HandleFunc("GET /posts/tag-list", s.handlePostTagList)
	mux.Handle("GET /posts/{id}", s.getResource("posts", "Post", rows))
	mux.Handle("GET /posts/{section}/{value}", http.HandlerFunc(s.handlePostsNested))
}

// Register the comment list, detail and by-post view.
func (s *Server) registerComments(mux *http.ServeMux) {
	rows := s.store.Rows.Comments
	mux.Handle("GET /comments", s.listResource("comments", rows))
	mux.Handle("GET /comments/{id}", s.getResource("comments", "Comment", rows))
	mux.Handle("GET /comments/post/{postId}", s.listByField("comments", "postId", "postId", rows))
}

// Register the cart list, detail and by-user view.
func (s *Server) registerCarts(mux *http.ServeMux) {
	rows := s.store.Rows.Carts
	mux.Handle("GET /carts", s.listResource("carts", rows))
	mux.Handle("GET /carts/{id}", s.getResource("carts", "Cart", rows))
	mux.Handle("GET /carts/user/{userId}", http.HandlerFunc(s.handleUserCarts))
}

// Register the todo list, random paths, by-user view and detail.
func (s *Server) registerTodos(mux *http.ServeMux) {
	rows := s.store.Rows.Todos
	mux.Handle("GET /todos", s.listResource("todos", rows))
	mux.HandleFunc("GET /todos/random", func(w http.ResponseWriter, r *http.Request) {
		s.writeRandom(w, r, rows)
	})
	mux.HandleFunc("GET /todos/random/{n}", func(w http.ResponseWriter, r *http.Request) {
		s.writeRandom(w, r, rows)
	})
	mux.Handle("GET /todos/user/{userId}", s.listByField("todos", "userId", "userId", rows))
	mux.Handle("GET /todos/{id}", s.getResource("todos", "Todo", rows))
}

// Register the quote list, random paths and detail.
func (s *Server) registerQuotes(mux *http.ServeMux) {
	rows := s.store.Rows.Quotes
	mux.Handle("GET /quotes", s.listResource("quotes", rows))
	mux.HandleFunc("GET /quotes/random", func(w http.ResponseWriter, r *http.Request) {
		s.writeRandom(w, r, rows)
	})
	mux.HandleFunc("GET /quotes/random/{n}", func(w http.ResponseWriter, r *http.Request) {
		s.writeRandom(w, r, rows)
	})
	mux.Handle("GET /quotes/{id}", s.getResource("quotes", "Quote", rows))
}

// Register the recipe list, search, tags, tag and meal-type views and detail.
func (s *Server) registerRecipes(mux *http.ServeMux) {
	rows := s.store.Rows.Recipes
	mux.Handle("GET /recipes", s.listResource("recipes", rows))
	mux.HandleFunc("GET /recipes/search", s.handleSearchRecipes)
	mux.HandleFunc("GET /recipes/tags", s.handleRecipeTags)
	mux.Handle("GET /recipes/tag/{tag}", s.listByField("recipes", "tags", "tag", rows))
	mux.Handle("GET /recipes/meal-type/{type}", s.listByField("recipes", "mealType", "type", rows))
	mux.Handle("GET /recipes/{id}", s.getResource("recipes", "Recipe", rows))
}

// Register the four auth paths. Two of them need the token guard.
func (s *Server) registerAuth(mux *http.ServeMux) {
	mux.HandleFunc("POST /auth/login", s.handleLogin)
	mux.Handle("GET /auth/me", s.withAuth(http.HandlerFunc(s.handleMe)))
	mux.HandleFunc("POST /auth/refresh", s.handleRefresh)
	mux.Handle("POST /auth/logout", s.withAuth(http.HandlerFunc(s.handleLogout)))
}

// Register a second door to each room. Singular paths answer like plurals.
func (s *Server) registerAliases(mux *http.ServeMux) {
	aliases := []struct {
		path, resource, display string
		rows                    []map[string]any
	}{
		{"/product", "products", "Product", s.store.Rows.Products},
		{"/user", "users", "User", s.store.Rows.Users},
		{"/post", "posts", "Post", s.store.Rows.Posts},
		{"/comment", "comments", "Comment", s.store.Rows.Comments},
		{"/cart", "carts", "Cart", s.store.Rows.Carts},
		{"/todo", "todos", "Todo", s.store.Rows.Todos},
		{"/quote", "quotes", "Quote", s.store.Rows.Quotes},
		{"/recipe", "recipes", "Recipe", s.store.Rows.Recipes},
	}
	for _, alias := range aliases {
		mux.Handle("GET "+alias.path, s.listResource(alias.resource, alias.rows))
		mux.Handle("GET "+alias.path+"/{id}", s.getResource(alias.resource, alias.display, alias.rows))
	}
}

// Answer a health check.
func (s *Server) handleTest(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Break on purpose. This exists only to prove the recover middleware works.
func (s *Server) handlePanic(w http.ResponseWriter, r *http.Request) {
	panic("forced panic for test")
}
