package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"go.uber.org/zap"

	"auto-claude-code/internal/logger"
)

// Transport MCP传输层接口
type Transport interface {
	// Start 启动传输层
	Start(ctx context.Context) error

	// Stop 停止传输层
	Stop(ctx context.Context) error

	// GetType 获取传输类型
	GetType() string

	// GetAddress 获取传输地址（如果适用）
	GetAddress() string
}

// TransportType 传输类型
type TransportType string

const (
	TransportHTTP  TransportType = "http"
	TransportStdio TransportType = "stdio"
)

// TransportHandler 传输处理器
type TransportHandler interface {
	HandleRequest(ctx context.Context, req *JSONRPCRequest) *JSONRPCResponse
}

// StdioTransport stdio传输实现
type StdioTransport struct {
	logger  logger.Logger
	handler TransportHandler

	reader io.Reader
	writer io.Writer

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewStdioTransport 创建stdio传输
func NewStdioTransport(handler TransportHandler, logger logger.Logger, reader io.Reader, writer io.Writer) Transport {
	return &StdioTransport{
		logger:  logger,
		handler: handler,
		reader:  reader,
		writer:  writer,
	}
}

// Start 启动stdio传输
func (t *StdioTransport) Start(ctx context.Context) error {
	t.ctx, t.cancel = context.WithCancel(ctx)

	t.logger.Info("启动MCP stdio传输")

	// 启动消息处理循环
	t.wg.Add(1)
	go t.messageLoop()

	return nil
}

// Stop 停止stdio传输
func (t *StdioTransport) Stop(ctx context.Context) error {
	t.logger.Info("停止MCP stdio传输")

	if t.cancel != nil {
		t.cancel()
	}

	// 等待goroutine完成
	done := make(chan struct{})
	go func() {
		t.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// GetType 获取传输类型
func (t *StdioTransport) GetType() string {
	return string(TransportStdio)
}

// GetAddress 获取传输地址
func (t *StdioTransport) GetAddress() string {
	return "stdio"
}

// messageLoop 消息处理循环
func (t *StdioTransport) messageLoop() {
	defer t.wg.Done()

	scanner := bufio.NewScanner(t.reader)
	encoder := json.NewEncoder(t.writer)

	for {
		select {
		case <-t.ctx.Done():
			return
		default:
			if !scanner.Scan() {
				if err := scanner.Err(); err != nil {
					t.logger.Error("读取stdin失败", zap.Error(err))
				}
				return
			}

			line := scanner.Text()
			if line == "" {
				continue
			}

			// 解析JSON-RPC请求
			var req JSONRPCRequest
			if err := json.Unmarshal([]byte(line), &req); err != nil {
				t.logger.Error("解析JSON-RPC请求失败",
					zap.Error(err),
					zap.String("data", line))

				// 发送错误响应
				errorResp := &JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      nil,
					Error: &JSONRPCError{
						Code:    -32700,
						Message: "Parse error",
						Data:    err.Error(),
					},
				}
				encoder.Encode(errorResp)
				continue
			}

			t.logger.Debug("收到JSON-RPC请求",
				zap.String("method", req.Method),
				zap.Any("id", req.ID))

			// 处理请求
			resp := t.handler.HandleRequest(t.ctx, &req)

			// 发送响应
			if err := encoder.Encode(resp); err != nil {
				t.logger.Error("发送JSON-RPC响应失败", zap.Error(err))
			}

			t.logger.Debug("发送JSON-RPC响应",
				zap.Any("id", resp.ID),
				zap.Bool("hasError", resp.Error != nil))
		}
	}
}

// HTTPTransport HTTP传输实现（对现有代码的包装）
type HTTPTransport struct {
	server  *http.Server
	address string
	logger  logger.Logger
	handler TransportHandler
}

// NewHTTPTransport 创建HTTP传输
func NewHTTPTransport(server *http.Server, address string, handler TransportHandler, logger logger.Logger) Transport {
	return &HTTPTransport{
		server:  server,
		address: address,
		logger:  logger,
		handler: handler,
	}
}

// Start 启动HTTP传输
func (t *HTTPTransport) Start(ctx context.Context) error {
	t.logger.Info("启动MCP HTTP传输", zap.String("address", t.address))

	go func() {
		if err := t.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.logger.Error("HTTP服务器启动失败", zap.Error(err))
		}
	}()

	return nil
}

// Stop 停止HTTP传输
func (t *HTTPTransport) Stop(ctx context.Context) error {
	t.logger.Info("停止MCP HTTP传输")
	return t.server.Shutdown(ctx)
}

// GetType 获取传输类型
func (t *HTTPTransport) GetType() string {
	return string(TransportHTTP)
}

// GetAddress 获取传输地址
func (t *HTTPTransport) GetAddress() string {
	return t.address
}

// MultiTransport 多传输支持
type MultiTransport struct {
	transports []Transport
	logger     logger.Logger
}

// NewMultiTransport 创建多传输实例
func NewMultiTransport(logger logger.Logger) *MultiTransport {
	return &MultiTransport{
		transports: make([]Transport, 0),
		logger:     logger,
	}
}

// AddTransport 添加传输
func (mt *MultiTransport) AddTransport(transport Transport) {
	mt.transports = append(mt.transports, transport)
}

// Start 启动所有传输
func (mt *MultiTransport) Start(ctx context.Context) error {
	mt.logger.Info("启动多传输MCP服务器",
		zap.Int("transports", len(mt.transports)))

	for _, transport := range mt.transports {
		if err := transport.Start(ctx); err != nil {
			return fmt.Errorf("启动传输 %s 失败: %w", transport.GetType(), err)
		}

		mt.logger.Info("传输已启动",
			zap.String("type", transport.GetType()),
			zap.String("address", transport.GetAddress()))
	}

	return nil
}

// Stop 停止所有传输
func (mt *MultiTransport) Stop(ctx context.Context) error {
	mt.logger.Info("停止多传输MCP服务器")

	var lastErr error
	for _, transport := range mt.transports {
		if err := transport.Stop(ctx); err != nil {
			mt.logger.Error("停止传输失败",
				zap.String("type", transport.GetType()),
				zap.Error(err))
			lastErr = err
		}
	}

	return lastErr
}

// GetTransports 获取所有传输
func (mt *MultiTransport) GetTransports() []Transport {
	return mt.transports
}

// transportHandlerAdapter 传输处理器适配器
type transportHandlerAdapter struct {
	server *mcpServer
}

// HandleRequest 处理传输请求
func (t *transportHandlerAdapter) HandleRequest(ctx context.Context, req *JSONRPCRequest) *JSONRPCResponse {
	return t.server.processJSONRPCRequest(ctx, req)
}
