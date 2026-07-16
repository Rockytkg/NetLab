# NetLab

> 面向真实网络设备的综合运维管理平台，定位为企业网络管理中心。

NetLab 采用 **React 前端 + Go 后端** 架构，目标是统一纳管路由器、交换机、防火墙、负载均衡等真实网络设备，并逐步集成 SNMP 指标采集、Syslog 日志汇聚、RADIUS 认证审计、告警策略与运维视图，帮助网络团队完成资产管理、状态监控、日志分析和安全审计。

---

## ✨ 功能特性

- 🔐 **完整的认证体系**：登录 / 注册 / 找回密码 / Passkey（WebAuthn）/ OAuth 第三方登录 / 图形验证码
- 🌓 **主题与国际化**：明暗主题切换，中英文（zh-CN / en-US）双语支持
- 🧭 **设备资产管理**：真实网络设备、站点、分组、厂商型号和连接关系统一维护
- 📈 **SNMP 监控**：采集设备可用性、接口流量、CPU、内存和关键 OID 指标
- 🧾 **Syslog 汇聚**：集中接收、检索和分析设备日志，联动告警与审计
- 🛡️ **RADIUS 集成**：统一认证、授权和审计网络访问行为
- 🔒 **安全传输**：前后端预共享密钥加密（AES-256-CBC）+ HMAC-SHA256 请求签名
- ⚡ **速率限制**：多级 IP 限流保护，防止暴力攻击

---

## 🏗️ 技术栈

| 层次 | 技术栈 | 目录 |
|------|--------|------|
| 前端 | React 19 · TypeScript 6 · Ant Design 6.x · Vite 8 · Zustand 5 · React Router 7 · i18next | `NetLab-frontend/` |
| 后端 | Go 1.25 · Gin · GORM · PostgreSQL · Redis · JWT · Zap | `NetLab-backend/` |

---

## 📁 项目结构

```
NetLab/
├── CLAUDE.md                    # 项目开发指引（根级）
├── .gitignore                   # Git 忽略规则
├── NetLab-frontend/             # React 单页应用
│   ├── src/
│   │   ├── components/          # layout/、auth/、common/ 组件
│   │   ├── pages/               # login/、dashboard/、device-groups/、devices/、observability/、settings/ 等页面
│   │   ├── router/              # 单文件路由配置
│   │   ├── stores/              # Zustand 状态管理（appStore、authStore、operationsStore）
│   │   ├── services/            # Axios 实例 + API 服务
│   │   ├── hooks/               # useAuth、usePasskey、useI18n 等
│   │   ├── i18n/                # i18next 初始化 + 中英文语言包
│   │   ├── types/               # TypeScript DTO 定义
│   │   └── utils/               # crypto.ts、token.ts、constants.ts 等工具
│   └── docs/                    # 面向 AI 的 API 文档、UI 设计方案
└── NetLab-backend/              # Go API 服务
    ├── main.go                  # 入口：配置 → DB/Redis → 仓储 → 服务 → 处理器 → 路由
    ├── config/                  # 基于 Viper 的环境配置
    ├── internal/
    │   ├── router/              # Gin 路由（带限流分组）
    │   ├── middleware/          # 认证、CORS、加解密、i18n、限流、签名等中间件
    │   ├── handler/             # HTTP 处理器
    │   ├── service/             # 业务逻辑层
    │   ├── repository/          # 数据访问层（GORM + Redis）
    │   ├── model/               # GORM 模型
    │   ├── dto/                 # 请求/响应 DTO
    │   └── database/            # PostgreSQL + Redis 连接与自动迁移
    ├── pkg/                     # jwt、captcha、crypto、email、response 等通用包
    ├── migrations/              # SQL 迁移文件
    └── docker-compose.yml       # PostgreSQL 16 + Redis 7
```

---

## 🚀 快速开始

### 环境要求

