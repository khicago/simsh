package engine_test

import (
	"context"
	"testing"

	"github.com/khicago/simsh/pkg/contract"
)

func BenchmarkEngineExecuteEcho(b *testing.B) {
	eng := newTestEngine()
	ops := contract.OpsFromFilesystem(newTestFS())
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, code := eng.Execute(ctx, "echo hello", ops)
		if code != 0 || out != "hello" {
			b.Fatalf("unexpected result: code=%d out=%q", code, out)
		}
	}
}

func BenchmarkEngineExecutePreparedEcho(b *testing.B) {
	eng := newTestEngine()
	ops := contract.OpsFromFilesystem(newTestFS())
	prepared, err := eng.PrepareOps(context.Background(), ops)
	if err != nil {
		b.Fatalf("prepare ops failed: %v", err)
	}
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, code := eng.ExecutePrepared(ctx, "echo hello", prepared)
		if code != 0 || out != "hello" {
			b.Fatalf("unexpected result: code=%d out=%q", code, out)
		}
	}
}

func BenchmarkEngineExecuteRedirectWrite(b *testing.B) {
	eng := newTestEngine()
	fs := newTestFS()
	ops := writableOps(fs)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, code := eng.Execute(ctx, "echo hello > /workspace/out.txt", ops)
		if code != 0 || out != "" {
			b.Fatalf("unexpected result: code=%d out=%q", code, out)
		}
	}
}

func BenchmarkEngineExecutePreparedRedirectWrite(b *testing.B) {
	eng := newTestEngine()
	fs := newTestFS()
	ops := writableOps(fs)
	prepared, err := eng.PrepareOps(context.Background(), ops)
	if err != nil {
		b.Fatalf("prepare ops failed: %v", err)
	}
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, code := eng.ExecutePrepared(ctx, "echo hello > /workspace/out.txt", prepared)
		if code != 0 || out != "" {
			b.Fatalf("unexpected result: code=%d out=%q", code, out)
		}
	}
}
