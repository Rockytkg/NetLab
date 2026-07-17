# Coding Style Guide

> 此文件定义团队编码规范，所有 LLM 工具在修改代码时必须遵守。
> 提交到 Git，团队共享。

## General
- Prefer small, reviewable changes; avoid unrelated refactors.
- Keep functions short (<50 lines); avoid deep nesting (≤3 levels).
- Name things explicitly; no single-letter variables except loop counters.
- Handle errors explicitly; never swallow errors silently.

## Language-Specific

### TypeScript / React (NetLab-frontend)
- 遵循 `NetLab-frontend/CLAUDE.md` 前端开发宪法（组件模式、CSS 架构、design tokens）。
- `pnpm lint`（oxlint）零警告；提交前跑 `pnpm check`（i18n + lint + build）。
- 所有用户可见文案必须走 i18next（zh-CN / en-US 双语），禁止硬编码字符串。
- 状态管理用 Zustand store；跨组件共享状态不放在组件本地 state。

### Go (NetLab-backend)
- 严格分层：Handler（HTTP 解析）→ Service（业务逻辑）→ Repository（GORM/Redis）→ DB。Handler 不写业务逻辑。
- 错误使用 `pkg/apperrors` 类型化错误（带 i18n code）；响应统一走 `pkg/response` envelope。
- `make lint`（golangci-lint）通过；新逻辑需 `make test` 覆盖。
- 面向用户的消息需在 `messages/zh-CN.json` 与 `messages/en-US.json` 同步登记。

## Git Commits
- Conventional Commits, imperative mood.
- Atomic commits: one logical change per commit.

## Testing
- Every feat/fix MUST include corresponding tests.
- Coverage must not decrease.
- Fix flow: write failing test FIRST, then fix code.

## Security
- Never log secrets (tokens/keys/cookies/JWT).
- Validate inputs at trust boundaries.
