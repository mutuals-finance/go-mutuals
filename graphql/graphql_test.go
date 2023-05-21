//go:generate go get github.com/Khan/genqlient/generate
//go:generate go run github.com/Khan/genqlient
package graphql_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Khan/genqlient/graphql"
	genql "github.com/Khan/genqlient/graphql"

	"github.com/SplitFi/go-splitfi/server"
	"github.com/SplitFi/go-splitfi/service/auth"
	"github.com/SplitFi/go-splitfi/service/multichain"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/util"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type testCase struct {
	title    string
	run      func(t *testing.T)
	fixtures []fixture
}

func TestMain(t *testing.T) {
	tests := []testCase{
		{
			title:    "test GraphQL",
			run:      testGraphQL,
			fixtures: []fixture{useDefaultEnv, usePostgres, useRedis, useTokenQueue, useNotificationTopics},
		},
		{
			title:    "test syncing tokens",
			run:      testTokenSyncs,
			fixtures: []fixture{useDefaultEnv, usePostgres, useRedis, useTokenQueue, useTokenProcessing},
		},
	}
	for _, test := range tests {
		t.Run(test.title, testWithFixtures(test.run, test.fixtures...))
	}
}

func testGraphQL(t *testing.T) {
	tests := []testCase{
		{title: "should create a user", run: testCreateUser},
		{title: "should be able to login", run: testLogin},
		{title: "should be able to logout", run: testLogout},
		{title: "should get user by ID", run: testUserByID},
		{title: "should get user by username", run: testUserByUsername},
		{title: "should get user by address", run: testUserByAddress},
		{title: "should get viewer", run: testViewer},
		{title: "should add a wallet", run: testAddWallet},
		{title: "should remove a wallet", run: testRemoveWallet},
		{title: "should create a collection", run: testCreateCollection},
		{title: "views from multiple users are rolled up", run: testViewsAreRolledUp},
		{title: "update split and ensure name still gets set when not sent in update", run: testUpdateSplitWithNoNameChange},
		{title: "update split with a new collection", run: testUpdateSplitWithNewCollection},
		{title: "should delete collection in split update", run: testUpdateSplitDeleteCollection},
		{title: "should update user experiences", run: testUpdateUserExperiences},
		{title: "should create split", run: testCreateSplit},
		{title: "should move collection to new split", run: testMoveCollection},
		{title: "should connect social account", run: testConnectSocialAccount},
	}
	for _, test := range tests {
		t.Run(test.title, testWithFixtures(test.run, test.fixtures...))
	}
}

func testTokenSyncs(t *testing.T) {
	tests := []testCase{
		{title: "should sync new tokens", run: testSyncNewTokens},
		{title: "should submit new tokens to tokenprocessing", run: testSyncOnlySubmitsNewTokens},
		{title: "should not submit old tokens to tokenprocessing", run: testSyncSkipsSubmittingOldTokens},
		{title: "should delete old tokens", run: testSyncDeletesOldTokens},
		{title: "should combine all tokens from providers", run: testSyncShouldCombineProviders},
		{title: "should merge duplicates within provider", run: testSyncShouldMergeDuplicatesInProvider},
		{title: "should merge duplicates across providers", run: testSyncShouldMergeDuplicatesAcrossProviders},
		{title: "should process media", run: testSyncShouldProcessMedia},
	}
	for _, test := range tests {
		t.Run(test.title, testWithFixtures(test.run, test.fixtures...))
	}
}

func testCreateUser(t *testing.T) {
	nonceF := newNonceFixture(t)
	c := defaultHandlerClient(t)
	username := "user" + persist.GenerateID().String()

	response, err := createUserMutation(context.Background(), c, authMechanismInput(nonceF.Wallet, nonceF.Nonce),
		CreateUserInput{
			Username: username,
		},
	)

	require.NoError(t, err)
	payload, _ := (*response.CreateUser).(*createUserMutationCreateUserCreateUserPayload)
	assert.Equal(t, username, *payload.Viewer.User.Username)
}

func testUserByUsername(t *testing.T) {
	userF := newUserFixture(t)
	response, err := userByUsernameQuery(context.Background(), defaultHandlerClient(t), userF.Username)

	require.NoError(t, err)
	payload, _ := (*response.UserByUsername).(*userByUsernameQueryUserByUsernameSplitFiUser)
	assert.Equal(t, userF.Username, *payload.Username)
	assert.Equal(t, userF.ID, payload.Dbid)
}

func testUserByAddress(t *testing.T) {
	userF := newUserFixture(t)
	c := authedHandlerClient(t, userF.ID)

	response, err := userByAddressQuery(context.Background(), c, chainAddressInput(userF.Wallet.Address))

	require.NoError(t, err)
	payload, _ := (*response.UserByAddress).(*userByAddressQueryUserByAddressSplitFiUser)
	assert.Equal(t, userF.Username, *payload.Username)
	assert.Equal(t, userF.ID, payload.Dbid)
}

func testUserByID(t *testing.T) {
	userF := newUserFixture(t)
	response, err := userByIdQuery(context.Background(), defaultHandlerClient(t), userF.ID)

	require.NoError(t, err)
	payload, _ := (*response.UserById).(*userByIdQueryUserByIdSplitFiUser)
	assert.Equal(t, userF.Username, *payload.Username)
	assert.Equal(t, userF.ID, payload.Dbid)
}

func testViewer(t *testing.T) {
	userF := newUserFixture(t)
	c := authedHandlerClient(t, userF.ID)

	response, err := viewerQuery(context.Background(), c)
	require.NoError(t, err)

	payload, _ := (*response.Viewer).(*viewerQueryViewer)
	assert.Equal(t, userF.Username, *payload.User.Username)
}

func testAddWallet(t *testing.T) {
	userF := newUserFixture(t)
	walletToAdd := newWallet(t)
	ctx := context.Background()
	c := authedHandlerClient(t, userF.ID)
	nonce := newNonce(t, ctx, c, walletToAdd)

	response, err := addUserWalletMutation(ctx, c, chainAddressInput(walletToAdd.Address), authMechanismInput(walletToAdd, nonce))

	require.NoError(t, err)
	payload, _ := (*response.AddUserWallet).(*addUserWalletMutationAddUserWalletAddUserWalletPayload)
	wallets := payload.Viewer.User.Wallets
	assert.Equal(t, walletToAdd.Address, *wallets[len(wallets)-1].ChainAddress.Address)
	assert.Equal(t, Chain("Ethereum"), *wallets[len(wallets)-1].ChainAddress.Chain)
	assert.Len(t, wallets, 2)
}

