package metadata

type ExtractorOption func(*Extractor) error

func WithExiftoolBinaryPath(path string) ExtractorOption {
	return func(e *Extractor) error {
		e.path = path
		return nil
	}
}
