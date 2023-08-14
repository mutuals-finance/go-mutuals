//go:generate go run github.com/gallery-so/dataloaden UserLoaderByID github.com/SplitFi/go-splitfi/service/persist.DBID github.com/SplitFi/go-splitfi/db/gen/coredb.User
//go:generate go run github.com/gallery-so/dataloaden UsersLoaderByID github.com/SplitFi/go-splitfi/service/persist.DBID []github.com/SplitFi/go-splitfi/db/gen/coredb.User
//go:generate go run github.com/gallery-so/dataloaden UserLoaderByAddress github.com/SplitFi/go-splitfi/db/gen/coredb.GetUserByAddressBatchParams github.com/SplitFi/go-splitfi/db/gen/coredb.User
//go:generate go run github.com/gallery-so/dataloaden UserLoaderByString string github.com/SplitFi/go-splitfi/db/gen/coredb.User
//go:generate go run github.com/gallery-so/dataloaden UsersLoaderByString string []github.com/SplitFi/go-splitfi/db/gen/coredb.User
//go:generate go run github.com/gallery-so/dataloaden SplitLoaderByID github.com/SplitFi/go-splitfi/service/persist.DBID github.com/SplitFi/go-splitfi/db/gen/coredb.Split
//go:generate go run github.com/gallery-so/dataloaden SplitsLoaderByID github.com/SplitFi/go-splitfi/service/persist.DBID []github.com/SplitFi/go-splitfi/db/gen/coredb.Split
//go:generate go run github.com/gallery-so/dataloaden WalletLoaderById github.com/SplitFi/go-splitfi/service/persist.DBID github.com/SplitFi/go-splitfi/db/gen/coredb.Wallet
//go:generate go run github.com/gallery-so/dataloaden WalletLoaderByChainAddress github.com/SplitFi/go-splitfi/service/persist.ChainAddress github.com/SplitFi/go-splitfi/db/gen/coredb.Wallet
//go:generate go run github.com/gallery-so/dataloaden WalletsLoaderByID github.com/SplitFi/go-splitfi/service/persist.DBID []github.com/SplitFi/go-splitfi/db/gen/coredb.Wallet
//go:generate go run github.com/gallery-so/dataloaden TokenLoaderByID github.com/SplitFi/go-splitfi/service/persist.DBID github.com/SplitFi/go-splitfi/db/gen/coredb.Token
//go:generate go run github.com/gallery-so/dataloaden TokensLoaderByID github.com/SplitFi/go-splitfi/service/persist.DBID []github.com/SplitFi/go-splitfi/db/gen/coredb.Token
//go:generate go run github.com/gallery-so/dataloaden TokensLoaderByIDAndLimit github.com/SplitFi/go-splitfi/graphql/dataloader.IDAndLimit []github.com/SplitFi/go-splitfi/db/gen/coredb.Token
//go:generate go run github.com/gallery-so/dataloaden TokensLoaderByContractID github.com/SplitFi/go-splitfi/db/gen/coredb.GetTokensByContractIdBatchPaginateParams []github.com/SplitFi/go-splitfi/db/gen/coredb.Token
//go:generate go run github.com/gallery-so/dataloaden TokensLoaderByIDTuple github.com/SplitFi/go-splitfi/service/persist.DBIDTuple []github.com/SplitFi/go-splitfi/db/gen/coredb.Token
//go:generate go run github.com/gallery-so/dataloaden TokensLoaderByIDAndChain github.com/SplitFi/go-splitfi/graphql/dataloader.IDAndChain []github.com/SplitFi/go-splitfi/db/gen/coredb.Token
//go:generate go run github.com/gallery-so/dataloaden AssetsByOwnerAddress github.com/SplitFi/go-splitfi/service/persist.ChainAddress []github.com/SplitFi/go-splitfi/db/gen/coredb.Asset
//go:generate go run github.com/gallery-so/dataloaden EventLoaderByID github.com/SplitFi/go-splitfi/service/persist.DBID github.com/SplitFi/go-splitfi/db/gen/coredb.FeedEvent
//go:generate go run github.com/gallery-so/dataloaden NotificationLoaderByID github.com/SplitFi/go-splitfi/service/persist.DBID github.com/SplitFi/go-splitfi/db/gen/coredb.Notification
//go:generate go run github.com/gallery-so/dataloaden NotificationsLoaderByUserID github.com/SplitFi/go-splitfi/db/gen/coredb.GetUserNotificationsBatchParams []github.com/SplitFi/go-splitfi/db/gen/coredb.Notification
//go:generate go run github.com/gallery-so/dataloaden IntLoaderByID github.com/SplitFi/go-splitfi/service/persist.DBID int