func testRemoveWallet(t *testing.T) {
	userF := newUserFixture(t)
	walletToRemove := newWallet(t)
	ctx := context.Background()
	c := authedHandlerClient(t, userF.ID)
	nonce := newNonce(t, ctx, c, walletToRemove)
	addResponse, err := addUserWalletMutation(ctx, c, chainAddressInput(walletToRemove.Address), authMechanismInput(walletToRemove, nonce))
	require.NoError(t, err)
	wallets := (*addResponse.AddUserWallet).(*addUserWalletMutationAddUserWalletAddUserWalletPayload).Viewer.User.Wallets
	lastWallet := wallets[len(wallets)-1]
	assert.Len(t, wallets, 2)

	removeResponse, err := removeUserWalletsMutation(ctx, c, []persist.DBID{lastWallet.Dbid})

	require.NoError(t, err)
	payload, _ := (*removeResponse.RemoveUserWallets).(*removeUserWalletsMutationRemoveUserWalletsRemoveUserWalletsPayload)
	assert.Len(t, payload.Viewer.User.Wallets, 1)
	assert.NotEqual(t, lastWallet.Dbid, payload.Viewer.User.Wallets[0].Dbid)
}

func testLogin(t *testing.T) {
	userF := newUserFixture(t)
	ctx := context.Background()
	c := defaultHandlerClient(t)
	nonce := newNonce(t, ctx, c, userF.Wallet)

	response, err := loginMutation(ctx, c, authMechanismInput(userF.Wallet, nonce))

	require.NoError(t, err)
	payload, _ := (*response.Login).(*loginMutationLoginLoginPayload)
	assert.NotEmpty(t, readCookie(t, c.response, auth.JWTCookieKey))
	assert.Equal(t, userF.Username, *payload.Viewer.User.Username)
	assert.Equal(t, userF.ID, payload.Viewer.User.Dbid)
}

func testLogout(t *testing.T) {
	userF := newUserFixture(t)
	c := authedHandlerClient(t, userF.ID)

	response, err := logoutMutation(context.Background(), c)

	require.NoError(t, err)
	assert.Empty(t, readCookie(t, c.response, auth.JWTCookieKey))
	assert.Nil(t, response.Logout.Viewer)
}

func testCreateCollection(t *testing.T) {
	userF := newUserWithTokensFixture(t)
	c := authedHandlerClient(t, userF.ID)

	response, err := createCollectionMutation(context.Background(), c, CreateCollectionInput{
		SplitId:        userF.SplitID,
		Name:           "newCollection",
		CollectorsNote: "this is a note",
		Tokens:         userF.TokenIDs,
		Layout:         defaultLayout(),
		TokenSettings:  defaultTokenSettings(userF.TokenIDs),
		Caption:        nil,
	})

	require.NoError(t, err)
	payload := (*response.CreateCollection).(*createCollectionMutationCreateCollectionCreateCollectionPayload)
	assert.NotEmpty(t, payload.Collection.Dbid)
	assert.Len(t, payload.Collection.Tokens, len(userF.TokenIDs))
}

func testUpdateSplitWithPublish(t *testing.T) {
	serverF := newServerFixture(t)
	userF := newUserWithTokensFixture(t)
	c := authedServerClient(t, serverF.URL, userF.ID)

	colResp, err := createCollectionMutation(context.Background(), c, CreateCollectionInput{
		SplitId:        userF.SplitID,
		Name:           "newCollection",
		CollectorsNote: "this is a note",
		Tokens:         userF.TokenIDs[:1],
		Layout:         defaultLayout(),
		TokenSettings:  defaultTokenSettings(userF.TokenIDs[:1]),
		Caption:        nil,
	})

	require.NoError(t, err)
	colPay := (*colResp.CreateCollection).(*createCollectionMutationCreateCollectionCreateCollectionPayload)
	assert.NotEmpty(t, colPay.Collection.Dbid)
	assert.Len(t, colPay.Collection.Tokens, 1)

	updateReponse, err := updateSplitMutation(context.Background(), c, UpdateSplitInput{
		SplitId: userF.SplitID,
		Name:    util.ToPointer("newName"),
		UpdatedCollections: []*UpdateCollectionInput{
			{
				Dbid:           colPay.Collection.Dbid,
				Tokens:         userF.TokenIDs[:2],
				Name:           "yes",
				CollectorsNote: "no",
				Layout: CollectionLayoutInput{
					Sections: []int{0},
					SectionLayout: []CollectionSectionLayoutInput{
						{
							Columns:    0,
							Whitespace: []int{},
						},
					},
				},
				TokenSettings: defaultTokenSettings(userF.TokenIDs[:2]),
			},
		},
		CreatedCollections: []*CreateCollectionInSplitInput{
			{
				GivenID:        "wow",
				Tokens:         userF.TokenIDs[:3],
				CollectorsNote: "this is a note",
				Name:           "newCollection",
				Layout: CollectionLayoutInput{
					Sections: []int{0},
					SectionLayout: []CollectionSectionLayoutInput{
						{
							Columns:    3,
							Whitespace: []int{},
						},
					},
				},
				TokenSettings: defaultTokenSettings(userF.TokenIDs[:3]),
			},
		},
		Order:  []persist.DBID{colPay.Collection.Dbid, "wow"},
		EditId: util.ToPointer("edit_id"),
	})

	require.NoError(t, err)
	require.NotNil(t, updateReponse.UpdateSplit)
	updatePayload, ok := (*updateReponse.UpdateSplit).(*updateSplitMutationUpdateSplitUpdateSplitPayload)
	if !ok {
		err := (*updateReponse.UpdateSplit).(*updateSplitMutationUpdateSplitErrInvalidInput)
		t.Fatal(err)
	}
	assert.NotEmpty(t, updatePayload.Split.Name)

	update2Reponse, err := updateSplitMutation(context.Background(), c, UpdateSplitInput{
		SplitId:     userF.SplitID,
		Description: util.ToPointer("newDesc"),
		EditId:      util.ToPointer("edit_id"),
	})

	require.NoError(t, err)
	require.NotNil(t, update2Reponse.UpdateSplit)

	// Wait for event handlers to store update events
	time.Sleep(time.Second)

	// publish
	publishResponse, err := publishSplitMutation(context.Background(), c, PublishSplitInput{
		SplitId: userF.SplitID,
		EditId:  "edit_id",
		Caption: util.ToPointer("newCaption"),
	})
	require.NoError(t, err)
	require.NotNil(t, publishResponse.PublishSplit)

	_, err = viewerQuery(context.Background(), c)
	require.NoError(t, err)
}

