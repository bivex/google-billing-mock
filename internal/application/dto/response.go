package dto

// SubscriptionPurchaseResponse mirrors the Google Play Developer API v3 response exactly.
// All millisecond timestamps are serialized as strings per Google's convention.
// Nullable fields use *string / *int so they serialize as JSON null.
type SubscriptionPurchaseResponse struct {
	Kind                          string  `json:"kind"`
	StartTimeMillis               string  `json:"startTimeMillis"`
	ExpiryTimeMillis              string  `json:"expiryTimeMillis"`
	AutoRenewing                  bool    `json:"autoRenewing"`
	PriceCurrencyCode             string  `json:"priceCurrencyCode"`
	PriceAmountMicros             string  `json:"priceAmountMicros"`
	CountryCode                   string  `json:"countryCode"`
	DeveloperPayload              string  `json:"developerPayload"`
	PaymentState                  *int    `json:"paymentState"`
	CancelReason                  *int    `json:"cancelReason"`
	UserCancellationTimeMillis    *string `json:"userCancellationTimeMillis"`
	CancelSurveyResult            *string `json:"cancelSurveyResult"`
	OrderID                       string  `json:"orderId"`
	LinkedPurchaseToken           *string `json:"linkedPurchaseToken"`
	PurchaseType                  *int    `json:"purchaseType"`
	PriceChange                   *string `json:"priceChange"`
	ProfileName                   *string `json:"profileName"`
	EmailAddress                  *string `json:"emailAddress"`
	GivenName                     *string `json:"givenName"`
	FamilyName                    *string `json:"familyName"`
	ProfileID                     *string `json:"profileId"`
	AcknowledgementState          int     `json:"acknowledgementState"`
	ExternalAccountID             *string `json:"externalAccountId"`
	PromotionType                 *int    `json:"promotionType"`
	PromotionCode                 *string `json:"promotionCode"`
	ObfuscatedExternalAccountID   *string `json:"obfuscatedExternalAccountId"`
	ObfuscatedExternalProfileID   *string `json:"obfuscatedExternalProfileId"`
	PurchaseState                 int     `json:"purchaseState"`
	PurchaseTimeMillis            string  `json:"purchaseTimeMillis"`
	ProductID                     string  `json:"productId"`
	RegionCode                    string  `json:"regionCode"`
	IntroductoryPriceInfo         *string `json:"introductoryPriceInfo"`
}

// ProductPurchaseResponse mirrors the Google Play Developer API v3 product purchase response.
type ProductPurchaseResponse struct {
	Kind                        string  `json:"kind"`
	PurchaseTimeMillis          string  `json:"purchaseTimeMillis"`
	PurchaseState               int     `json:"purchaseState"`
	ConsumptionState            int     `json:"consumptionState"`
	DeveloperPayload            string  `json:"developerPayload"`
	OrderID                     string  `json:"orderId"`
	PurchaseType                *int    `json:"purchaseType"`
	AcknowledgementState        int     `json:"acknowledgementState"`
	PurchaseToken               string  `json:"purchaseToken"`
	ProductID                   string  `json:"productId"`
	Quantity                    int     `json:"quantity"`
	ObfuscatedExternalAccountID *string `json:"obfuscatedExternalAccountId"`
	ObfuscatedExternalProfileID *string `json:"obfuscatedExternalProfileId"`
	RegionCode                  string  `json:"regionCode"`
}

// DeferSubscriptionResponse is the response body for the :defer action.
type DeferSubscriptionResponse struct {
	NewExpiryTimeMillis string `json:"newExpiryTimeMillis"`
}

// ErrorResponse is the standard Google API error envelope.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error details.
type ErrorDetail struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// DeferSubscriptionRequest is the request body for the defer action.
type DeferSubscriptionRequest struct {
	DeferralInfo DeferralInfo `json:"deferralInfo"`
}

// DeferralInfo holds the desired new expiry.
type DeferralInfo struct {
	DesiredExpiryTimeMillis string `json:"desiredExpiryTimeMillis"`
}

// ─── subscriptionsv2 ─────────────────────────────────────────────────────────

// SubscriptionPurchaseV2Response matches the SubscriptionPurchaseV2 schema.
type SubscriptionPurchaseV2Response struct {
	Kind                 string                            `json:"kind"`
	StartTime            string                            `json:"startTime"`            // RFC3339
	RegionCode           string                            `json:"regionCode"`
	SubscriptionState    string                            `json:"subscriptionState"`
	LatestOrderId        string                            `json:"latestOrderId"`
	LinkedPurchaseToken  *string                           `json:"linkedPurchaseToken,omitempty"`
	AcknowledgementState string                            `json:"acknowledgementState"`
	LineItems            []SubscriptionPurchaseV2LineItem  `json:"lineItems"`
	TestPurchase         *struct{}                         `json:"testPurchase,omitempty"`
}

// SubscriptionPurchaseV2LineItem is one item in the lineItems array.
type SubscriptionPurchaseV2LineItem struct {
	ProductId        string            `json:"productId"`
	ExpiryTime       string            `json:"expiryTime"`  // RFC3339
	AutoRenewingPlan *AutoRenewingPlan `json:"autoRenewingPlan,omitempty"`
}

