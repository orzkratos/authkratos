package ratekratoslimits_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/go-redis/redis_rate/v10"
	"github.com/google/uuid"
	"github.com/orzkratos/authkratos"
	"github.com/orzkratos/authkratos/authkratosroutes"
	"github.com/orzkratos/authkratos/internal/somestub"
	"github.com/orzkratos/authkratos/internal/utils"
	"github.com/orzkratos/authkratos/ratekratoslimits"
	"github.com/orzkratos/zapkratos"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/yyle88/must"
	"github.com/yyle88/rese"
	"github.com/yyle88/zaplog"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

var (
	httpPort string // Dynamic HTTP port // 动态分配的 HTTP 端口
	grpcPort string // Dynamic gRPC port // 动态分配的 gRPC 端口
)

// someStubService implements SomeStub service to test rate limiting
// someStubService 实现 SomeStub 服务以测试速率限制
type someStubService struct {
	somestub.UnimplementedSomeStubServer
}

func (s *someStubService) SelectSomething(ctx context.Context, req *wrapperspb.StringValue) (*wrapperspb.StringValue, error) {
	return wrapperspb.String(req.GetValue()), nil
}

func (s *someStubService) CreateSomething(ctx context.Context, req *wrapperspb.StringValue) (*wrapperspb.StringValue, error) {
	return wrapperspb.String("created:" + req.GetValue()), nil
}

func (s *someStubService) UpdateSomething(ctx context.Context, req *wrapperspb.StringValue) (*wrapperspb.StringValue, error) {
	return wrapperspb.String("updated:" + req.GetValue()), nil
}

func TestMain(m *testing.M) {
	authkratos.SetDebugMode(true)

	// Start miniredis
	// 启动 miniredis
	rdm := rese.P1(miniredis.Run())
	defer rdm.Close()

	// Create Redis client
	// 创建 Redis 客户端
	redisClient := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:        []string{rdm.Addr()},
		PoolSize:     10,
		MinIdleConns: 10,
	})
	must.Done(redisClient.Ping(context.Background()).Err())
	defer rese.F0(redisClient.Close)

	// Create rate limiter with 5 requests each second
	// 创建速率限制器，限制每秒 5 个请求
	rateLimiter := redis_rate.NewLimiter(redisClient)
	rateLimit := &redis_rate.Limit{
		Rate:   5,
		Burst:  5,
		Period: time.Second,
	}

	// Create logger
	// 创建 logger
	zapKratos := zapkratos.NewZapKratos(zaplog.LOGGER, zapkratos.NewOptions())

	// Create route scope - EXCLUDE SelectSomething from rate limiting
	// 创建路由范围 - 将 SelectSomething 排除在速率限制之外
	routeScope := authkratosroutes.NewExclude(
		somestub.OperationSomeStubSelectSomething,
	)

	// Key extraction function - use a fixed key in testing
	// 键提取函数 - 测试时使用固定键
	keyFromCtx := func(ctx context.Context) (string, bool) {
		return "test-key", true
	}

	// Create rate limit config
	// 创建速率限制配置
	rateConfig := ratekratoslimits.NewConfig(routeScope, rateLimiter, rateLimit, keyFromCtx).
		WithDebugMode(true)

	// Create rate limit middleware
	// 创建速率限制中间件
	rateMiddleware := ratekratoslimits.NewMiddleware(rateConfig, zapKratos.GetLogger("RATE"))

	// Create HTTP server with dynamic port
	// 使用动态端口创建 HTTP 服务器
	httpSrv := http.NewServer(
		http.Address(":0"),
		http.Middleware(
			recovery.Recovery(),
			rateMiddleware,
		),
		http.Timeout(time.Minute),
	)
	httpPort = utils.ExtractPort(rese.P1(httpSrv.Endpoint()))

	// Create gRPC server with dynamic port
	// 使用动态端口创建 gRPC 服务器
	grpcSrv := grpc.NewServer(
		grpc.Address(":0"),
		grpc.Middleware(
			recovery.Recovery(),
			rateMiddleware,
		),
		grpc.Timeout(time.Minute),
	)
	grpcPort = utils.ExtractPort(rese.P1(grpcSrv.Endpoint()))

	// Create test service
	// 创建测试服务
	stubService := &someStubService{}
	somestub.RegisterSomeStubHTTPServer(httpSrv, stubService)
	somestub.RegisterSomeStubServer(grpcSrv, stubService)

	app := kratos.New(
		kratos.Name("test-rate-kratos-limits"),
		kratos.Server(httpSrv, grpcSrv),
	)

	// Start server in background
	// 后台启动服务器
	go func() {
		must.Done(app.Run())
	}()
	defer rese.F0(app.Stop)

	// Wait a short time to ensure the server has started
	// 等待片刻以确保服务器已启动
	time.Sleep(time.Millisecond * 200)

	zaplog.LOG.Info("Starting test servers with dynamic ports",
		zap.String("http_port", httpPort),
		zap.String("grpc_port", grpcPort),
	)

	m.Run()
}