package dataloader

import (
	"context"
	"sync"
	"time"

	"github.com/SplitFi/go-splitfi/service/tracing"

	db "github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/jackc/pgx/v4"
)

type IDAndChain struct {
	ID    persist.DBID
	Chain persist.Chain
}

type IDAndLimit struct {
	ID    persist.DBID
	Limit *int
}

// Loaders will cache and batch lookups. They are short-lived and should never persist beyond
// a single request, nor should they be shared between requests (since the data returned is
// relative to the current request context, including the user and their auth status).
type Loaders struct {
	UserByUserID                     *UserLoaderByID
	UserByUsername                   *UserLoaderByString
	UserByAddress                    *UserLoaderByAddress
	UsersWithTrait                   *UsersLoaderByString
	SplitBySplitID                   *SplitLoaderByID
	SplitByCollectionID              *SplitLoaderByID
	SplitsByUserID                   *SplitsLoaderByID
	CollectionByCollectionID         *CollectionLoaderByID
	CollectionsBySplitID             *CollectionsLoaderByID
	MembershipByMembershipID         *MembershipLoaderById
	WalletByWalletID                 *WalletLoaderById
	WalletsByUserID                  *WalletsLoaderByID
	WalletByChainAddress             *WalletLoaderByChainAddress
	TokenByTokenID                   *TokenLoaderByID
	TokensByContractID               *TokensLoaderByID
	TokensByCollectionID             *TokensLoaderByIDAndLimit
	TokensByWalletID                 *TokensLoaderByID
	TokensByUserID                   *TokensLoaderByID
	TokensByUserIDAndContractID      *TokensLoaderByIDTuple
	TokensByContractIDWithPagination *TokensLoaderByContractID
	TokensByUserIDAndChain           *TokensLoaderByIDAndChain
	AssetsByOwnerAddress             *AssetsByOwnerAddress
	OwnerByTokenID                   *UserLoaderByID
	ContractByContractID             *ContractLoaderByID
	ContractsByUserID                *ContractsLoaderByID
	ContractByChainAddress           *ContractLoaderByChainAddress
	FollowersByUserID                *UsersLoaderByID
	FollowingByUserID                *UsersLoaderByID
	SharedFollowersByUserIDs         *SharedFollowersLoaderByIDs
	SharedContractsByUserIDs         *SharedContractsLoaderByIDs
	EventByEventID                   *EventLoaderByID
	NotificationByID                 *NotificationLoaderByID
	NotificationsByUserID            *NotificationsLoaderByUserID
	ContractsDisplayedByUserID       *ContractsLoaderByID
	OwnersByContractID               *UsersLoaderByContractID
}

