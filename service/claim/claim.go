package claim

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgtype"
	"github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/service/auth"
	"github.com/mutuals/go-mutuals/service/persist"
	"github.com/mutuals/go-mutuals/service/persist/allocation"
)

type CreateClaimInput struct {
	ID             persist.DBID
	PoolID         persist.DBID
	Label          string
	Data           pgtype.JSONB
	Parent         sql.NullString
	Children       []string
	ValidationID   persist.DBID
	DistributionID persist.DBID
}

// CreateClaim creates a new claim for a pool
func CreateClaim(ctx context.Context, queries *coredb.Queries, input CreateClaimInput) (coredb.Claim, error) {
	return queries.CreateClaim(ctx, coredb.CreateClaimParams{
		ID:             input.ID,
		PoolID:         input.PoolID,
		Label:          input.Label,
		Data:           input.Data,
		Parent:         input.Parent,
		Children:       input.Children,
		ValidationID:   input.ValidationID,
		DistributionID: input.DistributionID,
	})
}

type UpdateClaimInput struct {
	ClaimID        persist.DBID
	Data           pgtype.JSONB
	Parent         sql.NullString
	Children       []string
	ValidationID   persist.DBID
	DistributionID persist.DBID
}

// UpdateClaim updates an existing claim
func UpdateClaim(ctx context.Context, queries *coredb.Queries, input UpdateClaimInput) (coredb.Claim, error) {
	return queries.UpdateClaim(ctx, coredb.UpdateClaimParams{
		ID:             input.ClaimID,
		Data:           input.Data,
		Parent:         input.Parent,
		Children:       input.Children,
		ValidationID:   input.ValidationID,
		DistributionID: input.DistributionID,
	})
}

type DeleteClaimInput struct {
	ClaimID persist.DBID
}

// DeleteClaim soft deletes a claim
func DeleteClaim(ctx context.Context, queries *coredb.Queries, input DeleteClaimInput) (coredb.Claim, error) {
	return queries.DeleteClaim(ctx, input.ClaimID)
}

type GetClaimByIdInput struct {
	ClaimID persist.DBID
}

// GetClaimById retrieves a claim by ID
func GetClaimById(ctx context.Context, queries *coredb.Queries, input GetClaimByIdInput) (*coredb.Claim, error) {
	claim, err := queries.GetClaimById(ctx, input.ClaimID)
	if err != nil {
		return nil, err
	}
	return &claim, nil
}

type GetClaimsByPoolIdInput struct {
	PoolID persist.DBID
}

// GetClaimsByPoolId retrieves all claims for a pool
func GetClaimsByPoolId(ctx context.Context, queries *coredb.Queries, input GetClaimsByPoolIdInput) ([]coredb.Claim, error) {
	return queries.GetClaimsByPoolId(ctx, input.PoolID)
}

type GetClaimsByIdsInput struct {
	ClaimIDs []string
}

// GetClaimsByIds retrieves multiple claims by their IDs
func GetClaimsByIds(ctx context.Context, queries *coredb.Queries, input GetClaimsByIdsInput) ([]coredb.Claim, error) {
	return queries.GetClaimsByIds(ctx, input.ClaimIDs)
}

type BulkCreateClaimsInput struct {
	PoolID persist.DBID
	Claims []CreateClaimInput
}

// BulkCreateClaims creates multiple claims using SQLC's CreateClaims batch query
func BulkCreateClaims(ctx context.Context, queries *coredb.Queries, input BulkCreateClaimsInput) ([]coredb.Claim, error) {
	// Prepare batch parameters
	ids := make([]string, len(input.Claims))
	labels := make([]string, len(input.Claims))
	paths := make([]string, len(input.Claims))
	dataList := make([]pgtype.JSONB, len(input.Claims))
	parents := make([]sql.NullString, len(input.Claims))
	childrenList := make([][]string, len(input.Claims))
	validationIDs := make([]string, len(input.Claims))
	distributionIDs := make([]string, len(input.Claims))

	for i, claim := range input.Claims {
		ids[i] = claim.ID.String()
		labels[i] = claim.Label
		paths[i] = "" // Path will be set separately if needed
		dataList[i] = claim.Data
		parents[i] = claim.Parent
		childrenList[i] = claim.Children
		validationIDs[i] = claim.ValidationID.String()
		distributionIDs[i] = claim.DistributionID.String()
	}

	// Note: CreateClaims currently doesn't support parent/children in the SQL
	// This is handled by the allocation tree logic in publicapi
	return queries.CreateClaims(ctx, coredb.CreateClaimsParams{
		ID:             ids,
		PoolID:         input.PoolID,
		Label:          labels,
		Path:           paths,
		Data:           dataList,
		ValidationID:   validationIDs,
		DistributionID: distributionIDs,
	})
}

