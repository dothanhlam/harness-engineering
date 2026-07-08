package agent

import "testing"

func TestParseLlamaTokenUsage(t *testing.T) {
	stderr := `
build: 4589 (abcdef) with cc
llama_perf_sampler_print:    sampling time =      12.34 ms /   142 runs
llama_perf_context_print:        load time =     512.00 ms
llama_perf_context_print: prompt eval time =     123.45 ms /    42 tokens (    2.94 ms per token,   340.24 tokens per second)
llama_perf_context_print:        eval time =    1234.56 ms /   100 runs   (   12.35 ms per token,    81.00 tokens per second)
llama_perf_context_print:       total time =    1800.00 ms /   142 tokens
`
	got := parseLlamaTokenUsage(stderr)
	if got.PromptTokens != 42 {
		t.Errorf("PromptTokens = %d, want 42", got.PromptTokens)
	}
	if got.EvalTokens != 100 {
		t.Errorf("EvalTokens = %d, want 100", got.EvalTokens)
	}
}

func TestParseLlamaTokenUsage_NoTimings(t *testing.T) {
	got := parseLlamaTokenUsage("no perf output here")
	if got.PromptTokens != 0 || got.EvalTokens != 0 {
		t.Errorf("expected zero usage, got %+v", got)
	}
}