func testCreateSplit(t *testing.T) {
	userF := newUserWithTokensFixture(t)
	c := authedHandlerClient(t, userF.ID)

	response, err := createSplitMutation(context.Background(), c, CreateSplitInput{
		Name:        util.ToPointer("newSplit"),
		Description: util.ToPointer("this is a description"),
		Position:    "a1",
	})

	require.NoError(t, err)
	payload := (*response.CreateSplit).(*createSplitMutationCreateSplitCreateSplitPayload)
	assert.NotEmpty(t, payload.Split.Dbid)
	assert.Equal(t, "newSplit", *payload.Split.Name)
	assert.Equal(t, "this is a description", *payload.Split.Description)
	assert.Equal(t, "a1", *payload.Split.Position)
}

func testMoveCollection(t *testing.T) {
	userF := newUserWithTokensFixture(t)
	c := authedHandlerClient(t, userF.ID)

	createResp, err := createCollectionMutation(context.Background(), c, CreateCollectionInput{
		SplitId:        userF.SplitID,
		Name:           "newCollection",
		CollectorsNote: "this is a note",
		Tokens:         userF.TokenIDs,
		Layout:         defaultLayout(),
		TokenSettings:  defaultTokenSettings(userF.TokenIDs),
		Caption:        nil,
	})

	require.NoError(t, err)
	createPayload := (*createResp.CreateCollection).(*createCollectionMutationCreateCollectionCreateCollectionPayload)
	assert.NotEmpty(t, createPayload.Collection.Dbid)

	createGalResp, err := createSplitMutation(context.Background(), c, CreateSplitInput{
		Name:        util.ToPointer("newSplit"),
		Description: util.ToPointer("this is a description"),
		Position:    "a1",
	})

	require.NoError(t, err)
	createGalPayload := (*createGalResp.CreateSplit).(*createSplitMutationCreateSplitCreateSplitPayload)
	assert.NotEmpty(t, createGalPayload.Split.Dbid)

	response, err := moveCollectionToSplit(context.Background(), c, MoveCollectionToSplitInput{
		SourceCollectionId: createPayload.Collection.Dbid,
		TargetSplitId:      createGalPayload.Split.Dbid,
	})

	require.NoError(t, err)
	payload := (*response.MoveCollectionToSplit).(*moveCollectionToSplitMoveCollectionToSplitMoveCollectionToSplitPayload)
	assert.NotEmpty(t, payload.OldSplit.Dbid)
	assert.Len(t, payload.OldSplit.Collections, 0)
	assert.NotEmpty(t, payload.NewSplit.Dbid)
	assert.Len(t, payload.NewSplit.Collections, 1)

}

func testUpdateUserExperiences(t *testing.T) {
	userF := newUserFixture(t)
	c := authedHandlerClient(t, userF.ID)

	response, err := updateUserExperience(context.Background(), c, UpdateUserExperienceInput{
		ExperienceType: UserExperienceTypeMultisplitannouncement,
		Experienced:    true,
	})

	require.NoError(t, err)
	bs, _ := json.Marshal(response)
	require.NotNil(t, response.UpdateUserExperience, string(bs))
	payload := (*response.UpdateUserExperience).(*updateUserExperienceUpdateUserExperienceUpdateUserExperiencePayload)
	assert.NotEmpty(t, payload.Viewer.UserExperiences)
	for _, experience := range payload.Viewer.UserExperiences {
		if experience.Type == UserExperienceTypeMultisplitannouncement {
			assert.True(t, experience.Experienced)
		}
	}
}

func testConnectSocialAccount(t *testing.T) {
	userF := newUserFixture(t)
	c := authedHandlerClient(t, userF.ID)
	dc := defaultHandlerClient(t)

	connectResp, err := connectSocialAccount(context.Background(), c, SocialAuthMechanism{
		Debug: &DebugSocialAuth{
			Provider: SocialAccountTypeTwitter,
			Id:       "123",
			Username: "test",
		},
	}, true)
	require.NoError(t, err)

	payload := (*connectResp.ConnectSocialAccount).(*connectSocialAccountConnectSocialAccountConnectSocialAccountPayload)
	assert.Equal(t, payload.Viewer.SocialAccounts.Twitter.Username, "test")
	assert.True(t, payload.Viewer.SocialAccounts.Twitter.Display)

	viewerResp, err := viewerQuery(context.Background(), c)
	require.NoError(t, err)
	viewerPayload := (*viewerResp.Viewer).(*viewerQueryViewer)
	assert.Equal(t, viewerPayload.User.SocialAccounts.Twitter.Username, "test")

	updateDisplayedResp, err := updateSocialAccountDisplayed(context.Background(), c, UpdateSocialAccountDisplayedInput{
		Type:      SocialAccountTypeTwitter,
		Displayed: false,
	})

	require.NoError(t, err)

	updateDisplayedPayload := (*updateDisplayedResp.UpdateSocialAccountDisplayed).(*updateSocialAccountDisplayedUpdateSocialAccountDisplayedUpdateSocialAccountDisplayedPayload)
	assert.Equal(t, updateDisplayedPayload.Viewer.SocialAccounts.Twitter.Username, "test")
	assert.False(t, updateDisplayedPayload.Viewer.SocialAccounts.Twitter.Display)

	userResp, err := userByIdQuery(context.Background(), dc, userF.ID)
	require.NoError(t, err)
	userPayload := (*userResp.UserById).(*userByIdQueryUserByIdSplitFiUser)
	assert.Nil(t, userPayload.SocialAccounts.Twitter)

	disconnectResp, err := disconnectSocialAccount(context.Background(), c, SocialAccountTypeTwitter)
	require.NoError(t, err)

	disconnectPayload := (*disconnectResp.DisconnectSocialAccount).(*disconnectSocialAccountDisconnectSocialAccountDisconnectSocialAccountPayload)
	assert.Nil(t, disconnectPayload.Viewer.SocialAccounts.Twitter)

}

func testUpdateSplitDeleteCollection(t *testing.T) {
	userF := newUserWithTokensFixture(t)
	c := authedHandlerClient(t, userF.ID)

	colResp, err := createCollectionMutation(context.Background(), c, CreateCollectionInput{
		SplitId:        userF.SplitID,
		Name:           "newCollection",
		CollectorsNote: "this is a note",
		Tokens:         userF.TokenIDs[:1],
		Layout: CollectionLayoutInput{
			Sections: []int{0},
			SectionLayout: []CollectionSectionLayoutInput{
				{
					Columns:    0,
					Whitespace: []int{},
				},
			},
		},
		TokenSettings: []CollectionTokenSettingsInput{
			{
				TokenId:    userF.TokenIDs[0],
				RenderLive: false,
			},
		},
		Caption: nil,
	})

	require.NoError(t, err)
	colPay := (*colResp.CreateCollection).(*createCollectionMutationCreateCollectionCreateCollectionPayload)
	assert.NotEmpty(t, colPay.Collection.Dbid)
	assert.Len(t, colPay.Collection.Tokens, 1)

	response, err := updateSplitMutation(context.Background(), c, UpdateSplitInput{
		SplitId:            userF.SplitID,
		DeletedCollections: []persist.DBID{colPay.Collection.Dbid},
		Order:              []persist.DBID{},
	})

	require.NoError(t, err)
	require.NotNil(t, response.UpdateSplit)
	payload, ok := (*response.UpdateSplit).(*updateSplitMutationUpdateSplitUpdateSplitPayload)
	if !ok {
		err := (*response.UpdateSplit).(*updateSplitMutationUpdateSplitErrInvalidInput)
		t.Fatal(err)
	}
	assert.Len(t, payload.Split.Collections, 0)
}

