package llmjudge

import "context"

type contextKey struct{}

func WithJudge(ctx context.Context, judge LLMJudge) context.Context {
	return context.WithValue(ctx, contextKey{}, judge)
}

func FromContext(ctx context.Context) (LLMJudge, bool) {
	val := ctx.Value(contextKey{})
	if val == nil {
		return nil, false
	}

	judge, ok := val.(LLMJudge)
	if !ok {
		return nil, false
	}

	return judge, true
}
