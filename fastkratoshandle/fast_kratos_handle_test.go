package fastkratoshandle_test

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
	"github.com/orzkratos/authkratos/fastkratoshandle"
	"github.com/orzkratos/authkratos/internal/somestub"
	"github.com/orzkratos/authkratos/internal/utils"
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

// someStubService implements SomeStub service for timeout testing
// someStubService 实现 SomeStub 服务用于超时测试
type someStubService struct {
	somestub.UnimplementedSomeStubServer
}

// SelectSomething is a fast operation that completes quickly
// Tests routes with fast timeout
//
// SelectSomething 是快速完成的操作
// 测试快速超时的路由
func (s *someStubService) SelectSomething(ctx context.Context, req *wrapperspb.StringValue) (*wrapperspb.StringValue, error) {
	// Fast operation, completes immediately
	// 快速操作，立即完成
	return wrapperspb.String(req.GetValue()), nil
}

// CreateSomething simulates a slow operation
// Tests EXCLUDE mode where certain operations have longer timeout
//
// CreateSomething 模拟慢速操作
// 测试 EXCLUDE 模式，某些操作有更长的超时时间
func (s *someStubService) CreateSomething(ctx context.Context, req *wrapperspb.StringValue) (*wrapperspb.StringValue, error) {
	// Simulate slow operation
	// 模拟慢速操作
	select {
	case <-time.After(time.Millisecond * 500):
		return wrapperspb.String("created:" + req.GetValue()), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// UpdateSomething simulates a slow operation that is NOT excluded
// Tests timeout failure when operation exceeds fast timeout
//
// UpdateSomething 模拟慢速操作且未被排除
// 测试操作超过快速超时时的超时失败
func (s *someStubService) UpdateSomething(ctx context.Context, req *wrapperspb.StringValue) (*wrapperspb.StringValue, error) {
	// Simulate slow operation
	// 模拟慢速操作
	select {
	case <-time.After(time.Millisecond * 500):
		return wrapperspb.String("updated:" + req.GetValue()), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func TestMain(m *testing.M) {
	authkratos.SetDebugMode(true)

	// Create logger to show middleware logs
	// 创建 logger 以显示中间件日志
	zapKratos := zapkratos.NewZapKratos(zaplog.LOGGER, zapkratos.NewOptions())

	// Create route scope - EXCLUDE CreateSomething from fast timeout (give it longer timeout)
	// Other routes (Select/Update) will have fast timeout
	//
	// 创建路由范围 - 将 CreateSomething 排除在快速超时之外（给它更长的超时）
	// 其他路由（Select/Update）将有快速超时
	routeScope := authkratosroutes.NewExclude(
		somestub.OperationSomeStubCreateSomething,
	)

	// Create fast timeout config
	// Fast routes will timeout in 50ms, excluded routes keep default timeout
	//
	// 创建快速超时配置
	// 快速路由将在 50ms 后超时，排除的路由保持默认超时
	fastConfig := fastkratoshandle.NewConfig(routeScope, time.Millisecond*50).
		WithDebugMode(true)

	// Create fast timeout middleware
	// 创建快速超时中间件
	fastMiddleware := fastkratoshandle.NewMiddleware(fastConfig, zapKratos.GetLogger("FAST"))

	// Create HTTP server with dynamic port (port 0 = random available port)
	// 使用动态端口创建 HTTP 服务器（端口 0 表示随机可用端口）
	httpSrv := http.NewServer(
		http.Address(":0"),
		http.Middleware(
			recovery.Recovery(),
			fastMiddleware,
		),
		http.Timeout(time.Minute),
	)

	// Create gRPC server with dynamic port
	// 使用动态端口创建 gRPC 服务器
	grpcSrv := grpc.NewServer(
		grpc.Address(":0"),
		grpc.Middleware(
			recovery.Recovery(),
			fastMiddleware,
		),
		grpc.Timeout(time.Minute),
	)

	// Create test service to verify timeout middleware behavior
	// 创建测试服务以验证超时中间件行为
	stubService := &someStubService{}
	somestub.RegisterSomeStubHTTPServer(httpSrv, stubService)
	somestub.RegisterSomeStubServer(grpcSrv, stubService)

	app := kratos.New(
		kratos.Name("test-fast-kratos-handle"),
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

func TestFastHandle_SelectSomething_FastTimeout_HTTP(t *testing.T) {
	// Test fast operation with fast timeout
	// Operation completes immediately within 50ms timeout → success
	//
	// 测试快速操作与快速超时
	// 操作立即完成，在 50ms 超时内 → 快速执行成功
	conn := rese.P1(http.NewClient(
		context.Background(),
		http.WithMiddleware(recovery.Recovery()),
		http.WithEndpoint("127.0.0.1:"+httpPort),
	))
	defer rese.F0(conn.Close)

	stubClient := somestub.NewSomeStubHTTPClient(conn)
	ctx := context.Background()
	message := uuid.New().String()

	resp, err := stubClient.SelectSomething(ctx, wrapperspb.String(message))
	require.NoError(t, err)
	require.Equal(t, message, resp.GetValue())
}

func TestFastHandle_UpdateSomething_FastTimeout_HTTP(t *testing.T) {
	// Test slow operation that is NOT excluded from fast timeout
	// Operation takes 500ms but 50ms timeout triggers → timeout failure
	//
	// 测试未被排除的慢速操作
	// 操作需要 500ms 但 50ms 超时触发 → 执行 50ms 后超时失败
	conn := rese.P1(http.NewClient(
		context.Background(),
		http.WithMiddleware(recovery.Recovery()),
		http.WithEndpoint("127.0.0.1:"+httpPort),
	))
	defer rese.F0(conn.Close)

	stubClient := somestub.NewSomeStubHTTPClient(conn)
	ctx := context.Background()
	message := uuid.New().String()

	_, err := stubClient.UpdateSomething(ctx, wrapperspb.String(message))
	require.Error(t, err)

	erk := errors.FromError(err)
	require.Equal(t, int32(500), erk.Code)
	require.Equal(t, "context deadline exceeded", erk.Message)
}

func TestFastHandle_CreateSomething_SlowTimeout_HTTP(t *testing.T) {
	// Test excluded route with longer timeout (default server timeout)
	// Operation takes 500ms, excluded from fast timeout → success
	//
	// 测试被排除的路由，使用更长的超时（默认服务器超时）
	// 操作需要 500ms，被排除在快速超时外 → 执行 500ms 后成功完成
	conn := rese.P1(http.NewClient(
		context.Background(),
		http.WithMiddleware(recovery.Recovery()),
		http.WithEndpoint("127.0.0.1:"+httpPort),
	))
	defer rese.F0(conn.Close)

	stubClient := somestub.NewSomeStubHTTPClient(conn)
	ctx := context.Background()
	message := uuid.New().String()

	resp, err := stubClient.CreateSomething(ctx, wrapperspb.String(message))
	require.NoError(t, err)
	require.Equal(t, "created:"+message, resp.GetValue())
}

func TestFastHandle_CreateSomething_SlowTimeout_gRPC(t *testing.T) {
	// Test excluded route with longer timeout via gRPC
	// Operation takes 500ms, excluded from fast timeout → success
	//
	// 测试被排除的路由，使用更长的超时，通过 gRPC
	// 操作需要 500ms，被排除在快速超时外 → 执行 500ms 后成功完成
	conn := rese.P1(grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("127.0.0.1:"+grpcPort),
		grpc.WithMiddleware(recovery.Recovery()),
	))
	defer rese.F0(conn.Close)

	stubClient := somestub.NewSomeStubClient(conn)
	ctx := context.Background()
	message := uuid.New().String()

	resp, err := stubClient.CreateSomething(ctx, wrapperspb.String(message))
	require.NoError(t, err)
	require.Equal(t, "created:"+message, resp.GetValue())
}
