package publicapi

import (
	"context"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgtype"
	"github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/graphql/dataloader"
	claimService "github.com/mutuals/go-mutuals/service/claim"
	"github.com/mutuals/go-mutuals/service/persist"
	"github.com/mutuals/go-mutuals/service/persist/allocation"
	"github.com/mutuals/go-mutuals/service/persist/postgres"
	"github.com/mutuals/go-mutuals/validate"
)

type ClaimAPI struct {
	repos     *postgres.Repositories
	queries   *coredb.Queries
	loaders   *dataloader.Loaders
	validator *validator.Validate
	ethClient *ethclient.Client
}

// CreateClaim creates a new claim for a pool
func (api ClaimAPI) CreateClaim(ctx context.Context, poolID persist.DBID, label string, validationData, distributionData *persist.JSON, parent *persist.DBID, children []persist.DBID, validationID string, distributionID string) (coredb.Claim, error) {
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"poolID":         validate.WithTag(poolID, "required"),
		"label":          validate.WithTag(label, "required,max=200"),
		"validationID":   validate.WithTag(validationID, "required"),
		"distributionID": validate.WithTag(distributionID, "required"),
	}); err != nil {
		return coredb.Claim{}, err
	}

	return claimService.CreateClaimForPool(ctx, api.queries, claimService.CreateClaimForPoolInput{
		PoolID:           poolID,
		Label:            label,
		ValidationData:   validationData,
		DistributionData: distributionData,
		Parent:           parent,
		Children:         children,
		ValidationID:     validationID,
		DistributionID:   distributionID,
	})
}

// UpdateClaim updates an existing claim
func (api ClaimAPI) UpdateClaim(ctx context.Context, poolID persist.DBID, claimID persist.DBID, validationData, distributionData *persist.JSON, parent *persist.DBID, children []persist.DBID, validationID string, distributionID string) (coredb.Claim, error) {
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"poolID":  validate.WithTag(poolID, "required"),
		"claimID": validate.WithTag(claimID, "required"),
	}); err != nil {
		return coredb.Claim{}, err
	}

	return claimService.UpdateClaimForPool(ctx, api.queries, claimService.UpdateClaimForPoolInput{
		PoolID:           poolID,
		ClaimID:          claimID,
		ValidationData:   validationData,
		DistributionData: distributionData,
		Parent:           parent,
		Children:         children,
		ValidationID:     validationID,
		DistributionID:   distributionID,
	})
}

// DeleteClaim deletes a claim
func (api ClaimAPI) DeleteClaim(ctx context.Context, poolID persist.DBID, claimID persist.DBID) (coredb.Claim, error) {
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"poolID":  validate.WithTag(poolID, "required"),
		"claimID": validate.WithTag(claimID, "required"),
	}); err != nil {
		return coredb.Claim{}, err
	}

	return claimService.DeleteClaimForPool(ctx, api.queries, claimService.DeleteClaimForPoolInput{
		PoolID:  poolID,
		ClaimID: claimID,
	})
}

// GetClaimById retrieves a claim by ID
func (api ClaimAPI) GetClaimById(ctx context.Context, claimID persist.DBID) (*coredb.Claim, error) {
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"claimID": validate.WithTag(claimID, "required"),
	}); err != nil {
		return nil, err
	}

	claim, err := api.loaders.GetClaimByIdBatch.Load(claimID)
	if err != nil {
		return nil, err
	}

	return &claim, nil
}

// CreateClaimsWithAllocationTree creates claims with allocation tree handling
func (api ClaimAPI) CreateClaimsWithAllocationTree(ctx context.Context, poolID persist.DBID, claims []allocation.Claim) ([]coredb.Claim, error) {
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"poolID": validate.WithTag(poolID, "required"),
		"claims": validate.WithTag(claims, "required,min=1"),
	}); err != nil {
		return nil, err
	}

	return claimService.CreateClaimsWithAllocationTreeForPool(ctx, api.queries, claimService.CreateClaimsWithAllocationTreeForPoolInput{
		PoolID: poolID,
		Claims: claims,
	})
}

// GetClaimsByPoolID retrieves all claims for a pool
func (api ClaimAPI) GetClaimsByPoolID(ctx context.Context, poolID persist.DBID) ([]coredb.Claim, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"poolID": validate.WithTag(poolID, "required"),
	}); err != nil {
		return nil, err
	}

	claims, err := api.loaders.GetClaimsByPoolIdBatch.Load(poolID)
	if err != nil {
		return nil, err
	}

	return claims, nil
}