func NewLoaders(ctx context.Context, q *db.Queries, disableCaching bool) *Loaders {
	subscriptionRegistry := make([]interface{}, 0)
	mutexRegistry := make([]*sync.Mutex, 0)

	defaults := settings{
		ctx:                  ctx,
		maxBatchOne:          100,
		maxBatchMany:         10,
		waitTime:             2 * time.Millisecond,
		disableCaching:       disableCaching,
		publishResults:       true,
		preFetchHook:         tracing.DataloaderPreFetchHook,
		postFetchHook:        tracing.DataloaderPostFetchHook,
		subscriptionRegistry: &subscriptionRegistry,
		mutexRegistry:        &mutexRegistry,
	}

	//---------------------------------------------------------------------------------------------------
	// HOW TO ADD A NEW DATALOADER
	//---------------------------------------------------------------------------------------------------
	// 1) If you need a new loader type, add it to the top of the file and use the "go generate" command
	//    to generate it. The convention is to name your loader <ValueType>LoaderBy<KeyType>, where
	//    <ValueType> should be plural if your loader returns a slice. Note that a loader type can be
	//    used by multiple dataloaders: UserLoaderByID is the correct generated type for both a
	//    "UserByUserID" dataloader and a "UserBySplitID" dataloader.
	//
	// 2) Add your dataloader to the Loaders struct above
	//
	// 3) Initialize your loader below. Dataloaders that don't return slices can subscribe to automatic
	//    cache priming by specifying an AutoCacheWithKey function (which should return the key to use
	//    when caching). If your dataloader needs to cache a single value with multiple keys (e.g. a
	//    SplitByCollectionID wants to cache a single Split by many collection IDs), you can use
	//    AutoCacheWithKeys instead. When other dataloaders return the type you've subscribed to, your
	//    dataloader will automatically cache those results.
	//
	//    Note: dataloaders that return slices can't subscribe to automatic caching, because it's
	//          unlikely that the grouping of results returned by one dataloader will make sense for
	//          another. E.g. the results of TokensByWalletID have little to do with the results
	//			of TokensByCollectionID, even though they both return slices of Tokens.
	//
	// 4) The "defaults" struct has sufficient settings for most use cases, but if you need to override
	//	  any default settings, all NewLoader methods accept these option args:
	//		- withMaxBatch(batchSize int)		<-- set the max batch size for a loader
	//		- withMaxWait(wait time.Duration)	<-- set the max wait time for a loader
	//		- withPublishResults(publish bool)  <-- whether this loader should publish its results for
	//  											other loaders to subscribe to and cache
	//---------------------------------------------------------------------------------------------------

	loaders := &Loaders{}

	loaders.UserByUserID = NewUserLoaderByID(defaults, loadUserByUserId(q), UserLoaderByIDCacheSubscriptions{
		AutoCacheWithKey: func(user db.User) persist.DBID { return user.ID },
	})

	loaders.UserByUsername = NewUserLoaderByString(defaults, loadUserByUsername(q), UserLoaderByStringCacheSubscriptions{
		AutoCacheWithKey: func(user db.User) string { return user.Username.String },
	})

	loaders.UserByAddress = NewUserLoaderByAddress(defaults, loadUserByAddress(q), UserLoaderByAddressCacheSubscriptions{})

	loaders.UsersWithTrait = NewUsersLoaderByString(defaults, loadUsersWithTrait(q))

	loaders.OwnersByContractID = NewUsersLoaderByContractID(defaults, loadOwnersByContractIDs(q))

	loaders.SplitBySplitID = NewSplitLoaderByID(defaults, loadSplitBySplitId(q), SplitLoaderByIDCacheSubscriptions{
		AutoCacheWithKey: func(split db.Split) persist.DBID { return split.ID },
	})

	loaders.SplitsByUserID = NewSplitsLoaderByID(defaults, loadSplitsByUserId(q))

	loaders.WalletByWalletID = NewWalletLoaderById(defaults, loadWalletByWalletId(q), WalletLoaderByIdCacheSubscriptions{
		AutoCacheWithKey: func(wallet db.Wallet) persist.DBID { return wallet.ID },
	})

	loaders.WalletsByUserID = NewWalletsLoaderByID(defaults, loadWalletsByUserId(q))

	loaders.WalletByChainAddress = NewWalletLoaderByChainAddress(defaults, loadWalletByChainAddress(q), WalletLoaderByChainAddressCacheSubscriptions{
		AutoCacheWithKey: func(wallet db.Wallet) persist.ChainAddress {
			return persist.NewChainAddress(wallet.Address, wallet.Chain)
		},
	})

	loaders.TokenByTokenID = NewTokenLoaderByID(defaults, loadTokenByTokenID(q), TokenLoaderByIDCacheSubscriptions{
		AutoCacheWithKey: func(token db.Token) persist.DBID { return token.ID },
	})

	loaders.TokensByWalletID = NewTokensLoaderByID(defaults, loadTokensByWalletID(q))

	loaders.TokensByContractID = NewTokensLoaderByID(defaults, loadTokensByContractID(q))

	loaders.TokensByContractIDWithPagination = NewTokensLoaderByContractID(defaults, loadTokensByContractIDWithPagination(q))

	loaders.TokensByUserID = NewTokensLoaderByID(defaults, loadTokensByUserID(q))

	loaders.TokensByUserIDAndContractID = NewTokensLoaderByIDTuple(defaults, loadTokensByUserIDAndContractID(q))

	loaders.TokensByUserIDAndChain = NewTokensLoaderByIDAndChain(defaults, loadTokensByUserIDAndChain(q))

	loaders.TokensByUserIDAndChain = NewTokensLoaderByIDAndChain(defaults, loadTokensByUserIDAndChain(q))

	loaders.OwnerByTokenID = NewUserLoaderByID(defaults, loadOwnerByTokenID(q), UserLoaderByIDCacheSubscriptions{
		AutoCacheWithKey: func(user db.User) persist.DBID { return user.ID },
	})

	loaders.NotificationsByUserID = NewNotificationsLoaderByUserID(defaults, loadUserNotifications(q))

	loaders.NotificationByID = NewNotificationLoaderByID(defaults, loadNotificationById(q), NotificationLoaderByIDCacheSubscriptions{
		AutoCacheWithKey: func(notification db.Notification) persist.DBID { return notification.ID },
	})

	return loaders
}

