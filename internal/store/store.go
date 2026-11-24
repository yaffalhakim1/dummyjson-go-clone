package store

import "errors"

var ErrNotFound = errors.New("not found")

// All eight tables, loaded once at boot.
type Rows struct {
	Products []map[string]any
	Users    []map[string]any
	Posts    []map[string]any
	Comments []map[string]any
	Carts    []map[string]any
	Todos    []map[string]any
	Quotes   []map[string]any
	Recipes  []map[string]any
}

// Lists derived from the rows at boot. Never read from disk later.
type Derived struct {
	ProductCategories   []map[string]any
	ProductCategoryList []string
	PostTags            []map[string]any
	PostTagList         []string
	RecipeTags          []string
}

// One store holds the rows, two id maps and the derived lists.
type Store struct {
	Rows         Rows
	ProductsByID map[int]map[string]any
	UsersByID    map[int]map[string]any
	Derived      Derived
}

// Find a product by id, or return ErrNotFound.
func (s *Store) FindProduct(id int) (map[string]any, error) {
	product, ok := s.ProductsByID[id]
	if !ok {
		return nil, ErrNotFound
	}
	return product, nil
}

// Find a user by id, or return ErrNotFound.
func (s *Store) FindUser(id int) (map[string]any, error) {
	user, ok := s.UsersByID[id]
	if !ok {
		return nil, ErrNotFound
	}
	return user, nil
}
