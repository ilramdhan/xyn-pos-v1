package grpc

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	tenantv1 "github.com/xyn-pos/gen/tenant/v1"
	"github.com/xyn-pos/services/tenant/internal/application/command"
	"github.com/xyn-pos/services/tenant/internal/application/query"
	"github.com/xyn-pos/services/tenant/internal/domain/user"
)

// UserHandler implements tenantv1.UserServiceServer.
type UserHandler struct {
	tenantv1.UnimplementedUserServiceServer
	registerH   *command.RegisterUserHandler
	loginH      *command.LoginUserHandler
	logoutH     *command.LogoutHandler
	setPINH     *command.SetPINHandler
	verifyPINH  *command.VerifyPINHandler
	deactivateH *command.DeactivateUserHandler
	getUserH    *query.GetUserHandler
	listUsersH  *query.ListUsersHandler
}

// NewUserHandler assembles a UserHandler from its application layer dependencies.
func NewUserHandler(
	registerH *command.RegisterUserHandler,
	loginH *command.LoginUserHandler,
	logoutH *command.LogoutHandler,
	setPINH *command.SetPINHandler,
	verifyPINH *command.VerifyPINHandler,
	deactivateH *command.DeactivateUserHandler,
	getUserH *query.GetUserHandler,
	listUsersH *query.ListUsersHandler,
) *UserHandler {
	return &UserHandler{
		registerH:   registerH,
		loginH:      loginH,
		logoutH:     logoutH,
		setPINH:     setPINH,
		verifyPINH:  verifyPINH,
		deactivateH: deactivateH,
		getUserH:    getUserH,
		listUsersH:  listUsersH,
	}
}

// RegisterUser creates a new user in Keycloak and persists the local record.
func (h *UserHandler) RegisterUser(ctx context.Context, req *tenantv1.RegisterUserRequest) (*tenantv1.RegisterUserResponse, error) {
	tenantID, err := extractTenantIDFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant context")
	}

	var branchScope []uuid.UUID
	for _, s := range req.BranchScope {
		if id, parseErr := uuid.Parse(s); parseErr == nil {
			branchScope = append(branchScope, id)
		}
	}

	u, err := h.registerH.Handle(ctx, command.RegisterUserInput{
		TenantID:       tenantID,
		Email:          req.Email,
		Password:       req.Password,
		FullName:       req.FullName,
		Role:           protoRoleToDomain(req.Role),
		BranchScope:    branchScope,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		return nil, mapUserError(err)
	}
	return &tenantv1.RegisterUserResponse{User: domainUserToProto(u)}, nil
}

// Login exchanges a Keycloak token for a PASETO token.
func (h *UserHandler) Login(ctx context.Context, req *tenantv1.LoginRequest) (*tenantv1.LoginResponse, error) {
	result, err := h.loginH.Handle(ctx, command.LoginInput{KeycloakToken: req.KeycloakToken})
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}
	return &tenantv1.LoginResponse{
		PasetoToken: result.PASETOToken,
		UserId:      result.UserID.String(),
		ExpiresAt:   result.ExpiresAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}

// Logout blacklists the PASETO token's JTI to revoke it.
func (h *UserHandler) Logout(ctx context.Context, req *tenantv1.LogoutRequest) (*emptypb.Empty, error) {
	if err := h.logoutH.Handle(ctx, command.LogoutInput{PASETOToken: req.PasetoToken}); err != nil {
		return nil, status.Error(codes.Internal, "logout failed")
	}
	return &emptypb.Empty{}, nil
}

// GetUser retrieves a single user by ID. Enforces cross-tenant isolation.
func (h *UserHandler) GetUser(ctx context.Context, req *tenantv1.GetUserRequest) (*tenantv1.GetUserResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}
	u, err := h.getUserH.Handle(ctx, userID)
	if err != nil {
		return nil, mapUserError(err)
	}
	claims, err := extractClaimsFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing auth context")
	}
	if u.TenantID != claims.TenantID {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	return &tenantv1.GetUserResponse{User: domainUserToProto(u)}, nil
}

// ListUsers returns all users belonging to the caller's tenant (derived from JWT claims).
func (h *UserHandler) ListUsers(ctx context.Context, _ *tenantv1.ListUsersRequest) (*tenantv1.ListUsersResponse, error) {
	claims, err := extractClaimsFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing auth context")
	}
	users, err := h.listUsersH.Handle(ctx, claims.TenantID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	protoUsers := make([]*tenantv1.User, len(users))
	for i, u := range users {
		protoUsers[i] = domainUserToProto(u)
	}
	return &tenantv1.ListUsersResponse{Users: protoUsers}, nil
}

