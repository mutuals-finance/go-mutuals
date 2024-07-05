package operation

import (
	"context"
	db "github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/service/logger"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/util"
	"github.com/jackc/pgtype"
	"sort"
)

type TokenFullDetails struct {
	Instance db.Token
	Metadata db.TokenMetadata
}

func InsertTokenMetadatas(ctx context.Context, q *db.Queries, tokens []db.TokenMetadata) ([]db.TokenMetadata, []bool, error) {
	// Sort to ensure consistent insertion order
	sort.SliceStable(tokens, func(i, j int) bool {
		if tokens[i].Chain != tokens[j].Chain {
			return tokens[i].Chain < tokens[j].Chain
		}
		return tokens[i].ContractAddress < tokens[j].ContractAddress
	})

	var p db.UpsertTokenMetadatasParams
	var errors []error

	for i := range tokens {
		t := &tokens[i]
		p.Dbid = append(p.Dbid, persist.GenerateID().String())
		p.Name = append(p.Name, t.Name.String)
		p.Symbol = append(p.Symbol, t.Symbol.String)
		p.Chain = append(p.Chain, t.Chain)
		p.Thumbnail = append(p.Thumbnail, t.Thumbnail.String)
		p.Logo = append(p.Logo, t.Logo.String)
		p.ContractAddress = append(p.ContractAddress, t.ContractAddress)

		if len(errors) > 0 {
			return nil, nil, errors[0]
		}
	}

	added, err := q.UpsertTokenMetadatas(ctx, p)
	if err != nil {
		return nil, nil, err
	}

	logger.For(ctx).Infof("added %d new metadata(s) to the db", len(added))

	metadatas := make([]db.TokenMetadata, len(added))
	isNewMetadata := make([]bool, len(added))
	for i, t := range added {
		metadatas[i] = t.TokenMetadata
		isNewMetadata[i] = t.IsNewMetadata
	}

	return metadatas, isNewMetadata, nil
}

func InsertTokens(ctx context.Context, q *db.Queries, tokens []db.Token) ([]TokenFullDetails, error) {
	tokens = excludeZeroQuantityTokens(ctx, tokens)

	// If we're not upserting anything, we still need to return the current database time
	// since it may be used by the caller and is assumed valid if err == nil
	if len(tokens) == 0 {
		return []TokenFullDetails{}, nil
	}

	// Sort to ensure consistent insertion order
	sort.SliceStable(tokens, func(i, j int) bool {
		if tokens[i].OwnerAddress != tokens[j].OwnerAddress {
			return tokens[i].OwnerAddress < tokens[j].OwnerAddress
		}
		if tokens[i].Chain != tokens[j].Chain {
			return tokens[i].Chain < tokens[j].Chain
		}
		return tokens[i].TokenAddress < tokens[j].TokenAddress
	})

	p := db.UpsertTokensParams{}

	for i := range tokens {
		t := &tokens[i]
		p.Dbid = append(p.Dbid, persist.GenerateID())
		p.Version = append(p.Version, t.Version.Int32)
		p.Balance = append(p.Balance, t.Balance.String())
		p.OwnerAddress = append(p.OwnerAddress, t.OwnerAddress)
		p.Chain = append(p.Chain, t.Chain)
		p.TokenAddress = append(p.TokenAddress, t.TokenAddress)
	}

	added, err := q.UpsertTokens(ctx, p)

	if err != nil {
		return nil, err
	}

	logger.For(ctx).Infof("added %d new token instance(s) to the db", len(added))

	addedTokens := make([]TokenFullDetails, len(added))
	for i, t := range added {
		addedTokens[i] = TokenFullDetails{
			Instance: t.Token,
			Metadata: t.TokenMetadata,
		}
	}

	return addedTokens, nil
}

func excludeZeroQuantityTokens(ctx context.Context, tokens []db.Token) []db.Token {
	return util.Filter(tokens, func(t db.Token) bool {
		if t.Balance == "" || t.Balance == "0" {
			logger.For(ctx).Warnf("%s has 0 quantity", persist.NewTokenChainAddress(t.TokenAddress, t.Chain))
			return false
		}
		return true
	}, false)
}

func appendIndices(startIndices *[]int32, endIndices *[]int32, entryLength int) {
	// Postgres uses 1-based indexing
	startIndex := int32(1)
	if len(*endIndices) > 0 {
		startIndex = (*endIndices)[len(*endIndices)-1] + 1
	}
	*startIndices = append(*startIndices, startIndex)
	*endIndices = append(*endIndices, startIndex+int32(entryLength)-1)
}

func appendJSONB(dest *[]pgtype.JSONB, src any, errs *[]error) error {
	jsonb, err := persist.ToJSONB(src)
	if err != nil {
		*errs = append(*errs, err)
		return err
	}
	*dest = append(*dest, jsonb)
	return nil
}

func appendDBIDList(dest *[]string, src []persist.DBID, startIndices, endIndices *[]int32) {
	for _, id := range src {
		*dest = append(*dest, id.String())
	}
	appendIndices(startIndices, endIndices, len(src))
}
