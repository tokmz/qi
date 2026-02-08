package qi

const (
	// ContextTraceIDKey 链路追踪trace_id键
	ContextTraceIDKey = "trace_id"
	// ContextUidKey 用户uid键
	ContextUidKey = "uid"
	// ContextLanguageKey 用户语言键
	ContextLanguageKey = "language"
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
