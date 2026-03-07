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
	"github.com/gin-gonic/gin"
	"github.com/mutuals/go-mutuals/publicapi"
	"github.com/mutuals/go-mutuals/service/redis"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	genql "github.com/Khan/genqlient/graphql"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mutuals/go-mutuals/server"
	"github.com/mutuals/go-mutuals/service/auth"
	"github.com/mutuals/go-mutuals/service/multichain"
	"github.com/mutuals/go-mutuals/service/persist"
	"github.com/mutuals/go-mutuals/util"
	"github.com/stretchr/testify/assert"
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
			fixtures: []fixture{useDefaultEnv, usePostgres, useRedis, useCloudTasksDirectDispatch, useNotificationTopics},
		},
		/*		{
					title: "test syncing tokens",
					run:   testTokenSyncs,
					fixtures: []fixture{useDefaultEnv, usePostgres, useRedis, useCloudTasksDirectDispatch},
				},
		*/}
	for _, test := range tests {
		t.Run(test.title, testWithFixtures(test.run, test.fixtures...))
	}
}

func testGraphQL(t *testing.T) {
	tests := []testCase{
		{title: "should create a user", run: testCreateUser},
		{title: "should get user by ID", run: testUserByID},
		{title: "should get user by username", run: testUserByUsername},
		{title: "should get user by address", run: testUserByAddress},
		{title: "should get viewer", run: testViewer},
		{title: "should add a wallet", run: testAddWallet},
		{title: "should remove a wallet", run: testRemoveWallet},
		{title: "should update pool and ensure name still gets set when not sent in update", run: testUpsertPoolWithNoNameChange},
		{title: "should update user experiences", run: testUpdateUserExperiences},
		{title: "should create pool", run: testCreatePool},
		//{title: "should send notifications", run: testSendNotifications, fixtures: []fixture{usePostgres, useRedis}},
	}
	for _, test := range tests {
		t.Run(test.title, testWithFixtures(test.run, test.fixtures...))
	}
}

/*
	func testTokenSyncs(t *testing.T) {
		tests := []testCase{
			// TODO
		}
		for _, test := range tests {
			t.Run(test.title, testWithFixtures(test.run, test.fixtures...))
		}
	}
*/