func loadUserByUserId(q *db.Queries) func(context.Context, []persist.DBID) ([]db.User, []error) {
	return func(ctx context.Context, userIds []persist.DBID) ([]db.User, []error) {
		users := make([]db.User, len(userIds))
		errors := make([]error, len(userIds))

		b := q.GetUserByIdBatch(ctx, userIds)
		defer b.Close()

		b.QueryRow(func(i int, user db.User, err error) {
			if err == pgx.ErrNoRows {
				err = persist.ErrUserNotFound{UserID: userIds[i]}
			}

			users[i], errors[i] = user, err
		})

		return users, errors
	}
}

func loadUserByUsername(q *db.Queries) func(context.Context, []string) ([]db.User, []error) {
	return func(ctx context.Context, usernames []string) ([]db.User, []error) {
		users := make([]db.User, len(usernames))
		errors := make([]error, len(usernames))

		b := q.GetUserByUsernameBatch(ctx, usernames)
		defer b.Close()

		b.QueryRow(func(i int, user db.User, err error) {
			if err == pgx.ErrNoRows {
				err = persist.ErrUserNotFound{Username: usernames[i]}
			}

			users[i], errors[i] = user, err
		})

		return users, errors
	}
}

func loadUserByAddress(q *db.Queries) func(context.Context, []db.GetUserByAddressBatchParams) ([]db.User, []error) {
	return func(ctx context.Context, params []db.GetUserByAddressBatchParams) ([]db.User, []error) {
		users := make([]db.User, len(params))
		errors := make([]error, len(params))

		b := q.GetUserByAddressBatch(ctx, params)
		defer b.Close()

		b.QueryRow(func(i int, user db.User, err error) {
			if err == pgx.ErrNoRows {
				err = persist.ErrUserNotFound{ChainAddress: persist.NewChainAddress(params[i].Address, persist.Chain(params[i].Chain))}
			}

			users[i], errors[i] = user, err
		})

		return users, errors
	}
}

