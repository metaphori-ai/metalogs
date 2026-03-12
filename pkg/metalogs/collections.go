package metalogs

import (
	"fmt"
	"strings"
)

// CreateCollection creates a named collection of site+layer pairs.
// Name, site, and layer values are normalized to lowercase.
func (s *Store) CreateCollection(name string, members []SiteLayer) error {
	name = strings.ToLower(name)

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Remove existing members for this collection name
	if _, err := tx.Exec("DELETE FROM collections WHERE name = ?", name); err != nil {
		return err
	}

	stmt, err := tx.Prepare("INSERT INTO collections (name, site, layer) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, m := range members {
		if _, err := stmt.Exec(name, strings.ToLower(m.Site), strings.ToLower(m.Layer)); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetCollection returns a collection by name.
func (s *Store) GetCollection(name string) (*Collection, error) {
	name = strings.ToLower(name)
	rows, err := s.db.Query("SELECT site, layer FROM collections WHERE name = ? ORDER BY site, layer", name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	coll := &Collection{Name: name}
	for rows.Next() {
		var sl SiteLayer
		if err := rows.Scan(&sl.Site, &sl.Layer); err != nil {
			return nil, err
		}
		coll.Members = append(coll.Members, sl)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(coll.Members) == 0 {
		return nil, fmt.Errorf("collection %q not found", name)
	}
	return coll, nil
}

// ListCollections returns all collections.
func (s *Store) ListCollections() ([]Collection, error) {
	rows, err := s.db.Query("SELECT name, site, layer FROM collections ORDER BY name, site, layer")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	collMap := make(map[string]*Collection)
	var order []string
	for rows.Next() {
		var name string
		var sl SiteLayer
		if err := rows.Scan(&name, &sl.Site, &sl.Layer); err != nil {
			return nil, err
		}
		if _, ok := collMap[name]; !ok {
			collMap[name] = &Collection{Name: name}
			order = append(order, name)
		}
		collMap[name].Members = append(collMap[name].Members, sl)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var result []Collection
	for _, name := range order {
		result = append(result, *collMap[name])
	}
	return result, nil
}

// DeleteCollection removes a collection by name.
func (s *Store) DeleteCollection(name string) error {
	name = strings.ToLower(name)
	result, err := s.db.Exec("DELETE FROM collections WHERE name = ?", name)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("collection %q not found", name)
	}
	return nil
}
