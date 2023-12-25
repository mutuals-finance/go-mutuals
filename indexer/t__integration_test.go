package indexer

import (
	"context"
	"testing"
	"time"

	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/getsentry/sentry-go"
	"github.com/stretchr/testify/assert"
)

func TestIndexLogs_Success(t *testing.T) {
	a, db, pgx := setupTest(t)
	i := newMockIndexer(db, pgx)

	// Run the Indexer
	i.catchUp(sentry.SetHubOnContext(context.Background(), sentry.CurrentHub()), eventsToTopics(i.eventHashes))
	/*
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		ethClient := rpc.NewEthClient()
			ipfsShell := rpc.NewIPFSShell()
			arweaveClient := rpc.NewArweaveClient()
			stg := newStorageClient(ctx)
	*/
	t.Run("it updates its state", func(t *testing.T) {
		a.EqualValues(testBlockTo-blocksPerLogsCall, i.lastSyncedChunk)
	})

	t.Run("it stores splits in the db", func(t *testing.T) {
		t.SkipNow()
		splits := addressHasSplitsInDB(t, a, i.splitRepo, persist.Address(testAddress), expectedSplitsForAddress(persist.Address(testAddress)))
		for _, split := range splits {
			splitMatchesExpected(t, a, split)
		}
	})

	t.Run("it saves native and ERC-20 tokens to the db", func(t *testing.T) {
		for _, address := range expectedTokens() {
			token := tokenExistsInDB(t, a, i.tokenRepo, address)
			a.NotEmpty(token.ID)
			a.Equal(address, token.ContractAddress)
		}
	})

	t.Run("it updates an accounts assets", func(t *testing.T) {
		t.SkipNow()
		assets := addressHasAssetsInDB(t, a, i.assetRepo, persist.Address(contribAddress), persist.ChainETH, expectedTokensForAddress(persist.Address(testAddress)))
		for _, asset := range assets {
			assetMatchesExpected(t, a, asset)
		}
	})

	t.Run("it can create image media", func(t *testing.T) {
		// TODO
		/*

					token := tokenExistsInDB(t, a, i.tokenRepo, "0xbc4ca0eda7647a8ab7c2061c2e118a18a936f13d", "1")
			uri, err := rpc.GetTokenURI(ctx, token.TokenType, token.ContractAddress, token.TokenID, ethClient)
			tokenURIHasExpectedType(t, a, err, uri, persist.URITypeIPFS)

			metadata, err := rpc.GetMetadataFromURI(ctx, uri, ipfsShell, arweaveClient)
			mediaHasContent(t, a, err, metadata)

			predicted, _, _, err := media.PredictMediaType(ctx, metadata["image"].(string))
			mediaTypeHasExpectedType(t, a, err, persist.MediaTypeImage, predicted)

			imageData, err := rpc.GetDataFromURI(ctx, persist.TokenURI(metadata["image"].(string)), ipfsShell, arweaveClient)
			a.NoError(err)
			a.NotEmpty(imageData)

			predicted, _ = persist.SniffMediaType(imageData)
			mediaTypeHasExpectedType(t, a, nil, persist.MediaTypeImage, predicted)

			image, animation := media.KeywordsForChain(persist.ChainETH, imageKeywords, animationKeywords)
			med, err := media.MakePreviewsForMetadata(ctx, metadata, persist.Address(token.ContractAddress), token.TokenID, uri, persist.ChainETH, ipfsShell, arweaveClient, stg, env.GetString("GCLOUD_TOKEN_CONTENT_BUCKET"), image, animation)
			mediaTypeHasExpectedType(t, a, err, persist.MediaTypeImage, med.MediaType)
			a.Empty(med.ThumbnailURL)
			a.NotEmpty(med.MediaURL)

			mediaType, _, _, err := media.PredictMediaType(ctx, med.MediaURL.String())
			mediaTypeHasExpectedType(t, a, err, persist.MediaTypeImage, mediaType)

		*/
	})

	t.Run("it can create svg media", func(t *testing.T) {
		// TODO
		/*
			token := tokenExistsInDB(t, a, i.tokenRepo, "0x69c40e500b84660cb2ab09cb9614fa2387f95f64", "1")

					uri, err := rpc.GetTokenURI(ctx, token.TokenType, token.ContractAddress, token.TokenID, ethClient)
					tokenURIHasExpectedType(t, a, err, uri, persist.URITypeBase64JSON)

					metadata, err := rpc.GetMetadataFromURI(ctx, uri, ipfsShell, arweaveClient)
					mediaHasContent(t, a, err, metadata)

					image, animation := media.KeywordsForChain(persist.ChainETH, imageKeywords, animationKeywords)
					med, err := media.MakePreviewsForMetadata(ctx, metadata, persist.Address(token.ContractAddress), token.TokenID, uri, persist.ChainETH, ipfsShell, arweaveClient, stg, env.GetString("GCLOUD_TOKEN_CONTENT_BUCKET"), image, animation)
					mediaTypeHasExpectedType(t, a, err, persist.MediaTypeSVG, med.MediaType)
					a.Empty(med.ThumbnailURL)
					a.Contains(med.MediaURL.String(), "https://")
		*/
	})

}

