package grpc

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	commonv1 "github.com/xyn-pos/gen/common/v1"
	tenantv1 "github.com/xyn-pos/gen/tenant/v1"
	"github.com/xyn-pos/services/tenant/internal/application/command"
	"github.com/xyn-pos/services/tenant/internal/application/query"
	domain "github.com/xyn-pos/services/tenant/internal/domain/tenant"
	"github.com/xyn-pos/shared/pkg/middleware"
	sharedtelemetry "github.com/xyn-pos/shared/pkg/telemetry"
)

var tracer trace.Tracer

func init() {
	tracer = otel.Tracer("tenant-service.grpc")
}

// TenantHandler implements tenantv1.TenantServiceServer.
type TenantHandler struct {
	tenantv1.UnimplementedTenantServiceServer
	createTenant *command.CreateTenantHandler
	createBranch *command.CreateBranchHandler
	getTenant    *query.GetTenantHandler
	listBranches *query.ListBranchesHandler
	updatePlanH  *command.UpdatePlanHandler
}

// NewTenantHandler constructs a TenantHandler with all required use-case handlers.
func NewTenantHandler(
	createTenant *command.CreateTenantHandler,
	createBranch *command.CreateBranchHandler,
	getTenant *query.GetTenantHandler,
	listBranches *query.ListBranchesHandler,
	updatePlanH *command.UpdatePlanHandler,
) *TenantHandler {
	return &TenantHandler{
		createTenant: createTenant,
		createBranch: createBranch,
		getTenant:    getTenant,
		listBranches: listBranches,
		updatePlanH:  updatePlanH,
	}
}

// CreateTenant is a public registration endpoint — no auth required.
func (h *TenantHandler) CreateTenant(ctx context.Context, req *tenantv1.CreateTenantRequest) (*tenantv1.CreateTenantResponse, error) {
	ctx, span := tracer.Start(ctx, "TenantHandler.CreateTenant")
	defer span.End()

	cmd := command.CreateTenantCommand{
		IdempotencyKey: req.GetIdempotencyKey(),
		Name:           req.GetName(),
		Slug:           req.GetSlug(),
		Plan:           protoTierToDomain(req.GetPlan()),
	}

	result, err := h.createTenant.Handle(ctx, cmd)
	if err != nil {
		sharedtelemetry.RecordError(span, err)
		return nil, mapTenantError(err)
	}

	return &tenantv1.CreateTenantResponse{Tenant: tenantToProto(result)}, nil
}

// GetTenant retrieves a tenant by ID. Requires authentication.
func (h *TenantHandler) GetTenant(ctx context.Context, req *tenantv1.GetTenantRequest) (*tenantv1.GetTenantResponse, error) {
	ctx, span := tracer.Start(ctx, "TenantHandler.GetTenant")
	defer span.End()

	// Auth check must come before any I/O.
	claims, ok := middleware.ClaimsFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	// Tenant isolation: callers may only retrieve their own tenant.
	if claims.TenantID != tenantID {
		return nil, status.Error(codes.NotFound, "tenant not found")
	}

	result, err := h.getTenant.Handle(ctx, query.GetTenantQuery{TenantID: tenantID})
	if err != nil {
		sharedtelemetry.RecordError(span, err)
		return nil, mapTenantError(err)
	}

	return &tenantv1.GetTenantResponse{Tenant: tenantToProto(result)}, nil
}

