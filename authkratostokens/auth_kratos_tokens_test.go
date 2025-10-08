package authkratostokens_test

import (
	"context"
	nethttp "net/http"
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
	"github.com/orzkratos/authkratos/authkratostokens"
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

const (
	testUsername = "kratos-username-001"
	testPassword = "secret-password-123"
	invalidToken = "invalid-token-99999"
)

var (
	httpPort string // Dynamic HTTP port // 动态分配的 HTTP 端口
	grpcPort string // Dynamic gRPC port // 动态分配的 gRPC 端口
)

// someStubService implements SomeStub service for auth middleware testing
// someStubService 实现 SomeStub 服务用于认证中间件测试
type someStubService struct {
	somestub.UnimplementedSomeStubServer
}

// SelectSomething handles query operations without authentication requirement
// Tests EXCLUDE mode where certain operations are explicitly excluded from auth
//
// SelectSomething 处理查询操作，不需要认证
// 测试 EXCLUDE 模式，某些操作明确排除认证
func (s *someStubService) SelectSomething(ctx context.Context, req *wrapperspb.StringValue) (*wrapperspb.StringValue, error) {
	// Public endpoint, just echo the request
	// 公开端点，直接回显请求
	return wrapperspb.String(req.GetValue()), nil
}

// CreateSomething handles write operations that require authentication
// Returns guest info from context to verify context injection works
// Tests INCLUDE mode where operations require authentication
//
// CreateSomething 处理需要认证的写操作
// 从 context 返回用户信息以验证上下文注入
// 测试 INCLUDE 模式，操作需要认证
func (s *someStubService) CreateSomething(ctx context.Context, req *wrapperspb.StringValue) (*wrapperspb.StringValue, error) {
	// Extract username from context to prove auth middleware injected it
	// 从 context 提取 username 以证明认证中间件已注入
	username, ok := authkratostokens.GetUsername(ctx)
	if !ok {
		username = "unknown"
	}

	// Return format: "created:<input>,guest:<username>"
	// 返回格式: "created:<输入>,guest:<用户名>"
	return wrapperspb.String("created:" + req.GetValue() + ",guest:" + username), nil
}

// UpdateSomething handles write operations that require authentication
// Returns guest info from context to verify context injection works
// Tests INCLUDE mode where operations require authentication
//
// UpdateSomething 处理需要认证的写操作
// 从 context 返回用户信息以验证上下文注入
// 测试 INCLUDE 模式，操作需要认证
func (s *someStubService) UpdateSomething(ctx context.Context, req *wrapperspb.StringValue) (*wrapperspb.StringValue, error) {
	// Extract username from context to prove auth middleware injected it
	// 从 context 提取 username 以证明认证中间件已注入
	username, ok := authkratostokens.GetUsername(ctx)
	if !ok {
		username = "unknown"
	}

	// Return format: "updated:<input>,guest:<username>"
	// 返回格式: "updated:<输入>,guest:<用户名>"
	return wrapperspb.String("updated:" + req.GetValue() + ",guest:" + username), nil
}

func TestMain(m *testing.M) {
	authkratos.SetDebugMode(true)

	// Create logger to show auth middleware logs
	// 创建 logger 以显示认证中间件日志
	zapKratos := zapkratos.NewZapKratos(zaplog.LOGGER, zapkratos.NewOptions())

	// Create username-token map for authentication
	// 创建用户名-令牌映射用于认证
	usernameToTokenMap := map[string]string{
		testUsername: testPassword,
	}

	// Create route scope - protect write operations (Create/Update) but not query operations (Select)
	// 创建路由范围 - 保护写操作（Create/Update）但不保护查询操作（Select）
	routeScope := authkratosroutes.NewInclude(
		somestub.OperationSomeStubCreateSomething,
		somestub.OperationSomeStubUpdateSomething,
	)

	// Create auth config with username-token map
	// 使用用户名-令牌映射创建认证配置
	authConfig := authkratostokens.NewConfig(routeScope, usernameToTokenMap).
		WithFieldName("Authorization").
		WithDebugMode(true)

	// Create auth middleware
	// 创建认证中间件
	authMiddleware := authkratostokens.NewMiddleware(authConfig, zapKratos.GetLogger("AUTH"))

	// Create HTTP server with dynamic port (port 0 = random available port)
	// 使用动态端口创建 HTTP 服务器（端口 0 表示随机可用端口）
	httpSrv := http.NewServer(
		http.Address(":0"),
		http.Middleware(
			recovery.Recovery(),
			authMiddleware,
		),
		http.Timeout(time.Minute),
	)

	// Create gRPC server with dynamic port
	// 使用动态端口创建 gRPC 服务器
	grpcSrv := grpc.NewServer(
		grpc.Address(":0"),
		grpc.Middleware(
			recovery.Recovery(),
			authMiddleware,
		),
		grpc.Timeout(time.Minute),
	)

	// Create test service to verify auth middleware behavior
	// 创建测试服务以验证认证中间件行为
	stubService := &someStubService{}
	somestub.RegisterSomeStubHTTPServer(httpSrv, stubService)
	somestub.RegisterSomeStubServer(grpcSrv, stubService)

	app := kratos.New(
		kratos.Name("test-auth-kratos-tokens"),
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

func TestAuthTokens_SelectSomething_NoAuth_HTTP(t *testing.T) {
	// Test public endpoint that does not require authentication (EXCLUDE mode)
	// 测试不需要认证的公开端点（EXCLUDE 模式）
	conn := rese.P1(http.NewClient(
		context.Background(),
		http.WithMiddleware(recovery.Recovery()),
		http.WithEndpoint("127.0.0.1:"+httpPort),
	))
	defer rese.F0(conn.Close)

	stubClient := somestub.NewSomeStubHTTPClient(conn)
	ctx := context.Background()
	message := uuid.New().String()

	// SelectSomething should work without token
	// SelectSomething 应该在没有令牌时也能工作
	resp, err := stubClient.SelectSomething(ctx, wrapperspb.String(message))
	require.NoError(t, err)
	require.Equal(t, message, resp.GetValue())
}

func TestAuthTokens_CreateSomething_SimpleToken_HTTP(t *testing.T) {
	// Test protected endpoint with simple token format
	// 测试使用简单令牌格式的受保护端点
	conn := rese.P1(http.NewClient(
		context.Background(),
		http.WithMiddleware(recovery.Recovery()),
		http.WithEndpoint("127.0.0.1:"+httpPort),
	))
	defer rese.F0(conn.Close)

	stubClient := somestub.NewSomeStubHTTPClient(conn)
	ctx := context.Background()
	message := uuid.New().String()

	// Set simple token in header
	// 设置简单令牌到请求头
	headers := nethttp.Header{}
	headers.Set("Authorization", testPassword)

	// CreateSomething requires auth and should return guest info from context
	// CreateSomething 需要认证并应从 context 返回用户信息
	resp, err := stubClient.CreateSomething(ctx, wrapperspb.String(message), http.Header(&headers))
	require.NoError(t, err)
	require.Equal(t, "created:"+message+",guest:"+testUsername, resp.GetValue())
}

func TestAuthTokens_CreateSomething_BearerToken_HTTP(t *testing.T) {
	// Test protected endpoint with Bearer token format
	// 测试使用 Bearer 令牌格式的受保护端点
	conn := rese.P1(http.NewClient(
		context.Background(),
		http.WithMiddleware(recovery.Recovery()),
		http.WithEndpoint("127.0.0.1:"+httpPort),
	))
	defer rese.F0(conn.Close)

	stubClient := somestub.NewSomeStubHTTPClient(conn)
	ctx := context.Background()
	message := uuid.New().String()

	// Set Bearer token in header
	// 设置 Bearer 令牌到请求头
	headers := nethttp.Header{}
	headers.Set("Authorization", "Bearer "+testPassword)

	resp, err := stubClient.CreateSomething(ctx, wrapperspb.String(message), http.Header(&headers))
	require.NoError(t, err)
	require.Equal(t, "created:"+message+",guest:"+testUsername, resp.GetValue())
}

func TestAuthTokens_CreateSomething_BasicAuth_HTTP(t *testing.T) {
	// Test protected endpoint with Basic Auth format
	// 测试使用 Basic Auth 格式的受保护端点
	conn := rese.P1(http.NewClient(
		context.Background(),
		http.WithMiddleware(recovery.Recovery()),
		http.WithEndpoint("127.0.0.1:"+httpPort),
	))
	defer rese.F0(conn.Close)

	stubClient := somestub.NewSomeStubHTTPClient(conn)
	ctx := context.Background()
	message := uuid.New().String()

	// Set Basic Auth token in header
	// 设置 Basic Auth 令牌到请求头
	headers := nethttp.Header{}
	headers.Set("Authorization", utils.BasicAuth(testUsername, testPassword))

	resp, err := stubClient.CreateSomething(ctx, wrapperspb.String(message), http.Header(&headers))
	require.NoError(t, err)
	require.Equal(t, "created:"+message+",guest:"+testUsername, resp.GetValue())
}

func TestAuthTokens_CreateSomething_InvalidToken_HTTP(t *testing.T) {
	// Test protected endpoint with invalid token
	// 测试带无效令牌的受保护端点
	conn := rese.P1(http.NewClient(
		context.Background(),
		http.WithMiddleware(recovery.Recovery()),
		http.WithEndpoint("127.0.0.1:"+httpPort),
	))
	defer rese.F0(conn.Close)

	stubClient := somestub.NewSomeStubHTTPClient(conn)
	ctx := context.Background()
	message := uuid.New().String()

	headers := nethttp.Header{}
	headers.Set("Authorization", invalidToken)

	// Should fail with UNAUTHORIZED
	// 应返回 UNAUTHORIZED 错误
	_, err := stubClient.CreateSomething(ctx, wrapperspb.String(message), http.Header(&headers))
	require.Error(t, err)

	erk := errors.FromError(err)
	require.Equal(t, int32(401), erk.Code)
	require.Equal(t, "UNAUTHORIZED", erk.Reason)
}

func TestAuthTokens_CreateSomething_MissingToken_HTTP(t *testing.T) {
	// Test protected endpoint without token
	// 测试不带令牌的受保护端点
	conn := rese.P1(http.NewClient(
		context.Background(),
		http.WithMiddleware(recovery.Recovery()),
		http.WithEndpoint("127.0.0.1:"+httpPort),
	))
	defer rese.F0(conn.Close)

	stubClient := somestub.NewSomeStubHTTPClient(conn)
	ctx := context.Background()
	message := uuid.New().String()

	// Should fail with UNAUTHORIZED
	// 应返回 UNAUTHORIZED 错误
	_, err := stubClient.CreateSomething(ctx, wrapperspb.String(message))
	require.Error(t, err)

	erk := errors.FromError(err)
	require.Equal(t, int32(401), erk.Code)
	require.Equal(t, "UNAUTHORIZED", erk.Reason)
}

func TestAuthTokens_UpdateSomething_SimpleToken_HTTP(t *testing.T) {
	// Test another protected endpoint with simple token
	// 测试另一个使用简单令牌的受保护端点
	conn := rese.P1(http.NewClient(
		context.Background(),
		http.WithMiddleware(recovery.Recovery()),
		http.WithEndpoint("127.0.0.1:"+httpPort),
	))
	defer rese.F0(conn.Close)

	stubClient := somestub.NewSomeStubHTTPClient(conn)
	ctx := context.Background()
	message := uuid.New().String()

	headers := nethttp.Header{}
	headers.Set("Authorization", testPassword)

	// UpdateSomething requires auth and should return guest info from context
	// UpdateSomething 需要认证并应从 context 返回用户信息
	resp, err := stubClient.UpdateSomething(ctx, wrapperspb.String(message), http.Header(&headers))
	require.NoError(t, err)
	require.Equal(t, "updated:"+message+",guest:"+testUsername, resp.GetValue())
}

func TestAuthTokens_SelectSomething_NoAuth_gRPC(t *testing.T) {
	// Test public endpoint via gRPC without authentication
	// 通过 gRPC 测试不需要认证的公开端点
	conn := rese.P1(grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("127.0.0.1:"+grpcPort),
		grpc.WithMiddleware(recovery.Recovery()),
	))
	defer rese.F0(conn.Close)

	stubClient := somestub.NewSomeStubClient(conn)
	ctx := context.Background()
	message := uuid.New().String()

	resp, err := stubClient.SelectSomething(ctx, wrapperspb.String(message))
	require.NoError(t, err)
	require.Equal(t, message, resp.GetValue())
}
