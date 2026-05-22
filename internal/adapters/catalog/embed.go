package catalog

import (
	officialcatalog "elydelva/one/catalog"
)

// NewCatalogEmbed returns a catalog backed by the services embedded into the
// one binary at build time (the "official" catalog).
func NewCatalogEmbed() *CatalogFS {
	return NewCatalogFromFS(officialcatalog.Services())
}