// CreateBranch creates a branch for the authenticated tenant.
// TenantID is always extracted from claims — never from the request body.
func (h *TenantHandler) CreateBranch(ctx context.Context, req *tenantv1.CreateBranchRequest) (*tenantv1.CreateBranchResponse, error) {
	ctx, span := tracer.Start(ctx, "TenantHandler.CreateBranch")
	defer span.End()

	// Auth check must come before any I/O.
	claims, ok := middleware.ClaimsFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	addr := domain.Address{}
	if a := req.GetAddress(); a != nil {
		addr = domain.Address{
			Street:     a.GetStreet(),
			City:       a.GetCity(),
			Province:   a.GetProvince(),
			PostalCode: a.GetPostalCode(),
			Country:    a.GetCountry(),
		}
	}

	cmd := command.CreateBranchCommand{
		TenantID: claims.TenantID, // always from claims — never from req.TenantId
		Name:     req.GetName(),
		Address:  addr,
		Timezone: req.GetTimezone(),
	}

	result, err := h.createBranch.Handle(ctx, cmd)
	if err != nil {
		sharedtelemetry.RecordError(span, err)
		return nil, mapTenantError(err)
	}

	return &tenantv1.CreateBranchResponse{Branch: branchToProto(*result)}, nil
}

// ListBranches lists branches for the authenticated tenant.
func (h *TenantHandler) ListBranches(ctx context.Context, req *tenantv1.ListBranchesRequest) (*tenantv1.ListBranchesResponse, error) {
	ctx, span := tracer.Start(ctx, "TenantHandler.ListBranches")
	defer span.End()

	// Auth check must come before any I/O.
	claims, ok := middleware.ClaimsFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	// Tenant isolation: callers may only list branches of their own tenant.
	if claims.TenantID != tenantID {
		return nil, status.Error(codes.NotFound, "tenant not found")
	}

	result, err := h.listBranches.Handle(ctx, query.ListBranchesQuery{TenantID: tenantID})
	if err != nil {
		sharedtelemetry.RecordError(span, err)
		return nil, mapTenantError(err)
	}

	branches := make([]*tenantv1.Branch, len(result))
	for i, b := range result {
		branches[i] = branchToProto(b)
	}

	return &tenantv1.ListBranchesResponse{Branches: branches}, nil
}

// UpdatePlan upgrades the authenticated tenant's subscription plan.
func (h *TenantHandler) UpdatePlan(ctx context.Context, req *tenantv1.UpdatePlanRequest) (*tenantv1.UpdatePlanResponse, error) {
	ctx, span := tracer.Start(ctx, "TenantHandler.UpdatePlan")
	defer span.End()

	// Auth check must come before any I/O.
	claims, ok := middleware.ClaimsFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "authentication required")
	}

	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	// Tenant isolation: callers may only modify their own tenant.
	if claims.TenantID != tenantID {
		return nil, status.Error(codes.NotFound, "tenant not found")
	}

	cmd := command.UpdatePlanCommand{
		TenantID: tenantID,
		NewTier:  protoTierToDomain(req.GetNewTier()),
	}

	if err := h.updatePlanH.Handle(ctx, cmd); err != nil {
		sharedtelemetry.RecordError(span, err)
		return nil, mapTenantError(err)
	}

	// Reload the tenant to return its updated state.
	updated, err := h.getTenant.Handle(ctx, query.GetTenantQuery{TenantID: tenantID})
	if err != nil {
		sharedtelemetry.RecordError(span, err)
		return nil, mapTenantError(err)
	}

	return &tenantv1.UpdatePlanResponse{Tenant: tenantToProto(updated)}, nil
}

// --- proto ↔ domain conversion helpers ---

func protoTierToDomain(t tenantv1.PlanTier) domain.PlanTier {
	switch t {
	case tenantv1.PlanTier_PLAN_TIER_GROWTH:
		return domain.PlanTierGrowth
	case tenantv1.PlanTier_PLAN_TIER_ENTERPRISE:
		return domain.PlanTierEnterprise
	default:
		return domain.PlanTierFree
	}
}

func domainTierToProto(t domain.PlanTier) tenantv1.PlanTier {
	switch t {
	case domain.PlanTierGrowth:
		return tenantv1.PlanTier_PLAN_TIER_GROWTH
	case domain.PlanTierEnterprise:
		return tenantv1.PlanTier_PLAN_TIER_ENTERPRISE
	default:
		return tenantv1.PlanTier_PLAN_TIER_FREE
	}
}