func testUpdateSplitWithNoNameChange(t *testing.T) {
	userF := newUserWithTokensFixture(t)
	c := authedHandlerClient(t, userF.ID)

	response, err := updateSplitMutation(context.Background(), c, UpdateSplitInput{
		SplitId: userF.SplitID,
		Name:    util.ToPointer("newName"),
	})

	require.NoError(t, err)
	payload, ok := (*response.UpdateSplit).(*updateSplitMutationUpdateSplitUpdateSplitPayload)
	if !ok {
		err := (*response.UpdateSplit).(*updateSplitMutationUpdateSplitErrInvalidInput)
		t.Fatal(err)
	}
	assert.NotEmpty(t, payload.Split.Name)

	response, err = updateSplitMutation(context.Background(), c, UpdateSplitInput{
		SplitId: userF.SplitID,
	})

	require.NoError(t, err)
	payload, ok = (*response.UpdateSplit).(*updateSplitMutationUpdateSplitUpdateSplitPayload)
	if !ok {
		err := (*response.UpdateSplit).(*updateSplitMutationUpdateSplitErrInvalidInput)
		t.Fatal(err)
	}
	assert.NotEmpty(t, payload.Split.Name)
}

func testUpdateSplitWithNewCollection(t *testing.T) {
	userF := newUserWithTokensFixture(t)
	c := authedHandlerClient(t, userF.ID)

	response, err := updateSplitMutation(context.Background(), c, UpdateSplitInput{
		SplitId: userF.SplitID,

		CreatedCollections: []*CreateCollectionInSplitInput{
			{
				Name:           "yay",
				CollectorsNote: "this is a note",
				Tokens:         userF.TokenIDs[:1],
				Hidden:         false,
				Layout: CollectionLayoutInput{
					Sections: []int{0},
					SectionLayout: []CollectionSectionLayoutInput{
						{
							Columns:    1,
							Whitespace: []int{},
						},
					},
				},
				TokenSettings: []CollectionTokenSettingsInput{},
				GivenID:       "wow",
			},
		},
		Order: []persist.DBID{"wow"},
	})

	require.NoError(t, err)
	payload, ok := (*response.UpdateSplit).(*updateSplitMutationUpdateSplitUpdateSplitPayload)
	if !ok {
		err := (*response.UpdateSplit).(*updateSplitMutationUpdateSplitErrInvalidInput)
		t.Fatal(err)
	}
	assert.Len(t, payload.Split.Collections, 1)
	assert.Len(t, payload.Split.Collections[0].Tokens, 1)
}

func testViewsAreRolledUp(t *testing.T) {
	serverF := newServerFixture(t)
	userF := newUserFixture(t)
	bob := newUserFixture(t)
	alice := newUserFixture(t)
	ctx := context.Background()
	// bob views split
	client := authedServerClient(t, serverF.URL, bob.ID)
	viewSplit(t, ctx, client, userF.SplitID)
	// // alice views split
	client = authedServerClient(t, serverF.URL, alice.ID)
	viewSplit(t, ctx, client, userF.SplitID)

	// TODO: Actually verify that the views get rolled up
}

func testSyncOnlySubmitsNewTokens(t *testing.T) {
	userF := newUserFixture(t)
	provider := defaultStubProvider(userF.Wallet.Address)
	tokenRecorder := sendTokensRecorder{}
	h := handlerWithProviders(t, tokenRecorder.Send, provider)
	c := customHandlerClient(t, h, withJWTOpt(t, userF.ID))
	tokenRecorder.On("Send", mock.Anything, mock.Anything).Times(1).Return(nil)

	_, err := syncTokensMutation(context.Background(), c, []Chain{ChainEthereum})

	require.NoError(t, err)
	tokenRecorder.AssertExpectations(t)
	assert.Len(t, tokenRecorder.Tasks[0].TokenIDs, len(provider.Tokens))
}

func testSyncSkipsSubmittingOldTokens(t *testing.T) {
	userF := newUserWithTokensFixture(t)
	tokenRecorder := sendTokensRecorder{}
	h := handlerWithProviders(t, tokenRecorder.Send, defaultStubProvider(userF.Wallet.Address))
	c := customHandlerClient(t, h, withJWTOpt(t, userF.ID))

	_, err := syncTokensMutation(context.Background(), c, []Chain{ChainEthereum})

	require.NoError(t, err)
	tokenRecorder.AssertNotCalled(t, "Send", mock.Anything, mock.Anything)
}

func testSyncDeletesOldTokens(t *testing.T) {
	userF := newUserWithTokensFixture(t)
	provider := newStubProvider(withContractTokens(multichain.ChainAgnosticContract{
		Address: "0x1337",
		Name:    "someContract",
	}, userF.Wallet.Address, 4))
	h := handlerWithProviders(t, sendTokensNOOP, provider)
	c := customHandlerClient(t, h, withJWTOpt(t, userF.ID))

	response, err := syncTokensMutation(context.Background(), c, []Chain{ChainEthereum})

	assertSyncedTokens(t, response, err, 4)
}

func testSyncShouldCombineProviders(t *testing.T) {
	userF := newUserFixture(t)
	providerA := newStubProvider(withContractTokens(multichain.ChainAgnosticContract{
		Address: "0x1337",
		Name:    "someContract",
	}, userF.Wallet.Address, 4))
	providerB := newStubProvider(withContractTokens(multichain.ChainAgnosticContract{
		Address: "0x1234",
		Name:    "anotherContract",
	}, userF.Wallet.Address, 2))
	h := handlerWithProviders(t, sendTokensNOOP, providerA, providerB)
	c := customHandlerClient(t, h, withJWTOpt(t, userF.ID))

	response, err := syncTokensMutation(context.Background(), c, []Chain{ChainEthereum})

	assertSyncedTokens(t, response, err, len(providerA.Tokens)+len(providerB.Tokens))
}

