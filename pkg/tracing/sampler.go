package tracing

import (
	"os"
	"strconv"

	"go.opentelemetry.io/otel/sdk/trace"
)

// newSampler 根据配置创建采样器
func newSampler(cfg *Config) trace.Sampler {
	// 从环境变量读取采样配置
	if samplerType := os.Getenv("OTEL_TRACES_SAMPLER"); samplerType != "" {
		return newSamplerFromEnv(samplerType)
	}

	// 使用配置文件的采样策略
	switch cfg.SamplingType {
	case "always":
		return trace.AlwaysSample()
	case "never":
		return trace.NeverSample()
	case "ratio":
		return trace.TraceIDRatioBased(cfg.SamplingRate)
	case "parent_based":
		return trace.ParentBased(trace.TraceIDRatioBased(cfg.SamplingRate))
	default:
		// 默认使用 parent_based
		return trace.ParentBased(trace.TraceIDRatioBased(cfg.SamplingRate))
	}
}

// newSamplerFromEnv 从环境变量创建采样器
func newSamplerFromEnv(samplerType string) trace.Sampler {
	switch samplerType {
	case "always_on":
		return trace.AlwaysSample()
	case "always_off":
		return trace.NeverSample()
	case "traceidratio":
		ratio := getSamplingRatioFromEnv()
		return trace.TraceIDRatioBased(ratio)
	case "parentbased_always_on":
		return trace.ParentBased(trace.AlwaysSample())
	case "parentbased_always_off":
		return trace.ParentBased(trace.NeverSample())
	case "parentbased_traceidratio":
		ratio := getSamplingRatioFromEnv()
		return trace.ParentBased(trace.TraceIDRatioBased(ratio))
	default:
		// 默认使用 parent_based with 100% sampling
		return trace.ParentBased(trace.AlwaysSample())
	}
}

// getSamplingRatioFromEnv 从环境变量获取采样率
func getSamplingRatioFromEnv() float64 {
	ratioStr := os.Getenv("OTEL_TRACES_SAMPLER_ARG")
	if ratioStr == "" {
		return 1.0
	}

	ratio, err := strconv.ParseFloat(ratioStr, 64)
	if err != nil || ratio < 0 || ratio > 1 {
		return 1.0
	}

	return ratio
}