func loadOwnersByContractIDs(q *db.Queries) func(context.Context, []db.GetOwnersByContractIdBatchPaginateParams) ([][]db.User, []error) {
	return func(ctx context.Context, params []db.GetOwnersByContractIdBatchPaginateParams) ([][]db.User, []error) {
		users := make([][]db.User, len(params))
		errors := make([]error, len(params))

		b := q.GetOwnersByContractIdBatchPaginate(ctx, params)
		defer b.Close()

		b.Query(func(i int, user []db.User, err error) {
			users[i], errors[i] = user, err
		})

		return users, errors
	}
}

func loadUsersWithTrait(q *db.Queries) func(context.Context, []string) ([][]db.User, []error) {
	return func(ctx context.Context, trait []string) ([][]db.User, []error) {
		users := make([][]db.User, len(trait))
		errors := make([]error, len(trait))

		b := q.GetUsersWithTraitBatch(ctx, trait)
		defer b.Close()

		b.Query(func(i int, user []db.User, err error) {
			users[i], errors[i] = user, err
		})

		return users, errors
	}
}

func loadSplitBySplitId(q *db.Queries) func(context.Context, []persist.DBID) ([]db.Split, []error) {
	return func(ctx context.Context, splitIds []persist.DBID) ([]db.Split, []error) {
		splits := make([]db.Split, len(splitIds))
		errors := make([]error, len(splitIds))

		b := q.GetSplitByIdBatch(ctx, splitIds)
		defer b.Close()

		b.QueryRow(func(i int, g db.Split, err error) {
			splits[i] = g
			errors[i] = err

			if errors[i] == pgx.ErrNoRows {
				errors[i] = persist.ErrSplitNotFound{ID: splitIds[i]}
			}
		})

		return splits, errors
	}
}

func loadSplitsByUserId(q *db.Queries) func(context.Context, []persist.DBID) ([][]db.Split, []error) {
	return func(ctx context.Context, userIds []persist.DBID) ([][]db.Split, []error) {
		splits := make([][]db.Split, len(userIds))
		errors := make([]error, len(userIds))

		b := q.GetSplitsByUserIdBatch(ctx, userIds)
		defer b.Close()

		b.Query(func(i int, g []db.Split, err error) {
			splits[i] = g
			errors[i] = err
		})

		return splits, errors
	}
}

func loadNotificationById(q *db.Queries) func(context.Context, []persist.DBID) ([]db.Notification, []error) {
	return func(ctx context.Context, ids []persist.DBID) ([]db.Notification, []error) {
		notifs := make([]db.Notification, len(ids))
		errors := make([]error, len(ids))

		b := q.GetNotificationByIDBatch(ctx, ids)
		defer b.Close()

		b.QueryRow(func(i int, n db.Notification, err error) {
			errors[i] = err
			notifs[i] = n
		})

		return notifs, errors
	}
}

func loadWalletByWalletId(q *db.Queries) func(context.Context, []persist.DBID) ([]db.Wallet, []error) {
	return func(ctx context.Context, walletIds []persist.DBID) ([]db.Wallet, []error) {
		wallets := make([]db.Wallet, len(walletIds))
		errors := make([]error, len(walletIds))

		b := q.GetWalletByIDBatch(ctx, walletIds)
		defer b.Close()

		b.QueryRow(func(i int, wallet db.Wallet, err error) {
			// TODO err for not found by ID
			wallets[i], errors[i] = wallet, err
		})

		return wallets, errors
	}
}

