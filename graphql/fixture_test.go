package graphql_test

import (
	"context"
	"github.com/SplitFi/go-splitfi/env"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	migrate "github.com/SplitFi/go-splitfi/db"
	"github.com/SplitFi/go-splitfi/docker"
	"github.com/SplitFi/go-splitfi/graphql/dummymetadata"
	"github.com/SplitFi/go-splitfi/server"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/service/persist/postgres"
	"github.com/SplitFi/go-splitfi/service/pubsub/gcp"
	"github.com/stretchr/testify/require"
)

// fixture runs setup and teardown for a test
type fixture func(t *testing.T)

// testWithFixtures sets up each fixture before running the test
func testWithFixtures(test func(t *testing.T), fixtures ...fixture) func(t *testing.T) {
	return func(t *testing.T) {
		for _, fixture := range fixtures {
			fixture(t)
		}
		test(t)
	}
}

// useDefaultEnv sets the test environment to the default server environment
func useDefaultEnv(t *testing.T) {
	prevValues := make(map[string]string)
	for _, envVar := range os.Environ() {
		kv := strings.Split(envVar, "=")
		prevValues[kv[0]] = kv[1]
	}

	server.SetDefaults()
	curValues := os.Environ()

	t.Cleanup(func() {
		for _, envVar := range curValues {
			k := strings.Split(envVar, "=")[0]
			if prevVal, ok := prevValues[k]; ok {
				os.Setenv(k, prevVal)
			} else {
				os.Unsetenv(k)
			}
		}
	})
}

// usePostgres starts a running Postgres Docker container with migrations applied.
// The passed testing.T arg resets the environment and deletes the container
// when the test and its subtests complete.
func usePostgres(t *testing.T) {
	t.Helper()
	r, err := docker.StartPostgres()
	require.NoError(t, err)
	hostAndPort := strings.Split(r.GetHostPort("5432/tcp"), ":")
	t.Setenv("POSTGRES_HOST", hostAndPort[0])
	t.Setenv("POSTGRES_PORT", hostAndPort[1])

	err = migrate.RunMigrations(postgres.MustCreateClient(postgres.WithUser("postgres")), "./db/migrations/core")
	require.NoError(t, err)

	t.Cleanup(func() { r.Close() })
}

// useRedis starts a running Redis Docker container and stops the instance
// when the test and its subtests complete
func useRedis(t *testing.T) {
	t.Helper()
	r, err := docker.StartRedis()
	require.NoError(t, err)
	t.Setenv("REDIS_URL", r.GetHostPort("6379/tcp"))
	t.Cleanup(func() { r.Close() })
}

// useCloudTasksDirectDispatch is a fixture that sends tasks directly to their targets instead of using the Cloud Tasks emulator
func useCloudTasksDirectDispatch(t *testing.T) {
	t.Helper()

	// Skip these queues -- don't dispatch their tasks at all
	t.Setenv("CLOUD_TASKS_SKIP_QUEUES", strings.Join([]string{
		env.GetString("AUTOSOCIAL_QUEUE"),
	}, ","))

	t.Setenv("CLOUD_TASKS_DIRECT_DISPATCH_ENABLED", "true")
}

// useNotificationTopics is a fixture that creates dummy PubSub topics for notifications
func useNotificationTopics(t *testing.T) {
	t.Helper()
	usePubSub(t)
	ctx := context.Background()
	client := gcp.NewClient(ctx)

	newNotificationsTopic := "new-notifications" + persist.GenerateID().String()
	_, err := client.CreateTopic(ctx, newNotificationsTopic)
	require.NoError(t, err)
	t.Setenv("PUBSUB_TOPIC_NEW_NOTIFICATIONS", newNotificationsTopic)

	updatedNotificationsTopic := "updated-notifications" + persist.GenerateID().String()
	_, err = client.CreateTopic(ctx, updatedNotificationsTopic)
	require.NoError(t, err)
	t.Setenv("PUBSUB_TOPIC_UPDATED_NOTIFICATIONS", updatedNotificationsTopic)
}

// useCloudTasks starts a running Cloud Tasks emulator
func useCloudTasks(t *testing.T) {
	t.Helper()
	r, err := docker.StartCloudTasks()
	require.NoError(t, err)
	t.Setenv("TASK_QUEUE_HOST", r.GetHostPort("8123/tcp"))
	t.Cleanup(func() { r.Close() })
}