func domainStatusToProto(s domain.Status) tenantv1.TenantStatus {
	switch s {
	case domain.StatusActive:
		return tenantv1.TenantStatus_TENANT_STATUS_ACTIVE
	case domain.StatusSuspended:
		return tenantv1.TenantStatus_TENANT_STATUS_SUSPENDED
	case domain.StatusDeleted:
		return tenantv1.TenantStatus_TENANT_STATUS_DELETED
	default:
		return tenantv1.TenantStatus_TENANT_STATUS_UNSPECIFIED
	}
}

func domainSubStatusToProto(s domain.SubscriptionStatus) tenantv1.SubscriptionStatus {
	switch s {
	case domain.SubscriptionActive:
		return tenantv1.SubscriptionStatus_SUBSCRIPTION_STATUS_ACTIVE
	case domain.SubscriptionTrial:
		return tenantv1.SubscriptionStatus_SUBSCRIPTION_STATUS_TRIAL
	case domain.SubscriptionExpired:
		return tenantv1.SubscriptionStatus_SUBSCRIPTION_STATUS_EXPIRED
	case domain.SubscriptionCancelled:
		return tenantv1.SubscriptionStatus_SUBSCRIPTION_STATUS_CANCELLED
	default:
		return tenantv1.SubscriptionStatus_SUBSCRIPTION_STATUS_UNSPECIFIED
	}
}

func tenantToProto(t *domain.Tenant) *tenantv1.Tenant {
	var trialEndsAt string
	if t.TrialEndsAt != nil {
		trialEndsAt = t.TrialEndsAt.UTC().Format(time.RFC3339)
	}

	return &tenantv1.Tenant{
		Id:                 t.ID.String(),
		Name:               t.Name,
		Slug:               t.Slug,
		Plan:               domainTierToProto(t.Plan.Tier),
		Status:             domainStatusToProto(t.Status),
		CreatedAt:          t.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:          t.UpdatedAt.UTC().Format(time.RFC3339),
		SubscriptionStatus: domainSubStatusToProto(t.SubscriptionStatus),
		TrialEndsAt:        trialEndsAt,
	}
}

func branchToProto(b domain.Branch) *tenantv1.Branch {
	return &tenantv1.Branch{
		Id:       b.ID.String(),
		TenantId: b.TenantID.String(),
		Name:     b.Name,
		Address: &commonv1.Address{
			Street:     b.Address.Street,
			City:       b.Address.City,
			Province:   b.Address.Province,
			PostalCode: b.Address.PostalCode,
			Country:    b.Address.Country,
		},
		Timezone:  b.Timezone,
		IsActive:  b.IsActive,
		CreatedAt: b.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: b.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

// mapTenantError maps domain sentinel errors to gRPC status errors.
func mapTenantError(err error) error {
	switch {
	case errors.Is(err, domain.ErrTenantNotFound):
		return status.Error(codes.NotFound, "tenant not found")
	case errors.Is(err, domain.ErrSlugAlreadyTaken):
		return status.Error(codes.AlreadyExists, "slug already taken")
	case errors.Is(err, domain.ErrBranchLimitReached):
		return status.Error(codes.ResourceExhausted, "branch limit reached for plan")
	case errors.Is(err, domain.ErrInvalidTenantName):
		return status.Error(codes.InvalidArgument, "tenant name cannot be empty")
	case errors.Is(err, domain.ErrInvalidSlug):
		return status.Error(codes.InvalidArgument, "invalid slug format")
	case errors.Is(err, domain.ErrSubscriptionExpired):
		return status.Error(codes.PermissionDenied, "subscription has expired or been cancelled")
	case errors.Is(err, domain.ErrDowngradeNotAllowed):
		return status.Error(codes.FailedPrecondition, "plan downgrade is not allowed")
	case errors.Is(err, domain.ErrSameTierUpgrade):
		return status.Error(codes.FailedPrecondition, "tenant is already on this plan tier")
	case errors.Is(err, domain.ErrSubscriptionAlreadyCancelled):
		return status.Error(codes.FailedPrecondition, "subscription is already cancelled")
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
