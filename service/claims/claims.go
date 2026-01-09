package claims

import (
	"context"

	db "github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/service/persist"
	"github.com/mutuals/go-mutuals/service/persist/allocation"
)

func CreateClaims(ctx context.Context, queries *db.Queries, poolID persist.DBID, claims []allocation.Claim) (result []db.Claim, err error) {
	tree, err := allocation.NewTree(claims)
	if err != nil {
		return nil, err
	}

	if err := tree.Prepare(ctx); err != nil {
		return nil, err
	}

	if err := tree.Validate(ctx); err != nil {
		return nil, err
	}

	params := db.CreateClaimsParams{PoolID: poolID}

	for _, claim := range tree.Claims {
		params.ID = append(params.ID, claim.ID.String())
		params.Label = append(params.Label, claim.Label)
		params.RecipientAddress = append(params.RecipientAddress, claim.RecipientAddress.String())
		params.Path = append(params.Path, claim.Path)
		params.StateID = append(params.StateID, claim.StateID)
		params.StrategyID = append(params.StrategyID, claim.StrategyID)
		// params.Data = append(params.Data, claim.Data)
	}

	return queries.CreateClaims(ctx, params)
}
