package goodreads

import _ "embed"

//go:embed fixtures/54493401_legacy_id.json
var GetBookByLegacyIDResponse54493401 []byte

//go:embed fixtures/autocomplete.json
var AutoCompleteResponse []byte
