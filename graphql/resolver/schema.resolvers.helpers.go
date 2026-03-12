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

// resolvePool looks up a pool by any combination of id, slug, or contractId.
// Uses model.ToDBIDPtr for nil-safe GqlID → DBID conversion.
func resolvePool(ctx context.Context, id *model.GqlID, slug *string, contractID *model.GqlID) (model.PoolResult, error) {
	pool, err := publicapi.For(ctx).Pool.GetPool(ctx, model.ToDBIDPtr(id), slug, model.ToDBIDPtr(contractID))
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

// resolveUserPools returns a PoolConnection for the authenticated viewer.
func resolveUserPools(ctx context.Context, obj *model.User, first *int, after *string) (*model.PoolConnection, error) {
	pools, err := publicapi.For(ctx).Pool.GetViewerPools(ctx)
	if err != nil {
		return nil, err
	}
	edges := poolsToEdges(ctx, *pools)
	return &model.PoolConnection{
		Edges: edges,
		PageInfo: &model.PageInfo{
			HasPreviousPage: false,
			HasNextPage:     false,
			Size:            len(edges),
		},
	}, nil
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
	return &model.Deposit{DBID: id}, nil
}

func resolveEVMAccountByID(ctx context.Context, id persist.DBID) (*model.EVMAccount, error) {
	return &model.EVMAccount{DBID: id}, nil
}

func resolveModuleByID(ctx context.Context, id persist.DBID) (*model.Module, error) {
	return &model.Module{DBID: id}, nil
}

func resolveModuleRegistryByID(ctx context.Context, id persist.DBID) (*model.ModuleRegistry, error) {
	return &model.ModuleRegistry{DBID: id}, nil
}

func resolvePoolContractByID(ctx context.Context, id persist.DBID) (*model.PoolContract, error) {
	return &model.PoolContract{DBID: id}, nil
}

func resolvePoolDayBalanceByID(ctx context.Context, id persist.DBID) (*model.PoolDayBalance, error) {
	return &model.PoolDayBalance{DBID: id}, nil
}

func resolvePoolFactoryByID(ctx context.Context, id persist.DBID) (*model.PoolFactory, error) {
	return &model.PoolFactory{DBID: id}, nil
}

func resolvePoolHourBalanceByID(ctx context.Context, id persist.DBID) (*model.PoolHourBalance, error) {
	return &model.PoolHourBalance{DBID: id}, nil
}

func resolveTokenByID(ctx context.Context, id persist.DBID) (*model.Token, error) {
	return &model.Token{DBID: id}, nil
}

func resolveTokenBalanceByID(ctx context.Context, id persist.DBID) (*model.TokenBalance, error) {
	return &model.TokenBalance{DBID: id}, nil
}

func resolveTxByID(ctx context.Context, id persist.DBID) (*model.Tx, error) {
	return &model.Tx{DBID: id}, nil
}

func resolveWithdrawalByID(ctx context.Context, id persist.DBID) (*model.Withdrawal, error) {
	return &model.Withdrawal{DBID: id}, nil
}

func resolveViewer(ctx context.Context) (*model.User, error) {
	user, err := publicapi.For(ctx).User.GetViewer(ctx)
	if err != nil {
		return nil, err
	}
	return userToModel(ctx, *user), nil
}

func resolveDeletedNodeByID(ctx context.Context, id persist.DBID) (*model.DeletedNode, error) {
	return &model.DeletedNode{DBID: id}, nil
}

// --- Model converters ---

func poolToModel(_ context.Context, pool db.Pool) *model.Pool {
	return &model.Pool{
		DBID:        pool.ID,
		Name:        pool.Name,
		Description: pool.Description,
		Image:       pool.Image,
		DonationBps: int(pool.DonationBps),
		Slug:        pool.Slug,
		Status:      model.PoolStatusDraft, // TODO: map from pool.Status
		CreatedAt:   pool.CreatedAt,
		UpdatedAt:   pool.UpdatedAt,
	}
}

func poolsToModels(ctx context.Context, pools []db.Pool) []*model.Pool {
	models := make([]*model.Pool, len(pools))
	for i, pool := range pools {
		models[i] = poolToModel(ctx, pool)
	}
	return models
}

func poolsToEdges(ctx context.Context, pools []db.Pool) []*model.PoolEdge {
	edges := make([]*model.PoolEdge, len(pools))
	for i, pool := range pools {
		edges[i] = &model.PoolEdge{
			Node:   poolToModel(ctx, pool),
			Cursor: pool.ID.String(),
		}
	}
	return edges
}

func claimToModel(_ context.Context, claim db.Claim) *model.Claim {
	path := ""
	if claim.Path.Valid {
		path = claim.Path.String
	}
	return &model.Claim{
		DBID:             claim.ID,
		Label:            claim.Label,
		ValidationData:   persist.JSONBToJSON(claim.ValidationData),
		DistributionData: persist.JSONBToJSON(claim.DistributionData),
		Path:             path,
		CreatedAt:        claim.CreatedAt,
		UpdatedAt:        claim.UpdatedAt,
	}
}

func claimsToModels(ctx context.Context, claims []db.Claim) []*model.Claim {
	models := make([]*model.Claim, len(claims))
	for i, claim := range claims {
		models[i] = claimToModel(ctx, claim)
	}
	return models
}

func userToModel(_ context.Context, user db.User) *model.User {
	return &model.User{
		DBID: user.ID,
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
			Cursor: user.ID.String(),
		}
	}
	return edges
}

func evmAccountToModel(_ context.Context, account indexerdb.Account) *model.EVMAccount {
	return &model.EVMAccount{
		DBID:        account.ID,
		Address:     persist.Address(account.Address),
		AccountType: model.EVMAccountType(account.AccountType),
		CreatedAt:   account.CreatedAt,
		UpdatedAt:   account.UpdatedAt,
	}
}

func evmAccountsToModels(ctx context.Context, accounts []indexerdb.Account) []*model.EVMAccount {
	models := make([]*model.EVMAccount, len(accounts))
	for i, account := range accounts {
		models[i] = evmAccountToModel(ctx, account)
	}
	return models
}

// strPtr returns nil for empty string, a pointer otherwise — used for optional cursor fields.
func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func pageInfoToModel(_ context.Context, pageInfo publicapi.PageInfo) *model.PageInfo {
	return &model.PageInfo{
		Total:           pageInfo.Total,
		Size:            pageInfo.Size,
		HasPreviousPage: pageInfo.HasPreviousPage,
		HasNextPage:     pageInfo.HasNextPage,
		StartCursor:     strPtr(pageInfo.StartCursor),
		EndCursor:       strPtr(pageInfo.EndCursor),
	}
}