type BulkUpdateClaimsInput struct {
	Claims []UpdateClaimInput
}

// BulkUpdateClaims updates multiple claims using SQLC's UpdateClaims batch query
func BulkUpdateClaims(ctx context.Context, queries *coredb.Queries, input BulkUpdateClaimsInput) ([]coredb.Claim, error) {
	// Prepare batch parameters
	ids := make([]string, len(input.Claims))
	dataList := make([]pgtype.JSONB, len(input.Claims))
	parents := make([]sql.NullString, len(input.Claims))
	childrenList := make([][]string, len(input.Claims))
	validationIDs := make([]string, len(input.Claims))
	distributionIDs := make([]string, len(input.Claims))
	deleted := make([]bool, len(input.Claims))
	paths := make([]string, len(input.Claims))
	labels := make([]string, len(input.Claims))

	for i, claim := range input.Claims {
		ids[i] = claim.ClaimID.String()
		dataList[i] = claim.Data
		parents[i] = claim.Parent
		childrenList[i] = claim.Children
		validationIDs[i] = claim.ValidationID.String()
		distributionIDs[i] = claim.DistributionID.String()
		deleted[i] = false
		paths[i] = ""  // Maintain existing path
		labels[i] = "" // Maintain existing label
	}

	return queries.UpdateClaims(ctx, coredb.UpdateClaimsParams{
		ID:             ids,
		ValidationID:   validationIDs,
		DistributionID: distributionIDs,
		Data:           dataList,
		Label:          labels,
		Path:           paths,
		Deleted:        deleted,
	})
}

type BulkDeleteClaimsInput struct {
	ClaimIDs []persist.DBID
}

// BulkDeleteClaims soft-deletes multiple claims using UpdateClaims with deleted=true
func BulkDeleteClaims(ctx context.Context, queries *coredb.Queries, input BulkDeleteClaimsInput) (int, error) {
	// Prepare batch parameters for soft delete
	ids := make([]string, len(input.ClaimIDs))
	deleted := make([]bool, len(input.ClaimIDs))
	// Empty arrays for other fields (will maintain existing values)
	validationIDs := make([]string, len(input.ClaimIDs))
	distributionIDs := make([]string, len(input.ClaimIDs))
	dataList := make([]pgtype.JSONB, len(input.ClaimIDs))
	labels := make([]string, len(input.ClaimIDs))
	paths := make([]string, len(input.ClaimIDs))

	for i, claimID := range input.ClaimIDs {
		ids[i] = claimID.String()
		deleted[i] = true
		// Other fields will use existing values via SQL query
	}

	claims, err := queries.UpdateClaims(ctx, coredb.UpdateClaimsParams{
		ID:             ids,
		ValidationID:   validationIDs,
		DistributionID: distributionIDs,
		Data:           dataList,
		Label:          labels,
		Path:           paths,
		Deleted:        deleted,
	})
	if err != nil {
		return 0, err
	}

	return len(claims), nil
}

type CreateClaimsWithAllocationTreeInput struct {
	PoolID persist.DBID
	Claims []allocation.Claim
}

