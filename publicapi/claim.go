package publicapi

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-playground/validator/v10"
	"github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/graphql/dataloader"
	claimService "github.com/mutuals/go-mutuals/service/claim"
	"github.com/mutuals/go-mutuals/service/persist"
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
func (api ClaimAPI) CreateClaim(ctx context.Context, poolID persist.DBID, label string, data persist.JSON, parent *persist.DBID, children []persist.DBID, validationID string, distributionID string) (coredb.Claim, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"poolID":         validate.WithTag(poolID, "required"),
		"label":          validate.WithTag(label, "required,max=200"),
		"validationID":   validate.WithTag(validationID, "required"),
		"distributionID": validate.WithTag(distributionID, "required"),
	}); err != nil {
		return coredb.Claim{}, err
	}

	// Check authorization (user must be pool owner)
	userId, err := getAuthenticatedUserId(ctx)
	if err != nil {
		return coredb.Claim{}, err
	}

	pool, err := api.queries.GetPoolById(ctx, poolID)
	if err != nil {
		return coredb.Claim{}, err
	}

	if pool.OwnerID != userId {
		return coredb.Claim{}, fmt.Errorf("user is not the owner of the pool")
	}

	return claimService.CreateClaim(ctx, api.queries, claimService.CreateClaimInput{
		ID:             persist.GenerateID(),
		PoolID:         poolID,
		Label:          label,
		Data:           persist.JSONToJSONB(data),
		Parent:         persist.DBIDPtrToSQLNullString(parent),
		Children:       persist.DBIDSliceToStringSlice(children),
		ValidationID:   persist.DBID(validationID),
		DistributionID: persist.DBID(distributionID),
	})
}

// UpdateClaim updates an existing claim
func (api ClaimAPI) UpdateClaim(ctx context.Context, poolID persist.DBID, claimID persist.DBID, data persist.JSON, parent *persist.DBID, children []persist.DBID, validationID string, distributionID string) (coredb.Claim, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"poolID":  validate.WithTag(poolID, "required"),
		"claimID": validate.WithTag(claimID, "required"),
	}); err != nil {
		return coredb.Claim{}, err
	}

	// Check authorization (user must be pool owner)
	userID, err := getAuthenticatedUserId(ctx)
	if err != nil {
		return coredb.Claim{}, err
	}

	pool, err := api.queries.GetPoolById(ctx, poolID)
	if err != nil {
		return coredb.Claim{}, err
	}

	if pool.OwnerID != userID {
		return coredb.Claim{}, fmt.Errorf("user is not the owner of the pool")
	}

	return claimService.UpdateClaim(ctx, api.queries, claimService.UpdateClaimInput{
		ClaimID:        claimID,
		Data:           persist.JSONToJSONB(data),
		Parent:         persist.DBIDPtrToSQLNullString(parent),
		Children:       persist.DBIDSliceToStringSlice(children),
		ValidationID:   persist.DBID(validationID),
		DistributionID: persist.DBID(distributionID),
	})
}

// DeleteClaim deletes a claim
func (api ClaimAPI) DeleteClaim(ctx context.Context, poolID persist.DBID, claimID persist.DBID) (coredb.Claim, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"poolID":  validate.WithTag(poolID, "required"),
		"claimID": validate.WithTag(claimID, "required"),
	}); err != nil {
		return coredb.Claim{}, err
	}

	// Check authorization (user must be pool owner)
	userID, err := getAuthenticatedUserId(ctx)
	if err != nil {
		return coredb.Claim{}, err
	}

	pool, err := api.queries.GetPoolById(ctx, poolID)
	if err != nil {
		return coredb.Claim{}, err
	}

	if pool.OwnerID != userID {
		return coredb.Claim{}, fmt.Errorf("user is not the owner of the pool")
	}

	return claimService.DeleteClaim(ctx, api.queries, claimService.DeleteClaimInput{
		ClaimID: claimID,
	})
}

// GetClaimById retrieves a claim by ID
func (api ClaimAPI) GetClaimById(ctx context.Context, claimID persist.DBID) (*coredb.Claim, error) {
	// Validate
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
	Data           persist.JSON
	Parent         *persist.DBID
	Children       []persist.DBID
	ValidationID   string
	DistributionID string
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

	// Check authorization (user must be pool owner)
	userID, err := getAuthenticatedUserId(ctx)
	if err != nil {
		return nil, err
	}

	pool, err := api.queries.GetPoolById(ctx, poolID)
	if err != nil {
		return nil, err
	}

	if pool.OwnerID != userID {
		return nil, fmt.Errorf("user is not the owner of the pool")
	}

	// Convert inputs to service inputs
	serviceInputs := make([]claimService.CreateClaimInput, len(inputs))
	for i, input := range inputs {
		serviceInputs[i] = claimService.CreateClaimInput{
			ID:             persist.GenerateID(),
			PoolID:         poolID,
			Data:           persist.JSONToJSONB(input.Data),
			Parent:         persist.DBIDPtrToSQLNullString(input.Parent),
			Children:       persist.DBIDSliceToStringSlice(input.Children),
			ValidationID:   persist.DBID(input.ValidationID),
			DistributionID: persist.DBID(input.DistributionID),
		}
	}

	return claimService.BulkCreateClaims(ctx, api.queries, claimService.BulkCreateClaimsInput{
		PoolID: poolID,
		Claims: serviceInputs,
	})
}

type BulkUpdateClaimInput struct {
	ClaimID        persist.DBID
	Data           persist.JSON
	Parent         *persist.DBID
	Children       []persist.DBID
	ValidationID   string
	DistributionID string
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

	// Check authorization (user must be pool owner)
	userID, err := getAuthenticatedUserId(ctx)
	if err != nil {
		return nil, err
	}

	pool, err := api.queries.GetPoolById(ctx, poolID)
	if err != nil {
		return nil, err
	}

	if pool.OwnerID != userID {
		return nil, fmt.Errorf("user is not the owner of the pool")
	}

	// Convert inputs to service inputs
	serviceInputs := make([]claimService.UpdateClaimInput, len(inputs))
	for i, input := range inputs {
		serviceInputs[i] = claimService.UpdateClaimInput{
			ClaimID:        input.ClaimID,
			Data:           persist.JSONToJSONB(input.Data),
			Parent:         persist.DBIDPtrToSQLNullString(input.Parent),
			Children:       persist.DBIDSliceToStringSlice(input.Children),
			ValidationID:   persist.DBID(input.ValidationID),
			DistributionID: persist.DBID(input.DistributionID),
		}
	}

	return claimService.BulkUpdateClaims(ctx, api.queries, claimService.BulkUpdateClaimsInput{
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

	// Check authorization (user must be pool owner)
	userID, err := getAuthenticatedUserId(ctx)
	if err != nil {
		return 0, err
	}

	pool, err := api.queries.GetPoolById(ctx, poolID)
	if err != nil {
		return 0, err
	}

	if pool.OwnerID != userID {
		return 0, fmt.Errorf("user is not the owner of the pool")
	}

	return claimService.BulkDeleteClaims(ctx, api.queries, claimService.BulkDeleteClaimsInput{
		ClaimIDs: claimIDs,
	})
}
