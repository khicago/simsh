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
	Output   string `json:"output"`
	ExitCode int    `json:"exit_code"`
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
		return ExecuteResponse{Output: "execute: command is required", ExitCode: contract.ExitCodeUsage}, nil
	}
	ops, err := s.OpsFactory(ctx, req)
	if err != nil {
		return ExecuteResponse{}, err
	}
	out, code := s.Engine.Execute(ctx, command, ops)
	return ExecuteResponse{Output: out, ExitCode: code}, nil
}