- Node.js（配合 [pnpm](https://pnpm.io/)）
- Go 1.25+
- Docker & Docker Compose（用于 PostgreSQL 与 Redis）

### 1. 启动基础设施（PostgreSQL 16 + Redis 7）

```bash
cd NetLab-backend
make docker-up
```

PostgreSQL 运行于 `5432` 端口（用户：`netlab`，数据库：`netlab`），Redis 运行于 `6379` 端口。

### 2. 启动后端

```bash
cd NetLab-backend
cp .env.example .env   # 按需修改配置
make dev               # 使用 air 热重载
```

> `debug` 模式下后端会自动迁移 GORM 模型并初始化默认 OAuth 配置。

### 3. 启动前端

```bash
cd NetLab-frontend
cp .env.example .env.local   # 配置 API 地址与预共享密钥
pnpm install
pnpm dev                     # 默认 http://localhost:5173
```

---

## 🛠️ 常用命令

### 前端（`NetLab-frontend/`）

```bash
pnpm dev              # 启动 Vite 开发服务器
pnpm build            # 生产构建
pnpm lint             # oxlint 检查（要求零警告）
pnpm preview          # 预览生产构建
pnpm i18n:check       # 运行 i18n 审计脚本
pnpm check            # 完整检查：i18n + lint + build
```

### 后端（`NetLab-backend/`）

```bash
make build            # 编译到 bin/netlab-server
make run              # 编译并运行
make dev              # air 热重载开发
make test             # go test ./... -v -cover
make test-race        # 竞态检测测试
make lint             # golangci-lint 检查
make swagger          # 根据注解生成 Swagger 文档
make docker-up        # 启动 PostgreSQL + Redis
make docker-down      # 停止基础设施容器
make migrate          # 应用初始 SQL 迁移
```

---

## 🔗 前后端约定

### 预共享密钥（前后端必须一致）

以下三个环境变量必须在前后端保持一致：

| 前端（.env.local） | 后端（.env） | 用途 |
|--------------------|--------------|------|
| `VITE_AUTH_PRESHARED_KEY` | `AUTH_PRESHARED_KEY` | 密码字段 AES-256-CBC 加密 |
| `VITE_AUTH_SIGNATURE_KEY` | `AUTH_SIGNATURE_KEY` | HMAC-SHA256 请求签名 |
| `VITE_AUTH_SIGNATURE_SALT` | `AUTH_SIGNATURE_SALT` | 签名载荷盐值 |

### API 基础地址

前端将 `/api` 代理到后端。在 `.env.local` 中设置 `VITE_API_BASE_URL`（默认 `http://localhost:8080/api`）。

### 统一响应结构

所有接口返回 `{ code: number, data: T, message: string }`，成功码为 `0` 或 `200`。前端 Axios 拦截器会自动解包，服务函数直接拿到 `data`。

---

## 🔐 认证流程概览

| 类型 | 端点示例 | 说明 |
|------|----------|------|
| 预共享密钥端点 | `/auth/login`、`/auth/register`、`/auth/reset-password` | 客户端加密敏感字段并签名，后端中间件解密校验后再进入处理器 |
| 公开端点 | `/auth/refresh`、`/auth/captcha`、`/auth/send-code` | 可选 JWT，携带有效令牌则附加用户信息 |
| 认证端点 | `/auth/userinfo`、`/auth/logout` | `RequireAuth` 强制校验 JWT + 黑名单 |

- 访问令牌有效期 15 分钟，刷新令牌 7 天。
- 前端在过期前 5 分钟主动刷新，并在 401 时排队重试以避免并发刷新风暴。

---

## 📌 开发路线图

| 阶段 | 状态 | 范围 |
|------|------|------|
| Phase 1 | ✅ 已完成 | 布局框架、主题、路由、i18n、认证体系、仪表盘设备分组列表、占位页面 |
| Phase 2 | 🔲 规划中 | 设备资产台账、站点/分组管理、设备详情、真实网络拓扑视图 |
| Phase 3 | 🔲 规划中 | SNMP 采集、指标趋势、接口监控、Syslog 接入与检索 |
| Phase 4 | 🔲 规划中 | RADIUS 认证审计、告警策略、通知联动、响应式运维工作台 |

---

## 📄 许可证

本项目遵循仓库中的许可证声明。

---

> 更详细的开发规范请参阅根目录 `CLAUDE.md` 及前端 `NetLab-frontend/CLAUDE.md`。
