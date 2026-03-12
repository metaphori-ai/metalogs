package metalogs

// ListSites returns all distinct site values.
func (s *Store) ListSites() ([]string, error) {
	rows, err := s.db.Query("SELECT DISTINCT site FROM logs ORDER BY site")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sites []string
	for rows.Next() {
		var site string
		if err := rows.Scan(&site); err != nil {
			return nil, err
		}
		sites = append(sites, site)
	}
	return sites, rows.Err()
}

// ListLayers returns all distinct layer values for a given site.
func (s *Store) ListLayers(site string) ([]string, error) {
	rows, err := s.db.Query("SELECT DISTINCT layer FROM logs WHERE site = ? ORDER BY layer", site)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var layers []string
	for rows.Next() {
		var layer string
		if err := rows.Scan(&layer); err != nil {
			return nil, err
		}
		layers = append(layers, layer)
	}
	return layers, rows.Err()
}

// ListSiteLayers returns all distinct site+layer pairs.
func (s *Store) ListSiteLayers() ([]SiteLayer, error) {
	rows, err := s.db.Query("SELECT DISTINCT site, layer FROM logs ORDER BY site, layer")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pairs []SiteLayer
	for rows.Next() {
		var sl SiteLayer
		if err := rows.Scan(&sl.Site, &sl.Layer); err != nil {
			return nil, err
		}
		pairs = append(pairs, sl)
	}
	return pairs, rows.Err()
}