func TestRateLimit_SelectSomething_NoLimit_HTTP(t *testing.T) {
	// Test excluded route with no rate limit
	// Operation should always succeed
	//
	// 测试被排除的路由，无速率限制
	// 操作应始终成功
	conn := rese.P1(http.NewClient(
		context.Background(),
		http.WithMiddleware(recovery.Recovery()),
		http.WithEndpoint("127.0.0.1:"+httpPort),
	))
	defer rese.F0(conn.Close)

	stubClient := somestub.NewSomeStubHTTPClient(conn)
	ctx := context.Background()

	// Test multiple times to ensure no rate limit
	// 多次测试确保无速率限制
	for i := 0; i < 20; i++ {
		message := uuid.New().String()
		resp, err := stubClient.SelectSomething(ctx, wrapperspb.String(message))
		require.NoError(t, err)
		require.Equal(t, message, resp.GetValue())
	}
}

func TestRateLimit_CreateSomething_Limited_HTTP(t *testing.T) {
	// Test rate-limited route (5 requests each second)
	// First 5 requests should pass, 6th should be blocked
	//
	// 测试速率限制的路由（每秒 5 个请求）
	// 前 5 个请求应该通过，第 6 个应该被拦截
	conn := rese.P1(http.NewClient(
		context.Background(),
		http.WithMiddleware(recovery.Recovery()),
		http.WithEndpoint("127.0.0.1:"+httpPort),
	))
	defer rese.F0(conn.Close)

	stubClient := somestub.NewSomeStubHTTPClient(conn)
	ctx := context.Background()

	// First 5 requests should succeed
	// 前 5 个请求应该成功
	for i := 0; i < 5; i++ {
		message := uuid.New().String()
		resp, err := stubClient.CreateSomething(ctx, wrapperspb.String(message))
		require.NoError(t, err)
		require.Equal(t, "created:"+message, resp.GetValue())
	}

	// 6th request should be blocked with rate limit
	// 第 6 个请求应因速率限制被阻断
	message := uuid.New().String()
	_, err := stubClient.CreateSomething(ctx, wrapperspb.String(message))
	require.Error(t, err)

	erk := errors.FromError(err)
	require.Equal(t, int32(429), erk.Code)
}

func TestRateLimit_SelectSomething_NoLimit_gRPC(t *testing.T) {
	// Test excluded route via gRPC with no rate limit
	// Operation should always succeed
	//
	// 通过 gRPC 测试被排除的路由，无速率限制
	// 操作应始终成功
	conn := rese.P1(grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("127.0.0.1:"+grpcPort),
		grpc.WithMiddleware(recovery.Recovery()),
	))
	defer rese.F0(conn.Close)

	stubClient := somestub.NewSomeStubClient(conn)
	ctx := context.Background()

	// Test multiple times to ensure no rate limit
	// 多次测试确保无速率限制
	for i := 0; i < 20; i++ {
		message := uuid.New().String()
		resp, err := stubClient.SelectSomething(ctx, wrapperspb.String(message))
		require.NoError(t, err)
		require.Equal(t, message, resp.GetValue())
	}
}
