package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
)

// HandleInit handles GET /v1/initialization.
// It returns the Kobo API resource map pointing back to this server.
func (h *KoboHandler) HandleInit(w http.ResponseWriter, r *http.Request) {
	tokenValue := auth.KoboTokenFromContext(r.Context())
	base := schemeAndHost(r)
	tb := base + "/kobo/" + tokenValue // token base URL

	resources := map[string]any{
		"account_page":           "https://www.kobo.com/account/settings",
		"account_page_rakuten":   "https://my.rakuten.co.jp/",
		"add_entitlement":        tb + "/v1/library/{RevisionIds}",
		"affiliate":              tb + "/v1/affiliate",
		"assets":                 tb + "/v1/assets",
		"audiobook":              tb + "/v1/products/audiobooks/{ProductId}",
		"audiobook_landing_page": "https://www.kobo.com/ebooks",
		"audiobook_subscription_orange_deal_inclusion_url": "https://authorize.kobo.com/inclusion",
		"authorproduct_recommendations":                    tb + "/v1/products/books/authors/recommendations",
		"autocomplete":                                     tb + "/v1/products/autocomplete",
		"blackstone_header": map[string]any{
			"key":   "x-amz-request-payer",
			"value": "requester",
		},
		"book":                            tb + "/v1/products/books/{ProductId}",
		"book_landing_page":               "https://www.kobo.com/ebooks",
		"browse_history":                  tb + "/v1/user/browsehistory",
		"categories":                      tb + "/v1/categories",
		"checkout_borrowed_book":          tb + "/v1/library/borrow",
		"client_authd_referral":           "https://authorize.kobo.com/api/AuthenticatedReferral/client/v1/getLink",
		"configuration_data":              tb + "/v1/configuration",
		"content_access_book":             tb + "/v1/products/books/{ProductId}/access",
		"daily_deal":                      tb + "/v1/products/dailydeal",
		"deals":                           tb + "/v1/deals",
		"delete_entitlement":              tb + "/v1/library/{Ids}",
		"delete_tag":                      tb + "/v1/library/tags/{TagId}",
		"delete_tag_items":                tb + "/v1/library/tags/{TagId}/items/delete",
		"device_auth":                     tb + "/v1/auth/device",
		"device_refresh":                  tb + "/v1/auth/refresh",
		"dictionary_host":                 "https://ereaderfiles.kobo.com",
		"discovery_host":                  "https://discovery.kobobooks.com",
		"exchange_auth":                   tb + "/v1/auth/exchange",
		"external_book":                   tb + "/v1/products/books/external/{Ids}",
		"featured_list":                   tb + "/v1/products/featured/{FeaturedListId}",
		"featured_lists":                  tb + "/v1/products/featured",
		"get_download_keys":               tb + "/v1/library/downloadkeys",
		"get_download_link":               tb + "/v1/library/downloadlink",
		"get_tests_request":               tb + "/v1/analytics/gettests",
		"gpb_flow_enabled":                "False",
		"help_page":                       "https://www.kobo.com/help",
		"image_host":                      base,
		"image_url_quality_template":      base + "/kobo/" + tokenValue + "/covers/{ImageId}/{Width}/{Height}/{Quality}/{IsGreyscale}/image.jpg",
		"image_url_template":              base + "/kobo/" + tokenValue + "/covers/{ImageId}/{Width}/{Height}/false/image.jpg",
		"instapaper_enabled":              "False",
		"library_metadata":                tb + "/v1/library/{Ids}",
		"library_prices":                  tb + "/v1/library/{Ids}/prices",
		"library_stack":                   tb + "/v1/user/library/stack",
		"library_sync":                    tb + "/v1/library/sync",
		"new_entitlement":                 tb + "/v1/library/{RevisionId}",
		"new_recommendation":              tb + "/v1/user/recommendations",
		"new_wishlist_item":               tb + "/v1/user/wishlist",
		"partner_agreements":              tb + "/v1/user/partneragreements",
		"product_nextread":                tb + "/v1/products/{ProductId}/nextread",
		"product_prices":                  tb + "/v1/products/{ProductIds}/prices",
		"product_recommendations":         tb + "/v1/products/{ProductId}/recommendations",
		"product_reviews":                 tb + "/v1/products/{ProductId}/reviews",
		"products_v2":                     tb + "/v2/products",
		"reading_services_host":           "https://readingservices.kobo.com",
		"recommendations":                 tb + "/v1/user/recommendations",
		"review_sentiment":                tb + "/v1/user/reviews/ratings",
		"search":                          tb + "/v1/products/search",
		"social_authorization":            "https://social.kobo.com",
		"store_home":                      "https://www.kobo.com/ebooks",
		"store_host":                      "https://www.kobo.com",
		"tag_items":                       tb + "/v1/library/tags/{TagId}/items",
		"tag_list":                        tb + "/v1/library/tags",
		"taste_profile":                   tb + "/v1/products/tasteprofile",
		"update_accessibility_to_preview": tb + "/v1/user/library/accessibility/{EntitlementId}",
		"user_loyalty_benefits":           tb + "/v1/user/loyalty/benefits",
		"user_platform":                   tb + "/v1/user/platform",
		"user_profile":                    tb + "/v1/user/profile",
		"user_ratings":                    tb + "/v1/user/ratings",
		"user_recommendations":            tb + "/v1/user/recommendations",
		"user_reviews":                    tb + "/v1/user/reviews",
		"user_wishlist":                   tb + "/v1/user/wishlist",
		"userguide_host":                  "https://ereaderfiles.kobo.com",
		"wishlist_list":                   tb + "/v1/user/wishlist",
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	// x-kobo-apitoken is required by Kobo devices; "e30=" is base64("{}")
	w.Header().Set("x-kobo-apitoken", "e30=")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]any{"Resources": resources}); err != nil {
		slog.ErrorContext(r.Context(), "failed to encode kobo init response", slog.Any(otelkeys.Error, err))
	}
}