func testSyncShouldMergeDuplicatesInProvider(t *testing.T) {
	userF := newUserFixture(t)
	token := defaultToken(userF.Wallet.Address)
	provider := newStubProvider(withTokens([]multichain.ChainAgnosticToken{token, token}))
	h := handlerWithProviders(t, sendTokensNOOP, provider)
	c := customHandlerClient(t, h, withJWTOpt(t, userF.ID))

	response, err := syncTokensMutation(context.Background(), c, []Chain{ChainEthereum})

	assertSyncedTokens(t, response, err, 1)
}

func testSyncShouldMergeDuplicatesAcrossProviders(t *testing.T) {
	userF := newUserFixture(t)
	token := defaultToken(userF.Wallet.Address)
	providerA := newStubProvider(withTokens([]multichain.ChainAgnosticToken{token}))
	providerB := newStubProvider(withTokens([]multichain.ChainAgnosticToken{token}))
	h := handlerWithProviders(t, sendTokensNOOP, providerA, providerB)
	c := customHandlerClient(t, h, withJWTOpt(t, userF.ID))

	response, err := syncTokensMutation(context.Background(), c, []Chain{ChainEthereum})

	assertSyncedTokens(t, response, err, 1)
}

func testSyncShouldProcessMedia(t *testing.T) {
	metadataServer := newMetadataServerFixture(t)

	patchMetadata := func(t *testing.T, ctx context.Context, address, endpoint string) http.Handler {
		contract := multichain.ChainAgnosticContract{Address: "0x123", Name: "testContract"}
		clients := server.ClientInit(ctx)
		provider := newStubProvider(
			withContractTokens(contract, address, 1),
			withFetchMetadata(fetchFromDummyEndpoint(metadataServer.URL, endpoint, clients.IPFSClient, clients.ArweaveClient)),
		)
		mc := newMultichainProvider(clients, sendTokensNOOP, []any{provider})
		t.Cleanup(clients.Close)
		return handlerWithProviders(t, sendTokensToTokenProcessing(clients, &mc), provider)
	}

	t.Run("should process image", func(t *testing.T) {
		ctx := context.Background()
		userF := newUserFixture(t)
		h := patchMetadata(t, ctx, userF.Wallet.Address, "/metadata/image")
		c := customHandlerClient(t, h, withJWTOpt(t, userF.ID))

		response, err := syncTokensMutation(ctx, c, []Chain{ChainEthereum})

		tokens := assertSyncedTokens(t, response, err, 1)
		media := (*tokens[0].Media).(*syncTokensMutationSyncTokensSyncTokensPayloadViewerUserSplitFiUserTokensTokenMediaImageMedia)
		assert.Equal(t, string(persist.MediaTypeImage), *media.MediaType)
		assert.NotEmpty(t, *media.MediaURL)
	})

	t.Run("should process video", func(t *testing.T) {
		ctx := context.Background()
		userF := newUserFixture(t)
		h := patchMetadata(t, ctx, userF.Wallet.Address, "/metadata/video")
		c := customHandlerClient(t, h, withJWTOpt(t, userF.ID))

		response, err := syncTokensMutation(ctx, c, []Chain{ChainEthereum})

		tokens := assertSyncedTokens(t, response, err, 1)
		media := (*tokens[0].Media).(*syncTokensMutationSyncTokensSyncTokensPayloadViewerUserSplitFiUserTokensTokenMediaVideoMedia)
		assert.Equal(t, string(persist.MediaTypeVideo), *media.MediaType)
		assert.NotEmpty(t, *media.MediaURL)
	})

	t.Run("should process iframe", func(t *testing.T) {
		ctx := context.Background()
		userF := newUserFixture(t)
		h := patchMetadata(t, ctx, userF.Wallet.Address, "/metadata/iframe")
		c := customHandlerClient(t, h, withJWTOpt(t, userF.ID))

		response, err := syncTokensMutation(ctx, c, []Chain{ChainEthereum})

		tokens := assertSyncedTokens(t, response, err, 1)
		media := (*tokens[0].Media).(*syncTokensMutationSyncTokensSyncTokensPayloadViewerUserSplitFiUserTokensTokenMediaHtmlMedia)
		assert.Equal(t, string(persist.MediaTypeHTML), *media.MediaType)
		assert.NotEmpty(t, *media.MediaURL)
	})

	t.Run("should process gif", func(t *testing.T) {
		ctx := context.Background()
		userF := newUserFixture(t)
		h := patchMetadata(t, ctx, userF.Wallet.Address, "/metadata/gif")
		c := customHandlerClient(t, h, withJWTOpt(t, userF.ID))

		response, err := syncTokensMutation(ctx, c, []Chain{ChainEthereum})

		tokens := assertSyncedTokens(t, response, err, 1)
		media := (*tokens[0].Media).(*syncTokensMutationSyncTokensSyncTokensPayloadViewerUserSplitFiUserTokensTokenMediaGIFMedia)
		assert.Equal(t, string(persist.MediaTypeGIF), *media.MediaType)
		assert.NotEmpty(t, *media.MediaURL)
	})

	t.Run("should process bad metadata", func(t *testing.T) {
		ctx := context.Background()
		userF := newUserFixture(t)
		h := patchMetadata(t, ctx, userF.Wallet.Address, "/metadata/bad")
		c := customHandlerClient(t, h, withJWTOpt(t, userF.ID))

		response, err := syncTokensMutation(ctx, c, []Chain{ChainEthereum})

		tokens := assertSyncedTokens(t, response, err, 1)
		media := (*tokens[0].Media).(*syncTokensMutationSyncTokensSyncTokensPayloadViewerUserSplitFiUserTokensTokenMediaInvalidMedia)
		assert.Equal(t, "", *media.MediaType)
		assert.Empty(t, *media.MediaURL)
	})

	t.Run("should process missing metadata", func(t *testing.T) {
		ctx := context.Background()
		userF := newUserFixture(t)
		h := patchMetadata(t, ctx, userF.Wallet.Address, "/metadata/notfound")
		c := customHandlerClient(t, h, withJWTOpt(t, userF.ID))

		response, err := syncTokensMutation(ctx, c, []Chain{ChainEthereum})

		tokens := assertSyncedTokens(t, response, err, 1)
		media := (*tokens[0].Media).(*syncTokensMutationSyncTokensSyncTokensPayloadViewerUserSplitFiUserTokensTokenMediaInvalidMedia)
		assert.Equal(t, "", *media.MediaType)
		assert.Empty(t, *media.MediaURL)
	})

	t.Run("should process bad media", func(t *testing.T) {
		ctx := context.Background()
		userF := newUserFixture(t)
		h := patchMetadata(t, ctx, userF.Wallet.Address, "/metadata/media/bad")
		c := customHandlerClient(t, h, withJWTOpt(t, userF.ID))

		response, err := syncTokensMutation(ctx, c, []Chain{ChainEthereum})

		tokens := assertSyncedTokens(t, response, err, 1)
		media := (*tokens[0].Media).(*syncTokensMutationSyncTokensSyncTokensPayloadViewerUserSplitFiUserTokensTokenMediaUnknownMedia)
		assert.Equal(t, string(persist.MediaTypeUnknown), *media.MediaType)
		assert.NotEmpty(t, *media.MediaURL)
	})

	t.Run("should process missing media", func(t *testing.T) {
		ctx := context.Background()
		userF := newUserFixture(t)
		h := patchMetadata(t, ctx, userF.Wallet.Address, "/metadata/media/notfound")
		c := customHandlerClient(t, h, withJWTOpt(t, userF.ID))

		response, err := syncTokensMutation(ctx, c, []Chain{ChainEthereum})

		tokens := assertSyncedTokens(t, response, err, 1)
		media := (*tokens[0].Media).(*syncTokensMutationSyncTokensSyncTokensPayloadViewerUserSplitFiUserTokensTokenMediaUnknownMedia)
		assert.Equal(t, string(persist.MediaTypeUnknown), *media.MediaType)
		assert.NotEmpty(t, *media.MediaURL)
	})

	t.Run("should process svg", func(t *testing.T) {
		ctx := context.Background()
		userF := newUserFixture(t)
		h := patchMetadata(t, ctx, userF.Wallet.Address, "/metadata/svg")
		c := customHandlerClient(t, h, withJWTOpt(t, userF.ID))

		response, err := syncTokensMutation(ctx, c, []Chain{ChainEthereum})

		tokens := assertSyncedTokens(t, response, err, 1)
		media := (*tokens[0].Media).(*syncTokensMutationSyncTokensSyncTokensPayloadViewerUserSplitFiUserTokensTokenMediaImageMedia)
		assert.Equal(t, string(persist.MediaTypeSVG), *media.MediaType)
		assert.NotEmpty(t, *media.MediaURL)
	})

	t.Run("should process base64 encoded svg", func(t *testing.T) {
		ctx := context.Background()
		userF := newUserFixture(t)
		h := patchMetadata(t, ctx, userF.Wallet.Address, "/metadata/base64svg")
		c := customHandlerClient(t, h, withJWTOpt(t, userF.ID))

		response, err := syncTokensMutation(ctx, c, []Chain{ChainEthereum})

		tokens := assertSyncedTokens(t, response, err, 1)
		media := (*tokens[0].Media).(*syncTokensMutationSyncTokensSyncTokensPayloadViewerUserSplitFiUserTokensTokenMediaImageMedia)
		assert.Equal(t, string(persist.MediaTypeSVG), *media.MediaType)
		assert.NotEmpty(t, *media.MediaURL)
	})

	t.Run("should process base64 encoded metadata", func(t *testing.T) {
		ctx := context.Background()
		userF := newUserFixture(t)
		h := patchMetadata(t, ctx, userF.Wallet.Address, "/metadata/base64")
		c := customHandlerClient(t, h, withJWTOpt(t, userF.ID))

		response, err := syncTokensMutation(ctx, c, []Chain{ChainEthereum})

		tokens := assertSyncedTokens(t, response, err, 1)
		media := (*tokens[0].Media).(*syncTokensMutationSyncTokensSyncTokensPayloadViewerUserSplitFiUserTokensTokenMediaImageMedia)
		assert.Equal(t, *media.MediaType, string(persist.MediaTypeImage))
		assert.NotEmpty(t, *media.MediaURL)
	})

	t.Run("should process ipfs", func(t *testing.T) {
		ctx := context.Background()
		userF := newUserFixture(t)
		h := patchMetadata(t, ctx, userF.Wallet.Address, "/metadata/media/ipfs")
		c := customHandlerClient(t, h, withJWTOpt(t, userF.ID))

		response, err := syncTokensMutation(ctx, c, []Chain{ChainEthereum})

		tokens := assertSyncedTokens(t, response, err, 1)
		media := (*tokens[0].Media).(*syncTokensMutationSyncTokensSyncTokensPayloadViewerUserSplitFiUserTokensTokenMediaImageMedia)
		assert.Equal(t, *media.MediaType, string(persist.MediaTypeImage))
		assert.NotEmpty(t, *media.MediaURL)
	})

	t.Run("should process bad dns", func(t *testing.T) {
		ctx := context.Background()
		userF := newUserFixture(t)
		h := patchMetadata(t, ctx, userF.Wallet.Address, "/metadata/media/dnsbad")
		c := customHandlerClient(t, h, withJWTOpt(t, userF.ID))

		response, err := syncTokensMutation(ctx, c, []Chain{ChainEthereum})

		tokens := assertSyncedTokens(t, response, err, 1)
		media := (*tokens[0].Media).(*syncTokensMutationSyncTokensSyncTokensPayloadViewerUserSplitFiUserTokensTokenMediaImageMedia)
		assert.Equal(t, string(persist.MediaTypeImage), *media.MediaType)
		assert.NotEmpty(t, *media.MediaURL)
	})

	t.Run("should process different keyword", func(t *testing.T) {
		ctx := context.Background()
		userF := newUserFixture(t)
		h := patchMetadata(t, ctx, userF.Wallet.Address, "/metadata/differentkeyword")
		c := customHandlerClient(t, h, withJWTOpt(t, userF.ID))

		response, err := syncTokensMutation(ctx, c, []Chain{ChainEthereum})

		tokens := assertSyncedTokens(t, response, err, 1)
		media := (*tokens[0].Media).(*syncTokensMutationSyncTokensSyncTokensPayloadViewerUserSplitFiUserTokensTokenMediaImageMedia)
		assert.Equal(t, string(persist.MediaTypeImage), *media.MediaType)
		assert.NotEmpty(t, *media.MediaURL)
	})

	t.Run("should process wrong keyword", func(t *testing.T) {
		ctx := context.Background()
		userF := newUserFixture(t)
		h := patchMetadata(t, ctx, userF.Wallet.Address, "/metadata/wrongkeyword")
		c := customHandlerClient(t, h, withJWTOpt(t, userF.ID))

		response, err := syncTokensMutation(ctx, c, []Chain{ChainEthereum})

		tokens := assertSyncedTokens(t, response, err, 1)
		media := (*tokens[0].Media).(*syncTokensMutationSyncTokensSyncTokensPayloadViewerUserSplitFiUserTokensTokenMediaVideoMedia)
		assert.Equal(t, string(persist.MediaTypeVideo), *media.MediaType)
		assert.NotEmpty(t, *media.MediaURL)
	})

	t.Run("should process animation", func(t *testing.T) {
		ctx := context.Background()
		userF := newUserFixture(t)
		h := patchMetadata(t, ctx, userF.Wallet.Address, "/metadata/animation")
		c := customHandlerClient(t, h, withJWTOpt(t, userF.ID))

		response, err := syncTokensMutation(ctx, c, []Chain{ChainEthereum})

		tokens := assertSyncedTokens(t, response, err, 1)
		media := (*tokens[0].Media).(*syncTokensMutationSyncTokensSyncTokensPayloadViewerUserSplitFiUserTokensTokenMediaGltfMedia)
		assert.Equal(t, string(persist.MediaTypeAnimation), *media.MediaType)
		assert.NotEmpty(t, *media.MediaURL)
	})

	t.Run("should process pdf", func(t *testing.T) {
		ctx := context.Background()
		userF := newUserFixture(t)
		h := patchMetadata(t, ctx, userF.Wallet.Address, "/metadata/pdf")
		c := customHandlerClient(t, h, withJWTOpt(t, userF.ID))

		response, err := syncTokensMutation(ctx, c, []Chain{ChainEthereum})

		tokens := assertSyncedTokens(t, response, err, 1)
		media := (*tokens[0].Media).(*syncTokensMutationSyncTokensSyncTokensPayloadViewerUserSplitFiUserTokensTokenMediaPdfMedia)
		assert.Equal(t, string(persist.MediaTypePDF), *media.MediaType)
		assert.NotEmpty(t, *media.MediaURL)
	})

	t.Run("should process text", func(t *testing.T) {
		ctx := context.Background()
		userF := newUserFixture(t)
		h := patchMetadata(t, ctx, userF.Wallet.Address, "/metadata/text")
		c := customHandlerClient(t, h, withJWTOpt(t, userF.ID))

		response, err := syncTokensMutation(ctx, c, []Chain{ChainEthereum})

		tokens := assertSyncedTokens(t, response, err, 1)
		media := (*tokens[0].Media).(*syncTokensMutationSyncTokensSyncTokensPayloadViewerUserSplitFiUserTokensTokenMediaTextMedia)
		assert.Equal(t, string(persist.MediaTypeText), *media.MediaType)
		assert.NotEmpty(t, *media.MediaURL)
	})

	t.Run("should process bad image", func(t *testing.T) {
		ctx := context.Background()
		userF := newUserFixture(t)
		h := patchMetadata(t, ctx, userF.Wallet.Address, "/metadata/badimage")
		c := customHandlerClient(t, h, withJWTOpt(t, userF.ID))

		response, err := syncTokensMutation(ctx, c, []Chain{ChainEthereum})

		tokens := assertSyncedTokens(t, response, err, 1)
		media := (*tokens[0].Media).(*syncTokensMutationSyncTokensSyncTokensPayloadViewerUserSplitFiUserTokensTokenMediaImageMedia)
		assert.Equal(t, string(persist.MediaTypeImage), *media.MediaType)
		assert.NotEmpty(t, *media.MediaURL)
	})
}

