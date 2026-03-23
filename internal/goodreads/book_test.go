package goodreads

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/Khan/genqlient/graphql"
	"github.com/stretchr/testify/require"
)

func Test_GetBookByASIN(t *testing.T) {
	var response GetBookByAsinResponse
	err := json.Unmarshal(GetBookByAsin_B08FHBV4ZX, &response)
	if err != nil {
		t.Fatalf("failed to unmarshal JSON response: %v", err)
	}
	client := &Client{
		client: &mockGraphQLClient{
			handler: func(req *graphql.Request, resp *graphql.Response) error {
				data := resp.Data.(*GetBookByAsinResponse)
				*data = response
				return nil
			},
		},
	}

	r, err := client.GetBookByASIN(t.Context(), response.GetBookByAsin.Work.BestBook.Details.Asin)
	if err != nil {
		t.Fatalf("failed to search by ASIN: %v", err)
	}

	expected, err := loadBookResult(t.Context(), response.GetBookByAsin.Work)
	if err != nil {
		t.Fatalf("failed to load expected BookResult from response: %v", err)
	}

	require.Equal(t, expected, r)
	require.Equal(t, expected.WorkID, r.WorkID)
	require.Equal(t, expected.WorkLegacyID, r.WorkLegacyID)
	require.Equal(t, expected.BookID, r.BookID)
	require.Equal(t, expected.BookLegacyID, r.BookLegacyID)
	require.Equal(t, expected.BookImageURL, r.BookImageURL)
	require.Equal(t, expected.BookTitle, r.BookTitle)
	require.Equal(t, expected.BookASIN, r.BookASIN)
	require.Equal(t, expected.BookISBN, r.BookISBN)
	require.Equal(t, expected.BookISBN13, r.BookISBN13)
	require.Equal(t, expected.BookLanguage, r.BookLanguage)
	require.Equal(t, expected.AuthorID, r.AuthorID)
	require.Equal(t, expected.AuthorName, r.AuthorName)
	require.Equal(t, expected.AuthorLegacyID, r.AuthorLegacyID)
	require.Equal(t, expected.AuthorProfileImageURL, r.AuthorProfileImageURL)
}

func Test_GetBookByASINError(t *testing.T) {
	client := &Client{
		client: &mockGraphQLClient{
			handler: func(req *graphql.Request, resp *graphql.Response) error {
				return errors.New("mock error")
			},
		},
	}

	_, err := client.GetBookByASIN(t.Context(), "mock-asin")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func Test_GetBook(t *testing.T) {
	var response GetBookResponse
	err := json.Unmarshal(GetBook_7WmufEffpivF1XTp, &response)
	if err != nil {
		t.Fatalf("failed to unmarshal JSON response: %v", err)
	}
	client := &Client{
		client: &mockGraphQLClient{
			handler: func(req *graphql.Request, resp *graphql.Response) error {
				data := resp.Data.(*GetBookResponse)
				*data = response
				return nil
			},
		},
	}

	r, err := client.GetBookByID(t.Context(), response.GetBook.Work.BestBook.Id)
	if err != nil {
		t.Fatalf("failed to search by ID: %v", err)
	}

	expected, err := loadBookResult(t.Context(), response.GetBook.Work)
	if err != nil {
		t.Fatalf("failed to load expected BookResult from response: %v", err)
	}

	require.Equal(t, expected, r)
	require.Equal(t, expected.WorkID, r.WorkID)
	require.Equal(t, expected.WorkLegacyID, r.WorkLegacyID)
	require.Equal(t, expected.BookID, r.BookID)
	require.Equal(t, expected.BookLegacyID, r.BookLegacyID)
	require.Equal(t, expected.BookImageURL, r.BookImageURL)
	require.Equal(t, expected.BookTitle, r.BookTitle)
	require.Equal(t, expected.BookASIN, r.BookASIN)
	require.Equal(t, expected.BookISBN, r.BookISBN)
	require.Equal(t, expected.BookISBN13, r.BookISBN13)
	require.Equal(t, expected.BookLanguage, r.BookLanguage)
	require.Equal(t, expected.AuthorID, r.AuthorID)
	require.Equal(t, expected.AuthorName, r.AuthorName)
	require.Equal(t, expected.AuthorLegacyID, r.AuthorLegacyID)
	require.Equal(t, expected.AuthorProfileImageURL, r.AuthorProfileImageURL)
}
