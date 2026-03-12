package graphql

// schema.resolvers.go gets updated when generating gqlgen bindings and should not contain
// helper functions. schema.resolvers.helpers.go is a companion file that can contain
// helper functions without interfering with code generation.

import (
	"context"

	db "github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/db/gen/indexerdb"
	"github.com/mutuals/go-mutuals/graphql/model"
	"github.com/mutuals/go-mutuals/publicapi"
	"github.com/mutuals/go-mutuals/service/auth"
	"github.com/mutuals/go-mutuals/service/persist"
	"github.com/mutuals/go-mutuals/validate"
)


var nodeFetcher = model.NodeFetcher{
	OnClaim:           resolveClaimByID,
	OnDeletedNode:     resolveDeletedNodeByID,
	OnDeposit:         resolveDepositByID,
	OnEVMAccount:      resolveEVMAccountByID,
	OnModule:          resolveModuleByID,
	OnModuleRegistry:  resolveModuleRegistryByID,
	OnPool:            resolvePoolByID,
	OnPoolContract:    resolvePoolContractByID,
	OnPoolDayBalance:  resolvePoolDayBalanceByID,
	OnPoolFactory:     resolvePoolFactoryByID,
	OnPoolHourBalance: resolvePoolHourBalanceByID,
	OnToken:           resolveTokenByID,
	OnTokenBalance:    resolveTokenBalanceByID,
	OnTx:              resolveTxByID,
	OnUser:            resolveUserByID,
	OnWithdrawal:      resolveWithdrawalByID,
}

func init() {
	nodeFetcher.ValidateHandlers()
}

// errorToGraphqlType converts a golang error to its matching type from our GraphQL schema.
// If no matching type is found, ok will return false.
// This function is used by middleware to remap errors to their GraphQL union types.
func errorToGraphqlType(ctx context.Context, err error, gqlTypeName string) (gqlModel interface{}, ok bool) {
	message := err.Error()
	var mappedErr model.Error = nil

	switch e := err.(type) {
	case auth.ErrAuthenticationFailed:
		mappedErr = model.ErrAuthenticationFailed{Message: message}
	case persist.ErrUserNotFound:
		mappedErr = model.ErrUserNotFound{Message: message}
	case persist.ErrUserAlreadyExists:
		mappedErr = model.ErrUserAlreadyExists{Message: message}
	case persist.ErrUsernameNotAvailable:
		mappedErr = model.ErrUsernameNotAvailable{Message: message}
	case persist.ErrTokenNotFoundByID:
		mappedErr = model.ErrTokenNotFound{Message: message}
	case persist.ErrAddressOwnedByUser:
		mappedErr = model.ErrAddressOwnedByUser{Message: message}
	case validate.ErrInvalidInput:
		mappedErr = model.ErrInvalidInput{
			Message:    message,
			Parameters: e.Parameters,
			Reasons:    e.Reasons,
		}
	case persist.ErrPoolNotFound:
		mappedErr = model.ErrPoolNotFound{Message: message}
	}

	if mappedErr != nil {
		if converted, ok := model.ConvertToModelType(mappedErr, gqlTypeName); ok {
			return converted, true
		}
	}

	return nil, false
}

func resolveUserByID(ctx context.Context, id persist.DBID) (*model.User, error) {
	user, err := publicapi.For(ctx).User.GetUserById(ctx, id)
	if err != nil {
		return nil, err
	}
	return userToModel(ctx, *user), nil
}

func resolveUserByAddress(ctx context.Context, address persist.Address) (*model.User, error) {
	user, err := publicapi.For(ctx).User.GetUserByAddress(ctx, address)
	if err != nil {
		return nil, err
	}
	return userToModel(ctx, *user), nil
}

