package grpcapi

import (
	"context"
	"grinex-adapter-service/internal/metrics"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// LoggingInterceptor logs method, status code and duration of each unary call.
func LoggingInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)

		code := status.Code(err)
		logger.Info("grpc request", zap.String("method", info.FullMethod), zap.String("code", code.String()), zap.Duration("duration", time.Since(start)))
		return resp, err
	}
}

// MetricsInterceptor records request count and latency of each unary call.
func MetricsInterceptor(m *metrics.Metrics) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)

		m.GRPCRequests.WithLabelValues(info.FullMethod, status.Code(err).String()).Inc()
		m.GRPCDuration.WithLabelValues(info.FullMethod).Observe(time.Since(start).Seconds())

		return resp, err
	}
}