func assertSyncedTokens(t *testing.T, response *syncTokensMutationResponse, err error, expectedLen int) []*syncTokensMutationSyncTokensSyncTokensPayloadViewerUserSplitFiUserTokensToken {
	t.Helper()
	require.NoError(t, err)
	payload := (*response.SyncTokens).(*syncTokensMutationSyncTokensSyncTokensPayload)
	assert.Len(t, payload.Viewer.User.Tokens, expectedLen)
	return payload.Viewer.User.Tokens
}

// authMechanismInput signs a nonce with an ethereum wallet
func authMechanismInput(w wallet, nonce string) AuthMechanism {
	return AuthMechanism{
		Eoa: &EoaAuth{
			Nonce:     nonce,
			Signature: w.Sign(nonce),
			ChainPubKey: ChainPubKeyInput{
				PubKey: w.Address,
				Chain:  "Ethereum",
			},
		},
	}
}

func chainAddressInput(address string) ChainAddressInput {
	return ChainAddressInput{Address: address, Chain: "Ethereum"}
}

type wallet struct {
	PKey    *ecdsa.PrivateKey
	PubKey  *ecdsa.PublicKey
	Address string
}

func (w *wallet) Sign(msg string) string {
	sig, err := crypto.Sign(crypto.Keccak256([]byte(msg)), w.PKey)
	if err != nil {
		panic(err)
	}
	return "0x" + hex.EncodeToString(sig)
}

