package grpcserver

import (
	"context"
	"net"
	"strings"
	"testing"

	"github.com/glebb1331/shortener-practicum/internal/audit"
	"github.com/glebb1331/shortener-practicum/internal/logger"
	"github.com/glebb1331/shortener-practicum/internal/middleware"
	"github.com/glebb1331/shortener-practicum/internal/storage"
	"github.com/glebb1331/shortener-practicum/internal/usecase"
	pb "github.com/glebb1331/shortener-practicum/pkg/shortenerpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"
)

func init() {
	_ = logger.Initialize("error")
}

func startTestServer(t *testing.T) (pb.ShortenerServiceClient, func()) {
	t.Helper()

	store := storage.NewMemoryStorage()
	svc := usecase.NewURLService(store, "http://localhost:8080")
	auditSvc := audit.NewAuditService()

	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer(grpc.UnaryInterceptor(AuthInterceptor()))
	pb.RegisterShortenerServiceServer(srv, NewShortenerServer(svc, auditSvc))

	go func() {
		_ = srv.Serve(lis)
	}()

	dialer := func(context.Context, string) (net.Conn, error) { return lis.Dial() }
	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	client := pb.NewShortenerServiceClient(conn)
	return client, func() {
		conn.Close()
		srv.Stop()
	}
}

func TestShortenURL_NewUserGetsToken(t *testing.T) {
	client, stop := startTestServer(t)
	defer stop()

	var headers metadata.MD
	resp, err := client.ShortenURL(
		context.Background(),
		&pb.URLShortenRequest{Url: "https://example.com"},
		grpc.Header(&headers),
	)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(resp.GetResult(), "http://localhost:8080/"))

	tokens := headers.Get(authMetadataKey)
	require.NotEmpty(t, tokens, "interceptor must issue token for new user")
	_, ok := middleware.ParseToken(tokens[0])
	assert.True(t, ok)
}

func TestShortenURL_EmptyURLReturnsInvalidArgument(t *testing.T) {
	client, stop := startTestServer(t)
	defer stop()

	_, err := client.ShortenURL(context.Background(), &pb.URLShortenRequest{Url: "  "})
	require.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestExpandURL_RoundTrip(t *testing.T) {
	client, stop := startTestServer(t)
	defer stop()

	var headers metadata.MD
	shortResp, err := client.ShortenURL(
		context.Background(),
		&pb.URLShortenRequest{Url: "https://example.org"},
		grpc.Header(&headers),
	)
	require.NoError(t, err)

	id := shortResp.GetResult()[len("http://localhost:8080/"):]

	expandResp, err := client.ExpandURL(context.Background(), &pb.URLExpandRequest{Id: id})
	require.NoError(t, err)
	assert.Equal(t, "https://example.org", expandResp.GetResult())
}

func TestExpandURL_NotFound(t *testing.T) {
	client, stop := startTestServer(t)
	defer stop()

	_, err := client.ExpandURL(context.Background(), &pb.URLExpandRequest{Id: "missingid"})
	require.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestListUserURLs_ReturnsUserURLs(t *testing.T) {
	client, stop := startTestServer(t)
	defer stop()

	userID, err := middleware.GenerateUserID()
	require.NoError(t, err)
	token, err := middleware.CreateToken(userID)
	require.NoError(t, err)

	ctx := metadata.AppendToOutgoingContext(context.Background(), authMetadataKey, token)

	_, err = client.ShortenURL(ctx, &pb.URLShortenRequest{Url: "https://one.example"})
	require.NoError(t, err)
	_, err = client.ShortenURL(ctx, &pb.URLShortenRequest{Url: "https://two.example"})
	require.NoError(t, err)

	listResp, err := client.ListUserURLs(ctx, &emptypb.Empty{})
	require.NoError(t, err)
	require.Len(t, listResp.GetUrl(), 2)

	got := map[string]bool{}
	for _, u := range listResp.GetUrl() {
		got[u.GetOriginalUrl()] = true
	}
	assert.True(t, got["https://one.example"])
	assert.True(t, got["https://two.example"])
}

func TestShortenURL_ExistingURLReturnsAlreadyExists(t *testing.T) {
	client, stop := startTestServer(t)
	defer stop()

	userID, err := middleware.GenerateUserID()
	require.NoError(t, err)
	token, err := middleware.CreateToken(userID)
	require.NoError(t, err)

	ctx := metadata.AppendToOutgoingContext(context.Background(), authMetadataKey, token)

	_, err = client.ShortenURL(ctx, &pb.URLShortenRequest{Url: "https://duplicate.example"})
	require.NoError(t, err)

	_, err = client.ShortenURL(ctx, &pb.URLShortenRequest{Url: "https://duplicate.example"})
	require.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.AlreadyExists, st.Code())
}

func TestExpandURL_EmptyID(t *testing.T) {
	client, stop := startTestServer(t)
	defer stop()

	_, err := client.ExpandURL(context.Background(), &pb.URLExpandRequest{Id: ""})
	require.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestAuthInterceptor_BearerPrefixAccepted(t *testing.T) {
	client, stop := startTestServer(t)
	defer stop()

	userID, err := middleware.GenerateUserID()
	require.NoError(t, err)
	token, err := middleware.CreateToken(userID)
	require.NoError(t, err)

	ctx := metadata.AppendToOutgoingContext(context.Background(), authMetadataKey, "Bearer "+token)
	_, err = client.ShortenURL(ctx, &pb.URLShortenRequest{Url: "https://bearer.example"})
	require.NoError(t, err)
}

func TestAuthInterceptor_InvalidTokenRejected(t *testing.T) {
	client, stop := startTestServer(t)
	defer stop()

	ctx := metadata.AppendToOutgoingContext(context.Background(), authMetadataKey, "garbage")
	_, err := client.ShortenURL(ctx, &pb.URLShortenRequest{Url: "https://x.example"})
	require.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Unauthenticated, st.Code())
}