// DeactivateUser soft-deletes a user. Only the owner role may call this.
func (h *UserHandler) DeactivateUser(ctx context.Context, req *tenantv1.DeactivateUserRequest) (*emptypb.Empty, error) {
	claims, err := extractClaimsFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing auth context")
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}
	if err := h.deactivateH.Handle(ctx, command.DeactivateUserInput{
		UserID:          userID,
		RequestedByRole: user.Role(claims.Role),
	}); err != nil {
		return nil, mapUserError(err)
	}
	return &emptypb.Empty{}, nil
}

// SetPIN stores a bcrypt-hashed PIN for fast cashier login.
// Only the target user themselves, or an owner/manager, may set a PIN.
func (h *UserHandler) SetPIN(ctx context.Context, req *tenantv1.SetPINRequest) (*emptypb.Empty, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}
	claims, err := extractClaimsFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing auth context")
	}
	if claims.UserID != userID && claims.Role != string(user.RoleOwner) && claims.Role != string(user.RoleManager) {
		return nil, status.Error(codes.PermissionDenied, "cannot set PIN for another user")
	}
	if err := h.setPINH.Handle(ctx, command.SetPINInput{UserID: userID, PIN: req.Pin}); err != nil {
		return nil, mapUserError(err)
	}
	return &emptypb.Empty{}, nil
}

// VerifyPIN checks a PIN against the stored bcrypt hash. Callers may only verify their own PIN.
func (h *UserHandler) VerifyPIN(ctx context.Context, req *tenantv1.VerifyPINRequest) (*tenantv1.VerifyPINResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}
	claims, err := extractClaimsFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing auth context")
	}
	if claims.UserID != userID {
		return nil, status.Error(codes.PermissionDenied, "cannot verify PIN for another user")
	}
	valid, err := h.verifyPINH.Handle(ctx, command.VerifyPINInput{UserID: userID, PIN: req.Pin})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &tenantv1.VerifyPINResponse{Valid: valid}, nil
}

// RefreshToken is not yet implemented.
func (h *UserHandler) RefreshToken(_ context.Context, _ *tenantv1.RefreshTokenRequest) (*tenantv1.LoginResponse, error) {
	return nil, status.Error(codes.Unimplemented, "RefreshToken not yet implemented")
}

func domainUserToProto(u *user.User) *tenantv1.User {
	scope := make([]string, len(u.BranchScope))
	for i, id := range u.BranchScope {
		scope[i] = id.String()
	}
	return &tenantv1.User{
		Id:          u.ID.String(),
		TenantId:    u.TenantID.String(),
		Email:       u.Email,
		FullName:    u.FullName,
		Role:        domainRoleToProto(u.Role),
		BranchScope: scope,
		IsActive:    u.IsActive,
		CreatedAt:   u.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   u.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func protoRoleToDomain(r tenantv1.Role) user.Role {
	switch r {
	case tenantv1.Role_ROLE_OWNER:
		return user.RoleOwner
	case tenantv1.Role_ROLE_MANAGER:
		return user.RoleManager
	case tenantv1.Role_ROLE_CASHIER:
		return user.RoleCashier
	case tenantv1.Role_ROLE_KITCHEN_STAFF:
		return user.RoleKitchenStaff
	default:
		return user.RoleCashier
	}
}

func domainRoleToProto(r user.Role) tenantv1.Role {
	switch r {
	case user.RoleOwner:
		return tenantv1.Role_ROLE_OWNER
	case user.RoleManager:
		return tenantv1.Role_ROLE_MANAGER
	case user.RoleCashier:
		return tenantv1.Role_ROLE_CASHIER
	case user.RoleKitchenStaff:
		return tenantv1.Role_ROLE_KITCHEN_STAFF
	default:
		return tenantv1.Role_ROLE_UNSPECIFIED
	}
}

func mapUserError(err error) error {
	switch {
	case errors.Is(err, user.ErrUserNotFound):
		return status.Error(codes.NotFound, "user not found")
	case errors.Is(err, user.ErrEmailTaken):
		return status.Error(codes.AlreadyExists, "email already taken")
	case errors.Is(err, user.ErrUserInactive):
		return status.Error(codes.FailedPrecondition, "user is inactive")
	case errors.Is(err, user.ErrUnauthorized):
		return status.Error(codes.PermissionDenied, "insufficient role")
	case errors.Is(err, user.ErrInvalidPIN):
		return status.Error(codes.InvalidArgument, "invalid PIN format")
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