// usePubSub starts a running PubSub emulator
func usePubSub(t *testing.T) {
	t.Helper()
	r, err := docker.StartPubSub()
	require.NoError(t, err)
	t.Setenv("PUBSUB_EMULATOR_HOST", r.GetHostPort("8085/tcp"))
	t.Cleanup(func() { r.Close() })
}

type serverFixture struct {
	*httptest.Server
}

// newServerFixture starts a new HTTP server for end-to-end tests
func newServerFixture(t *testing.T) serverFixture {
	t.Helper()
	server := httptest.NewServer(defaultHandler(t))
	t.Cleanup(func() { server.Close() })
	return serverFixture{server}
}

// newMetadataServerFixture starts a HTTP server for fetching static metadata
func newMetadataServerFixture(t *testing.T) serverFixture {
	t.Helper()
	server := httptest.NewServer(dummymetadata.CoreInitServer())
	t.Cleanup(func() { server.Close() })
	return serverFixture{server}
}

type nonceFixture struct {
	Wallet  wallet
	Nonce   string
	Message string
}

// newNonceFixture generates a new nonce
func newNonceFixture(t *testing.T) nonceFixture {
	t.Helper()
	wallet := newWallet(t)
	ctx := context.Background()
	c := defaultHandlerClient(t)
	nonce, message := newNonce(t, ctx, c)
	return nonceFixture{wallet, nonce, message}
}

type userFixture struct {
	Wallet   wallet
	Username string
	ID       persist.DBID
	SplitID  persist.DBID
}

// newUserFixture generates a new user
func newUserFixture(t *testing.T) userFixture {
	t.Helper()
	wallet := newWallet(t)
	ctx := context.Background()
	c := defaultHandlerClient(t)
	userID, username, splitID := newUser(t, ctx, c, wallet)
	return userFixture{wallet, username, userID, splitID}
}

type userWithTokensFixture struct {
	userFixture
	TokenIDs []persist.DBID
}

// newUserWithTokensFixture generates a new user with tokens synced
func newUserWithTokensFixture(t *testing.T) userWithTokensFixture {
	t.Helper()
	user := newUserFixture(t)
	/*	ctx := context.Background()
		providers := multichain.ProviderLookup{persist.ChainETH: defaultStubProvider(user.Wallet.Address)}
		h := handlerWithProviders(t, submitUserTokensNoop, providers)
		c := customHandlerClient(t, h, withJWTOpt(t, user.ID))
		tokenIDs := syncTokens(t, ctx, c, user.ID)
		return userWithTokensFixture{user, tokenIDs}
	*/
	return userWithTokensFixture{user, []persist.DBID{}}

}

/*type pushNotificationServiceFixture struct {
	SentNotificationBodies []string
	Errors                 []error
	mu                     sync.Mutex
}

func (p *pushNotificationServiceFixture) appendNotificationBody(title string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.SentNotificationBodies = append(p.SentNotificationBodies, title)
}

func (p *pushNotificationServiceFixture) appendError(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Errors = append(p.Errors, err)
}

// newPushNotificationServiceFixture creates a mock push notification service that records the bodies of messages that would be sent
func newPushNotificationServiceFixture(t *testing.T) *pushNotificationServiceFixture {
	t.Helper()
	ctx := context.Background()
	service := &pushNotificationServiceFixture{}

	expoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var messages []expo.PushMessage
		if err := json.NewDecoder(r.Body).Decode(&messages); err != nil {
			service.appendError(err)
			return
		}

		response := expo.SendMessagesResponse{}
		for _, message := range messages {
			service.appendNotificationBody(message.Body)
			response.Data = append(response.Data, expo.PushTicket{
				TicketID: persist.GenerateID().String(),
				Status:   expo.StatusOK,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK) // HTTP 200
		if err := json.NewEncoder(w).Encode(response); err != nil {
			service.appendError(err)
			return
		}
	}))
	t.Setenv("EXPO_PUSH_API_URL", expoServer.URL)

	pushServer := httptest.NewServer(pushnotifications.CoreInitServer(ctx))
	t.Setenv("PUSH_NOTIFICATIONS_URL", pushServer.URL)

	t.Cleanup(func() {
		pushServer.Close()
		expoServer.Close()
	})

	return service
}*/