func loadWalletsByUserId(q *db.Queries) func(context.Context, []persist.DBID) ([][]db.Wallet, []error) {
	return func(ctx context.Context, userIds []persist.DBID) ([][]db.Wallet, []error) {
		wallets := make([][]db.Wallet, len(userIds))
		errors := make([]error, len(userIds))

		b := q.GetWalletsByUserIDBatch(ctx, userIds)
		defer b.Close()

		b.Query(func(i int, w []db.Wallet, err error) {
			// TODO err for not found by user ID
			wallets[i], errors[i] = w, err
		})

		return wallets, errors
	}
}

func loadWalletByChainAddress(q *db.Queries) func(context.Context, []persist.ChainAddress) ([]db.Wallet, []error) {
	return func(ctx context.Context, chainAddresses []persist.ChainAddress) ([]db.Wallet, []error) {
		wallets := make([]db.Wallet, len(chainAddresses))
		errors := make([]error, len(chainAddresses))

		sqlChainAddress := make([]db.GetWalletByChainAddressBatchParams, len(chainAddresses))
		for i, chainAddress := range chainAddresses {
			sqlChainAddress[i] = db.GetWalletByChainAddressBatchParams{
				Address: chainAddress.Address(),
				Chain:   chainAddress.Chain(),
			}
		}

		b := q.GetWalletByChainAddressBatch(ctx, sqlChainAddress)
		defer b.Close()

		b.QueryRow(func(i int, wallet db.Wallet, err error) {
			if err == pgx.ErrNoRows {
				err = persist.ErrWalletNotFound{ChainAddress: chainAddresses[i]}
			}

			wallets[i], errors[i] = wallet, err
		})

		return wallets, errors
	}
}

func loadTokenByTokenID(q *db.Queries) func(context.Context, []persist.DBID) ([]db.Token, []error) {
	return func(ctx context.Context, tokenIDs []persist.DBID) ([]db.Token, []error) {
		tokens := make([]db.Token, len(tokenIDs))
		errors := make([]error, len(tokenIDs))

		b := q.GetTokenByIdBatch(ctx, tokenIDs)
		defer b.Close()

		b.QueryRow(func(i int, t db.Token, err error) {
			tokens[i], errors[i] = t, err

			if errors[i] == pgx.ErrNoRows {
				errors[i] = persist.ErrTokenNotFoundByID{ID: tokenIDs[i]}
			}
		})

		return tokens, errors
	}
}

func loadTokensByContractID(q *db.Queries) func(context.Context, []persist.DBID) ([][]db.Token, []error) {
	return func(ctx context.Context, contractIDs []persist.DBID) ([][]db.Token, []error) {
		tokens := make([][]db.Token, len(contractIDs))
		errors := make([]error, len(contractIDs))

		b := q.GetTokensByContractIdBatch(ctx, contractIDs)
		defer b.Close()

		b.Query(func(i int, t []db.Token, err error) {
			tokens[i], errors[i] = t, err
		})

		return tokens, errors
	}
}

func loadOwnerByTokenID(q *db.Queries) func(context.Context, []persist.DBID) ([]db.User, []error) {
	return func(ctx context.Context, tokenIDs []persist.DBID) ([]db.User, []error) {
		users := make([]db.User, len(tokenIDs))
		errors := make([]error, len(tokenIDs))

		b := q.GetTokenOwnerByIDBatch(ctx, tokenIDs)
		defer b.Close()

		b.QueryRow(func(i int, u db.User, err error) {
			users[i], errors[i] = u, err
		})

		return users, errors
	}
}

func loadTokensByContractIDWithPagination(q *db.Queries) func(context.Context, []db.GetTokensByContractIdBatchPaginateParams) ([][]db.Token, []error) {
	return func(ctx context.Context, params []db.GetTokensByContractIdBatchPaginateParams) ([][]db.Token, []error) {
		tokens := make([][]db.Token, len(params))
		errors := make([]error, len(params))

		b := q.GetTokensByContractIdBatchPaginate(ctx, params)
		defer b.Close()

		b.Query(func(i int, gtbcibpr []db.Token, err error) {
			tokens[i], errors[i] = gtbcibpr, err
		})

		return tokens, errors
	}
}

