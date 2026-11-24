package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Read all eight files, index products and users by id, and derive the
// category and tag lists. Nothing here runs again after boot.
func Load(dataDir string) (*Store, error) {
	products, err := loadRows(filepath.Join(dataDir, "products.json"))
	if err != nil {
		return nil, err
	}
	users, err := loadRows(filepath.Join(dataDir, "users.json"))
	if err != nil {
		return nil, err
	}
	posts, err := loadRows(filepath.Join(dataDir, "posts.json"))
	if err != nil {
		return nil, err
	}
	comments, err := loadRows(filepath.Join(dataDir, "comments.json"))
	if err != nil {
		return nil, err
	}
	carts, err := loadRows(filepath.Join(dataDir, "carts.json"))
	if err != nil {
		return nil, err
	}
	todos, err := loadRows(filepath.Join(dataDir, "todos.json"))
	if err != nil {
		return nil, err
	}
	quotes, err := loadRows(filepath.Join(dataDir, "quotes.json"))
	if err != nil {
		return nil, err
	}
	recipes, err := loadRows(filepath.Join(dataDir, "recipes.json"))
	if err != nil {
		return nil, err
	}

	categories, categoryList := deriveProductCategories(products)
	postTags, postTagList := derivePostTags(posts)

	return &Store{
		Rows: Rows{
			Products: products, Users: users, Posts: posts, Comments: comments,
			Carts: carts, Todos: todos, Quotes: quotes, Recipes: recipes,
		},
		ProductsByID: indexByID(products),
		UsersByID:    indexByID(users),
		Derived: Derived{
			ProductCategories:   categories,
			ProductCategoryList: categoryList,
			PostTags:            postTags,
			PostTagList:         postTagList,
			RecipeTags:          deriveRecipeTags(recipes),
		},
	}, nil
}

// Read one JSON array file into rows.
func loadRows(path string) ([]map[string]any, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rows []map[string]any
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

// Put rows into a map by id.
func indexByID(rows []map[string]any) map[int]map[string]any {
	index := make(map[int]map[string]any, len(rows))
	for _, row := range rows {
		id, ok := row["id"].(float64)
		if !ok {
			continue
		}
		index[int(id)] = row
	}
	return index
}

// Turn a slug into a readable name. "mens-shirts" becomes "Men's Shirts".
func titleFromSlug(slug string) string {
	words := strings.FieldsFunc(slug, func(letter rune) bool {
		return letter == '-' || letter == '_'
	})
	for i, word := range words {
		words[i] = strings.ToUpper(word[:1]) + word[1:]
	}
	name := strings.Join(words, " ")
	name = strings.ReplaceAll(name, "Mens ", "Men's ")
	return strings.ReplaceAll(name, "Womens ", "Women's ")
}

// Build the category objects and the slug list from the product rows.
func deriveProductCategories(products []map[string]any) ([]map[string]any, []string) {
	seen := map[string]bool{}
	for _, row := range products {
		category, ok := row["category"].(string)
		if ok && category != "" {
			seen[category] = true
		}
	}
	slugs := make([]string, 0, len(seen))
	for slug := range seen {
		slugs = append(slugs, slug)
	}
	sort.Strings(slugs)
	categories := make([]map[string]any, 0, len(slugs))
	for _, slug := range slugs {
		categories = append(categories, map[string]any{
			"slug": slug,
			"name": titleFromSlug(slug),
			"url":  "https://dummyjson.com/products/category/" + slug,
		})
	}
	return categories, slugs
}

// Build the tag objects and the slug list from the post rows.
func derivePostTags(posts []map[string]any) ([]map[string]any, []string) {
	seen := map[string]bool{}
	for _, row := range posts {
		tags, ok := row["tags"].([]any)
		if !ok {
			continue
		}
		for _, item := range tags {
			tag, ok := item.(string)
			if ok && tag != "" {
				seen[tag] = true
			}
		}
	}
	slugs := make([]string, 0, len(seen))
	for slug := range seen {
		slugs = append(slugs, slug)
	}
	sort.Strings(slugs)
	result := make([]map[string]any, 0, len(slugs))
	for _, slug := range slugs {
		result = append(result, map[string]any{
			"slug": slug,
			"name": titleFromSlug(slug),
			"url":  "https://dummyjson.com/posts/tag/" + slug,
		})
	}
	return result, slugs
}

// Collect the distinct recipe tags, sorted, as plain strings.
func deriveRecipeTags(recipes []map[string]any) []string {
	seen := map[string]bool{}
	for _, row := range recipes {
		tags, ok := row["tags"].([]any)
		if !ok {
			continue
		}
		for _, item := range tags {
			tag, ok := item.(string)
			if ok && tag != "" {
				seen[tag] = true
			}
		}
	}
	result := make([]string, 0, len(seen))
	for tag := range seen {
		result = append(result, tag)
	}
	sort.Strings(result)
	return result
}
