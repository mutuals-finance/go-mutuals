package publicapi

import (
	db "github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/graphql/dataloader"
	"github.com/go-playground/validator/v10"
)

const maxSearchQueryLength = 256
const maxSearchResults = 100

type SearchAPI struct {
	queries   *db.Queries
	loaders   *dataloader.Loaders
	validator *validator.Validate
}

/*
TODO custom search
// SearchSplits searches for splits with the given query, limit, and optional weights. Weights may be nil to accept default values.
// Weighting will probably be removed after we settle on defaults that feel correct!
func (api SearchAPI) SearchSplits(ctx context.Context, query string, limit int, nameWeight float32, descriptionWeight float32) ([]db.Split, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"query":             {query, fmt.Sprintf("required,min=1,max=%d", maxSearchQueryLength)},
		"limit":             {limit, fmt.Sprintf("min=1,max=%d", maxSearchResults)},
		"nameWeight":        {nameWeight, "gte=0.0,lte=1.0"},
		"descriptionWeight": {descriptionWeight, "gte=0.0,lte=1.0"},
	}); err != nil {
		return nil, err
	}

	// Sanitize
	query = validate.SanitizationPolicy.Sanitize(query)

	return api.queries.SearchSplits(ctx, db.SearchSplitsParams{
		Limit:             int32(limit),
		Query:             query,
		NameWeight:        nameWeight,
		DescriptionWeight: descriptionWeight,
	})
}


// SearchContracts searches for contracts with the given query, limit, and optional weights. Weights may be nil to accept default values.
// Weighting will probably be removed after we settle on defaults that feel correct!
func (api SearchAPI) SearchContracts(ctx context.Context, query string, limit int, nameWeight float32, descriptionWeight float32) ([]db.Contract, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"query":             {query, fmt.Sprintf("required,min=1,max=%d", maxSearchQueryLength)},
		"limit":             {limit, fmt.Sprintf("min=1,max=%d", maxSearchResults)},
		"nameWeight":        {nameWeight, "gte=0.0,lte=1.0"},
		"descriptionWeight": {descriptionWeight, "gte=0.0,lte=1.0"},
	}); err != nil {
		return nil, err
	}

	// Sanitize
	query = validate.SanitizationPolicy.Sanitize(query)

	return api.queries.SearchContracts(ctx, db.SearchContractsParams{
		Limit:             int32(limit),
		Query:             query,
		NameWeight:        nameWeight,
		DescriptionWeight: descriptionWeight,
	})
}
*/
