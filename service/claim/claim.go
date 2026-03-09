package claim

import (
	"context"
	"database/sql"

	"github.com/jackc/pgtype"
	"github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/service/persist"
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

// BulkCreateClaims creates multiple claims in one operation
func BulkCreateClaims(ctx context.Context, queries *coredb.Queries, input BulkCreateClaimsInput) ([]coredb.Claim, error) {
	claims := make([]coredb.Claim, 0, len(input.Claims))
	for _, claimInput := range input.Claims {
		claim, err := CreateClaim(ctx, queries, claimInput)
		if err != nil {
			return nil, err
		}
		claims = append(claims, claim)
	}
	return claims, nil
}

type BulkUpdateClaimsInput struct {
	Claims []UpdateClaimInput
}

// BulkUpdateClaims updates multiple claims in one operation
func BulkUpdateClaims(ctx context.Context, queries *coredb.Queries, input BulkUpdateClaimsInput) ([]coredb.Claim, error) {
	claims := make([]coredb.Claim, 0, len(input.Claims))
	for _, claimInput := range input.Claims {
		claim, err := UpdateClaim(ctx, queries, claimInput)
		if err != nil {
			return nil, err
		}
		claims = append(claims, claim)
	}
	return claims, nil
}

type BulkDeleteClaimsInput struct {
	ClaimIDs []persist.DBID
}

// BulkDeleteClaims deletes multiple claims in one operation
func BulkDeleteClaims(ctx context.Context, queries *coredb.Queries, input BulkDeleteClaimsInput) (int, error) {
	count := 0
	for _, claimID := range input.ClaimIDs {
		_, err := DeleteClaim(ctx, queries, DeleteClaimInput{ClaimID: claimID})
		if err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}
