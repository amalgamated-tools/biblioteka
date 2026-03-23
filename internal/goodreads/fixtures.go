package goodreads

import _ "embed"

//go:embed fixtures/getBookByLegacyId_54493401.json
var GetBookByLegacyID_54493401 []byte

//go:embed fixtures/autocomplete.json
var AutoComplete []byte

//go:embed fixtures/getBook_amzn1.gr.book.v3.7WmufEffpivF1XTp.json
var GetBook_7WmufEffpivF1XTp []byte

//go:embed fixtures/getBookByAsin_B08FHBV4ZX.json
var GetBookByAsin_B08FHBV4ZX []byte
