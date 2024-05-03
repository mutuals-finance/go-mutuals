package operation

import (
	"context"
	db "github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/service/logger"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/util"
	"sort"
	"time"
)

type AssetFullDetails struct {
	Instance db.Asset
	Token    db.Token
}

type UpsertAsset struct {
	Asset db.Asset
	// Identifiers aren't saved to the database, but are used for joining the token to its definitions
	Identifiers persist.AssetChainAddress
}

func InsertAssets(ctx context.Context, q *db.Queries, assets []UpsertAsset) (time.Time, []AssetFullDetails, error) {
	assets = excludeZeroQuantityTokens(ctx, assets)

	// If we're not upserting anything, we still need to return the current database time
	// since it may be used by the caller and is assumed valid if err == nil
	if len(assets) == 0 {
		currentTime, err := q.GetCurrentTime(ctx)
		if err != nil {
			return time.Time{}, nil, err
		}
		return currentTime, []AssetFullDetails{}, nil
	}

	// Sort to ensure consistent insertion order
	sort.SliceStable(assets, func(i, j int) bool {
		if assets[i].Identifiers.OwnerAddress != assets[j].Identifiers.OwnerAddress {
			return assets[i].Identifiers.OwnerAddress < assets[j].Identifiers.OwnerAddress
		}
		if assets[i].Identifiers.Chain != assets[j].Identifiers.Chain {
			return assets[i].Identifiers.Chain < assets[j].Identifiers.Chain
		}
		return assets[i].Identifiers.TokenAddress < assets[j].Identifiers.TokenAddress
	})

	p := db.UpsertAssetsParams{}

	for i := range assets {
		a := &assets[i].Asset
		aID := &assets[i].Identifiers
		p.ID = append(p.ID, persist.GenerateID().String())
		p.Version = append(p.Version, a.Version.Int32)
		p.Balance = append(p.Balance, a.Balance.String())
		p.BlockNumber = append(p.BlockNumber, a.BlockNumber.Int64)
		p.OwnerAddress = append(p.OwnerAddress, aID.OwnerAddress.String())
		p.TokenAddress = append(p.TokenAddress, aID.TokenAddress.String())
	}

	added, err := q.UpsertAssets(ctx, p)
	if err != nil {
		return time.Time{}, nil, err
	}

	logger.For(ctx).Infof("added %d new asset instance(s) to the db", len(added))

	// If we're not upserting anything, we still need to return the current database time
	// since it may be used by the caller and is assumed valid if err == nil
	if len(added) == 0 {
		currentTime, err := q.GetCurrentTime(ctx)
		if err != nil {
			return time.Time{}, nil, err
		}
		return currentTime, []AssetFullDetails{}, nil
	}

	addedTokens := make([]AssetFullDetails, len(added))
	for i, a := range added {
		addedTokens[i] = AssetFullDetails{
			Instance: a.Asset,
			Token:    a.Token,
		}
	}

	return addedTokens[0].Instance.LastUpdated, addedTokens, nil
}

func excludeZeroQuantityTokens(ctx context.Context, tokens []UpsertAsset) []UpsertAsset {
	return util.Filter(tokens, func(t UpsertAsset) bool {
		if t.Asset.Balance == "" || t.Asset.Balance == "0" {
			logger.For(ctx).Warnf("Asset(chain=%d, token=%s, owner=%s) has balance of 0", t.Identifiers.Chain, t.Identifiers.TokenAddress, t.Identifiers.OwnerAddress)
			return false
		}
		return true
	}, false)
}