func resolvePool(ctx context.Context, id *model.GqlID, slug *string, contractID *model.GqlID) (model.PoolResult, error) {
	poolDBID := id.DBID()
	contractDBID := contractID.DBID()

	pool, err := publicapi.For(ctx).Pool.GetPool(ctx, &poolDBID, slug, &contractDBID)
	if err != nil {
		return nil, err
	}

	return poolToModel(ctx, *pool), nil
}

func resolvePoolByID(ctx context.Context, id persist.DBID) (*model.Pool, error) {
	pool, err := publicapi.For(ctx).Pool.GetPool(ctx, &id, nil, nil)
	if err != nil {
		return nil, err
	}
	return poolToModel(ctx, *pool), nil
}

func resolveUserPools(ctx context.Context, obj *model.User) ([]*model.Pool, error) {
	// For now, we do not support querying pools for arbitrary users.
	pools, err := publicapi.For(ctx).Pool.GetViewerPools(ctx)
	if err != nil {
		return nil, err
	}
	return poolsToModels(ctx, *pools), nil
}

func resolveClaimByID(ctx context.Context, id persist.DBID) (*model.Claim, error) {
	claim, err := publicapi.For(ctx).Claim.GetClaimById(ctx, id)
	if err != nil {
		return nil, err
	}
	return claimToModel(ctx, *claim), nil
}

func resolveClaimsByPoolID(ctx context.Context, poolID persist.DBID) ([]*model.Claim, error) {
	claims, err := publicapi.For(ctx).Claim.GetClaimsByPoolID(ctx, poolID)
	if err != nil {
		return nil, err
	}
	return claimsToModels(ctx, claims), nil
}

func resolveDepositByID(ctx context.Context, id persist.DBID) (*model.Deposit, error) {
	// TODO: implement
	return &model.Deposit{}, nil
}

func resolveEVMAccountByID(ctx context.Context, id persist.DBID) (*model.EVMAccount, error) {
	// TODO: implement
	return &model.EVMAccount{}, nil
}

func resolveModuleByID(ctx context.Context, id persist.DBID) (*model.Module, error) {
	// TODO: implement
	return &model.Module{}, nil
}

func resolveModuleRegistryByID(ctx context.Context, id persist.DBID) (*model.ModuleRegistry, error) {
	// TODO: implement
	return &model.ModuleRegistry{}, nil
}

func resolvePoolContractByID(ctx context.Context, id persist.DBID) (*model.PoolContract, error) {
	// TODO: implement
	return &model.PoolContract{}, nil
}

func resolvePoolDayBalanceByID(ctx context.Context, id persist.DBID) (*model.PoolDayBalance, error) {
	// TODO: implement
	return &model.PoolDayBalance{}, nil
}

func resolvePoolFactoryByID(ctx context.Context, id persist.DBID) (*model.PoolFactory, error) {
	// TODO: implement
	return &model.PoolFactory{}, nil
}

func resolvePoolHourBalanceByID(ctx context.Context, id persist.DBID) (*model.PoolHourBalance, error) {
	// TODO: implement
	return &model.PoolHourBalance{}, nil
}

func resolveTokenByID(ctx context.Context, id persist.DBID) (*model.Token, error) {
	// TODO: implement
	return &model.Token{}, nil
}

func resolveTokenBalanceByID(ctx context.Context, id persist.DBID) (*model.TokenBalance, error) {
	// TODO: implement
	return &model.TokenBalance{}, nil
}

func resolveTxByID(ctx context.Context, id persist.DBID) (*model.Tx, error) {
	// TODO: implement
	return &model.Tx{}, nil
}

func resolveWithdrawalByID(ctx context.Context, id persist.DBID) (*model.Withdrawal, error) {
	// TODO: implement
	return &model.Withdrawal{}, nil
}

func resolveViewer(ctx context.Context) (*model.User, error) {
	user, err := publicapi.For(ctx).User.GetViewer(ctx)
	if err != nil {
		return nil, err
	}
	return userToModel(ctx, *user), nil
}

func resolveDeletedNodeByID(ctx context.Context, id persist.DBID) (*model.DeletedNode, error) {
	return &model.DeletedNode{}, nil
}

