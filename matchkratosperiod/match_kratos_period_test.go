package matchkratosperiod_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/google/uuid"
	"github.com/orzkratos/authkratos"
	"github.com/orzkratos/authkratos/authkratosroutes"
	"github.com/orzkratos/authkratos/internal/somestub"
	"github.com/orzkratos/authkratos/internal/utils"
	"github.com/orzkratos/authkratos/matchkratosperiod"
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

type sampleMark struct{}

// someStubService implements SomeStub service for periodic sampling testing
// someStubService 实现 SomeStub 服务用于周期采样测试
type someStubService struct {
	somestub.UnimplementedSomeStubServer
}

// SelectSomething handles query operations without sampling
// Tests EXCLUDE mode where certain operations are excluded from sampling
//
// SelectSomething 处理查询操作，不受采样影响
// 测试 EXCLUDE 模式，某些操作排除在采样之外
func (s *someStubService) SelectSomething(ctx context.Context, req *wrapperspb.StringValue) (*wrapperspb.StringValue, error) {
	// Check if sampled marker is set in context
	// 检查 context 中是否设置了采样标记
	sampled := ctx.Value(sampleMark{})
	if sampled != nil {
		return wrapperspb.String("sampled:" + req.GetValue()), nil
	}
	return wrapperspb.String(req.GetValue()), nil
}

// CreateSomething handles operations subject to periodic sampling
// Tests periodic sampling control (every Nth request is sampled)
//
// CreateSomething 处理受周期采样影响的操作
// 测试周期采样控制（每 N 个请求采样一次）
func (s *someStubService) CreateSomething(ctx context.Context, req *wrapperspb.StringValue) (*wrapperspb.StringValue, error) {
	// Check if sampled marker is set in context
	// 检查 context 中是否设置了采样标记
	sampled := ctx.Value(sampleMark{})
	if sampled != nil {
		return wrapperspb.String("sampled:created:" + req.GetValue()), nil
	}
	return wrapperspb.String("created:" + req.GetValue()), nil
}

// UpdateSomething handles operations subject to periodic sampling
// Tests periodic sampling control
//
// UpdateSomething 处理受周期采样影响的操作
// 测试周期采样控制
func (s *someStubService) UpdateSomething(ctx context.Context, req *wrapperspb.StringValue) (*wrapperspb.StringValue, error) {
	// Check if sampled marker is set in context
	// 检查 context 中是否设置了采样标记
	sampled := ctx.Value(sampleMark{})
	if sampled != nil {
		return wrapperspb.String("sampled:updated:" + req.GetValue()), nil
	}
	return wrapperspb.String("updated:" + req.GetValue()), nil
}

func TestMain(m *testing.M) {
	authkratos.SetDebugMode(true)

	// Create logger to show middleware logs
	// 创建 logger 以显示中间件日志
	zapKratos := zapkratos.NewZapKratos(zaplog.LOGGER, zapkratos.NewOptions())

	// Create route scope - EXCLUDE SelectSomething from periodic sampling
	// Create/Update operations will have periodic sampling (every 3rd request is sampled)
	//
	// 创建路由范围 - 将 SelectSomething 排除在周期采样之外
	// Create/Update 操作将受周期采样影响（每 3 个请求采样一次）
	routeScope := authkratosroutes.NewExclude(
		somestub.OperationSomeStubSelectSomething,
	)

	// Create periodic sampling config with N=3 (sample every 3rd request)
	// WithMatchFirst(true) means first request is immediately sampled
	//
	// 创建周期采样配置，N=3（每 3 个请求采样一次）
	// WithMatchFirst(true) 表示第一个请求立即被采样
	periodConfig := matchkratosperiod.NewConfig(routeScope, 3).
		WithMatchFirst(true).
		WithDebugMode(true)

	// Create match function for periodic sampling
	// 创建周期采样的匹配函数
	matchFunc := matchkratosperiod.NewMatchFunc(periodConfig, zapKratos.GetLogger("PERIOD"))

	// Create middleware that sets a marker in context when matched
	// This simulates what logging/tracing middleware would do
	//
	// 创建一个在命中时设置标记的中间件
	// 模拟日志/追踪中间件的行为
	samplingMiddleware := func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// Set sampled marker in context
			// 在 context 中设置采样标记
			ctx = context.WithValue(ctx, sampleMark{}, true)
			return handler(ctx, req)
		}
	}

	// Combine sampling middleware with match function
	// Only matched requests will have the marker set
	//
	// 组合采样中间件和匹配函数
	// 只有命中的请求会设置标记
	periodMiddleware := selector.Server(samplingMiddleware).Match(matchFunc).Build()

	// Create HTTP server with dynamic port (port 0 = random available port)
	// 使用动态端口创建 HTTP 服务器（端口 0 表示随机可用端口）
	httpSrv := http.NewServer(
		http.Address(":0"),
		http.Middleware(
			recovery.Recovery(),
			periodMiddleware,
		),
		http.Timeout(time.Minute),
	)

	// Create gRPC server with dynamic port
	// 使用动态端口创建 gRPC 服务器
	grpcSrv := grpc.NewServer(
		grpc.Address(":0"),
		grpc.Middleware(
			recovery.Recovery(),
			periodMiddleware,
		),
		grpc.Timeout(time.Minute),
	)

	// Create test service to verify periodic sampling middleware behavior
	// 创建测试服务以验证周期采样中间件行为
	stubService := &someStubService{}
	somestub.RegisterSomeStubHTTPServer(httpSrv, stubService)
	somestub.RegisterSomeStubServer(grpcSrv, stubService)

	app := kratos.New(
		kratos.Name("test-match-kratos-period"),
		kratos.Server(httpSrv, grpcSrv),
	)

	// Start server in background
	// 后台启动服务器
	go func() {
		must.Done(app.Run())
	}()
	defer rese.F0(app.Stop)

	// Wait for server to start and extract actual listening ports
	// 等待服务器启动并获取实际监听端口
	time.Sleep(time.Millisecond * 200)

	// Extract actual port from server endpoint
	// 从服务器端点提取实际端口
	httpPort = utils.ExtractPort(rese.P1(httpSrv.Endpoint()))
	grpcPort = utils.ExtractPort(rese.P1(grpcSrv.Endpoint()))

	zaplog.LOG.Info("Starting test servers with dynamic ports",
		zap.String("http_port", httpPort),
		zap.String("grpc_port", grpcPort),
	)

	m.Run()
}

