package passkratosrandom_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/google/uuid"
	"github.com/orzkratos/authkratos"
	"github.com/orzkratos/authkratos/authkratosroutes"
	"github.com/orzkratos/authkratos/internal/somestub"
	"github.com/orzkratos/authkratos/internal/utils"
	"github.com/orzkratos/authkratos/passkratosrandom"
	"github.com/orzkratos/zapkratos"
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

// someStubService implements SomeStub service for random pass rate testing
// someStubService 实现 SomeStub 服务用于随机放行率测试
type someStubService struct {
	somestub.UnimplementedSomeStubServer
}

// SelectSomething handles query operations without random blocking
// Tests EXCLUDE mode where certain operations are excluded from random blocking
//
// SelectSomething 处理查询操作，不受随机阻断影响
// 测试 EXCLUDE 模式，某些操作排除在随机阻断之外
func (s *someStubService) SelectSomething(ctx context.Context, req *wrapperspb.StringValue) (*wrapperspb.StringValue, error) {
	return wrapperspb.String(req.GetValue()), nil
}

// CreateSomething handles operations subject to random blocking
// Tests random pass rate control
//
// CreateSomething 处理受随机阻断影响的操作
// 测试随机放行率控制
func (s *someStubService) CreateSomething(ctx context.Context, req *wrapperspb.StringValue) (*wrapperspb.StringValue, error) {
	return wrapperspb.String("created:" + req.GetValue()), nil
}

// UpdateSomething handles operations subject to random blocking
// Tests random pass rate control
//
// UpdateSomething 处理受随机阻断影响的操作
// 测试随机放行率控制
func (s *someStubService) UpdateSomething(ctx context.Context, req *wrapperspb.StringValue) (*wrapperspb.StringValue, error) {
	return wrapperspb.String("updated:" + req.GetValue()), nil
}

func TestMain(m *testing.M) {
	authkratos.SetDebugMode(true)

	// Create logger to show middleware logs
	// 创建 logger 以显示中间件日志
	zapKratos := zapkratos.NewZapKratos(zaplog.LOGGER, zapkratos.NewOptions())

	// Create route scope - EXCLUDE SelectSomething from random blocking
	// Create/Update operations will have random blocking
	//
	// 创建路由范围 - 将 SelectSomething 排除在随机阻断之外
	// Create/Update 操作将受随机阻断影响
	routeScope := authkratosroutes.NewExclude(
		somestub.OperationSomeStubSelectSomething,
	)

	// Create random pass rate config with 60% pass rate
	// 60% of requests should pass, 40% should be blocked
	//
	// 创建随机放行率配置，放行率 60%
	// 60% 的请求应该通过，40% 的请求应该被拦截
	randomConfig := passkratosrandom.NewConfig(routeScope, 0.6).
		WithDebugMode(true)

	// Create random pass rate middleware
	// 创建随机放行率中间件
	randomMiddleware := passkratosrandom.NewMiddleware(randomConfig, zapKratos.GetLogger("RAND"))

	// Create HTTP server with dynamic port (port 0 = random available port)
	// 使用动态端口创建 HTTP 服务器（端口 0 表示随机可用端口）
	httpSrv := http.NewServer(
		http.Address(":0"),
		http.Middleware(
			recovery.Recovery(),
			randomMiddleware,
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
			randomMiddleware,
		),
		grpc.Timeout(time.Minute),
	)
	grpcPort = utils.ExtractPort(rese.P1(grpcSrv.Endpoint()))

	// Create test service to verify random pass rate middleware behavior
	// 创建测试服务以验证随机放行率中间件行为
	stubService := &someStubService{}
	somestub.RegisterSomeStubHTTPServer(httpSrv, stubService)
	somestub.RegisterSomeStubServer(grpcSrv, stubService)

	app := kratos.New(
		kratos.Name("test-pass-kratos-random"),
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

func TestPassRandom_SelectSomething_AlwaysPass_HTTP(t *testing.T) {
	// Test excluded route that is never blocked
	// Operation should always succeed regardless of random rate
	//
	// 测试被排除的路由，永不被拦截
	// 操作应始终成功，不受随机率影响
	conn := rese.P1(http.NewClient(
		context.Background(),
		http.WithMiddleware(recovery.Recovery()),
		http.WithEndpoint("127.0.0.1:"+httpPort),
	))
	defer rese.F0(conn.Close)

	stubClient := somestub.NewSomeStubHTTPClient(conn)
	ctx := context.Background()

	// Try multiple times to ensure it always passes
	// 多次尝试确保始终通过
	for i := 0; i < 10; i++ {
		message := uuid.New().String()
		resp, err := stubClient.SelectSomething(ctx, wrapperspb.String(message))
		require.NoError(t, err)
		require.Equal(t, message, resp.GetValue())
	}
}

func TestPassRandom_CreateSomething_RandomPass_HTTP(t *testing.T) {
	// Test random pass rate (60% pass rate configured)
	// Loop until we get at least one success and one failure
	//
	// 测试随机放行率（配置为 60% 放行率）
	// 循环直到至少出现一次成功和一次失败
	conn := rese.P1(http.NewClient(
		context.Background(),
		http.WithMiddleware(recovery.Recovery()),
		http.WithEndpoint("127.0.0.1:"+httpPort),
	))
	defer rese.F0(conn.Close)

	stubClient := somestub.NewSomeStubHTTPClient(conn)
	ctx := context.Background()

	hasSuccess := false
	hasFailure := false

	// Loop until we see both success and failure
	// 循环直到同时出现成功和失败
	for !hasSuccess || !hasFailure {
		message := uuid.New().String()
		resp, err := stubClient.CreateSomething(ctx, wrapperspb.String(message))
		if err != nil {
			hasFailure = true
			erk := errors.FromError(err)
			require.Equal(t, int32(503), erk.Code)
			require.Equal(t, "RANDOM_RATE_UNAVAILABLE", erk.Reason)
			continue
		}
		hasSuccess = true
		require.Equal(t, "created:"+message, resp.GetValue())
	}
}

func TestPassRandom_SelectSomething_AlwaysPass_gRPC(t *testing.T) {
	// Test excluded route via gRPC
	// Operation should always succeed
	//
	// 通过 gRPC 测试被排除的路由
	// 操作应始终成功
	conn := rese.P1(grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("127.0.0.1:"+grpcPort),
		grpc.WithMiddleware(recovery.Recovery()),
	))
	defer rese.F0(conn.Close)

	stubClient := somestub.NewSomeStubClient(conn)
	ctx := context.Background()

	// Try multiple times to ensure it always passes
	// 多次尝试确保始终通过
	for i := 0; i < 10; i++ {
		message := uuid.New().String()
		resp, err := stubClient.SelectSomething(ctx, wrapperspb.String(message))
		require.NoError(t, err)
		require.Equal(t, message, resp.GetValue())
	}
}
