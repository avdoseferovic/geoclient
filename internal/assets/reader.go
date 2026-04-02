package assets

// Reader loads named assets from a backing store.
type Reader interface {
	ReadFile(path string) ([]byte, error)
}