func TestMatchPeriod_SelectSomething_NeverSampled_HTTP(t *testing.T) {
	// Test excluded route that is never sampled
	// All requests pass through but middleware is never executed
	//
	// 测试被排除的路由，永不被采样
	// 所有请求都通过但中间件永不执行
	conn := rese.P1(http.NewClient(
		context.Background(),
		http.WithMiddleware(recovery.Recovery()),
		http.WithEndpoint("127.0.0.1:"+httpPort),
	))
	defer rese.F0(conn.Close)

	stubClient := somestub.NewSomeStubHTTPClient(conn)
	ctx := context.Background()

	// Try multiple times to ensure it's never sampled
	// 多次尝试确保永不被采样
	for i := 0; i < 10; i++ {
		message := uuid.New().String()
		resp, err := stubClient.SelectSomething(ctx, wrapperspb.String(message))
		require.NoError(t, err)
		// Should NOT have "sampled:" prefix
		// 不应该有 "sampled:" 前缀
		require.Equal(t, message, resp.GetValue())
	}
}

func TestMatchPeriod_CreateSomething_PeriodicSampling_HTTP(t *testing.T) {
	// Test periodic sampling (every 3rd request is sampled)
	// First request is sampled (matchFirst=true), then every 3rd request
	// Pattern: SAMPLED, NOT, NOT, SAMPLED, NOT, NOT, SAMPLED...
	//
	// 测试周期采样（每 3 个请求采样一次）
	// 第一个请求被采样（matchFirst=true），然后每 3 个请求采样一次
	// 模式：采样、不采样、不采样、采样、不采样、不采样、采样...
	conn := rese.P1(http.NewClient(
		context.Background(),
		http.WithMiddleware(recovery.Recovery()),
		http.WithEndpoint("127.0.0.1:"+httpPort),
	))
	defer rese.F0(conn.Close)

	stubClient := somestub.NewSomeStubHTTPClient(conn)
	ctx := context.Background()

	// Test pattern: 1st sampled, 2nd not, 3rd not, 4th sampled...
	// 测试模式：第 1 个被采样，第 2 个不采样，第 3 个不采样，第 4 个被采样...
	for i := 0; i < 3; i++ {
		// 1st request should be sampled (matchFirst or every 3rd)
		// 第 1 个请求应该被采样（首次匹配或每 3 个）
		message := uuid.New().String()
		resp, err := stubClient.CreateSomething(ctx, wrapperspb.String(message))
		require.NoError(t, err)
		require.Equal(t, "sampled:created:"+message, resp.GetValue())

		// 2nd request should NOT be sampled
		// 第 2 个请求不应该被采样
		message = uuid.New().String()
		resp, err = stubClient.CreateSomething(ctx, wrapperspb.String(message))
		require.NoError(t, err)
		require.Equal(t, "created:"+message, resp.GetValue())

		// 3rd request should NOT be sampled
		// 第 3 个请求不应该被采样
		message = uuid.New().String()
		resp, err = stubClient.CreateSomething(ctx, wrapperspb.String(message))
		require.NoError(t, err)
		require.Equal(t, "created:"+message, resp.GetValue())
	}
}

func TestMatchPeriod_SelectSomething_NeverSampled_gRPC(t *testing.T) {
	// Test excluded route via gRPC
	// All requests pass through but middleware is never executed
	//
	// 通过 gRPC 测试被排除的路由
	// 所有请求都通过但中间件永不执行
	conn := rese.P1(grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("127.0.0.1:"+grpcPort),
		grpc.WithMiddleware(recovery.Recovery()),
	))
	defer rese.F0(conn.Close)

	stubClient := somestub.NewSomeStubClient(conn)
	ctx := context.Background()

	// Try multiple times to ensure it's never sampled
	// 多次尝试确保永不被采样
	for i := 0; i < 10; i++ {
		message := uuid.New().String()
		resp, err := stubClient.SelectSomething(ctx, wrapperspb.String(message))
		require.NoError(t, err)
		// Should NOT have "sampled:" prefix
		// 不应该有 "sampled:" 前缀
		require.Equal(t, message, resp.GetValue())
	}
}
