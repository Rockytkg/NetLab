# 《NetLab 网络管理中心前端架构设计方案》

> **版本**: v2.0  
> **日期**: 2026-07-14  
> **设计系统**: Ant Design 6.x  
> **当前技术栈**: React 19 + TypeScript 6 + Vite 8 + Zustand 5 + React Router 7  
> **适用范围**: NetLab 网络管理中心前端

---

## 1. 产品定位

NetLab 的目标是成为真实网络环境中的综合运维管理平台。平台面向企业网络团队，统一纳管路由器、交换机、防火墙、负载均衡、无线设备和服务器等基础设施，并集成：

- SNMP 指标采集与接口监控
- Syslog 日志汇聚、检索、归档和告警联动
- RADIUS 认证、授权、计费和审计
- 设备资产、站点、分组、厂商型号和连接关系管理
- 告警策略、通知工作流和运维模板

---

## 2. 核心用户旅程

```
用户登录 → 运维概览
    ├── 查看设备资产、在线率、告警、SNMP 覆盖率
    ├── 进入设备分组 → 筛选站点/状态/告警 → 打开设备详情
    ├── 进入设备资产 → 纳管真实设备 → 配置 SNMP / Syslog / RADIUS
    ├── 进入设备拓扑 → 查看真实设备连接关系与链路状态
    ├── 进入监控与日志 → 检索指标、日志、认证审计事件
    └── 进入运维模板 → 复用巡检、告警、通知、审计策略
```

---

## 3. 信息架构

| 区域 | 路由 | 职责 |
|------|------|------|
| 运维概览 | `/dashboard` | 展示设备总量、在线设备、告警数量、SNMP 覆盖率和关键能力卡片 |
| 设备分组 | `/device-groups` | 按站点、业务域或管理域维护设备分组，支持状态筛选和批量监控操作 |
| 设备资产 | `/device-library` | 真实设备资产台账，维护厂商、型号、管理 IP、协议接入状态 |
| 设备拓扑 | `/devices/:deviceId/topology` | 展示真实设备关系、链路状态、接口连接和告警上下文 |
| 监控与日志 | `/observability` | 汇聚 SNMP 指标、Syslog 日志、RADIUS 审计和告警策略 |
| 运维模板 | `/operations-templates` | 管理巡检模板、告警模板、通知模板和审计策略模板 |
| 系统设置 | `/settings/**` | 用户、认证、安全、SMTP、OAuth、备案等系统配置 |

旧路由仅作兼容重定向：

```
/labs → /device-groups
/lab/:deviceId → /devices/:deviceId/topology
/lab/:deviceId/monitor → /observability
/monitor → /observability
/templates/** → /operations-templates/**
```

---

## 4. 前端领域模型

核心类型位于 `src/types/operations.ts`：

- `DeviceStatus`: `online | offline | warning | critical`
- `DeviceGroup`: 设备分组，包含设备数量、在线数量、告警数量、资源利用率和更新时间
- `ManagedDevice`: 真实设备资产，包含厂商型号、管理 IP、站点、SNMP/Syslog/RADIUS 接入状态
- `DeviceTopologyNode` / `DeviceTopologyLink`: 真实设备拓扑节点与链路
- `OperationsFilter`: 设备分组和资产列表的筛选条件

状态管理位于 `src/stores/operationsStore.ts`：

- `deviceGroups`
- `devices`
- `filter`
- `selectedGroupIds`
- `activeGroupId`
- `activeDeviceId`

该 store 只保存当前前端工作集和筛选状态；设备资产、监控指标、日志和审计数据以服务端为准。

---

## 5. i18n 架构

当前命名空间：

| Namespace | 用途 |
|-----------|------|
| `common` | 通用按钮、错误、主题、语言、全局提示 |
| `login` | 登录、注册、找回密码、Passkey、OAuth、两步验证 |
| `menu` | 主导航与菜单 |
| `operations` | 设备分组、资产、拓扑、SNMP、Syslog、RADIUS、告警、运维模板 |
| `settings` | 系统与账户设置 |

旧 `lab` 与 `topology` namespace 已迁移到 `operations`。新增运维功能时必须使用 `operations`，不得重新引入旧实验室语义。

---

## 6. Ant Design 组件策略

NetLab 是高频运维工具，界面应偏安静、密集、可扫描：

- 页面骨架：`Layout` + `Sider` + `Header` + `Content`
- 响应式：`Grid.useBreakpoint()` 和 `Flex`
- 数据视图：`Table`、`Statistic`、`Card`、`Descriptions`、`Tag`
- 筛选与配置：`Form`、`Input.Search`、`Select`、`DatePicker.RangePicker`、`Drawer`
- 状态反馈：`Alert`、`Result`、`Progress`、`Badge`
- 监控趋势：后续引入 `@ant-design/charts` 或等价图表库，并消费 Ant Design token

设计原则：

- 运维数据优先，不做营销式大卡片堆叠
- 表格、筛选器、状态标签应稳定紧凑
- 告警和异常必须具备清晰视觉层级
- 移动端以查看和处理轻量告警为主，复杂配置引导到桌面端

---

## 7. 分阶段路线图

| 阶段 | 范围 |
|------|------|
| Phase 1 | 完成认证、布局、主题、i18n、网络管理中心信息架构、登录页品牌迁移 |
| Phase 2 | 设备资产台账、设备分组、设备详情、SNMP/Syslog/RADIUS 接入配置 |
| Phase 3 | SNMP 指标采集、接口监控、Syslog 接入检索、RADIUS 审计事件 |
| Phase 4 | 告警策略、通知联动、运维模板、响应式运维工作台、权限细化 |

---

## 8. 迁移约束

- 不再新增旧实验/画布领域类型；统一使用 `operationsStore` 与 `DeviceGroup`、`ManagedDevice`、`DeviceTopologyNode`。
- 不再新增 `/lab`、`/labs`、`/templates` 作为主功能路由，只允许兼容重定向。
- 不再新增旧实验/画布 i18n 文件；统一使用 `operations.json`。
- 所有新功能必须以真实设备、协议接入、监控日志、认证审计或告警模板为业务对象。
- 页面目录优先使用 `device-groups`、`device-library`、`devices`、`observability`、`operations-templates`。

---

## 9. 后续落地建议

1. 设计后端设备资产模型：device、site、group、credential、protocol_config。
2. 设计 SNMP 采集任务模型：poller、oid_profile、metric_sample、interface_metric。
3. 设计 Syslog 接入模型：receiver、log_event、parser_rule、retention_policy。
4. 设计 RADIUS 审计模型：auth_event、accounting_session、policy_result。
5. 设计告警模型：alert_rule、alert_event、notification_channel、incident。
6. 将 `operationsStore` 接入真实 API，替换当前空数据占位。
