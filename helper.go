package qi

import "context"

const (
	// ContextTraceIDKey 链路追踪trace_id键（用于 Gin Context）
	ContextTraceIDKey = "trace_id"
	// ContextUidKey 用户uid键（用于 Gin Context）
	ContextUidKey = "uid"
	// ContextLanguageKey 用户语言键（用于 Gin Context）
	ContextLanguageKey = "language"

	// contextKeyTraceID 链路追踪trace_id键（用于标准库 context.Context）
	contextKeyTraceID = "qi:trace_id"
	// contextKeyUid 用户uid键（用于标准库 context.Context）
	contextKeyUid = "qi:uid"
	// contextKeyLanguage 用户语言键（用于标准库 context.Context）
	contextKeyLanguage = "qi:language"
)

// GetContextTraceID 获取上下文链路追踪trace_id
func GetContextTraceID(ctx *Context) string {
	return ctx.GetString(ContextTraceIDKey)
}

// SetContextTraceID 设置上下文链路追踪trace_id
func SetContextTraceID(ctx *Context, traceID string) {
	ctx.Set(ContextTraceIDKey, traceID)
}

// GetContextUid 获取上下文用户uid
func GetContextUid(ctx *Context) int64 {
	return ctx.GetInt64(ContextUidKey)
}

// SetContextUid 设置上下文用户uid
func SetContextUid(ctx *Context, uid int64) {
	ctx.Set(ContextUidKey, uid)
}

// GetContextLanguage 获取上下文用户语言
func GetContextLanguage(ctx *Context) string {
	return ctx.GetString(ContextLanguageKey)
}

// SetContextLanguage 设置上下文用户语言
func SetContextLanguage(ctx *Context, language string) {
	ctx.Set(ContextLanguageKey, language)
}

// GetTraceIDFromContext 从标准库 context.Context 获取 TraceID
func GetTraceIDFromContext(ctx context.Context) string {
	if traceID, ok := ctx.Value(contextKeyTraceID).(string); ok {
		return traceID
	}
	return ""
}

// GetUidFromContext 从标准库 context.Context 获取 UID
func GetUidFromContext(ctx context.Context) int64 {
	if uid, ok := ctx.Value(contextKeyUid).(int64); ok {
		return uid
	}
	return 0
}

// GetLanguageFromContext 从标准库 context.Context 获取 Language
func GetLanguageFromContext(ctx context.Context) string {
	if lang, ok := ctx.Value(contextKeyLanguage).(string); ok {
		return lang
	}
	return ""
}
