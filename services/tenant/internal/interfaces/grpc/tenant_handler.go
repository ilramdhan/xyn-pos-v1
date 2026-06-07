package grpc

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/xyn-pos/services/tenant/internal/application/command"
	"github.com/xyn-pos/services/tenant/internal/application/query"
	domain "github.com/xyn-pos/services/tenant/internal/domain/tenant"
	sharederrors "github.com/xyn-pos/shared/pkg/errors"
	sharedtelemetry "github.com/xyn-pos/shared/pkg/telemetry"
)

var tracer trace.Tracer

func init() {
	tracer = otel.Tracer("tenant-service.grpc")
}

// TenantHandler implements TenantServiceServer (generated in Phase 4).
type TenantHandler struct {
	createTenant *command.CreateTenantHandler
	createBranch *command.CreateBranchHandler
	getTenant    *query.GetTenantHandler
	listBranches *query.ListBranchesHandler
}

// NewTenantHandler constructs a TenantHandler with all required use-case handlers.
func NewTenantHandler(
	createTenant *command.CreateTenantHandler,
	createBranch *command.CreateBranchHandler,
	getTenant *query.GetTenantHandler,
	listBranches *query.ListBranchesHandler,
) *TenantHandler {
	return &TenantHandler{
		createTenant: createTenant,
		createBranch: createBranch,
		getTenant:    getTenant,
		listBranches: listBranches,
	}
}

// CreateTenant handles tenant creation. Wired to pb.TenantServiceServer in Phase 4.
func (h *TenantHandler) CreateTenant(ctx context.Context, idempotencyKey, name, slug string, planTier domain.PlanTier) (*domain.Tenant, error) {
	ctx, span := tracer.Start(ctx, "TenantHandler.CreateTenant")
	defer span.End()

	cmd := command.CreateTenantCommand{
		IdempotencyKey: idempotencyKey,
		Name:           name,
		Slug:           slug,
		Plan:           planTier,
	}

	result, err := h.createTenant.Handle(ctx, cmd)
	if err != nil {
		sharedtelemetry.RecordError(span, err)
		return nil, sharederrors.MapSentinelToGRPCStatus(err).Err()
	}
	return result, nil
}

// GetTenant handles tenant retrieval. Wired to pb.TenantServiceServer in Phase 4.
func (h *TenantHandler) GetTenant(ctx context.Context, tenantIDStr string) (*domain.Tenant, error) {
	ctx, span := tracer.Start(ctx, "TenantHandler.GetTenant")
	defer span.End()

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return nil, fmt.Errorf("TenantHandler.GetTenant invalid tenant_id: %w", err)
	}

	result, err := h.getTenant.Handle(ctx, query.GetTenantQuery{TenantID: tenantID})
	if err != nil {
		sharedtelemetry.RecordError(span, err)
		return nil, sharederrors.MapSentinelToGRPCStatus(err).Err()
	}
	return result, nil
}

// CreateBranch handles branch creation. Wired to pb.TenantServiceServer in Phase 4.
func (h *TenantHandler) CreateBranch(ctx context.Context, idempotencyKey string, tenantID uuid.UUID, name string, address domain.Address, timezone string) (*domain.Branch, error) {
	ctx, span := tracer.Start(ctx, "TenantHandler.CreateBranch")
	defer span.End()

	cmd := command.CreateBranchCommand{
		IdempotencyKey: idempotencyKey,
		TenantID:       tenantID,
		Name:           name,
		Address:        address,
		Timezone:       timezone,
	}

	result, err := h.createBranch.Handle(ctx, cmd)
	if err != nil {
		sharedtelemetry.RecordError(span, err)
		return nil, sharederrors.MapSentinelToGRPCStatus(err).Err()
	}
	return result, nil
}

// ListBranches handles listing branches for a tenant. Wired to pb.TenantServiceServer in Phase 4.
func (h *TenantHandler) ListBranches(ctx context.Context, tenantIDStr string) ([]domain.Branch, error) {
	ctx, span := tracer.Start(ctx, "TenantHandler.ListBranches")
	defer span.End()

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return nil, fmt.Errorf("TenantHandler.ListBranches invalid tenant_id: %w", err)
	}

	result, err := h.listBranches.Handle(ctx, query.ListBranchesQuery{TenantID: tenantID})
	if err != nil {
		sharedtelemetry.RecordError(span, err)
		return nil, sharederrors.MapSentinelToGRPCStatus(err).Err()
	}
	return result, nil
}