// GetClaimsByIds retrieves multiple claims by their IDs
func (api ClaimAPI) GetClaimsByIds(ctx context.Context, claimIDs []persist.DBID) ([]coredb.Claim, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"claimIDs": validate.WithTag(claimIDs, "required,min=1"),
	}); err != nil {
		return nil, err
	}

	return claimService.GetClaimsByIds(ctx, api.queries, claimService.GetClaimsByIdsInput{
		ClaimIDs: persist.DBIDSliceToStringSlice(claimIDs),
	})
}

type BulkCreateClaimInput struct {
	ValidationData   *persist.JSON
	DistributionData *persist.JSON
	Parent           *persist.DBID
	Children         []persist.DBID
	ValidationID     string
	DistributionID   string
}

// BulkCreateClaims creates multiple claims
func (api ClaimAPI) BulkCreateClaims(ctx context.Context, poolID persist.DBID, inputs []BulkCreateClaimInput) ([]coredb.Claim, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"poolID": validate.WithTag(poolID, "required"),
		"inputs": validate.WithTag(inputs, "required,min=1"),
	}); err != nil {
		return nil, err
	}

	// Convert inputs to service inputs
	serviceInputs := make([]claimService.CreateClaimInput, len(inputs))
	for i, input := range inputs {
		var validationData, distributionData pgtype.JSONB
		if input.ValidationData != nil {
			validationData = persist.JSONToJSONB(*input.ValidationData)
		}
		if input.DistributionData != nil {
			distributionData = persist.JSONToJSONB(*input.DistributionData)
		}

		serviceInputs[i] = claimService.CreateClaimInput{
			ID:               persist.GenerateID(),
			PoolID:           poolID,
			ValidationData:   validationData,
			DistributionData: distributionData,
			Parent:           persist.DBIDPtrToSQLNullString(input.Parent),
			Children:         persist.DBIDSliceToStringSlice(input.Children),
			ValidationID:     persist.DBID(input.ValidationID),
			DistributionID:   persist.DBID(input.DistributionID),
		}
	}

	return claimService.BulkCreateClaimsForPool(ctx, api.queries, claimService.BulkCreateClaimsForPoolInput{
		PoolID: poolID,
		Claims: serviceInputs,
	})
}

type BulkUpdateClaimInput struct {
	ClaimID          persist.DBID
	ValidationData   *persist.JSON
	DistributionData *persist.JSON
	Parent           *persist.DBID
	Children         []persist.DBID
	ValidationID     string
	DistributionID   string
}

// BulkUpdateClaims updates multiple claims
func (api ClaimAPI) BulkUpdateClaims(ctx context.Context, poolID persist.DBID, inputs []BulkUpdateClaimInput) ([]coredb.Claim, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"poolID": validate.WithTag(poolID, "required"),
		"inputs": validate.WithTag(inputs, "required,min=1"),
	}); err != nil {
		return nil, err
	}

	// Convert inputs to service inputs
	serviceInputs := make([]claimService.UpdateClaimInput, len(inputs))
	for i, input := range inputs {
		var validationData, distributionData pgtype.JSONB
		if input.ValidationData != nil {
			validationData = persist.JSONToJSONB(*input.ValidationData)
		}
		if input.DistributionData != nil {
			distributionData = persist.JSONToJSONB(*input.DistributionData)
		}

		serviceInputs[i] = claimService.UpdateClaimInput{
			ClaimID:          input.ClaimID,
			ValidationData:   validationData,
			DistributionData: distributionData,
			Parent:           persist.DBIDPtrToSQLNullString(input.Parent),
			Children:         persist.DBIDSliceToStringSlice(input.Children),
			ValidationID:     persist.DBID(input.ValidationID),
			DistributionID:   persist.DBID(input.DistributionID),
		}
	}

	return claimService.BulkUpdateClaimsForPool(ctx, api.queries, claimService.BulkUpdateClaimsForPoolInput{
		PoolID: poolID,
		Claims: serviceInputs,
	})
}

// BulkDeleteClaims deletes multiple claims
func (api ClaimAPI) BulkDeleteClaims(ctx context.Context, poolID persist.DBID, claimIDs []persist.DBID) (int, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"poolID":   validate.WithTag(poolID, "required"),
		"claimIDs": validate.WithTag(claimIDs, "required,min=1"),
	}); err != nil {
		return 0, err
	}

	return claimService.BulkDeleteClaimsForPool(ctx, api.queries, claimService.BulkDeleteClaimsForPoolInput{
		PoolID:   poolID,
		ClaimIDs: claimIDs,
	})
}
