package model

// PasskeyCredential 存储 WebAuthn 凭证数据。
//
// Credential 字段以 JSON 形式完整保存 go-webauthn 的凭证记录
// （含公钥、签名计数器、认证器信息与备份标志），认证时据此进行
// 签名校验与防重放。CredentialID 为凭证 ID 的 base64url 编码，
// 用于按凭证快速定位。Passkey 现在与 OAuth 统一存储在 auth_bindings 表中。
type PasskeyCredential = AuthBinding