// newWallet generates a new wallet for testing purposes
func newWallet(t *testing.T) wallet {
	t.Helper()
	pk, err := crypto.GenerateKey()
	require.NoError(t, err)

	pubKey := pk.Public().(*ecdsa.PublicKey)
	address := strings.ToLower(crypto.PubkeyToAddress(*pubKey).Hex())

	return wallet{
		PKey:    pk,
		PubKey:  pubKey,
		Address: address,
	}
}

func newNonce(t *testing.T, ctx context.Context, c graphql.Client, w wallet) string {
	t.Helper()
	response, err := getAuthNonceMutation(ctx, c, chainAddressInput(w.Address))
	require.NoError(t, err)
	payload := (*response.GetAuthNonce).(*getAuthNonceMutationGetAuthNonce)
	return *payload.Nonce
}

// newUser makes a GraphQL request to generate a new user
func newUser(t *testing.T, ctx context.Context, c graphql.Client, w wallet) (userID persist.DBID, username string, splitID persist.DBID) {
	t.Helper()
	nonce := newNonce(t, ctx, c, w)
	username = "user" + persist.GenerateID().String()

	response, err := createUserMutation(ctx, c, authMechanismInput(w, nonce),
		CreateUserInput{Username: username},
	)

	require.NoError(t, err)
	payload := (*response.CreateUser).(*createUserMutationCreateUserCreateUserPayload)
	return payload.Viewer.User.Dbid, username, payload.Viewer.User.Splits[0].Dbid
}

// newJWT generates a JWT
func newJWT(t *testing.T, ctx context.Context, userID persist.DBID) string {
	jwt, err := auth.JWTGeneratePipeline(ctx, userID)
	require.NoError(t, err)
	return jwt
}

// syncTokens makes a GraphQL request to sync a user's wallet
func syncTokens(t *testing.T, ctx context.Context, c graphql.Client, userID persist.DBID) []persist.DBID {
	t.Helper()
	resp, err := syncTokensMutation(ctx, c, []Chain{"Ethereum"})
	require.NoError(t, err)
	payload := (*resp.SyncTokens).(*syncTokensMutationSyncTokensSyncTokensPayload)
	tokens := make([]persist.DBID, len(payload.Viewer.User.Tokens))
	for i, token := range payload.Viewer.User.Tokens {
		tokens[i] = token.Dbid
	}
	return tokens
}

