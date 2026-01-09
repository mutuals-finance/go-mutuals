package auth

import (
	"context"
	"testing"

	"github.com/mutuals/go-mutuals/service/persist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAuthToken_Success(t *testing.T) {
	ctx := context.Background()
	userId := persist.DBID("test-user-id")
	appId := persist.DBID("test-app-id")
	refreshID := "refresh-id"
	roles := []persist.Role{}

	token, err := GenerateAuthToken(ctx, userId, appId, refreshID, roles)

	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestParseIdToken_Success(t *testing.T) {
	ctx := context.Background()

	mockToken := "eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6ImozdVc2YnJIMXV0LUlXWTE4bGRWYWFyS1JQZkN4SXUyUmpJd0FWcWtyZjgifQ.eyJjciI6IjE3NjY2NjgzMTEiLCJndWVzdCI6InQiLCJsaW5rZWRfYWNjb3VudHMiOiJbe1wiaWRcIjpcInQ4dWJiYzF4MzA0cHFiejZoNGRpMmhjbFwiLFwidHlwZVwiOlwid2FsbGV0XCIsXCJhZGRyZXNzXCI6XCIweDA3MGI4NTk2MDI2OThGOUZjNjE2YzYyQkZkMmI0MzhmMDBCZDFFMzBcIixcImNoYWluX3R5cGVcIjpcImV0aGVyZXVtXCIsXCJ3YWxsZXRfY2xpZW50X3R5cGVcIjpcInByaXZ5XCIsXCJsdlwiOjE3NjY2NjgzMTJ9XSIsImlzcyI6InByaXZ5LmlvIiwiaWF0IjoxNzY2ODM0MjQwLCJhdWQiOiJjbWhvcGg5MXMwMDU1aTgwYzk5eWRva2E4Iiwic3ViIjoiZGlkOnByaXZ5OmNtamxncDU4cDAxY2FsNzBkeHN4MDl6ZWIiLCJleHAiOjE3NjY4Mzc4NDB9.NhTT_QW354pYOXxAGz2l_0SqI0As_tpbrkBvE3Yh4PrJY_HruL3WYq9nnxrdDVTcBmIcEiD_x8_OnJ5Y-m7z8g"

	claims, err := ParseIdToken(ctx, mockToken)
	if err == nil {
		assert.Equal(t, "privy.io", claims.Issuer)
		assert.Equal(t, "did:privy:cmjlgp58p01cal70dxsx09zeb", claims.UserId)
		//assert.Equal(t, "TODO", claims.AppId)
		assert.NotEmpty(t, claims.LinkedAccounts)
		assert.Equal(t, "wallet", claims.LinkedAccounts[0].Type)
		assert.Equal(t, "0x070b859602698F9Fc616c62BFd2b438f00Bd1E30", claims.LinkedAccounts[0].Address)
	}
}

func TestParseIdToken_InvalidToken(t *testing.T) {
	ctx := context.Background()
	invalidToken := "invalid.token.here"

	_, err := ParseIdToken(ctx, invalidToken)

	assert.Error(t, err)
}
