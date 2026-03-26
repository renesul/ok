package agent

import (
	"context"
	"errors"
	"fmt"

	"github.com/renesul/ok/domain"
	"go.uber.org/zap"
)

type DefaultExecutor struct {
	safetyGate     *SafetyGate
	auditLog       *AuditLog
	rateLimiter    *RateLimiter
	confirmManager *ConfirmationManager
	log            *zap.Logger
}

func NewDefaultExecutor(log *zap.Logger) *DefaultExecutor {
	return &DefaultExecutor{
		safetyGate:  NewSafetyGate(log),
		rateLimiter: NewRateLimiter(),
		log:         log.Named("agent.executor"),
	}
}

// SetSafetyGate configura um safety gate externo
func (e *DefaultExecutor) SetSafetyGate(gate *SafetyGate) {
	e.safetyGate = gate
}

// SetAuditLog configura o audit log
func (e *DefaultExecutor) SetAuditLog(audit *AuditLog) {
	e.auditLog = audit
}

// SetConfirmManager configura o confirmation manager para tools perigosas
func (e *DefaultExecutor) SetConfirmManager(cm *ConfirmationManager) {
	e.confirmManager = cm
}

func (e *DefaultExecutor) Execute(plan domain.Plan) (string, error) {
	return e.ExecuteWithContext(context.Background(), plan)
}

func (e *DefaultExecutor) ExecuteWithContext(ctx context.Context, plan domain.Plan) (string, error) {
	toolName := plan.Tool.Name()
	e.log.Debug("executing", zap.String("tool", toolName), zap.String("input", plan.Input))

	// Rate limit check
	if e.rateLimiter != nil {
		if err := e.rateLimiter.Allow(toolName); err != nil {
			e.auditRecord(toolName, plan.Input, "", "blocked", false)
			return "", err
		}
	}

	// Safety check
	if e.safetyGate != nil {
		if err := e.safetyGate.Check(plan.Tool, plan.Input); err != nil {
			if errors.Is(err, ErrRequiresConfirmation) && e.confirmManager != nil {
				e.log.Debug("requesting confirmation", zap.String("tool", toolName))
				conf := e.confirmManager.Request(toolName, plan.Input)
				approved, waitErr := e.confirmManager.WaitForResponse(conf)
				if waitErr != nil || !approved {
					e.auditRecord(toolName, plan.Input, "", "dangerous", false)
					return "", fmt.Errorf("execution not approved: %s", toolName)
				}
				e.log.Debug("execution approved", zap.String("tool", toolName))
			} else {
				e.log.Debug("safety blocked", zap.String("tool", toolName), zap.Error(err))
				safety := string(e.safetyGate.GetToolSafety(plan.Tool))
				e.auditRecord(toolName, plan.Input, "", safety, false)
				return "", err
			}
		}
	}

	// Usar RunWithContext se a tool suporta
	var result string
	var err error
	if ctxTool, ok := plan.Tool.(domain.ContextualTool); ok {
		result, err = ctxTool.RunWithContext(ctx, plan.Input)
	} else {
		result, err = plan.Tool.Run(plan.Input)
	}

	if err != nil {
		e.log.Debug("execution failed", zap.String("tool", toolName), zap.Error(err))
		e.auditRecord(toolName, plan.Input, err.Error(), "safe", true)
		return "", fmt.Errorf("tool '%s' failed: %w", toolName, err)
	}

	e.log.Debug("execution ok", zap.String("tool", toolName), zap.Int("result_len", len(result)))
	e.auditRecord(toolName, plan.Input, result, "safe", true)
	return result, nil
}

func (e *DefaultExecutor) auditRecord(tool, input, output, safety string, approved bool) {
	if e.auditLog == nil {
		return
	}
	e.auditLog.Record(AuditEntry{
		Tool:     tool,
		Input:    input,
		Output:   output,
		Safety:   safety,
		Approved: approved,
	})
}
