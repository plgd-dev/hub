package mongodb_test

import (
	"context"
	"testing"
	"time"

	"github.com/plgd-dev/hub/v2/m2m-oauth-server/pb"
	"github.com/plgd-dev/hub/v2/m2m-oauth-server/test"
	"github.com/plgd-dev/hub/v2/test/config"
	"github.com/stretchr/testify/require"
)

func TestGetTokens(t *testing.T) {
	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT*100)
	defer cancel()

	// Set the owner and request parameters
	owner := "testOwner"
	tokens := []*pb.Token{
		{
			Id:       "token1",
			Owner:    owner,
			Version:  0,
			Name:     "name1",
			IssuedAt: time.Now().Unix(),
			ClientId: "client1",
		},
		{
			Id:       "token2",
			Owner:    owner,
			Version:  0,
			Name:     "name2",
			IssuedAt: time.Now().Unix(),
			ClientId: "client1",
			Blacklisted: &pb.Token_BlackListed{
				Flag:      true,
				Timestamp: time.Now().Unix(),
			},
		},
	}

	type args struct {
		ctx   context.Context
		owner string
		req   *pb.GetTokensRequest
	}

	tests := []struct {
		name    string
		args    args
		wantLen int
	}{
		{
			name: "all tokens",
			args: args{
				ctx:   ctx,
				owner: owner,
				req:   &pb.GetTokensRequest{},
			},
			wantLen: 1,
		},
		{
			name: "all tokens including blacklisted",
			args: args{
				ctx:   ctx,
				owner: owner,
				req: &pb.GetTokensRequest{
					IncludeBlacklisted: true,
				},
			},
			wantLen: 2,
		},
		{
			name: "certain token",
			args: args{
				ctx:   ctx,
				owner: owner,
				req: &pb.GetTokensRequest{
					IdFilter:           []string{"token2"},
					IncludeBlacklisted: true,
				},
			},
			wantLen: 1,
		},
		{
			name: "all tokens another owner",
			args: args{
				ctx:   ctx,
				owner: "anotherOwner",
				req:   &pb.GetTokensRequest{},
			},
			wantLen: 0,
		},
	}

	for _, token := range tokens {
		_, err := s.CreateToken(ctx, owner, token)
		require.NoError(t, err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := make(map[string]*pb.Token)
			// Define a mock process function
			process := func(token *pb.Token) error {
				result[token.GetId()] = token
				return nil
			}

			// Call the GetTokens method
			err := s.GetTokens(tt.args.ctx, tt.args.owner, tt.args.req, process)
			require.NoError(t, err)
			require.Len(t, result, tt.wantLen)
		})
	}
}

func TestBlacklistTokens(t *testing.T) {
	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT*100)
	defer cancel()

	owner := "testOwner"
	tokens := []*pb.Token{
		{
			Id:       "token1",
			Owner:    owner,
			Version:  0,
			Name:     "name1",
			IssuedAt: time.Now().Unix(),
			ClientId: "client1",
		},
		{
			Id:       "token2",
			Owner:    owner,
			Version:  0,
			Name:     "name2",
			IssuedAt: time.Now().Unix(),
			ClientId: "client1",
		},
		{
			Id:       "token3",
			Owner:    owner,
			Version:  0,
			Name:     "name3",
			IssuedAt: time.Now().Unix(),
			ClientId: "client1",
		},
	}

	for _, token := range tokens {
		_, err := s.CreateToken(ctx, owner, token)
		require.NoError(t, err)
	}

	req := &pb.BlacklistTokensRequest{
		IdFilter: []string{"token1", "token2"},
	}

	resp, err := s.BlacklistTokens(ctx, owner, req)
	require.NoError(t, err)
	require.Equal(t, int64(2), resp.GetCount())

	blacklistedTokens := []*pb.Token{
		{
			Id:       "token1",
			Owner:    owner,
			Version:  0,
			Name:     "name1",
			IssuedAt: time.Now().Unix(),
			ClientId: "client1",
			Blacklisted: &pb.Token_BlackListed{
				Flag:      true,
				Timestamp: time.Now().Unix(),
			},
		},
		{
			Id:       "token2",
			Owner:    owner,
			Version:  0,
			Name:     "name2",
			IssuedAt: time.Now().Unix(),
			ClientId: "client1",
			Blacklisted: &pb.Token_BlackListed{
				Flag:      true,
				Timestamp: time.Now().Unix(),
			},
		},
	}

	for _, token := range blacklistedTokens {
		storedToken := make(map[string]*pb.Token)
		process := func(token *pb.Token) error {
			storedToken[token.GetId()] = token
			return nil
		}

		err := s.GetTokens(ctx, owner, &pb.GetTokensRequest{
			IdFilter:           []string{token.GetId()},
			IncludeBlacklisted: true,
		}, process)
		require.NoError(t, err)
		require.NotNil(t, storedToken)
		require.True(t, storedToken[token.GetId()].GetBlacklisted().GetFlag())
		require.Greater(t, storedToken[token.GetId()].GetBlacklisted().GetTimestamp(), int64(0))
	}
}

func TestDeleteTokens(t *testing.T) {
	s, cleanUpStore := test.NewMongoStore(t)
	defer cleanUpStore()

	ctx, cancel := context.WithTimeout(context.Background(), config.TEST_TIMEOUT*100)
	defer cancel()

	owner := "testOwner"
	tokens := []*pb.Token{
		{
			Id:         "token1",
			Owner:      owner,
			Version:    0,
			Name:       "name1",
			IssuedAt:   time.Now().Unix(),
			ClientId:   "client1",
			Expiration: time.Now().Add(time.Minute * 10).Unix(),
			Blacklisted: &pb.Token_BlackListed{
				Flag:      true,
				Timestamp: time.Now().Unix(),
			},
		},
		{
			Id:         "token2",
			Owner:      owner,
			Version:    0,
			Name:       "name2",
			IssuedAt:   time.Now().Unix(),
			ClientId:   "client1",
			Expiration: time.Now().Add(time.Minute * 10).Unix(),
			Blacklisted: &pb.Token_BlackListed{
				Flag:      true,
				Timestamp: time.Now().Add(time.Minute).Unix(),
			},
		},
		{
			Id:       "token3",
			Owner:    owner,
			Version:  0,
			Name:     "name3",
			IssuedAt: time.Now().Unix(),
			ClientId: "client1",
		},
	}

	for _, token := range tokens {
		_, err := s.CreateToken(ctx, owner, token)
		require.NoError(t, err)
	}

	err := s.DeleteTokens(ctx, time.Now().Add(time.Hour))
	require.NoError(t, err)

	remainingTokens := []*pb.Token{
		{
			Id:       "token3",
			Owner:    owner,
			Version:  0,
			Name:     "name3",
			IssuedAt: time.Now().Unix(),
			ClientId: "client1",
		},
	}

	result := make(map[string]*pb.Token)
	process := func(token *pb.Token) error {
		result[token.GetId()] = token
		return nil
	}

	err = s.GetTokens(ctx, owner, &pb.GetTokensRequest{
		IncludeBlacklisted: true,
	}, process)
	require.NoError(t, err)
	require.Len(t, result, len(remainingTokens))
	for _, token := range remainingTokens {
		require.Contains(t, result, token.GetId())
	}
}