// AutoRenewingPlan holds auto-renew flag.
type AutoRenewingPlan struct {
	AutoRenewEnabled bool `json:"autoRenewEnabled"`
}

// DeferSubscriptionPurchaseV2Response is returned by subscriptionsv2 :defer.
type DeferSubscriptionPurchaseV2Response struct {
	ItemExpiryTimeDetails []ItemExpiryTimeDetails `json:"itemExpiryTimeDetails"`
}

// ItemExpiryTimeDetails holds per-product new expiry info.
type ItemExpiryTimeDetails struct {
	ProductId  string `json:"productId"`
	ExpiryTime string `json:"expiryTime"` // RFC3339
}

// ─── productsv2 ───────────────────────────────────────────────────────────────

// ProductPurchaseV2Response matches the ProductPurchaseV2 schema.
type ProductPurchaseV2Response struct {
	Kind                        string                   `json:"kind"`
	OrderId                     string                   `json:"orderId"`
	RegionCode                  string                   `json:"regionCode"`
	AcknowledgementState        string                   `json:"acknowledgementState"`
	PurchaseCompletionTime      string                   `json:"purchaseCompletionTime"` // RFC3339
	ObfuscatedExternalAccountId *string                  `json:"obfuscatedExternalAccountId,omitempty"`
	ObfuscatedExternalProfileId *string                  `json:"obfuscatedExternalProfileId,omitempty"`
	ProductLineItem             []ProductLineItemV2      `json:"productLineItem"`
	PurchaseStateContext        PurchaseStateContextV2   `json:"purchaseStateContext"`
}

// ProductLineItemV2 is one item in ProductPurchaseV2.productLineItem.
type ProductLineItemV2 struct {
	ProductId string `json:"productId"`
	Quantity  int    `json:"quantity"`
}

// PurchaseStateContextV2 holds the v2 purchase state.
type PurchaseStateContextV2 struct {
	PurchaseState string `json:"purchaseState"` // PURCHASE_STATE_PURCHASED / CANCELED / PENDING
}

// ─── voidedpurchases ─────────────────────────────────────────────────────────

// VoidedPurchasesListResponse matches VoidedPurchasesListResponse schema.
type VoidedPurchasesListResponse struct {
	PageInfo        VoidedPageInfo        `json:"pageInfo"`
	TokenPagination VoidedTokenPagination `json:"tokenPagination"`
	VoidedPurchases []VoidedPurchase      `json:"voidedPurchases"`
}

// VoidedPageInfo is pagination metadata.
type VoidedPageInfo struct {
	TotalResults   int `json:"totalResults"`
	StartIndex     int `json:"startIndex"`
	ResultsPerPage int `json:"resultsPerPage"`
}

// VoidedTokenPagination holds page tokens.
type VoidedTokenPagination struct {
	NextPageToken     string `json:"nextPageToken"`
	PreviousPageToken string `json:"previousPageToken"`
}

// VoidedPurchase is a single voided purchase entry.
type VoidedPurchase struct {
	Kind               string `json:"kind"`
	PurchaseToken      string `json:"purchaseToken"`
	PurchaseTimeMillis string `json:"purchaseTimeMillis"`
	VoidedTimeMillis   string `json:"voidedTimeMillis"`
	OrderId            string `json:"orderId"`
	VoidedSource       int    `json:"voidedSource"`
	VoidedReason       int    `json:"voidedReason"`
}

// ─── orders ──────────────────────────────────────────────────────────────────

// OrderResponse matches the Order schema.
type OrderResponse struct {
	OrderId       string `json:"orderId"`
	PurchaseToken string `json:"purchaseToken"`
	State         string `json:"state"`
	CreateTime    string `json:"createTime"` // RFC3339
}

// BatchGetOrdersResponse matches BatchGetOrdersResponse schema.
type BatchGetOrdersResponse struct {
	Orders []OrderResponse `json:"orders"`
}

// SeedSubscriptionRequest is used by the admin API to directly seed a subscription.
type SeedSubscriptionRequest struct {
	Token                string  `json:"token"`
	SubscriptionID       string  `json:"subscriptionId"`
	PackageName          string  `json:"packageName"`
	PurchaseState        int     `json:"purchaseState"`
	PaymentState         *int    `json:"paymentState"`
	AcknowledgementState int     `json:"acknowledgementState"`
	AutoRenewing         bool    `json:"autoRenewing"`
	ExpiryTimeMillis     int64   `json:"expiryTimeMillis"`
	CancelReason         *int    `json:"cancelReason,omitempty"`
}

// SeedProductRequest is used by the admin API to directly seed a product purchase.
type SeedProductRequest struct {
	Token                string `json:"token"`
	ProductID            string `json:"productId"`
	PackageName          string `json:"packageName"`
	PurchaseState        int    `json:"purchaseState"`
	AcknowledgementState int    `json:"acknowledgementState"`
}