// viewSplit makes a GraphQL request to view a split
func viewSplit(t *testing.T, ctx context.Context, c graphql.Client, splitID persist.DBID) {
	t.Helper()
	resp, err := viewSplitMutation(ctx, c, splitID)
	require.NoError(t, err)
	_ = (*resp.ViewSplit).(*viewSplitMutationViewSplitViewSplitPayload)
}

// createCollection makes a GraphQL request to create a collection
func createCollection(t *testing.T, ctx context.Context, c graphql.Client, input CreateCollectionInput) persist.DBID {
	t.Helper()
	resp, err := createCollectionMutation(ctx, c, input)
	require.NoError(t, err)
	payload := (*resp.CreateCollection).(*createCollectionMutationCreateCollectionCreateCollectionPayload)
	return payload.Collection.Dbid
}

// defaultLayout returns a collection layout of one section with one column
func defaultLayout() CollectionLayoutInput {
	return CollectionLayoutInput{
		Sections: []int{0},
		SectionLayout: []CollectionSectionLayoutInput{
			{
				Columns:    0,
				Whitespace: []int{},
			},
		},
	}
}

// defaultToken returns a dummy token owned by the provided address
func defaultToken(address string) multichain.ChainAgnosticToken {
	return multichain.ChainAgnosticToken{
		Name:            "testToken1",
		TokenID:         "1",
		Quantity:        "1",
		ContractAddress: "0x123",
		OwnerAddress:    persist.Address(address),
	}
}

// defaultTokenSettings returns default display token settings
func defaultTokenSettings(tokens []persist.DBID) []CollectionTokenSettingsInput {
	settings := make([]CollectionTokenSettingsInput, len(tokens))
	for i, token := range tokens {
		settings[i] = CollectionTokenSettingsInput{TokenId: token}
	}
	return settings
}

// defaultHandler returns a backend GraphQL http.Handler
func defaultHandler(t *testing.T) http.Handler {
	c := server.ClientInit(context.Background())
	p := server.NewMultichainProvider(c)
	handler := server.CoreInit(c, p)
	t.Cleanup(c.Close)
	return handler
}

// handlerWithProviders returns a GraphQL http.Handler
func handlerWithProviders(t *testing.T, sendTokens multichain.SendTokens, providers ...any) http.Handler {
	c := server.ClientInit(context.Background())
	provider := newMultichainProvider(c, sendTokens, providers)
	t.Cleanup(c.Close)
	return server.CoreInit(c, &provider)
}

// newMultichainProvider a new multichain provider configured with the given providers
func newMultichainProvider(c *server.Clients, sendToken multichain.SendTokens, providers []any) multichain.Provider {
	return multichain.Provider{
		Repos:      c.Repos,
		Queries:    c.Queries,
		Chains:     map[persist.Chain][]any{persist.ChainETH: providers},
		SendTokens: sendToken,
	}
}

// defaultHandlerClient returns a GraphQL client attached to a backend GraphQL handler
func defaultHandlerClient(t *testing.T) *handlerClient {
	return customHandlerClient(t, defaultHandler(t))
}

// authedHandlerClient returns a GraphQL client with an authenticated JWT
func authedHandlerClient(t *testing.T, userID persist.DBID) *handlerClient {
	return customHandlerClient(t, defaultHandler(t), withJWTOpt(t, userID))
}

// customHandlerClient configures the client with the provided HTTP handler and client options
func customHandlerClient(t *testing.T, handler http.Handler, opts ...func(*http.Request)) *handlerClient {
	return &handlerClient{handler: handler, opts: opts, endpoint: "/glry/graphql/query"}
}

// defaultServerClient provides a client to a live server
func defaultServerClient(t *testing.T, host string) *serverClient {
	return customServerClient(t, host)
}

// authedServerClient provides an authenticated client to a live server
func authedServerClient(t *testing.T, host string, userID persist.DBID) *serverClient {
	return customServerClient(t, host, withJWTOpt(t, userID))
}

// customServerClient provides a client to a live server with custom options
func customServerClient(t *testing.T, host string, opts ...func(*http.Request)) *serverClient {
	return &serverClient{url: host + "/glry/graphql/query", opts: opts}
}

// withJWTOpt ddds a JWT cookie to the request headers
func withJWTOpt(t *testing.T, userID persist.DBID) func(*http.Request) {
	jwt, err := auth.JWTGeneratePipeline(context.Background(), userID)
	require.NoError(t, err)
	return func(r *http.Request) {
		r.AddCookie(&http.Cookie{Name: auth.JWTCookieKey, Value: jwt})
	}
}

// handlerClient records the server response for testing purposes
type handlerClient struct {
	handler  http.Handler
	endpoint string
	opts     []func(r *http.Request)
	response *http.Response
}

func (c *handlerClient) MakeRequest(ctx context.Context, req *genql.Request, resp *genql.Response) error {
	body, err := json.Marshal(map[string]any{
		"query":     req.Query,
		"variables": req.Variables,
	})
	if err != nil {
		return err
	}

	r := httptest.NewRequest(http.MethodPost, c.endpoint, io.NopCloser(bytes.NewBuffer(body)))
	r.Header.Set("Content-Type", "application/json")
	r.URL.Path = c.endpoint
	for _, opt := range c.opts {
		opt(r)
	}

	w := httptest.NewRecorder()
	c.handler.ServeHTTP(w, r)

	res := w.Result()
	c.response = res
	defer res.Body.Close()

	return json.Unmarshal(w.Body.Bytes(), resp)
}

// serverClient makes a request to a running server
type serverClient struct {
	url      string
	opts     []func(r *http.Request)
	response *http.Response
}

func (c *serverClient) MakeRequest(ctx context.Context, req *genql.Request, resp *genql.Response) error {
	body, err := json.Marshal(map[string]any{
		"query":     req.Query,
		"variables": req.Variables,
	})
	if err != nil {
		return err
	}

	r := httptest.NewRequest(http.MethodPost, c.url, io.NopCloser(bytes.NewBuffer(body)))
	r.Header.Set("Content-Type", "application/json")
	r.RequestURI = ""
	for _, opt := range c.opts {
		opt(r)
	}

	res, err := http.DefaultClient.Do(r)
	if err != nil {
		return err
	}
	c.response = res
	defer res.Body.Close()

	return json.NewDecoder(res.Body).Decode(resp)
}

// readCookie finds a cookie set in the response
func readCookie(t *testing.T, r *http.Response, name string) string {
	t.Helper()
	for _, c := range r.Cookies() {
		if c.Name == name {
			return c.Value
		}
	}
	require.NoError(t, fmt.Errorf("%s not set as a cookie", name))
	return ""
}
