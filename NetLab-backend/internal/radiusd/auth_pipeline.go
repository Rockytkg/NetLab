package radiusd

import (
	"context"
	"fmt"
	"sync"

	"layeh.com/radius"

	"netlab-backend/internal/model"
	"netlab-backend/internal/radiusd/vendors"
)

// AuthPipelineStage 是认证流水线中的可插拔处理单元。
type AuthPipelineStage interface {
	Name() string
	Execute(ctx *AuthPipelineContext) error
}

type stageFunc struct {
	name string
	fn   func(ctx *AuthPipelineContext) error
}

func (s *stageFunc) Name() string { return s.name }

func (s *stageFunc) Execute(ctx *AuthPipelineContext) error { return s.fn(ctx) }

func newStage(name string, fn func(ctx *AuthPipelineContext) error) AuthPipelineStage {
	return &stageFunc{name: name, fn: fn}
}

// AuthPipeline 管理有序 stage 的执行。
type AuthPipeline struct {
	mu     sync.RWMutex
	stages []AuthPipelineStage
}

// NewAuthPipeline 创建空流水线。
func NewAuthPipeline() *AuthPipeline {
	return &AuthPipeline{stages: make([]AuthPipelineStage, 0)}
}

// Use 追加一个 stage 到末尾。
func (p *AuthPipeline) Use(stage AuthPipelineStage) *AuthPipeline {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stages = append(p.stages, stage)
	return p
}

// Stages 返回已注册 stage 的副本。
func (p *AuthPipeline) Stages() []AuthPipelineStage {
	p.mu.RLock()
	defer p.mu.RUnlock()
	stages := make([]AuthPipelineStage, len(p.stages))
	copy(stages, p.stages)
	return stages
}

// Execute 顺序执行各 stage，直到完成或 ctx.Stop() 被调用。
func (p *AuthPipeline) Execute(ctx *AuthPipelineContext) error {
	p.mu.RLock()
	stages := make([]AuthPipelineStage, len(p.stages))
	copy(stages, p.stages)
	p.mu.RUnlock()

	for _, stage := range stages {
		if ctx.IsStopped() {
			break
		}
		if err := stage.Execute(ctx); err != nil {
			return fmt.Errorf("stage %s failed: %w", stage.Name(), err)
		}
	}
	return nil
}

// AuthPipelineContext 携带单次认证请求在各 stage 间共享的可变数据。
type AuthPipelineContext struct {
	Context context.Context
	Service *AuthService

	Writer   radius.ResponseWriter
	Request  *radius.Request
	Response *radius.Packet

	Username         string
	NasIdentifier    string
	CallingStationID string
	RemoteIP         string

	NAS           *model.RadiusNas
	VendorRequest *vendors.VendorRequest
	User          *model.RadiusUser

	IsEAP            bool
	EAPMethod        string
	IsMacAuth        bool
	RateLimitChecked bool

	stop bool
}

// NewAuthPipelineContext 构建带默认值的请求上下文。
func NewAuthPipelineContext(service *AuthService, w radius.ResponseWriter, r *radius.Request) *AuthPipelineContext {
	return &AuthPipelineContext{
		Context:       context.Background(),
		Service:       service,
		Writer:        w,
		Request:       r,
		VendorRequest: &vendors.VendorRequest{},
	}
}

// Stop 终止后续 stage 执行。
func (ctx *AuthPipelineContext) Stop() { ctx.stop = true }

// IsStopped 报告执行是否已终止。
func (ctx *AuthPipelineContext) IsStopped() bool { return ctx.stop }
