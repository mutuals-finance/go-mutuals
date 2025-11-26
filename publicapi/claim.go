package publicapi

import (
	"context"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-playground/validator/v10"
	db "github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/graphql/dataloader"
	"github.com/mutuals/go-mutuals/service/persist"
	"github.com/mutuals/go-mutuals/service/persist/postgres"
	"github.com/mutuals/go-mutuals/validate"
)

type ClaimAPI struct {
	repos     *postgres.Repositories
	queries   *db.Queries
	loaders   *dataloader.Loaders
	validator *validator.Validate
	ethClient *ethclient.Client
}

/*
	func (api ClaimAPI) CreateClaimBulk(ctx context.Context, input model.ClaimBulkCreateInput) ([]db.Claim, error) {
		if err := validate.ValidateFields(api.validator, validate.ValidationMap{
			"name":        validate.WithTag(input.Name, "max=200"),
			"description": validate.WithTag(input.Description, "max=600"),
		}); err != nil {
			return nil, err
		}

		claims, err := api.queries.UpsertClaims(ctx, db.UpsertClaimsParams{
			ID:               nil,
			PoolID:           util.FromPointer(input.PoolID),
			RecipientAddress: nil,
			Value:            nil,
			StateID:          nil,
			StrategyID:       nil,
			Label:            nil,
			Path:             nil,
			Deleted:          nil,
		})

		if err != nil {
			return nil, err
		}

		return claims, nil
	}
*/

func (api ClaimAPI) GetClaimsByPoolID(ctx context.Context, poolID persist.DBID) ([]db.Claim, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"poolID": validate.WithTag(poolID, "required"),
	}); err != nil {
		return nil, err
	}

	pools, err := api.loaders.GetClaimsByPoolIdBatch.Load(poolID)
	if err != nil {
		return nil, err
	}

	return pools, nil
}

func (api ClaimAPI) GetClaimById(ctx context.Context, id persist.DBID) (*db.Claim, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"id": validate.WithTag(id, "required"),
	}); err != nil {
		return nil, err
	}

	claim, err := api.loaders.GetClaimByIdBatch.Load(id)
	if err != nil {
		return nil, err
	}

	return &claim, nil
}

/*func processAllocations(poolID *persist.DBID, a []*model.ClaimAllocationInput) (allocationParams db.UpsertClaimAllocationsParams, aggregationParams db.UpsertClaimAggregatedAllocationsParams) {
	recipientToAllocations := make(map[persist.Address][]*model.ClaimAllocationInput)

	allocationParams.ClaimID = *poolID
	aggregationParams.ClaimID = *poolID

	var traverse func(node *model.ClaimAllocationInput, path string)
	traverse = func(node *model.ClaimAllocationInput, parentPath string) {
		id := node.ID
		if id == nil {
			id = util.ToPointer(persist.GenerateID())
		}
		// Determine the label: use RecipientAddress if not empty, otherwise use id
		label := node.RecipientAddress.String()
		if label == "" {
			label = id.String()
		}

		// Construct the current path
		path := parentPath
		if parentPath == "" {
			path = label
		} else {
			path = parentPath + "." + label
		}

		allocationParams.Ids = append(allocationParams.Ids, id.String())
		allocationParams.RecipientAddress = append(allocationParams.RecipientAddress, node.RecipientAddress.String())
		allocationParams.RecipientType = append(allocationParams.RecipientType, int32(node.RecipientType[0]))
		allocationParams.CalculationType = append(allocationParams.CalculationType, int32(node.CalculationType[0]))
		allocationParams.Value = append(allocationParams.Value, persist.MustHexString(node.Value.String()).String())
		allocationParams.Expression = append(allocationParams.Expression, "")
		allocationParams.Label = append(allocationParams.Label, label)
		allocationParams.Path = append(allocationParams.Path, path)

		if node.RecipientType[0] == persist.RecipientTypeDefaultItem && node.RecipientAddress != nil {
			recipientToAllocations[*node.RecipientAddress] = append(recipientToAllocations[*node.RecipientAddress], node)
		}

		// Recursively process children
		for _, child := range node.Children {
			traverse(child, path)
		}
	}

	// Process each top-level node
	for _, allocation := range a {
		traverse(allocation, "")
	}

	// Calculate aggregation
	for recipient, inputs := range recipientToAllocations {
		expression := ""
		for _, input := range inputs {
			expression = expression + input.Value.String()
		}
		aggregationParams.ID = append(aggregationParams.RecipientAddress, persist.GenerateID().String())
		aggregationParams.RecipientAddress = append(aggregationParams.RecipientAddress, recipient.String())
		aggregationParams.Expression = append(aggregationParams.Expression, expression)
	}

	return allocationParams, aggregationParams
}
*/
