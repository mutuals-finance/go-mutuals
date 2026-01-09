package claims

import (
	"context"

	db "github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/service/persist"
	"github.com/mutuals/go-mutuals/service/persist/allocation"
)

func CreateClaims(ctx context.Context, queries *db.Queries, poolID persist.DBID, claims []allocation.Claim) (result []db.Claim, err error) {
	tree, err := allocation.NewTree(claims, allocation.WithParamsBuildFunc(claimParamsBuilder))
	if err != nil {
		return nil, err
	}

	if err := tree.Validate(ctx); err != nil {
		return []db.Claim{}, err
	}

	if err := tree.Process(ctx); err != nil {
		return []db.Claim{}, err
	}

	params := tree.Params.(db.CreateClaimsParams)
	params.PoolID = poolID

	result, err = queries.CreateClaims(ctx, params)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func claimParamsBuilder(ctx context.Context, tree *allocation.Tree, claim *allocation.Claim) any {
	var params db.CreateClaimsParams

	if tree.Params == nil {
		params = db.CreateClaimsParams{}
	} else {
		params = tree.Params.(db.CreateClaimsParams)
	}

	params.ID = append(params.ID, claim.ID.String())
	params.Label = append(params.Label, claim.Label)
	params.RecipientAddress = append(params.RecipientAddress, claim.RecipientAddress.String())
	params.Path = append(params.Path, claim.Path)
	params.StateID = append(params.StateID, claim.StateID)
	params.StrategyID = append(params.StrategyID, claim.StrategyID)
	// params.Data = append(params.Data, claim.Data) // uncomment when ready

	return params
}