// CreateClaimsWithAllocationTree creates claims using allocation tree for hierarchy and path calculation
func CreateClaimsWithAllocationTree(ctx context.Context, queries *coredb.Queries, input CreateClaimsWithAllocationTreeInput) ([]coredb.Claim, error) {
	// Create tree, prepare (generate IDs & paths), and validate
	tree, err := allocation.NewTree(input.Claims)
	if err != nil {
		return nil, err
	}

	if err := tree.Prepare(ctx); err != nil {
		return nil, err
	}

	if err := tree.Validate(ctx); err != nil {
		return nil, err
	}

	// Bulk insert all claims in one query
	ids := make([]string, len(tree.Claims))
	labels := make([]string, len(tree.Claims))
	paths := make([]string, len(tree.Claims))
	dataList := make([]pgtype.JSONB, len(tree.Claims))
	validationIDs := make([]string, len(tree.Claims))
	distributionIDs := make([]string, len(tree.Claims))

	for i, claim := range tree.Claims {
		ids[i] = claim.ID.String()
		labels[i] = claim.Label
		paths[i] = claim.Path
		dataList[i] = persist.JSONToJSONB(claim.Data)
		validationIDs[i] = claim.ValidationID
		distributionIDs[i] = claim.DistributionID
	}

	return queries.CreateClaims(ctx, coredb.CreateClaimsParams{
		ID:             ids,
		PoolID:         input.PoolID,
		Label:          labels,
		Path:           paths,
		Data:           dataList,
		ValidationID:   validationIDs,
		DistributionID: distributionIDs,
	})
}

type CreateClaimForPoolInput struct {
	PoolID         persist.DBID
	Label          string
	Data           persist.JSON
	Parent         *persist.DBID
	Children       []persist.DBID
	ValidationID   string
	DistributionID string
}

// CreateClaimForPool creates a claim with authorization check
func CreateClaimForPool(ctx context.Context, queries *coredb.Queries, input CreateClaimForPoolInput) (coredb.Claim, error) {
	userID, err := auth.GetAuthenticatedUserId(ctx)
	if err != nil {
		return coredb.Claim{}, err
	}

	pool, err := queries.GetPoolById(ctx, input.PoolID)
	if err != nil {
		return coredb.Claim{}, err
	}

	if pool.OwnerID != userID {
		return coredb.Claim{}, fmt.Errorf("user is not the owner of the pool")
	}

	return CreateClaim(ctx, queries, CreateClaimInput{
		ID:             persist.GenerateID(),
		PoolID:         input.PoolID,
		Label:          input.Label,
		Data:           persist.JSONToJSONB(input.Data),
		Parent:         persist.DBIDPtrToSQLNullString(input.Parent),
		Children:       persist.DBIDSliceToStringSlice(input.Children),
		ValidationID:   persist.DBID(input.ValidationID),
		DistributionID: persist.DBID(input.DistributionID),
	})
}

type UpdateClaimForPoolInput struct {
	PoolID         persist.DBID
	ClaimID        persist.DBID
	Data           persist.JSON
	Parent         *persist.DBID
	Children       []persist.DBID
	ValidationID   string
	DistributionID string
}

// UpdateClaimForPool updates a claim with authorization check
func UpdateClaimForPool(ctx context.Context, queries *coredb.Queries, input UpdateClaimForPoolInput) (coredb.Claim, error) {
	userID, err := auth.GetAuthenticatedUserId(ctx)
	if err != nil {
		return coredb.Claim{}, err
	}

	pool, err := queries.GetPoolById(ctx, input.PoolID)
	if err != nil {
		return coredb.Claim{}, err
	}

	if pool.OwnerID != userID {
		return coredb.Claim{}, fmt.Errorf("user is not the owner of the pool")
	}

	return UpdateClaim(ctx, queries, UpdateClaimInput{
		ClaimID:        input.ClaimID,
		Data:           persist.JSONToJSONB(input.Data),
		Parent:         persist.DBIDPtrToSQLNullString(input.Parent),
		Children:       persist.DBIDSliceToStringSlice(input.Children),
		ValidationID:   persist.DBID(input.ValidationID),
		DistributionID: persist.DBID(input.DistributionID),
	})
}

type DeleteClaimForPoolInput struct {
	PoolID  persist.DBID
	ClaimID persist.DBID
}