func poolToModel(ctx context.Context, pool db.Pool) *model.Pool {
	return &model.Pool{
		Name:        pool.Name,
		Description: pool.Description,
		Image:       pool.Image,
		DonationBps: 0, // TODO: add to db.Pool
		Slug:        pool.Slug,
		Status:      model.PoolStatusDraft, // TODO: map pool.Status
		CreatedAt:   pool.CreatedAt,
		UpdatedAt:   pool.UpdatedAt,
		Owner:       nil, // handled by dedicated resolver
		Contract:    nil, // handled by dedicated resolver
		Claims:      nil, // handled by dedicated resolver
	}
}

func poolsToModels(ctx context.Context, pools []db.Pool) []*model.Pool {
	models := make([]*model.Pool, len(pools))
	for i, pool := range pools {
		models[i] = poolToModel(ctx, pool)
	}
	return models
}

func claimToModel(ctx context.Context, claim db.Claim) *model.Claim {
	path := ""
	if claim.Path.Valid {
		path = claim.Path.String
	}

	validationData := persist.JSONBToJSON(claim.ValidationData)
	distributionData := persist.JSONBToJSON(claim.DistributionData)

	return &model.Claim{
		Label:            claim.Label,
		ValidationData:   validationData,
		DistributionData: distributionData,
		Path:             path,
		CreatedAt:        claim.CreatedAt,
		UpdatedAt:        claim.UpdatedAt,
		Validation:       nil, // handled by dedicated resolver
		Distribution:     nil, // handled by dedicated resolver
		Pool:             nil, // handled by dedicated resolver
		Parent:           nil, // handled by dedicated resolver
		Children:         nil, // handled by dedicated resolver
	}
}

func claimsToModels(ctx context.Context, claims []db.Claim) []*model.Claim {
	models := make([]*model.Claim, len(claims))
	for i, claim := range claims {
		models[i] = claimToModel(ctx, claim)
	}
	return models
}

func userToModel(ctx context.Context, user db.User) *model.User {
	return &model.User{
		Pools: nil, // handled by dedicated resolver
		Roles: nil, // handled by dedicated resolver
	}
}

func usersToModels(ctx context.Context, users []db.User) []*model.User {
	models := make([]*model.User, len(users))
	for i, user := range users {
		models[i] = userToModel(ctx, user)
	}
	return models
}

func usersToEdges(ctx context.Context, users []db.User) []*model.UserEdge {
	edges := make([]*model.UserEdge, len(users))
	for i, user := range users {
		edges[i] = &model.UserEdge{
			Node:   userToModel(ctx, user),
			Cursor: nil,
		}
	}
	return edges
}

func evmAccountToModel(ctx context.Context, account indexerdb.Account) *model.EVMAccount {
	return &model.EVMAccount{
		Address:     persist.Address(account.Address),
		AccountType: model.EVMAccountType(account.AccountType),
		CreatedAt:   account.CreatedAt,
		UpdatedAt:   account.UpdatedAt,
		SelfPools:   nil, // handled by dedicated resolver
		Balances:    nil, // handled by dedicated resolver
	}
}

func evmAccountsToModels(ctx context.Context, wallets []indexerdb.Account) []*model.EVMAccount {
	models := make([]*model.EVMAccount, len(wallets))
	for i, wallet := range wallets {
		models[i] = evmAccountToModel(ctx, wallet)
	}
	return models
}

func pageInfoToModel(ctx context.Context, pageInfo publicapi.PageInfo) *model.PageInfo {
	return &model.PageInfo{
		Total:           pageInfo.Total,
		Size:            pageInfo.Size,
		HasPreviousPage: pageInfo.HasPreviousPage,
		HasNextPage:     pageInfo.HasNextPage,
		StartCursor:     pageInfo.StartCursor,
		EndCursor:       pageInfo.EndCursor,
	}
}
