package emails

import (
	"context"
	"database/sql"
	migrate "github.com/SplitFi/go-splitfi/db"
	"github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/docker"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/service/persist/postgres"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

var testUser = coredb.PiiUserView{
	Username:           sql.NullString{String: "test1", Valid: true},
	UsernameIdempotent: sql.NullString{String: "test1", Valid: true},
	PiiEmailAddress:    persist.Email("bc@splitfi.com"),
}

var testUser2 = coredb.PiiUserView{
	Username:           sql.NullString{String: "test2", Valid: true},
	UsernameIdempotent: sql.NullString{String: "test2", Valid: true},
	PiiEmailAddress:    persist.Email("bcc@splitfi.com"),
}

var followNotif coredb.Notification

var viewNotif coredb.Notification

var testSplit coredb.Split

func setupTest(t *testing.T) (*assert.Assertions, *sql.DB, *pgxpool.Pool) {
	setDefaults()
	r, err := docker.StartPostgres()
	if err != nil {
		t.Fatal(err)
	}

	hostAndPort := strings.Split(r.GetHostPort("5432/tcp"), ":")
	t.Setenv("POSTGRES_HOST", hostAndPort[0])
	t.Setenv("POSTGRES_PORT", hostAndPort[1])

	err = migrate.RunMigrations(postgres.MustCreateClient(postgres.WithUser("postgres")), "./db/migrations/core")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		r.Close()
	})

	db := postgres.MustCreateClient()
	pgx := postgres.NewPgxClient()

	seedNotifications(context.Background(), t, coredb.New(pgx), newRepos(db, pgx))

	return assert.New(t), db, pgx
}

func newRepos(pq *sql.DB, pgx *pgxpool.Pool) *postgres.Repositories {
	queries := coredb.New(pgx)

	return &postgres.Repositories{
		UserRepository:        postgres.NewUserRepository(pq, queries),
		NonceRepository:       postgres.NewNonceRepository(pq, queries),
		TokenRepository:       postgres.NewTokenSplitRepository(pq, queries),
		SplitRepository:       postgres.NewSplitRepository(queries),
		EarlyAccessRepository: postgres.NewEarlyAccessRepository(pq, queries),
		WalletRepository:      postgres.NewWalletRepository(pq, queries),
	}
}

func seedNotifications(ctx context.Context, t *testing.T, q *coredb.Queries, repos *postgres.Repositories) {

	email := testUser.PiiEmailAddress
	userID, err := repos.UserRepository.Create(ctx, persist.CreateUserInput{Username: testUser.Username.String, Email: &email, ChainAddress: persist.NewChainAddress("0x8914496dc01efcc49a2fa340331fb90969b6f1d2", persist.ChainETH)})
	if err != nil {
		t.Fatalf("failed to create user: %s", err)
	}

	email2 := testUser2.PiiEmailAddress
	userID2, err := repos.UserRepository.Create(ctx, persist.CreateUserInput{Username: testUser2.Username.String, Email: &email2, ChainAddress: persist.NewChainAddress("0x9a3f9764b21adaf3c6fdf6f947e6d3340a3f8ac5", persist.ChainETH)})
	if err != nil {
		t.Fatalf("failed to create user: %s", err)
	}

	testUser.ID = userID

	testUser2.ID = userID2

	splitInsert := coredb.SplitRepoCreateParams{OwnerUserID: userID, SplitID: persist.GenerateID(), Position: "0.1"}

	split, err := repos.SplitRepository.Create(ctx, splitInsert)
	if err != nil {
		t.Fatalf("failed to create split: %s", err)
	}

	splitInsert2 := coredb.SplitRepoCreateParams{OwnerUserID: userID2, SplitID: persist.GenerateID(), Position: "0.2"}

	_, err = repos.SplitRepository.Create(ctx, splitInsert2)
	if err != nil {
		t.Fatalf("failed to create split: %s", err)
	}

	collID, err := repos.CollectionRepository.Create(ctx, persist.CollectionDB{
		Name:        "test coll",
		OwnerUserID: userID,
		SplitID:     split.ID,
	})

	if err != nil {
		t.Fatalf("failed to create collection: %s", err)
	}

	err = repos.SplitRepository.Update(ctx, split.ID, userID, persist.SplitTokenUpdateInput{
		Collections: []persist.DBID{collID},
	})
	if err != nil {
		t.Fatalf("failed to update split: %s", err)
	}

	testSplit, err = q.GetSplitById(ctx, split.ID)
	if err != nil {
		t.Fatalf("failed to get split: %s", err)
	}

	_, err = q.CreateCollectionEvent(ctx, coredb.CreateCollectionEventParams{
		ID:             persist.GenerateID(),
		ActorID:        persist.DBIDToNullStr(userID),
		Action:         persist.ActionCollectionCreated,
		ResourceTypeID: persist.ResourceTypeCollection,
		CollectionID:   testSplit.Collections[0],
		SplitID:        split.ID,
	})

	if err != nil {
		t.Fatalf("failed to create collection event: %s", err)
	}

	seedViewNotif(ctx, t, q, repos, userID, userID2)
	seedFollowNotif(ctx, t, q, repos, userID, userID2)

}

func seedViewNotif(ctx context.Context, t *testing.T, q *coredb.Queries, repos *postgres.Repositories, userID persist.DBID, userID2 persist.DBID) {

	viewEvent, err := q.CreateSplitEvent(ctx, coredb.CreateSplitEventParams{
		ID:             persist.GenerateID(),
		ActorID:        persist.DBIDToNullStr(userID2),
		Action:         persist.ActionViewedSplit,
		ResourceTypeID: persist.ResourceTypeSplit,
		SplitID:        testSplit.ID,
	})

	if err != nil {
		t.Fatalf("failed to create view event: %s", err)
	}

	viewNotif, err = q.CreateViewSplitNotification(ctx, coredb.CreateViewSplitNotificationParams{
		ID:       persist.GenerateID(),
		OwnerID:  userID,
		Action:   persist.ActionViewedSplit,
		EventIds: []persist.DBID{viewEvent.ID},
		Data: persist.NotificationData{
			AuthedViewerIDs: []persist.DBID{userID2},
		},
		SplitID: testSplit.ID,
	})

	if err != nil {
		t.Fatalf("failed to create view event: %s", err)
	}

}

func seedFollowNotif(ctx context.Context, t *testing.T, q *coredb.Queries, repos *postgres.Repositories, userID persist.DBID, userID2 persist.DBID) {

	viewEvent, err := q.CreateUserEvent(ctx, coredb.CreateUserEventParams{
		ID:             persist.GenerateID(),
		ActorID:        persist.DBIDToNullStr(userID2),
		Action:         persist.ActionUserFollowedUsers,
		ResourceTypeID: persist.ResourceTypeUser,
		UserID:         userID,
	})

	if err != nil {
		t.Fatalf("failed to create follow event: %s", err)
	}

	followNotif, err = q.CreateFollowNotification(ctx, coredb.CreateFollowNotificationParams{
		ID:       persist.GenerateID(),
		OwnerID:  userID,
		Action:   persist.ActionUserFollowedUsers,
		EventIds: []persist.DBID{viewEvent.ID},
		Data: persist.NotificationData{
			FollowerIDs: []persist.DBID{userID2},
		},
	})

	if err != nil {
		t.Fatalf("failed to create follow event: %s", err)
	}

}
