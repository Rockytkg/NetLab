package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"netlab-backend/internal/model"
	"netlab-backend/internal/repository"
	"netlab-backend/pkg/apperrors"
)

// PasskeyService 处理 WebAuthn 的注册与认证。
type PasskeyService struct {
	passkeyRepo  *repository.PasskeyRepository
	userRepo     *repository.UserRepository
	tokenService *TokenService
	logger       *zap.Logger
	rpID         string
	rpName       string
}

// NewPasskeyService 创建一个新的 PasskeyService。
func NewPasskeyService(
	passkeyRepo *repository.PasskeyRepository,
	userRepo *repository.UserRepository,
	tokenService *TokenService,
	logger *zap.Logger,
	rpID, rpName string,
) *PasskeyService {
	return &PasskeyService{
		passkeyRepo:  passkeyRepo,
		userRepo:     userRepo,
		tokenService: tokenService,
		logger:       logger,
		rpID:         rpID,
		rpName:       rpName,
	}
}

// PasskeyRegisterOptions 包含 WebAuthn 注册质询数据。
type PasskeyRegisterOptions struct {
	Challenge              string
	RP                     PasskeyRPInfo
	User                   PasskeyUserInfo
	PubKeyCredParams       []PubKeyCredParam
	Timeout                int
	Attestation            string
	AuthenticatorSelection AuthenticatorSelectionInfo
}

// PasskeyRPInfo 是依赖方（relying party）信息。
type PasskeyRPInfo struct {
	Name string
	ID   string
}

// PasskeyUserInfo 是 WebAuthn 用户信息。
type PasskeyUserInfo struct {
	ID          string
	Name        string
	DisplayName string
}

// PubKeyCredParam 是一个公钥凭据参数。
type PubKeyCredParam struct {
	Type string
	Alg  int
}

// AuthenticatorSelectionInfo 配置认证器要求。
type AuthenticatorSelectionInfo struct {
	AuthenticatorAttachment string
	ResidentKey             string
	UserVerification        string
}

// PasskeyAuthOptions 包含认证质询数据。
type PasskeyAuthOptions struct {
	Challenge        string
	RPID             string
	Timeout          int
	UserVerification string
	AllowCredentials []AllowCredentialInfo
}

// AllowCredentialInfo 是用户可用于认证的一个凭据。
type AllowCredentialInfo struct {
	ID         string
	Type       string
	Transports []string
}

// GetRegisterOptions 生成 WebAuthn 注册质询选项。
func (s *PasskeyService) GetRegisterOptions(ctx context.Context, userID string) (*PasskeyRegisterOptions, *apperrors.AppError) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil || user == nil {
		return nil, apperrors.ErrUserNotFound
	}

	challenge, err := generateChallenge()
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to generate challenge", err)
	}

	userIDBytes, err := uuid.Parse(userID)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInvalidCredentials, "invalid user id", err)
	}

	return &PasskeyRegisterOptions{
		Challenge: challenge,
		RP: PasskeyRPInfo{
			Name: s.rpName,
			ID:   s.rpID,
		},
		User: PasskeyUserInfo{
			ID:          base64.RawURLEncoding.EncodeToString(userIDBytes[:]),
			Name:        user.Username,
			DisplayName: user.Username,
		},
		PubKeyCredParams: []PubKeyCredParam{
			{Type: "public-key", Alg: -7},   // ES256
			{Type: "public-key", Alg: -257}, // RS256
		},
		Timeout:     60000,
		Attestation: "none",
		AuthenticatorSelection: AuthenticatorSelectionInfo{
			AuthenticatorAttachment: "platform",
			ResidentKey:             "required",
			UserVerification:        "preferred",
		},
	}, nil
}

// VerifyRegistration 处理 WebAuthn 的 attestation 响应。
func (s *PasskeyService) VerifyRegistration(ctx context.Context, userID string, credentialData map[string]interface{}) *apperrors.AppError {
	rawID, _ := credentialData["rawId"].(string)
	attestationType, _ := credentialData["type"].(string)

	if rawID == "" {
		return apperrors.New(apperrors.ErrCodeInvalidCredentials, "missing credential ID")
	}

	transportsJSON, _ := json.Marshal(credentialData["transports"])
	flagsJSON, _ := json.Marshal(credentialData["flags"])
	authJSON, _ := json.Marshal(credentialData["authenticator"])

	uid, err := uuid.Parse(userID)
	if err != nil {
		return apperrors.Wrap(apperrors.ErrCodeInvalidCredentials, "invalid user id", err)
	}

	cred := &model.PasskeyCredential{
		UserID:          uid,
		CredentialID:    rawID,
		PublicKey:       "",
		AttestationType: attestationType,
		Transports:      string(transportsJSON),
		Flags:           string(flagsJSON),
		Authenticator:   string(authJSON),
	}

	if err := s.passkeyRepo.Create(ctx, cred); err != nil {
		return apperrors.Wrap(apperrors.ErrCodeDuplicateEntry, "failed to save credential", err)
	}

	s.logger.Info("passkey registered",
		zap.String("user_id", userID),
		zap.String("credential_id", rawID),
	)

	return nil
}

// GetAuthOptions 生成 WebAuthn 认证质询选项。
func (s *PasskeyService) GetAuthOptions(ctx context.Context) (*PasskeyAuthOptions, *apperrors.AppError) {
	challenge, err := generateChallenge()
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeOperationDenied, "failed to generate challenge", err)
	}

	return &PasskeyAuthOptions{
		Challenge:        challenge,
		RPID:             s.rpID,
		Timeout:          60000,
		UserVerification: "preferred",
	}, nil
}

// VerifyAuth 处理 WebAuthn 的 assertion 响应，成功时返回 token。
func (s *PasskeyService) VerifyAuth(ctx context.Context, assertionData map[string]interface{}) (*LoginServiceResult, *apperrors.AppError) {
	rawID, _ := assertionData["rawId"].(string)
	if rawID == "" {
		return nil, apperrors.ErrInvalidCredentials
	}

	cred, err := s.passkeyRepo.FindByCredentialID(ctx, rawID)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.ErrCodeInvalidCredentials, "database error", err)
	}
	if cred == nil {
		return nil, apperrors.ErrInvalidCredentials
	}

	user, err := s.userRepo.FindByID(ctx, cred.UserID.String())
	if err != nil || user == nil {
		return nil, apperrors.ErrUserNotFound
	}
	if !user.IsActive() {
		return nil, apperrors.ErrAccountDisabled
	}

	tokens, appErr := s.tokenService.IssueTokens(ctx, user)
	if appErr != nil {
		return nil, appErr
	}

	_ = s.userRepo.UpdateLoginSuccess(ctx, user.ID.String())

	s.logger.Info("passkey authentication successful",
		zap.String("user_id", user.ID.String()),
	)

	return &LoginServiceResult{
		Tokens: tokens,
		User:   userToInfo(user),
	}, nil
}

func generateChallenge() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