func testCreateUser(t *testing.T) {
	nonceF := newNonceFixture(t)
	c := defaultHandlerClient(t)
	username := "user" + persist.GenerateID().String()

	response, err := createUserMutation(context.Background(), c, authMechanismInput(nonceF.Wallet, nonceF.Nonce, nonceF.Message),
		CreateUserInput{
			Username: &username,
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
	payload, _ := (*response.UserByUsername).(*userByUsernameQueryUserByUsernameMutualsUser)
	assert.Equal(t, userF.Username, *payload.Username)
	assert.Equal(t, userF.ID, payload.Dbid)
}

func testUserByAddress(t *testing.T) {
	userF := newUserFixture(t)
	c := authedHandlerClient(t, userF.ID)

	response, err := userByAddressQuery(context.Background(), c, chainAddressInput(userF.Wallet.Address))

	require.NoError(t, err)
	payload, _ := (*response.UserByAddress).(*userByAddressQueryUserByAddressMutualsUser)
	assert.Equal(t, userF.Username, *payload.Username)
	assert.Equal(t, userF.ID, payload.Dbid)
}

func testUserByID(t *testing.T) {
	userF := newUserFixture(t)
	response, err := userByIdQuery(context.Background(), defaultHandlerClient(t), userF.ID)

	require.NoError(t, err)
	payload, _ := (*response.UserById).(*userByIdQueryUserByIdMutualsUser)
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
	nonce, message := newNonce(t, ctx, c)

	response, err := addUserWalletMutation(ctx, c, chainAddressInput(walletToAdd.Address), authMechanismInput(walletToAdd, nonce, message))

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
	nonce, message := newNonce(t, ctx, c)
	addResponse, err := addUserWalletMutation(ctx, c, chainAddressInput(walletToRemove.Address), authMechanismInput(walletToRemove, nonce, message))
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

func testUpsertPoolWithPublish(t *testing.T) {
	serverF := newServerFixture(t)
	userF := newUserWithTokensFixture(t)
	c := authedServerClient(t, serverF.URL, userF.ID)

	updateReponse, err := upsertPoolMutation(context.Background(), c, UpsertPoolInput{
		PoolId: util.ToPointer(userF.PoolID),
		Name:   util.ToPointer("newName"),
	})

	require.NoError(t, err)
	require.NotNil(t, updateReponse.UpsertPool)
	updatePayload, ok := (*updateReponse.UpsertPool).(*upsertPoolMutationUpsertPoolUpsertPoolPayload)
	if !ok {
		err := (*updateReponse.UpsertPool).(*upsertPoolMutationUpsertPoolErrInvalidInput)
		t.Fatal(err)
	}
	assert.NotEmpty(t, updatePayload.Pool.Name)

	update2Reponse, err := upsertPoolMutation(context.Background(), c, UpsertPoolInput{
		PoolId:      util.ToPointer(userF.PoolID),
		Description: util.ToPointer("newDesc"),
	})

	require.NoError(t, err)
	require.NotNil(t, update2Reponse.UpsertPool)

	// Wait for event handlers to store update events
	time.Sleep(time.Second)

	// publish
	publishResponse, err := publishPoolMutation(context.Background(), c, PublishPoolInput{
		PoolId:  userF.PoolID,
		EditId:  "edit_id",
		Caption: util.ToPointer("newCaption"),
	})
	require.NoError(t, err)
	require.NotNil(t, publishResponse.PublishPool)

	_, err = viewerQuery(context.Background(), c)
	require.NoError(t, err)
}

func testCreatePool(t *testing.T) {
	userF := newUserWithTokensFixture(t)
	c := authedHandlerClient(t, userF.ID)

	response, err := createPoolMutation(context.Background(), c, CreatePoolInput{
		Name:        util.ToPointer("newPool"),
		Description: util.ToPointer("this is a description"),
	})

	require.NoError(t, err)
	payload := (*response.CreatePool).(*createPoolMutationCreatePoolCreatePoolPayload)
	assert.NotEmpty(t, payload.Pool.Dbid)
	assert.Equal(t, "newPool", *payload.Pool.Name)
	assert.Equal(t, "this is a description", *payload.Pool.Description)
}

func testUpdateUserExperiences(t *testing.T) {
	userF := newUserFixture(t)
	c := authedHandlerClient(t, userF.ID)

	response, err := updateUserExperience(context.Background(), c, UpdateUserExperienceInput{
		ExperienceType: UserExperienceTypeEmailupsell,
		Experienced:    true,
	})

	require.NoError(t, err)
	bs, _ := json.Marshal(response)
	require.NotNil(t, response.UpdateUserExperience, string(bs))
	payload := (*response.UpdateUserExperience).(*updateUserExperienceUpdateUserExperienceUpdateUserExperiencePayload)
	assert.NotEmpty(t, payload.Viewer.UserExperiences)
	for _, experience := range payload.Viewer.UserExperiences {
		if experience.Type == UserExperienceTypeEmailupsell {
			assert.True(t, experience.Experienced)
		}
	}
}

func testUpsertPoolWithNoNameChange(t *testing.T) {
	userF := newUserWithTokensFixture(t)
	c := authedHandlerClient(t, userF.ID)

	response, err := upsertPoolMutation(context.Background(), c, UpsertPoolInput{
		PoolId: util.ToPointer(userF.PoolID),
		Name:   util.ToPointer("newName"),
	})

	require.NoError(t, err)
	payload, ok := (*response.UpsertPool).(*upsertPoolMutationUpsertPoolUpsertPoolPayload)
	if !ok {
		err := (*response.UpsertPool).(*upsertPoolMutationUpsertPoolErrInvalidInput)
		t.Fatal(err)
	}
	assert.NotEmpty(t, payload.Pool.Name)

	response, err = upsertPoolMutation(context.Background(), c, UpsertPoolInput{
		PoolId: util.ToPointer(userF.PoolID),
	})

	require.NoError(t, err)
	payload, ok = (*response.UpsertPool).(*upsertPoolMutationUpsertPoolUpsertPoolPayload)
	if !ok {
		err := (*response.UpsertPool).(*upsertPoolMutationUpsertPoolErrInvalidInput)
		t.Fatal(err)
	}
	assert.NotEmpty(t, payload.Pool.Name)
}

// authMechanismInput signs a nonce with an ethereum wallet
func authMechanismInput(w wallet, nonce string, message string) AuthMechanism {
	return AuthMechanism{
		Eoa: &EoaAuth{
			Nonce:     nonce,
			Message:   message,
			Signature: w.Sign(message),
			ChainPubKey: ChainPubKeyInput{
				PubKey: w.Address,
				Chain:  "Ethereum",
			},
		},
	}
}

func chainAddressInput(address string) ChainAddressInput {
	return ChainAddressInput{Address: persist.Address(address), Chain: "Ethereum"}
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

func newNonce(t *testing.T, ctx context.Context, c genql.Client) (string, string) {
	t.Helper()
	response, err := getAuthNonceMutation(ctx, c)
	require.NoError(t, err)
	payload := (*response.GetAuthNonce).(*getAuthNonceMutationGetAuthNonce)
	return *payload.Nonce, *payload.Message
}

// registerPushtoken makes a GraphQL request to register a push token for an authenticated user
func registerPushToken(t *testing.T, ctx context.Context, c genql.Client) {
	t.Helper()
	_, err := registerPushTokenMutation(ctx, c, persist.GenerateID().String())
	require.NoError(t, err)
}

// newUser makes a GraphQL request to generate a new user
func newUser(t *testing.T, ctx context.Context, c genql.Client, w wallet) (userId persist.DBID, username string, galleryID persist.DBID) {
	t.Helper()
	nonce, message := newNonce(t, ctx, c)
	username = "user" + persist.GenerateID().String()

	response, err := createUserMutation(ctx, c, authMechanismInput(w, nonce, message),
		CreateUserInput{Username: &username},
	)

	require.NoError(t, err)
	payload := (*response.CreateUser).(*createUserMutationCreateUserCreateUserPayload)
	return payload.Viewer.User.Dbid, username, payload.Viewer.User.Pools[0].Dbid
}

// defaultHandler returns a backend GraphQL http.Handler
func defaultHandler(t *testing.T) http.Handler {
	ctx := context.Background()
	c := server.ClientInit(ctx)
	handler := server.CoreInit(ctx, c)
	t.Cleanup(func() {
		c.Close()
	})
	return handler
}

// handlerWithProviders returns a GraphQL http.Handler
func handlerWithProviders(t *testing.T, p multichain.ProviderLookup) http.Handler {
	ctx := context.Background()
	c := server.ClientInit(context.Background())
	provider := newMultichainProvider(c, p)
	t.Cleanup(c.Close)

	lock := redis.NewLockClient(redis.NewCache(redis.NotificationLockCache))
	authRefreshCache := redis.NewCache(redis.AuthTokenForceRefreshCache)

	publicapiF := func(ctx context.Context, disableDataloaderCaching bool) *publicapi.PublicAPI {
		return publicapi.NewWithMultichainProvider(
			ctx,
			false,
			c.Repos,
			c.CoreQueries,
			c.IndexerQueries,
			c.HTTPClient,
			c.EthClient,
			c.IPFSClient,
			c.ArweaveClient,
			c.StorageClient,
			c.TaskClient,
			nil, // throttler
			c.SecretClient,
			nil, // apqCache
			authRefreshCache,
			&provider,
		)
	}

	handlerInitF := func(r *gin.Engine) {
		server.GraphqlHandlersInit(
			r,
			c.CoreQueries,
			c.IndexerQueries,
			c.TaskClient,
			c.PubSubClient,
			lock,             // redislock
			nil,              // apqCache
			authRefreshCache, // authRefreshCache
			publicapiF,
		)
	}

	return server.CoreInitHandlerF(ctx, handlerInitF)
}

// newMultichainProvider a new multichain provider configured with the given providers
func newMultichainProvider(c *server.Clients, p multichain.ProviderLookup) multichain.Provider {
	return multichain.Provider{
		Repos:   c.Repos,
		Queries: c.CoreQueries,
		Chains:  p,
	}
}

// defaultHandlerClient returns a GraphQL client attached to a backend GraphQL handler
func defaultHandlerClient(t *testing.T) *handlerClient {
	return customHandlerClient(t, defaultHandler(t))
}

// authedHandlerClient returns a GraphQL client with an authenticated JWT
func authedHandlerClient(t *testing.T, userId persist.DBID) *handlerClient {
	return customHandlerClient(t, defaultHandler(t), withJWTOpt(t, userId))
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
func authedServerClient(t *testing.T, host string, userId persist.DBID) *serverClient {
	return customServerClient(t, host, withJWTOpt(t, userId))
}

// customServerClient provides a client to a live server with custom options
func customServerClient(t *testing.T, host string, opts ...func(*http.Request)) *serverClient {
	return &serverClient{url: host + "/glry/graphql/query", opts: opts}
}

// withJWTOpt adds a JWT cookie to the request headers
// TODO: Update this to use Privy ID tokens for testing instead of old custom auth tokens
func withJWTOpt(t *testing.T, userID persist.DBID) func(*http.Request) {
	// This function needs to be updated to generate Privy ID tokens for testing
	// For now, tests using this will need to be updated or skipped
	t.Skip("withJWTOpt needs to be updated for Privy authentication")
	return func(r *http.Request) {
		// TODO: Generate valid Privy ID token for testing
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