func tokenExistsInDB(t *testing.T, a *assert.Assertions, tokenRepo persist.TokenRepository, address persist.Address) persist.Token {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tokens, err := tokenRepo.GetByTokenIdentifiers(ctx, address, 0, -1)
	a.NoError(err)
	a.Len(tokens, 1)
	return tokens[0]
}

func addressHasAssetsInDB(t *testing.T, a *assert.Assertions, assetRepo persist.AssetRepository, address persist.Address, chain persist.Chain, expected int) []persist.Asset {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	assets, err := assetRepo.GetByOwner(ctx, address, chain, -1, 0)
	a.NoError(err)
	a.Len(assets, expected)
	return assets
}

func addressHasSplitsInDB(t *testing.T, a *assert.Assertions, splitRepo persist.SplitRepository, address persist.Address, expected int) []persist.Split {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	splits, err := splitRepo.GetByRecipient(ctx, address, -1, 0)
	a.NoError(err)
	a.Len(splits, expected)
	return splits
}

func mediaHasContent(t *testing.T, a *assert.Assertions, err error, metadata persist.TokenMetadata) {
	t.Helper()
	a.NoError(err)
	a.NotEmpty(metadata["image"])
}

func mediaTypeHasExpectedType(t *testing.T, a *assert.Assertions, err error, expected, actual persist.MediaType) {
	t.Helper()
	a.NoError(err)
	a.Equal(expected, actual)
}

func splitMatchesExpected(t *testing.T, a *assert.Assertions, actual persist.Split) {
	t.Helper()
	expected, ok := expectedSplits[actual.Address]
	if !ok {
		t.Fatalf("split Address=%s not in expected splits", actual.Address.String())
	}
	a.NotEmpty(actual.ID)
	a.Equal(expected.Name, actual.Name)
	a.Equal(expected.Description, actual.Description)
	a.Equal(expected.CreatorAddress, actual.CreatorAddress)
	a.Equal(expected.Address, actual.Address)
	a.Equal(expected.Chain, actual.Chain)
	a.Equal(expected.BadgeURL, actual.BadgeURL)
	a.Equal(expected.BannerURL, actual.BannerURL)
	a.Equal(expected.Chain, actual.Chain)
	a.Len(actual.Recipients, len(expected.Recipients))
	for i, actualRecipient := range actual.Recipients {
		expectedRecipient := expected.Recipients[i]
		a.NotEmpty(actualRecipient.ID)
		a.Equal(expectedRecipient.Address, actualRecipient.Address)
		a.Equal(expectedRecipient.Ownership, actualRecipient.Ownership)
	}
}

func tokenMatchesExpected(t *testing.T, a *assert.Assertions, actual persist.Token) {
	t.Helper()
	id := persist.NewEthereumTokenIdentifiers(actual.ContractAddress)
	expected, ok := expectedResults[id]
	if !ok {
		t.Fatalf("tokenID=%s not in expected results", id)
	}
	a.NotEmpty(actual.ID)
	a.Equal(expected.BlockNumber, actual.BlockNumber)
	a.Equal(expected.Name, actual.Name)
	a.Equal(expected.Symbol, actual.Symbol)
	a.Equal(expected.Decimals, actual.Decimals)
	a.Equal(expected.ContractAddress, actual.ContractAddress)
	a.Equal(expected.TokenType, actual.TokenType)
}

func assetMatchesExpected(t *testing.T, a *assert.Assertions, actual persist.Asset) {
	t.Helper()
	id := persist.NewAssetIdentifiers(actual.Token.ContractAddress, actual.OwnerAddress)
	expected, ok := expectedAssets[id]
	if !ok {
		t.Fatalf("id=%s not in expected assets", id)
	}
	a.NotEmpty(actual.ID)
	a.Equal(expected.OwnerAddress, actual.OwnerAddress)
	a.Equal(expected.BlockNumber, actual.BlockNumber)
	a.Equal(expected.Balance, actual.Balance)
	a.Equal(expected.Token.ContractAddress, actual.Token.ContractAddress)
	a.Equal(expected.Token.Chain, actual.Token.Chain)
}
