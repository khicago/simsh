package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

type ExecuteRequest struct {
	Command string `json:"command"`
	RootDir string `json:"root_dir,omitempty"`
	Profile string `json:"profile,omitempty"`
	Policy  string `json:"policy,omitempty"`
}

type ExecuteResponse struct {
	contract.ExecutionResult
	Output string `json:"output,omitempty"`
}

type OpsFactory func(ctx context.Context, req ExecuteRequest) (contract.Ops, error)

type ExecutorService struct {
	Engine     *engine.Engine
	OpsFactory OpsFactory
}

func (s *ExecutorService) Execute(ctx context.Context, req ExecuteRequest) (ExecuteResponse, error) {
	if s == nil || s.Engine == nil {
		return ExecuteResponse{}, fmt.Errorf("executor service is not initialized")
	}
	if s.OpsFactory == nil {
		return ExecuteResponse{}, fmt.Errorf("ops factory is required")
	}
	command := strings.TrimSpace(req.Command)
	if command == "" {
		result := contract.ExecutionResult{ExitCode: contract.ExitCodeUsage, Stdout: "execute: command is required"}
		return ExecuteResponse{ExecutionResult: result, Output: result.FlattenOutput()}, nil
	}
	ops, err := s.OpsFactory(ctx, req)
	if err != nil {
		return ExecuteResponse{}, err
	}
	result := s.Engine.ExecuteResult(ctx, command, ops)
	return ExecuteResponse{ExecutionResult: result, Output: result.FlattenOutput()}, nil
}