// DeleteClaimForPool deletes a claim with authorization check
func DeleteClaimForPool(ctx context.Context, queries *coredb.Queries, input DeleteClaimForPoolInput) (coredb.Claim, error) {
	userID, err := auth.GetAuthenticatedUserId(ctx)
	if err != nil {
		return coredb.Claim{}, err
	}

	pool, err := queries.GetPoolById(ctx, input.PoolID)
	if err != nil {
		return coredb.Claim{}, err
	}

	if pool.OwnerID != userID {
		return coredb.Claim{}, fmt.Errorf("user is not the owner of the pool")
	}

	return DeleteClaim(ctx, queries, DeleteClaimInput{
		ClaimID: input.ClaimID,
	})
}

type BulkCreateClaimsForPoolInput struct {
	PoolID persist.DBID
	Claims []CreateClaimInput
}

// BulkCreateClaimsForPool creates multiple claims with authorization check
func BulkCreateClaimsForPool(ctx context.Context, queries *coredb.Queries, input BulkCreateClaimsForPoolInput) ([]coredb.Claim, error) {
	userID, err := auth.GetAuthenticatedUserId(ctx)
	if err != nil {
		return nil, err
	}

	pool, err := queries.GetPoolById(ctx, input.PoolID)
	if err != nil {
		return nil, err
	}

	if pool.OwnerID != userID {
		return nil, fmt.Errorf("user is not the owner of the pool")
	}

	return BulkCreateClaims(ctx, queries, BulkCreateClaimsInput{
		PoolID: input.PoolID,
		Claims: input.Claims,
	})
}

type BulkUpdateClaimsForPoolInput struct {
	PoolID persist.DBID
	Claims []UpdateClaimInput
}

// BulkUpdateClaimsForPool updates multiple claims with authorization check
func BulkUpdateClaimsForPool(ctx context.Context, queries *coredb.Queries, input BulkUpdateClaimsForPoolInput) ([]coredb.Claim, error) {
	userID, err := auth.GetAuthenticatedUserId(ctx)
	if err != nil {
		return nil, err
	}

	pool, err := queries.GetPoolById(ctx, input.PoolID)
	if err != nil {
		return nil, err
	}

	if pool.OwnerID != userID {
		return nil, fmt.Errorf("user is not the owner of the pool")
	}

	return BulkUpdateClaims(ctx, queries, BulkUpdateClaimsInput{
		Claims: input.Claims,
	})
}

type BulkDeleteClaimsForPoolInput struct {
	PoolID   persist.DBID
	ClaimIDs []persist.DBID
}

// BulkDeleteClaimsForPool deletes multiple claims with authorization check
func BulkDeleteClaimsForPool(ctx context.Context, queries *coredb.Queries, input BulkDeleteClaimsForPoolInput) (int, error) {
	userID, err := auth.GetAuthenticatedUserId(ctx)
	if err != nil {
		return 0, err
	}

	pool, err := queries.GetPoolById(ctx, input.PoolID)
	if err != nil {
		return 0, err
	}

	if pool.OwnerID != userID {
		return 0, fmt.Errorf("user is not the owner of the pool")
	}

	return BulkDeleteClaims(ctx, queries, BulkDeleteClaimsInput{
		ClaimIDs: input.ClaimIDs,
	})
}

type CreateClaimsWithAllocationTreeForPoolInput struct {
	PoolID persist.DBID
	Claims []allocation.Claim
}

// CreateClaimsWithAllocationTreeForPool creates claims with allocation tree and authorization
func CreateClaimsWithAllocationTreeForPool(ctx context.Context, queries *coredb.Queries, input CreateClaimsWithAllocationTreeForPoolInput) ([]coredb.Claim, error) {
	userID, err := auth.GetAuthenticatedUserId(ctx)
	if err != nil {
		return nil, err
	}

	pool, err := queries.GetPoolById(ctx, input.PoolID)
	if err != nil {
		return nil, err
	}

	if pool.OwnerID != userID {
		return nil, fmt.Errorf("user is not the owner of the pool")
	}

	return CreateClaimsWithAllocationTree(ctx, queries, CreateClaimsWithAllocationTreeInput{
		PoolID: input.PoolID,
		Claims: input.Claims,
	})
}