func loadTokensByWalletID(q *db.Queries) func(context.Context, []persist.DBID) ([][]db.Token, []error) {
	return func(ctx context.Context, walletIds []persist.DBID) ([][]db.Token, []error) {
		tokens := make([][]db.Token, len(walletIds))
		errors := make([]error, len(walletIds))

		convertedIds := make([]persist.DBIDList, len(walletIds))
		for i, id := range walletIds {
			convertedIds[i] = persist.DBIDList{id}
		}

		b := q.GetTokensByWalletIdsBatch(ctx, convertedIds)
		defer b.Close()

		b.Query(func(i int, t []db.Token, err error) {
			tokens[i], errors[i] = t, err
		})

		return tokens, errors
	}
}

func loadTokensByUserID(q *db.Queries) func(context.Context, []persist.DBID) ([][]db.Token, []error) {
	return func(ctx context.Context, userIDs []persist.DBID) ([][]db.Token, []error) {
		tokens := make([][]db.Token, len(userIDs))
		errors := make([]error, len(userIDs))

		b := q.GetTokensByUserIdBatch(ctx, userIDs)
		defer b.Close()

		b.Query(func(i int, t []db.Token, err error) {
			tokens[i], errors[i] = t, err
		})

		return tokens, errors
	}
}

func loadTokensByUserIDAndContractID(q *db.Queries) func(context.Context, []persist.DBIDTuple) ([][]db.Token, []error) {
	return func(ctx context.Context, idTuples []persist.DBIDTuple) ([][]db.Token, []error) {
		tokens := make([][]db.Token, len(idTuples))
		errors := make([]error, len(idTuples))

		params := make([]db.GetTokensByUserIdAndContractIDBatchParams, len(idTuples))
		for i, tuple := range idTuples {
			params[i] = db.GetTokensByUserIdAndContractIDBatchParams{
				OwnerUserID: tuple[0],
				Contract:    tuple[1],
			}
		}

		b := q.GetTokensByUserIdAndContractIDBatch(ctx, params)
		defer b.Close()

		b.Query(func(i int, t []db.Token, err error) {
			tokens[i], errors[i] = t, err
		})

		return tokens, errors
	}
}

func loadTokensByUserIDAndChain(q *db.Queries) func(context.Context, []IDAndChain) ([][]db.Token, []error) {
	return func(ctx context.Context, userIDsAndChains []IDAndChain) ([][]db.Token, []error) {
		tokens := make([][]db.Token, len(userIDsAndChains))
		errors := make([]error, len(userIDsAndChains))

		params := make([]db.GetTokensByUserIdAndChainBatchParams, len(userIDsAndChains))
		for i, userIDAndChain := range userIDsAndChains {
			params[i] = db.GetTokensByUserIdAndChainBatchParams{
				OwnerUserID: userIDAndChain.ID,
				Chain:       userIDAndChain.Chain,
			}
		}

		b := q.GetTokensByUserIdAndChainBatch(ctx, params)
		defer b.Close()

		b.Query(func(i int, t []db.Token, err error) {
			tokens[i], errors[i] = t, err
		})

		return tokens, errors
	}
}

func loadUserNotifications(q *db.Queries) func(context.Context, []db.GetUserNotificationsBatchParams) ([][]db.Notification, []error) {
	return func(ctx context.Context, params []db.GetUserNotificationsBatchParams) ([][]db.Notification, []error) {
		notifs := make([][]db.Notification, len(params))
		errors := make([]error, len(params))

		b := q.GetUserNotificationsBatch(ctx, params)
		defer b.Close()

		b.Query(func(i int, ntfs []db.Notification, err error) {
			notifs[i] = ntfs
			errors[i] = err
		})

		return notifs, errors
	}
}
