# 《网络模拟器前端界面改造设计方案》

> **版本**: v1.0  
> **日期**: 2026-07-06  
> **设计系统**: Ant Design 6.x  
> **当前技术栈**: React 19 + TypeScript 6 + Vite 8 + Zustand 5 + React Router 7  
> **适用范围**: NetLab 网络模拟器全前端界面

---

## 一、产品分析

### 1.1 产品定位与核心场景

NetLab 是一款类 EVE-NG 的网络模拟器，面向网络工程师、安全研究员及 IT 教育工作者，提供虚拟网络拓扑的构建、配置、运行与监控能力。其核心功能场景包括：

| 场景 | 描述 | 用户任务 |
|------|------|---------|
| **拓扑画布编辑** | 在画布上拖放、连接、布局网络设备 | 构建网络拓扑图，调整节点位置与连线 |
| **设备管理** | 管理路由器、交换机、防火墙等设备模板 | 浏览设备库，拖拽设备至画布，配置设备参数 |
| **网络连接配置** | 配置设备间的链路属性 | 设置接口、VLAN、IP 地址、带宽等 |
| **实验室会话管理** | 创建、启动、停止、保存实验环境 | 管理多个实验项目，控制运行状态 |
| **设备状态监控** | 实时查看设备 CPU、内存、端口状态 | 监控运行中设备的健康状态与流量 |

### 1.2 核心用户旅程

```
用户登录 → 仪表盘（实验室列表）
    ├── 新建实验室 → 选择模板 → 进入拓扑编辑器
    │       ├── 从设备面板拖拽设备至画布
    │       ├── 连接设备端口（拖拽连线）
    │       ├── 点击设备 → 打开配置面板 → 配置参数
    │       └── 保存拓扑 → 启动实验室
    └── 打开已有实验室 → 进入运行监控
            ├── 查看设备实时状态
            ├── 打开设备 Console/终端
            └── 停止/保存实验室
```

### 1.3 与 Ant Design 规范的差距分析

基于对 EVE-NG 经典界面的理解，当前 UI 模式存在以下与 Ant Design 设计价值观的差距：

| 维度 | 现状问题 | Ant Design 期望 |
|------|---------|----------------|
| **Natural（自然）** | 操作入口分散，右键菜单与工具栏功能重叠，用户认知路径不统一 | 交互行为应遵循操作系统惯例，减少意外 |
| **Certain（确定性）** | 设备状态缺乏明确的视觉反馈，运行中/停止/错误状态区分度不足 | 所有状态变化应有清晰的视觉信号 |
| **Meaningful（有意义）** | 信息层级扁平，工具栏图标密集堆叠，缺乏主次操作区分 | 视觉强调应服务于功能优先级 |
| **Growing（成长性）** | 界面采用固定布局，扩展新功能时布局容易破碎 | 采用弹性布局体系，支持功能持续叠加 |
| **布局** | 未采用标准化 Layout 组件，自定义 CSS 散落 | 使用 Layout + Sider + Content 标准化架构 |
| **组件一致性** | 控件样式不统一，表单、表格、弹窗风格各异 | 统一使用 antd 基础组件，通过 ConfigProvider 全局管控 |
| **响应式** | 基本不考虑小屏适配 | Grid 栅格 + 断点系统实现多屏适配 |

### 1.4 当前代码库现状

基于对 NetLab 现有代码的深度扫描，项目已具备以下基础设施，本设计方案将在此基础上增量改造：

| 已有能力 | 现状 | 改造方向 |
|---------|------|---------|
| **Layout 骨架** | `MainLayout` 已实现 Header(64px) + Sider(232px) + Content，支持折叠与移动端 Drawer 降级 | 保留布局架构，将 Sider 内容从"管理系统菜单"重构为"网络模拟器功能导航" |
| **ConfigProvider 主题** | 已配置 `colorPrimary: #1677FF`, `borderRadius: 6/8`, `fontSize: 14`，以及 Layout/Card/Table/Menu 组件令牌 | 扩展令牌以覆盖 Drawer、Tag、Segmented 等新增组件的样式 |
| **全局 CSS** | `index.css` 含 `netlab-` 前缀工具类，引用 antd CSS 变量（如 `var(--ant-color-bg-layout)`） | 保留现有工具类，新增拓扑画布、设备面板等专用样式类 |
| **行内 useToken() 样式** | 各组件广泛使用 `theme.useToken()` 获取设计令牌 | 保持此模式用于组件级定制 |
| **i18n** | zh-CN / en-US 双语言，含 common/login/menu 三个命名空间 | 扩展命名空间：`topology`（拓扑）、`device`（设备）、`lab`（实验室） |
| **路由** | React Router 7，含 AuthGuard + MainLayout 包裹的认证路由 | 新增 `/lab/:id`、`/lab/:id/monitor`、`/templates` 等路由 |
| **状态管理** | Zustand + persist（appStore、authStore） | 新增 `labStore`、`topologyStore` 管理拓扑编辑器状态 |
| **PWA** | vite-plugin-pwa 已配置（manifest 主题色为 `#667eea`） | 统一 PWA 主题色为 antd 主色 `#1677FF` |

---

## 二、改造策略

### 2.1 分阶段计划

#### 第一阶段：基础框架搭建（优先级 P0）

- 建立全局 Layout 骨架（Header + Sider + Content）
- 配置 ConfigProvider 全局主题令牌
- 搭建路由框架与导航菜单体系
- 实现实验室列表仪表盘页面
- **目标**: 建立可用的产品外壳，后续所有页面在此框架内开发

#### 第二阶段：核心工作区改造（优先级 P0）

- 拓扑画布编辑器（核心交互）
- 设备面板/组件库（侧边栏）
- 设备配置抽屉面板
- 画布工具栏（浮动式）
- **目标**: 完成最核心的网络拓扑编辑体验

#### 第三阶段：实验管理完善（优先级 P1）

- 实验室创建/编辑向导
- 设备模板管理
- 实验室运行监控面板
- 设备 Console 终端界面
- **目标**: 覆盖完整的实验生命周期

#### 第四阶段：体验打磨（优先级 P2）

- 响应式适配与移动端降级
- 暗色主题支持（通过 `darkAlgorithm`）
- 键盘快捷键体系
- 动画与微交互增强
- 国际化框架
- **目标**: 达到产品级交付质量

### 2.2 设计决策原则

每次设计决策以 Ant Design 四大价值观为裁决依据：

1. **Natural 优先**: 当存在多种交互方案时，选择最符合操作系统惯例的方案
2. **Certain 优先**: 任何状态变化必须有可感知的视觉反馈
3. **Meaningful 优先**: 移除纯装饰性元素，每个视觉元素都服务于信息传达
4. **Growing 前提**: 所有布局选择必须支持未来 2-3 个版本的扩展而不需重构

---

## 三、整体布局架构

### 3.1 布局方案选择

**采用方案: Header-Sider 模式（应用站点型）**

```
┌──────────────────────────────────────────────────────────┐
│  Header (64px)                                  🔔 👤 ⚙  │
│  Logo   实验室 ▼   项目 ▼   帮助                        │
├────────────┬─────────────────────────────────────────────┤
│  Sider     │  Content                                    │
│  (240px)   │                                             │
│            │  ┌───────────────────────────────────────┐  │
│  📁 设备库  │  │                                       │  │
│  📊 仪表盘  │  │        拓扑画布（核心工作区）           │  │
│  🧪 实验室  │  │                                       │  │
│  📋 模板   │  │                                       │  │
│  ⚙ 设置   │  └───────────────────────────────────────┘  │
│            │                                             │
│  ◀ 收起   │  ┌─ 设备面板（可收起） ─────────────────┐  │
│            │  │ 🔀 路由器  🔁 交换机  🛡 防火墙      │  │
│            │  └────────────────────────────────────┘  │
├────────────┴─────────────────────────────────────────────┤
│  Footer (48px)  状态栏: 实验运行中 | 节点: 12 | 在线: 10 │
└──────────────────────────────────────────────────────────┘
```

**选择理由:**

- **Header-Sider 模式** 是 Ant Design 推荐的企业级应用布局（引用 Layout 文档："Generally, the mainnav is placed on the left side of the page, and the secondary menu is placed on the top of the working area"）
- 侧边栏承载一级功能导航（实验室、设备库、模板、设置），顶部导航承载全局操作（新建实验、用户菜单、通知）
- 拓扑画布作为 Content 核心区，获得最大可视面积
- Sider 支持折叠（`collapsible`），在编辑拓扑时可收起以最大化画布空间

### 3.2 布局尺寸规范

| 区域 | 尺寸 | 依据 |
|------|------|------|
| Header 高度 | 64px | Ant Design 标准一级导航高度 |
| Sider 宽度 | 240px（展开）/ 80px（折叠） | `200 + 8n` 公式，取 n=5 |
| Footer 高度 | 48px | 状态栏，紧凑展示 |
| Content 内边距 | 24px | `lg` 间距级别 |
| 设备面板高度 | 200px（可拖拽调整） | 底部面板，不占用核心画布区 |

### 3.3 响应式策略

采用 Ant Design Grid 24 栏栅格系统，断点适配策略：

| 断点 | 宽度 | 布局适配 |
|------|------|---------|
| `xxl` | ≥1600px | 完整布局：Header + Sider(展开) + Content + 底部设备面板 |
| `xl` | ≥1200px | 标准布局：Header + Sider(展开) + Content |
| `lg` | ≥992px | Sider 默认折叠，设备面板隐藏，按需展开 |
| `md` | ≥768px | Sider 变为浮动抽屉（Drawer），设备面板以模态框呈现 |
| `sm` | ≥576px | 简化模式：隐藏非核心功能区 |
| `xs` | <576px | 仅可查看拓扑（只读模式），编辑功能引导至桌面端 |

**关键实现要点:**
- Sider 配置 `breakpoint="lg"` 和 `collapsedWidth={80}`，在宽度低于 992px 时自动折叠
- 拓扑画布采用 `ResizeObserver` 动态适配容器尺寸
- 设备面板在 `md` 以下断点转为 `Drawer` 组件

---

## 四、核心界面设计方案

### 4.1 拓扑画布（核心工作区）

#### 设计思路

拓扑画布是产品的核心交互区域，设计优先级最高。遵循 "Natural" 原则，交互模式参照主流图形编辑工具（Figma、Draw.io），用户凭直觉即可操作。画布需最大化可视区域，所有辅助控件以浮动或可收起方式存在。

**核心设计目标:**
- 画布占据 Content 区域的 100%，无内边距浪费
- 工具栏采用浮动条，悬停在画布左上角，半透明背景，不遮挡拓扑内容
- 迷你地图（Minimap）浮动在右下角，帮助在大拓扑中导航
- 画布支持无限滚动画布（通过鼠标中键拖拽或触控板手势）
- 缩放控件浮动在右下角，与 Minimap 相邻

#### 组件选型

| 功能 | 组件 | 说明 |
|------|------|------|
| 画布容器 | 自定义 `<canvas>` 或 SVG 层 | 使用 HTML5 Canvas 或 SVG 渲染拓扑图；推荐 Konva.js 或自研渲染层 |
| 浮动工具栏 | `FloatButton.Group` + `Tooltip` | 浮动在画布左上角，主要操作（选择、移动、连线、删除） |
| 缩放控件 | 自定义组件 | 配合 `Button` 的 `shape="circle"` + `MinusOutlined` / `PlusOutlined` |
| 迷你地图 | 自定义 Canvas 组件 | 浮动右下角，缩略图预览 |
| 右键菜单 | `Dropdown` | 设备/连线右键操作菜单 |
| 节点设备 | 自定义 SVG/HTML 组件 | 设备图标 + 名称标签，采用设备类型对应的预设颜色标签 |
| 连接线 | SVG `<path>` / `<line>` | 统一 2px 线宽，`#1677FF` 选中态，`#D9D9D9` 默认态 |
| 加载状态 | `Spin` | 大型拓扑加载时全画布遮罩 |

#### 样式说明

```
// 画布容器
.canvas-container {
  width: 100%;
  height: 100%;
  background: #FAFAFA;           // container 背景
  background-image: 
    linear-gradient(#F0F0F0 1px, transparent 1px),
    linear-gradient(90deg, #F0F0F0 1px, transparent 1px);
  background-size: 20px 20px;    // 网格点阵，辅助对齐
  cursor: grab;
  position: relative;
}

// 浮动工具栏
.canvas-toolbar {
  position: absolute;
  top: 16px;                     // md 间距
  left: 16px;
  display: flex;
  flex-direction: column;
  gap: 4px;                      // xs 间距
  background: rgba(255,255,255,0.92);
  backdrop-filter: blur(8px);    // glassmorphism 效果
  border: 1px solid #F0F0F0;
  border-radius: 8px;            // 表面圆角
  padding: 4px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.08);  // Card shadow tier
  z-index: 100;
}

// 选中设备节点
.device-node--selected {
  border: 2px solid #1677FF;     // 主色选中边框
  box-shadow: 0 0 0 4px rgba(22,119,255,0.12);  // 主色扩散光晕
}

// 设备节点默认态
.device-node {
  background: #FFFFFF;
  border: 1px solid #D9D9D9;
  border-radius: 8px;            // 表面圆角
  padding: 12px 16px;
  min-width: 80px;
  text-align: center;
  transition: all 0.1s ease;     // fast motion
  cursor: pointer;
}
.device-node:hover {
  border-color: #1677FF;
  box-shadow: 0 2px 8px rgba(22,119,255,0.15);
}

// 连接线默认
.connection-line {
  stroke: #D9D9D9;
  stroke-width: 2;
  fill: none;
  transition: stroke 0.1s ease;
}
.connection-line:hover {
  stroke: #1677FF;
  stroke-width: 2.5;
}
.connection-line--selected {
  stroke: #1677FF;
  stroke-width: 2.5;
}

// 迷你地图
.minimap {
  position: absolute;
  bottom: 16px;
  right: 16px;
  width: 200px;
  height: 150px;
  background: #FFFFFF;
  border: 1px solid #D9D9D9;
  border-radius: 6px;            // 控件圆角
  box-shadow: 0 2px 8px rgba(0,0,0,0.08);
  z-index: 100;
}
```

#### 交互规范

| 操作 | 行为 | 反馈 |
|------|------|------|
| 左键拖拽设备 | 移动设备节点 | 节点跟随光标，松手时吸附网格（20px grid snap） |
| 左键拖拽端口 → 另一端口 | 创建连线 | 拖拽时显示临时虚线，释放时创建连线并动画过渡 |
| 左键点击设备 | 选中设备 | 边框变为主色 + 光晕扩散（0.1s transition） |
| 双击设备 | 打开配置面板 | 右侧 Drawer 滑出（0.3s `motionEaseInOut`） |
| 右键设备/连线 | 弹出上下文菜单 | `Dropdown` 弹出，子项 hover 时背景变为 `#F5F5F5` |
| 鼠标中键拖拽 | 平移画布 | 光标变为 `grab` → `grabbing` |
| 滚轮 | 缩放画布 | 以光标位置为中心缩放，缩放比例实时显示 |
| Delete 键 | 删除选中元素 | `Modal.confirm` 确认弹窗（危险操作用 `okButtonProps={{ danger: true }}`） |
| Ctrl+Z / Ctrl+Y | 撤销/重做 | 底部状态栏短暂提示操作结果 |

---

### 4.2 设备面板/组件库

#### 设计思路

设备面板是用户构建拓扑的"建材市场"，需提供快速浏览、搜索、分类筛选能力，并支持拖拽至画布。在拓扑编辑场景下，设备面板应当可收起以最大化画布空间。采用底部可折叠面板设计（而非左侧），保持画布区域完整。

**核心设计目标:**
- 按设备类型分组（路由器、交换机、防火墙、服务器、终端等）
- 支持关键词搜索过滤
- 每类设备显示图标 + 名称 + 简要描述
- 支持拖拽至画布添加节点
- 面板高度可拖拽调整（默认 200px，可展开至 400px）

#### 组件选型

| 功能 | 组件 | 说明 |
|------|------|------|
| 面板容器 | 自定义可拖拽面板 | 基于 `Layout` 底部区域，搭配拖拽手柄调整高度 |
| 分类标签 | `Segmented` | 设备分类切换（全部/路由器/交换机/防火墙/…），Ant Design 6.0 推荐的分段控制器 |
| 搜索 | `Input.Search` | 带搜索图标的输入框，实时过滤设备列表 |
| 设备卡片列表 | `Card` + `List` (grid 模式) | 每项设备缩略图 + 名称，响应式列数 |
| 设备图标 | 自定义 SVG 图标 | 每种设备类型对应唯一图标，颜色取自预设调色板 |
| 拖拽源 | HTML5 Drag API | `draggable` 属性 + `onDragStart` 传递设备类型数据 |
| 空状态 | `Empty` | 无搜索结果时显示，"未找到匹配设备" |
| 面板折叠 | `Button` (icon only) | 面板顶部中央的折叠/展开手柄 |

#### 样式说明

```
// 设备面板容器
.device-panel {
  background: #FFFFFF;
  border-top: 1px solid #F0F0F0;
  padding: 0;
  transition: height 0.2s ease;  // mid motion for expand/collapse
}

// 面板头部（分类 + 搜索）
.device-panel__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 16px;             // sm 垂直间距
  border-bottom: 1px solid #F0F0F0;
}

// 拖拽手柄
.device-panel__handle {
  width: 32px;
  height: 4px;
  background: #D9D9D9;
  border-radius: 9999px;         // pill shape
  margin: 4px auto 0;
  cursor: ns-resize;
}

// 设备卡片（列表项）
.device-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 12px 8px;
  border: 1px solid #F0F0F0;
  border-radius: 6px;            // 控件圆角
  cursor: grab;
  transition: all 0.1s ease;     // fast
  user-select: none;
}
.device-card:hover {
  border-color: #1677FF;
  background: #E6F4FF;           // 与 Menu 选中项一致
}
.device-card:active {
  cursor: grabbing;
  transform: scale(0.95);
  box-shadow: 0 4px 12px rgba(0,0,0,0.15);  // 拖拽中 shadow
}

// 设备图标颜色映射（Ant Design 预设色板）
.device-icon--router     { color: #1677FF; }  // blue
.device-icon--switch     { color: #13C2C2; }  // cyan
.device-icon--firewall   { color: #F5222D; }  // red (预设色板)
.device-icon--server     { color: #722ED1; }  // purple
.device-icon--endpoint   { color: #52C41A; }  // green
.device-icon--cloud      { color: #2F54EB; }  // geekblue

// 搜索结果为空
.device-panel__empty {
  padding: 32px 0;
}
```

#### 交互规范

| 操作 | 行为 | 反馈 |
|------|------|------|
| 点击分类标签 | 切换设备列表 | `Segmented` 滑动切换动画（0.2s），列表内容渐隐再渐显 |
| 输入搜索词 | 实时过滤设备 | 300ms 防抖，匹配设备名称或描述 |
| 拖拽设备卡片 | 开始拖拽 | 卡片半透明（opacity: 0.6），光标变为 `grabbing`，生成拖拽预览图 |
| 拖拽至画布释放 | 在释放位置创建节点 | 节点以缩放动画（0.2s `motionEaseInOutBack`）出现在画布上 |
| 拖拽面板手柄上下 | 调整面板高度 | 高度实时跟随光标，松手时吸附到最近的预设高度（200px/300px/400px） |
| 点击面板折叠按钮 | 折叠/展开面板 | 面板高度过渡动画 0.2s |

---

### 4.3 设备配置面板

#### 设计思路

设备配置是网络模拟器的高频操作。采用右侧滑出式 `Drawer`，而非 `Modal`，因为用户在配置设备时可能需要同时查看拓扑画布上的其他设备信息——`Drawer` 不会完全遮挡画布，支持这种"对照配置"场景。

遵循 Ant Design 表单设计规范，配置表单按功能分区组织，使用 `Tabs` 或 `Collapse` 面板分组。复杂配置分层级展示，避免单页表单过长。

**核心设计目标:**
- 配置面板不遮挡拓扑画布核心区域
- 表单分区清晰，按逻辑分组（基本、网络、高级）
- 实时表单校验，错误即时提示
- 支持"应用"（不关闭面板）和"确定"（保存并关闭）两种提交模式

#### 组件选型

| 功能 | 组件 | 说明 |
|------|------|------|
| 配置面板容器 | `Drawer` | 右侧滑出，宽度 480px（xl 断点）或 100%（<md） |
| 内容分区 | `Tabs` | 基本配置 / 网络配置 / 高级配置 |
| 配置表单 | `Form` | `layout="vertical"` 垂直布局，适配 Drawer 窄空间 |
| 设备名称 | `Input` | 必填，maxLength=64，实时校验唯一性 |
| 设备类型显示 | `Tag` + 图标 | 只读标签，显示设备类型和图标，使用设备对应色 |
| 接口列表 | `Form.List` + `Table` | 动态增减网络接口，每行含接口名、IP、掩码、VLAN |
| 接口类型选择 | `Select` | 下拉选择接口类型（Ethernet/Serial/Loopback） |
| VLAN 配置 | `InputNumber` | 范围 1-4094 |
| 开关选项 | `Switch` | 启用/禁用功能开关 |
| 模板选择 | `Select` (searchable) | 从预设模板加载配置 |
| 启动配置文本 | `Input.TextArea` | 高级模式下的自由配置文本，等宽字体，`rows={12}` |
| 操作按钮 | `Button` + `Space` | "应用"（default）、"确定"（primary）、"取消" |
| 加载状态 | `Skeleton` | 初次加载设备详情时的骨架屏 |
| 校验反馈 | `Form.Item` 的 `validateStatus` | 实时显示字段校验状态图标 |

#### 样式说明

```
// Drawer 定制
.device-config-drawer {
  // 通过 ConfigProvider theme.components.Drawer 定制
}

// Drawer 内表单布局
.device-config-form {
  // Form layout="vertical" — 标签在上，输入框在下
  // labelCol/wrapperCol 不设置（垂直布局默认全宽）
}

// 接口列表表格（Form.List 内）
.interface-table {
  // 紧凑型表格，size="small"
  // 操作列：删除按钮（danger type, text variant）
}

// 表单操作按钮区
.config-form-actions {
  display: flex;
  justify-content: flex-start;
  gap: 8px;                      // sm 间距
  padding-top: 16px;             // md 间距
  border-top: 1px solid #F0F0F0;
  margin-top: 16px;
}

// 只读信息标签
.device-info-tag {
  // Tag 组件，device 类型对应颜色
  font-size: 12px;               // 标签字体
  border-radius: 4px;            // md 圆角
}
```

#### 交互规范

| 操作 | 行为 | 反馈 |
|------|------|------|
| 双击设备/点击"配置" | 打开 Drawer | Drawer 从右侧滑入（0.3s `motionEaseInOut`），画布左侧留出空间 |
| 切换 Tab | 切换配置分区 | Tab 下划线滑动过渡（0.2s），内容区淡入淡出 |
| 修改字段值 | 表单值变更 | `onChange` 触发校验（`validateTrigger: 'onChange'`），错误信息即时显示 |
| 添加接口 | 点击"+ 添加接口" | `Form.List.add()` 插入新行，自动滚动到新行 |
| 删除接口 | 点击删除按钮 | `Popconfirm` 二次确认（"确定移除此接口？"） |
| 点击"应用" | 保存配置但保持 Drawer 打开 | Button loading 状态 → `message.success("配置已应用")` |
| 点击"确定" | 保存并关闭 | Button loading → Drawer 滑出 → `message.success("配置已保存")` |
| 点击"取消" | 关闭 Drawer | 若有未保存修改，`Modal.confirm` 提示确认 |
| 表单校验失败 | 阻止提交 | `scrollToFirstError` 滚动到第一个错误字段，字段标红并显示错误信息 |

---

### 4.4 实验室列表/仪表盘

#### 设计思路

实验室列表是用户登录后的默认着陆页，需清晰展示所有实验室的状态、资源使用情况和关键操作入口。采用 Ant Design `Table` 组件展示数据，配合状态标签、进度条、操作按钮，实现高效的数据浏览与管理。

**核心设计目标:**
- 一屏展示尽可能多的实验室信息（信息密度优先，符合 Ant Design 14px 基础字号设计哲学）
- 状态一目了然：运行中（绿色）、已停止（灰色）、错误（红色）
- 快速筛选与搜索
- 支持批量操作（启动、停止、删除）

#### 组件选型

| 功能 | 组件 | 说明 |
|------|------|------|
| 页面标题区 | `Typography.Title` + `Button` | 标题"我的实验室" + 主操作"新建实验室"（primary） |
| 搜索与筛选栏 | `Input.Search` + `Select` + `DatePicker.RangePicker` | 搜索关键词 + 状态筛选 + 日期范围 |
| 数据表格 | `Table` | 列表展示，支持排序、分页、行选择 |
| 状态标签 | `Tag` | 运行中 `#52C41A`、已停止 `#BFBFBF`、错误 `#FF4D4F`、暂停 `#FAAD14` |
| 进度条 | `Progress` | 显示资源使用率（CPU/内存） |
| 行操作 | `Button` (type="link") + `Dropdown` | 主要操作：启动/停止/打开；更多操作：重命名/复制/导出/删除 |
| 批量操作栏 | `Alert` (type="info") | 选中行后出现，显示已选数量 + 批量操作按钮 |
| 分页 | Table 内置 `pagination` | 位置 `bottomCenter`，默认每页 20 条 |
| 新建向导 | `Modal` (多步) 或 `Steps` | 创建实验室的分步向导 |
| 空状态 | `Empty` | 无实验室时引导创建 |
| 加载 | `Spin` + `Skeleton` | 列表加载中 |
| 删除确认 | `Popconfirm` | 单行删除二次确认 |

#### 样式说明

```
// 页面容器
.dashboard-page {
  padding: 24px;                 // lg 间距
  background: #F5F5F5;          // 页面背景
  min-height: 100%;
}

// 页面标题区
.dashboard-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;          // lg 间距
}

// 筛选栏
.dashboard-filters {
  display: flex;
  gap: 16px;                     // md 间距
  margin-bottom: 16px;
  flex-wrap: wrap;
}

// 表格容器
.dashboard-table {
  background: #FFFFFF;
  border-radius: 8px;           // 表面圆角
  padding: 0;                   // Table 自带内边距
  // Table header: surface-container bg, title-md (14px/600)
}

// 状态标签（确保使用语义色）
.status-tag--running   { color: #52C41A; background: #F6FFED; border-color: #B7EB8F; }
.status-tag--stopped   { color: #8C8C8C; background: #FAFAFA; border-color: #D9D9D9; }
.status-tag--error     { color: #FF4D4F; background: #FFF2F0; border-color: #FFCCC7; }
.status-tag--paused    { color: #FAAD14; background: #FFFBE6; border-color: #FFE58F; }

// 批量操作 Alert
.dashboard-batch-bar {
  margin-bottom: 16px;
  // 使用 Ant Design Alert info 类型
}

// 空状态
.dashboard-empty {
  padding: 80px 0;
  // 配合 Empty 组件的 description 和 children（CTA 按钮）
}
```

#### 表格列定义

| 列名 | 数据字段 | 宽度 | 排序 | 筛选 | 说明 |
|------|---------|------|------|------|------|
| ☐ | selection | 48px | - | - | 复选框选择 |
| 实验室名称 | `name` | auto | ✓ | - | `ellipsis: true`，点击进入实验室 |
| 状态 | `status` | 100px | ✓ | ✓ | Tag 组件，预设 filter |
| 设备数量 | `nodeCount` | 90px | ✓ | - | 右对齐数字 |
| CPU 使用 | `cpuUsage` | 140px | ✓ | - | `Progress` 组件，`size="small"`，超过 80% 变红 |
| 内存使用 | `memUsage` | 140px | ✓ | - | `Progress` 组件，`size="small"` |
| 创建时间 | `createdAt` | 160px | ✓ | - | 格式化日期 |
| 更新时间 | `updatedAt` | 160px | ✓ | - | 格式化日期，默认降序排列 |
| 操作 | `actions` | 180px | - | - | 固定列 `fixed: 'end'` |

#### 交互规范

| 操作 | 行为 | 反馈 |
|------|------|------|
| 点击"新建实验室" | 打开创建向导 | `Modal` 弹出（0.3s），第一步选择模板 |
| 搜索框输入 | 实时过滤列表 | 300ms 防抖，表格数据过滤刷新 |
| 点击状态标签筛选 | 按状态过滤 | `filteredValue` 更新，表格重新渲染 |
| 点击实验室名称 | 进入实验室 | 路由跳转，加载实验室拓扑 |
| 点击行操作"启动" | 启动实验室 | Button loading → 成功后 Tag 变为绿色 + `message.success` |
| 勾选多行 | 批量选择 | 顶部出现 `Alert` 批量操作栏 |
| 批量删除 | 确认后删除 | `Modal.confirm({ title: "确认删除", content: "已选择 N 个实验室...", okButtonProps: { danger: true } })` |
| 表格排序 | 按列排序 | 表头排序图标变化，`onChange({ action: 'sort' })` |
| 分页切换 | 翻页 | `onChange({ action: 'paginate' })` |

---

### 4.5 全局导航

#### 设计思路

全局导航采用 Ant Design 推荐的 Header-Sider 模式，提供清晰的信息架构与层级导航。遵循 "Certain" 原则，当前所在位置需有明确的视觉标识（Menu 选中态）。

**导航信息架构:**

```
Header（全局导航栏）
├── Logo + 产品名称
├── 主菜单（一级）
│   ├── 工作台（默认）
│   ├── 实验室
│   ├── 模板市场
│   └── 帮助
└── 右侧工具区
    ├── 全局搜索 (Search)
    ├── 通知 (Badge + Bell icon)
    ├── 帮助 (QuestionCircle icon)
    └── 用户菜单 (Avatar + Dropdown)

Sider（侧边菜单 — 根据 Header 选中项变化）
├── 工作台下
│   ├── 📊 仪表盘
│   ├── 📋 最近实验
│   └── ⭐ 收藏的模板
├── 实验室下
│   ├── 🧪 我的实验室
│   ├── 📁 设备库
│   ├── 📈 运行监控
│   └── 📝 实验报告
├── 模板市场下
│   ├── 🔍 浏览模板
│   ├── 📤 我的上传
│   └── ⬇ 已安装
└── 帮助下
    ├── 📖 使用文档
    ├── 🎓 教程
    └── 💬 社区
```

#### 组件选型

| 功能 | 组件 | 说明 |
|------|------|------|
| 全局布局 | `Layout` | 顶层包裹组件 |
| 顶部导航 | `Layout.Header` + `Menu` (mode="horizontal") | 一级导航，`theme="dark"` |
| 侧边菜单 | `Layout.Sider` + `Menu` (mode="inline") | 二级导航，`theme="light"` |
| Logo | 自定义 SVG/图片 | 32×32px，左侧留 24px 间距 |
| 全局搜索 | `Input.Search` 或 `AutoComplete` | 搜索实验室、设备、模板、文档 |
| 通知 | `Badge` + `BellOutlined` | 未读通知数红点 |
| 用户头像 | `Avatar` + `Dropdown` | 下拉菜单：个人设置、许可证、退出登录 |
| 面包屑 | `Breadcrumb` | Content 区域顶部，显示当前页面路径 |

#### 样式说明

```
// Header
.global-header {
  height: 64px;                  // 标准高度
  padding: 0 24px;              // xl 水平间距
  background: #001529;          // dark theme headerBg
  display: flex;
  align-items: center;
  box-shadow: 0 1px 4px rgba(0,0,0,0.15);  // tertiary shadow
  z-index: 1000;
  position: sticky;
  top: 0;
}

// Logo 区域
.header-logo {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-right: 40px;
  color: #FFFFFF;
  font-size: 18px;
  font-weight: 600;              // 标题字重
}

// Header 一级菜单
.header-menu {
  // theme="dark"
  // 当前选中项: 背景色 #1677FF（colorblock emphasis）
  // 非选中项: 白色文字 rgba(255,255,255,0.65)
  flex: 1;
  border-bottom: none;
}

// 右侧工具区
.header-tools {
  display: flex;
  align-items: center;
  gap: 16px;                     // md 间距
}

// Sider
.global-sider {
  background: #FFFFFF;          // light theme
  border-right: 1px solid #F0F0F0;
  // 展开: 240px, 折叠: 80px
  // transition: all 0.2s ease
}

// Sider 菜单
.sider-menu {
  // theme="light"
  // 选中项: 背景 #E6F4FF + 主色文字（highlight match stick 样式）
  // 菜单项高度: 40px
  // 图标尺寸: 16px
  // 字体: 14px / 400（常规）
}

// 面包屑
.page-breadcrumb {
  padding: 16px 24px 0;
  // 最后一项为当前页面（主文本色 #1F1F1F，非链接）
  // 前面项为链接（主色 #1677FF）
}
```

#### 交互规范

| 操作 | 行为 | 反馈 |
|------|------|------|
| 点击 Header 菜单项 | 切换一级导航，Sider 菜单联动变化 | Header 菜单项高亮（colorblock），Sider 菜单项刷新 |
| 点击 Sider 菜单项 | 路由跳转到对应页面 | 菜单项变为选中态（`#E6F4FF` 背景 + 主色文字） |
| 点击 Sider 折叠按钮 | 折叠/展开侧边栏 | 过渡动画 0.2s，折叠后仅显示图标，hover 显示 Tooltip |
| 点击通知图标 | 展开通知面板 | `Popover` 弹出，显示最近 5 条通知，底部"查看全部"链接 |
| 点击用户头像 | 展开用户菜单 | `Dropdown` 弹出，菜单项 hover 背景 `#F5F5F5` |
| 全局搜索聚焦 | 展开搜索面板 | `AutoComplete` 下拉显示搜索结果，分组显示（实验室/设备/模板） |
| 窗口宽度 <992px | Sider 自动折叠 | `breakpoint="lg"` 触发，Sider 变为 `collapsedWidth={80}` |

---

### 4.6 实验室创建向导

#### 设计思路

创建实验室是一个多步骤的决策流程，采用 `Steps` + `Modal` 组合实现分步向导。每一步聚焦一个决策点，降低用户的认知负担。遵循 "Meaningful" 原则，每步仅呈现必要信息。

#### 组件选型

| 功能 | 组件 | 说明 |
|------|------|------|
| 向导容器 | `Modal` (width=720px) | 居中模态框，较大宽度容纳表单 |
| 步骤指示 | `Steps` (current step) | 置于 Modal 内容区顶部 |
| 步骤内容 | 条件渲染组件 | 根据 current step 切换内容 |
| 模板选择 | `Card` (grid) | 预设模板卡片网格，选中态蓝色边框 + 光晕 |
| 基本配置表单 | `Form` (layout="vertical") | 名称、描述、设备数量限制 |
| 资源配置表单 | `Form` + `Slider` + `InputNumber` | 总 CPU、总内存、存储空间 |
| 底部操作 | `Button` + `Space` | "上一步"（default）、"下一步"（primary）、"取消" |

#### 交互规范

| 步骤 | 内容 | 操作 |
|------|------|------|
| 1. 选择模板 | 预设模板卡片网格 + "空白实验"选项 | 点击卡片选中 → "下一步" |
| 2. 基本信息 | 实验室名称（必填）、描述、标签 | 填写表单 → "下一步" |
| 3. 资源配置 | CPU/内存/存储限制滑块 | 调整资源 → "创建实验室"（primary） |
| 完成 | 创建成功提示 + 操作入口 | "进入实验室" 或 "返回列表" |

---

### 4.7 运行监控面板

#### 设计思路

当实验室处于运行状态时，用户需要监控各设备的实时状态。采用 `Card` 网格布局展示设备状态卡片，配合 `Badge` 状态点和实时更新的 `Statistic` 数值。

#### 组件选型

| 功能 | 组件 | 说明 |
|------|------|------|
| 设备状态卡片 | `Card` + `Grid` (`List` grid) | 响应式网格，每卡片显示一个设备 |
| 在线指示灯 | `Badge` (status="processing") | 绿色脉冲表示在线，灰色表示离线，红色表示错误 |
| CPU/内存数值 | `Statistic` | 实时数值 + 单位 |
| 实时图表 | 自定义 Chart（ECharts/ReCharts） | CPU/内存趋势小图（sparkline） |
| 自动刷新 | `Switch` + 定时器 | 开启后每 5 秒刷新数据 |
| 悬浮信息 | `Tooltip` | 悬停设备卡片显示详细状态 |
| 设备操作 | `Dropdown` (右键) | 启动/停止/重启/打开 Console |
| 连接终端 | `Modal` (全屏) | Console 终端，等宽字体，黑底绿字 |

---

## 五、技术架构与核心库选型

### 5.0 选型方法论

网络模拟器前端的技术架构需同时满足三个维度的需求：**编辑构建**（拖拽、连线、配置）、**可视化监控**（实时状态、数据刷新）、**协作与持久化**（多人协同、离线存储）。单一库难以同时覆盖这三个维度，因此采用**分层选型、组合使用**策略。

### 5.1 拓扑画布引擎选型

#### 候选库全景对比

| 维度 | **AntV X6** | **AntV G6** | **React Flow** | **Vis.js Network** | **Cytoscape.js** | **D3.js** |
|------|-------------|-------------|----------------|-------------------|------------------|-----------|
| **核心定位** | 图编辑引擎 | 图可视化分析 | 节点编辑器 | 网络可视化 | 图论分析 | 底层绑定库 |
| **渲染方式** | SVG/HTML | Canvas/WebGL | SVG/HTML | Canvas | Canvas/WebGL | SVG/Canvas/WebGL |
| **React 集成** | ⭐⭐⭐⭐⭐ 原生 `@antv/x6-react-shape` | ⭐⭐⭐ 手动桥接或 `@ant-design/graphs` | ⭐⭐⭐⭐⭐ 原生 React 组件 | ⭐⭐ 第三方 wrapper | ⭐⭐ 手动桥接 | ⭐ 需大量封装 |
| **编辑能力** | ⭐⭐⭐⭐⭐ 对齐线/撤销重做/框选/剪贴板/快捷键 | ⭐⭐ 需自行实现 | ⭐⭐⭐⭐ 拖拽/连线/选择 | ⭐⭐ 基础编辑 | ⭐ 仅渲染 | ⭐ 需全部自行实现 |
| **端口/连接桩** | ⭐⭐⭐⭐⭐ 内置 Port 系统 | ⭐⭐ 需自定义 | ⭐⭐⭐⭐ Handle 组件 | ⭐⭐ 需 hack | ⭐ 无原生支持 | ⭐ 需自行实现 |
| **布局算法** | ⭐⭐ 基础（复用 G6 布局） | ⭐⭐⭐⭐⭐ 20+ 内置（力导向/树/环形/层次） | ⭐⭐ 需搭配 dagre/elkjs | ⭐⭐⭐ 层次/物理 | ⭐⭐⭐⭐ 10+ 图论布局 | ⭐⭐ 需搭配 d3-force |
| **大规模性能** | ⭐⭐⭐ 数百节点流畅 | ⭐⭐⭐⭐⭐ GPU/Rust 加速，万级节点 | ⭐⭐⭐ 数百节点，虚拟化 | ⭐⭐ 数百节点开始卡顿 | ⭐⭐⭐⭐ 千级节点 | ⭐⭐⭐⭐⭐ 取决于实现 |
| **连线样式** | 直线/曲线/正交/ER 线 + 10+ 箭头 | 直线/曲线/折线 | 贝塞尔/直线/阶梯/平滑阶梯 | 直线/曲线 | 贝塞尔/直线 | 完全自定义 |
| **许可** | MIT | MIT | MIT | MIT/Apache | MIT | BSD |
| **npm 周下载量** | ~50K | ~180K | ~2.7M | ~180K | ~150K | ~3.5M |
| **GitHub Stars** | 6.2K | 22.3K | 35K+ | 10K+ | 10K+ | 110K+ |

#### 选型结论：X6 为主 + G6 为辅

```
拓扑编辑器（核心工作区）
└── AntV X6 (@antv/x6 + @antv/x6-react-shape)
    ├── 设备节点：React 组件渲染（图标 + 名称 + 状态指示灯 + 端口列表）
    ├── 连接线：正交路由，带端口标签
    ├── 编辑交互：拖拽移动、对齐线吸附、框选、撤销/重做、复制粘贴
    ├── 画布能力：无限滚动画布、缩放、网格背景、Minimap
    └── 数据导入导出：JSON 序列化/反序列化

运行监控视图（大屏展示）
└── AntV G6 (@antv/g6)
    ├── 大规模拓扑渲染（Canvas/WebGL，支持 GPU 加速）
    ├── 实时数据更新（增量 updateItem，不重建整个图）
    ├── 动画效果（数据流线、节点呼吸灯、链路闪烁）
    └── 布局算法（力导向自动排列、层次布局）
```

**选型理由（基于 Ant Design 设计价值观）:**

| 价值观 | X6 选型依据 |
|--------|------------|
| **Natural** | X6 的对齐线、网格吸附、框选等交互行为与主流图形编辑工具（Figma、Draw.io）一致，用户凭直觉即可操作 |
| **Certain** | X6 的 Port（连接桩）系统提供明确的"可连接"视觉反馈；撤销/重做让每次编辑都可预期、可回退 |
| **Meaningful** | X6 的 React 节点支持允许在设备节点内嵌入有意义的交互控件（状态灯、端口指示器），而非纯装饰 |
| **Growing** | X6 的插件架构（History、Clipboard、Snapline、Minimap）支持功能按需叠加，与产品迭代路径匹配 |

**为什么不选其他库:**

| 库 | 排除原因 |
|----|---------|
| **React Flow** | 偏工作流编排（节点→节点线性流转），缺少 Port 多端口系统和专业的网络连线验证，设备端口管理需要大量自定义开发 |
| **Vis.js** | 社区活跃度下降，React 集成仅靠第三方 wrapper（周下载 ~5 次），且物理引擎布局不适合精确的网络拓扑排列 |
| **Cytoscape.js** | 偏图论分析和生物网络，不是为编辑场景设计的，交互模型与网络模拟器不匹配 |
| **D3.js** | 学习曲线陡峭，需要从头实现所有编辑功能，开发成本过高。仅推荐在需要极特殊定制时作为底层补充 |

### 5.2 核心库详细技术方案

#### 5.2.1 AntV X6 — 拓扑编辑器引擎

```bash
pnpm add @antv/x6 @antv/x6-react-shape @antv/x6-plugin-history @antv/x6-plugin-selection @antv/x6-plugin-snapline @antv/x6-plugin-clipboard @antv/x6-plugin-minimap @antv/x6-plugin-scroller @antv/x6-plugin-transform
```

**架构设计:**

```
┌─────────────────────────────────────────────────────────┐
│                    React App (Zustand)                   │
│  ┌─────────────┐  ┌─────────────┐  ┌────────────────┐  │
│  │ topologyStore │  │  labStore   │  │   authStore    │  │
│  │ (节点/边数据) │  │ (实验元数据) │  │  (认证状态)    │  │
│  └──────┬──────┘  └─────────────┘  └────────────────┘  │
│         │                                                │
│  ┌──────▼──────────────────────────────────────────┐    │
│  │           X6GraphCanvas (React Component)        │    │
│  │  ┌────────────────────────────────────────────┐  │    │
│  │  │          X6 Graph Instance                  │  │    │
│  │  │  ┌──────────┐ ┌──────────┐ ┌───────────┐   │  │    │
│  │  │  │ History  │ │Selection │ │ Snapline  │   │  │    │
│  │  │  │ Plugin   │ │ Plugin   │ │  Plugin   │   │  │    │
│  │  │  └──────────┘ └──────────┘ └───────────┘   │  │    │
│  │  │  ┌──────────┐ ┌──────────┐ ┌───────────┐   │  │    │
│  │  │  │ Minimap  │ │ Clipboard│ │ Scroller  │   │  │    │
│  │  │  │ Plugin   │ │ Plugin   │ │  Plugin   │   │  │    │
│  │  │  └──────────┘ └──────────┘ └───────────┘   │  │    │
│  │  └────────────────────────────────────────────┘  │    │
│  └──────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────┘
```

**设备节点 React 组件示例:**

```tsx
// DeviceNode.tsx — 使用 @antv/x6-react-shape 将 React 组件注册为 X6 节点
import { register } from '@antv/x6-react-shape';
import { Tag, Badge, Tooltip } from 'antd';

interface DeviceNodeProps {
  node: X6Node;
  data: {
    name: string;
    type: 'router' | 'switch' | 'firewall' | 'server' | 'endpoint';
    status: 'running' | 'stopped' | 'error';
    ports: { id: string; label: string; connected: boolean }[];
  };
}

const DeviceNodeComponent: React.FC<DeviceNodeProps> = ({ data }) => {
  const statusMap = { running: 'processing', stopped: 'default', error: 'error' };
  const colorMap = {
    router: '#1677FF', switch: '#13C2C2', firewall: '#F5222D',
    server: '#722ED1', endpoint: '#52C41A',
  };

  return (
    <div className="device-node" style={{ borderColor: colorMap[data.type] }}>
      <div className="device-node__header">
        <Badge status={statusMap[data.status]} />
        <span className="device-node__name">{data.name}</span>
      </div>
      <Tag color={colorMap[data.type]}>{typeLabels[data.type]}</Tag>
      {/* 端口列表在 X6 中通过 Port 系统渲染，此处仅展示设备信息 */}
    </div>
  );
};

register({
  shape: 'device-node',
  component: DeviceNodeComponent,
  effect: ['data'],
});
```

**X6 核心交互实现要点:**

| 功能 | X6 实现方式 |
|------|------------|
| 设备从面板拖入画布 | 监听 `drop` 事件 → `graph.addNode({ x, y, shape: 'device-node', data })` |
| 端口连线 | X6 Port 系统 + `connecting` 配置，`validateConnection` 校验端口兼容性 |
| 对齐线 | `Snapline` 插件，自动吸附到其他节点的边/中心 |
| 撤销/重做 | `History` 插件，绑定 Ctrl+Z / Ctrl+Y |
| 框选多节点 | `Selection` 插件，启用 `showNodeSelectionBox` |
| 复制/粘贴 | `Clipboard` 插件，绑定 Ctrl+C / Ctrl+V |
| 网格背景 | `graph.drawBackground({ color: '#F0F0F0' })` + Grid 配置 |
| 右键菜单 | X6 `node:contextmenu` / `edge:contextmenu` 事件 → Ant Design `Dropdown` |
| 自动布局 | 调用 G6 布局算法（dagre/force）对 X6 节点位置进行批量更新 |
| 拓扑保存/加载 | `graph.toJSON()` 序列化 → 发送至 API；`graph.fromJSON(data)` 恢复 |

#### 5.2.2 AntV G6 — 运行监控大屏

```bash
pnpm add @antv/g6
```

**使用场景:**
- 实验室运行时的"监控大屏"模式（只读，大规模数据实时刷新）
- 节点数量 >200 时的性能优化渲染
- 需要力导向自动布局的复杂拓扑分析

**G6 与 X6 协作模式:**

```
┌──────────────────────────────────────────────────┐
│              编辑模式 (X6)                        │
│  用户拖拽、连线、配置设备                          │
│  graph.toJSON() → 保存到后端                      │
└──────────────────┬───────────────────────────────┘
                   │ 切换监控模式
                   ▼
┌──────────────────────────────────────────────────┐
│            监控模式 (G6 Canvas/WebGL)              │
│  只读视图 + 实时状态更新                           │
│  ┌────────────────────────────────────────────┐  │
│  │  WebSocket 推送实时数据                      │  │
│  │  → graph.updateItem(targetNode, newStyle)   │  │
│  │  → 状态变化：颜色/动画/标签联动              │  │
│  │  → 数据流：Edge 动画（流动虚线/光点）        │  │
│  └────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────┘
```

**G6 WebGL 3D 拓扑（可选高级特性）:**
```bash
pnpm add @antv/g6-extension-3d
```
当拓扑规模 >500 节点时，可启用 3D 视图，利用 GPU 并行渲染，保持 60fps。

### 5.3 高级 Web 技术应用

#### 5.3.1 实时通信：WebSocket

```
┌──────────────────────────────────────────────────────┐
│                  WebSocket 架构                        │
│                                                       │
│  前端 (Browser)          后端 (Go/Node.js)             │
│  ┌──────────────┐       ┌──────────────────────┐     │
│  │ Socket.IO    │◄─────►│ WS Server             │     │
│  │ Client       │       │ ┌──────────────────┐ │     │
│  │              │       │ │ 设备状态推送       │ │     │
│  │ 用途：       │       │ │ Console 终端数据   │ │     │
│  │ • 设备状态   │       │ │ 实验室事件广播     │ │     │
│  │ • Console    │       │ │ 心跳检测           │ │     │
│  │ • 协作通知   │       │ └──────────────────┘ │     │
│  └──────────────┘       └──────────────────────┘     │
└──────────────────────────────────────────────────────┘
```

| 场景 | 技术方案 | 说明 |
|------|---------|------|
| 设备状态实时推送 | Socket.IO | 服务端推送 CPU/内存/端口状态，前端 `graph.updateItem()` 增量更新 |
| Console 终端 | WebSocket + xterm.js | 双向流式数据传输，终端模拟器渲染 |
| 心跳检测 | WebSocket ping/pong | 维持连接活性，断线自动重连（Socket.IO 内置） |
| 多人协作通知 | WebSocket 广播 | 用户 A 添加设备 → 广播 → 用户 B 看到更新 |

#### 5.3.2 终端模拟：xterm.js

```bash
pnpm add @xterm/xterm @xterm/addon-fit @xterm/addon-web-links @xterm/addon-search
```

设备 Console/SSH 终端使用 xterm.js 渲染，通过 WebSocket 与后端代理通信：

```
Browser                     Backend
┌──────────┐   WSS    ┌──────────────┐   SSH/Telnet   ┌──────────┐
│ xterm.js │◄────────►│ WS ↔ PTY     │◄──────────────►│ 设备容器  │
│          │  双向流   │ Proxy        │   原生协议      │ (qemu/   │
│          │          │              │                │  docker) │
└──────────┘          └──────────────┘                └──────────┘
```

#### 5.3.3 计算卸载：Web Workers

将计算密集型任务移出主线程，确保 UI 不卡顿：

```
┌──────────────────────────────────────────────────┐
│                   Main Thread (UI)                 │
│  React + X6/G6 + Ant Design                       │
│  ┌────────────────────────────────────────────┐  │
│  │  用户交互：拖拽、点击、滚动                  │  │
│  │  始终保持 60fps                             │  │
│  └────────────────────────────────────────────┘  │
└─────────────┬────────────────────────────────────┘
              │ postMessage
┌─────────────▼────────────────────────────────────┐
│             Web Workers Pool                      │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────┐ │
│  │ Layout       │ │ Topology     │ │ Data     │ │
│  │ Worker       │ │ Validate     │ │ Process  │ │
│  │              │ │ Worker       │ │ Worker   │ │
│  │ • 力导向布局  │ │ • 环路检测    │ │ • JSON   │ │
│  │ • 层次布局   │ │ • 连通性检查  │ │   解析   │ │
│  │ • 树形布局   │ │ • IP 冲突检测 │ │ • 导入   │ │
│  └──────────────┘ └──────────────┘ └──────────┘ │
└──────────────────────────────────────────────────┘
```

| Worker 类型 | 计算任务 | 预期收益 |
|------------|---------|---------|
| **Layout Worker** | dagre/force 布局算法计算（200+ 节点） | 布局时间从 800ms → 50ms（主线程阻塞为 0） |
| **Topology Worker** | 环路检测、连通性验证、IP 冲突检查 | 大规模拓扑保存前毫秒级校验 |
| **Data Worker** | 大型 JSON 解析、CSV/Excel 导入解析 | 导入 1000 节点拓扑不阻塞 UI |

**技术实现:** 使用 Vite 原生 Web Worker 支持 (`new Worker(new URL('./worker.ts', import.meta.url), { type: 'module' })`) + Comlink 库简化 RPC 通信。

```bash
pnpm add comlink
```

#### 5.3.4 离线存储与缓存：IndexedDB + Service Worker

```
┌──────────────────────────────────────────────┐
│            数据持久化分层策略                   │
│                                               │
│  L1: Zustand (内存)                           │
│  ├── 编辑中的拓扑数据（即时读写）               │
│  └── UI 状态（侧栏折叠、面板尺寸等）            │
│                                               │
│  L2: IndexedDB (本地持久化)                    │
│  ├── 自动保存草稿（30s 防抖）                  │
│  ├── 离线编辑队列（恢复网络后自动同步）          │
│  └── 设备模板/图标缓存                         │
│                                               │
│  L3: Service Worker (离线可用)                 │
│  ├── 应用 Shell 缓存（PWA 已配置）              │
│  ├── API 响应缓存（NetworkFirst 策略）          │
│  └── 静态资源预缓存                            │
│                                               │
│  L4: 后端 API (远程持久化)                     │
│  ├── 实验室/拓扑 CRUD                          │
│  └── 用户数据同步                              │
└──────────────────────────────────────────────┘
```

```bash
pnpm add idb dexie  # IndexedDB 封装库
```

**Dexie.js 使用示例:**
```typescript
// stores/topologyDraftStore.ts
import Dexie, { type EntityTable } from 'dexie';

interface TopologyDraft {
  id?: number;
  labId: string;
  graphJSON: object;      // X6 graph.toJSON() 序列化结果
  updatedAt: number;
}

const db = new Dexie('NetLabDB') as Dexie & {
  drafts: EntityTable<TopologyDraft, 'id'>;
};
db.version(1).stores({ drafts: '++id, labId, updatedAt' });

// 自动保存草稿（30s 防抖）
export async function autoSaveDraft(labId: string, graphJSON: object) {
  await db.drafts.put({ labId, graphJSON, updatedAt: Date.now() });
}
```

#### 5.3.5 离线画布渲染：OffscreenCanvas

```
┌─────────────────────────────────────────────────┐
│          OffscreenCanvas 架构                    │
│                                                  │
│  Main Thread              Worker Thread          │
│  ┌──────────┐            ┌──────────────────┐   │
│  │ X6 DOM   │            │ OffscreenCanvas  │   │
│  │ (编辑)   │            │ (导出/缩略图)     │   │
│  └──────────┘            │                  │   │
│       │                  │ • 高清截图导出    │   │
│       │ transferControl  │ • 缩略图生成      │   │
│       └─────────────────►│ • Minimap 渲染    │   │
│         to Offscreen     │ • PDF 打印        │   │
│                          └──────────────────┘   │
└─────────────────────────────────────────────────┘
```

| 场景 | 收益 |
|------|------|
| 大型拓扑截图导出（PNG/SVG） | 导出不阻塞 UI，支持超大分辨率（4K+） |
| Minimap 实时渲染 | 缩略图渲染不占用主线程 |
| 打印/PDF 导出 | 高 DPI 渲染不卡界面 |

#### 5.3.6 WebAssembly (WASM) — 浏览器内轻量模拟

```bash
# 潜在方案：编译网络模拟逻辑到 WASM
# 使用 Rust/Go → WASM，在浏览器内运行轻量网络协议栈
```

| 场景 | 技术 | 可行性 |
|------|------|--------|
| 基础网络连通性模拟（ping/traceroute） | Rust → WASM | ✅ 高，已有 mio/netstack-wasm 等实现 |
| 轻量路由协议（RIP/OSPF 子集） | Go → WASM (TinyGo) | ⚠️ 中，计算量可控但内存受限 |
| 流量生成与分析 | Rust → WASM | ✅ 高，pcap 解析可在浏览器完成 |
| 完整 VM 模拟（QEMU 级别） | ❌ 不适合浏览器 | 需后端 Docker/QEMU 支持 |

**WASM 策略:**
- **推荐**: 使用 WASM 运行**网络连通性校验**和**轻量协议模拟**，作为后端重型模拟的补充
- **不推荐**: 在浏览器中运行完整 VM——这应由后端 Docker/QEMU 基础设施承担
- **过渡方案**: 通过 WebSocket 将模拟指令发送至后端，WASM 仅处理离线/本地场景

#### 5.3.7 多人协同编辑（远期规划）

```bash
pnpm add yjs y-websocket @y/richtext
```

使用 **Yjs (CRDT)** + WebSocket 实现多人实时协同编辑拓扑：

```
User A                        Server                     User B
┌──────────┐   WebSocket   ┌──────────┐   WebSocket   ┌──────────┐
│ Y.Doc    │◄─────────────►│ y-websocket│◄────────────►│ Y.Doc    │
│          │   增量同步     │  Server  │               │          │
│ X6 Graph │               │          │               │ X6 Graph │
│ ↔ Yjs   │               │ 冲突自动  │               │ ↔ Yjs   │
│  Binding │               │  合并    │               │  Binding │
└──────────┘               └──────────┘               └──────────┘
```

CRDT 确保离线编辑不丢数据，网络恢复后自动合并冲突。此功能属于 Phase 4 远期规划。

### 5.4 技术栈全景图（最终版）

```
┌─────────────────────────────────────────────────────────────────┐
│                      NetLab 前端技术栈                           │
├─────────────────────────────────────────────────────────────────┤
│  🏗 框架层                                                       │
│  ├── React 19 + TypeScript 6                                   │
│  ├── Vite 8 (构建)                                              │
│  └── React Router 7 (路由)                                      │
├─────────────────────────────────────────────────────────────────┤
│  🎨 UI 层                                                        │
│  ├── Ant Design 6.x (组件库 + 主题系统)                          │
│  ├── @ant-design/icons (图标)                                    │
│  └── @ant-design/charts (图表，监控面板)                         │
├─────────────────────────────────────────────────────────────────┤
│  🕸 拓扑引擎层                                                    │
│  ├── @antv/x6 (图编辑引擎 — 拓扑编辑器)                          │
│  │   ├── @antv/x6-react-shape (React 节点)                     │
│  │   └── @antv/x6-plugin-* (插件生态)                           │
│  ├── @antv/g6 (图可视化引擎 — 监控大屏)                          │
│  │   └── @antv/g6-extension-3d (3D 视图，可选)                  │
│  └── dagre / elkjs (自动布局算法)                               │
├─────────────────────────────────────────────────────────────────┤
│  🔌 通信层                                                       │
│  ├── Axios (HTTP API)                                           │
│  ├── Socket.IO (WebSocket 实时推送 + Console 终端)               │
│  └── xterm.js + addons (设备终端模拟器)                          │
├─────────────────────────────────────────────────────────────────┤
│  💾 数据层                                                       │
│  ├── Zustand 5 (内存状态管理)                                    │
│  ├── Dexie.js / IndexedDB (本地持久化 + 离线草稿)                │
│  └── Yjs + y-websocket (协同编辑 CRDT，远期)                     │
├─────────────────────────────────────────────────────────────────┤
│  ⚡ 性能层                                                       │
│  ├── Web Workers + Comlink (布局计算/数据校验卸载)               │
│  ├── OffscreenCanvas (离线渲染/导出)                             │
│  └── WASM (轻量网络协议模拟，远期)                                │
├─────────────────────────────────────────────────────────────────┤
│  📱 离线 & PWA                                                   │
│  ├── vite-plugin-pwa (已配置)                                    │
│  └── Service Worker (资源缓存 + API 代理)                        │
├─────────────────────────────────────────────────────────────────┤
│  🧪 质量保障                                                     │
│  ├── Vitest + React Testing Library (单元/组件测试)              │
│  ├── Playwright (E2E 测试，拓扑交互)                             │
│  └── oxlint (已配置，代码规范)                                    │
└─────────────────────────────────────────────────────────────────┘
```

### 5.5 库选型与 Ant Design 生态的协同优势

选择 AntV X6 + G6 而非 React Flow 或 Vis.js 的核心原因之一，是它们与 Ant Design 同属蚂蚁集团开源生态，享有以下协同优势：

| 协同维度 | 具体表现 |
|---------|---------|
| **设计令牌互通** | AntV 图表库原生支持 Ant Design 主题变量，`colorPrimary` 等令牌自动同步 |
| **暗色主题联动** | ConfigProvider `darkAlgorithm` 切换时，G6 内置暗色主题可同步切换 |
| **CSS 变量兼容** | NetLab 已启用 `cssVar: { prefix: 'ant' }`，AntV 图表可消费同组变量 |
| **版本兼容性** | Ant Design 6.x ↔ @antv/x6 ↔ @antv/g6 ↔ @ant-design/charts 经内部验证 |
| **社区与文档** | 中文文档完善，案例丰富，问题排查路径短 |

---

## 六、设计规范对齐检查

### 6.1 设计令牌合规检查

| 令牌类别 | 规定值 | 本方案使用情况 | 合规 |
|---------|--------|---------------|------|
| 主色 | `#1677FF` | 选中态、链接、主按钮、焦点环、选中菜单 | ✅ |
| 成功色 | `#52C41A` | 运行中状态 Tag、成功消息 | ✅ |
| 警告色 | `#FAAD14` | 暂停状态、警告提示 | ✅ |
| 错误色 | `#FF4D4F` | 错误状态 Tag、危险按钮、表单校验错误 | ✅ |
| 页面背景 | `#F5F5F5` | Content 区域背景 | ✅ |
| 容器背景 | `#FAFAFA` | 画布背景、面板背景 | ✅ |
| 卡片表面 | `#FFFFFF` | 表格背景、Drawer 背景、Card、Sider 背景 | ✅ |
| 主文本 | `#1F1F1F` (或 `rgba(0,0,0,0.88)`) | 标题、正文 | ✅ |
| 次要文本 | `#595959` (或 `rgba(0,0,0,0.65)`) | 描述文字、表单标签 | ✅ |
| 禁用文本 | `#BFBFBF` (或 `rgba(0,0,0,0.25)`) | 禁用态文字 | ✅ |
| 默认边框 | `#D9D9D9` | 输入框边框、连接线默认色 | ✅ |
| 浅色边框 | `#F0F0F0` | 面板分割线、卡片边框 | ✅ |
| 标题字号 | 16-38px / 600 | 页面标题 24px/600，模块标题 16px/600 | ✅ |
| 正文字号 | 12-16px / 400 | 表格/表单 14px/400，标签 12px/400 | ✅ |
| 基础间距 | 4px | 全部间距为 4 的倍数 | ✅ |
| 间距 scale | xs(4)/sm(8)/md(16)/lg(24)/xl(32) | 严格遵守 | ✅ |
| 控件圆角 | 6px | Button/Input/Select | ✅ |
| 表面圆角 | 8px | Card/Modal/Drawer | ✅ |
| 标签圆角 | 4px | Tag | ✅ |
| 全圆角 | 9999px | Badge dot / Avatar | ✅ |

### 6.2 四大价值观对齐检查

| 价值观 | 关键设计决策 | 对齐状态 |
|--------|-------------|---------|
| **Natural** | 画布操作遵循主流图形编辑工具惯例（拖拽、缩放、右键菜单）；设备面板拖拽至画布符合直觉 | ✅ |
| **Certain** | 设备选中态（蓝色边框+光晕）、状态 Tag（语义色+图标）、表单实时校验（onChange）、删除二次确认 | ✅ |
| **Meaningful** | 画布工具栏仅保留 5 个核心操作；配置面板按 Tab 分组而非一次展示所有字段；表格列按优先级排列，次要信息可折叠 | ✅ |
| **Growing** | Layout 弹性架构支持新增菜单项和页面；Table 列可通过 `responsive` 按需显隐；Drawer 宽度可配置；主题通过 ConfigProvider 统一扩展 | ✅ |

### 6.3 组件使用合规检查

| 规范要求 | 本方案执行情况 |
|---------|--------------|
| 每屏只有一个 primary 按钮 | ✅ 每个页面/面板仅一个主操作按钮 |
| Table 不采用斑马纹 | ✅ 仅 hover 行高亮 |
| Tag 不使用语义色于关键状态 | ✅ 关键状态使用 Badge status dot + Alert |
| 不自定义 cubic-bezier | ✅ 统一使用 `motionEaseInOut` / `motionEaseOut` 等令牌 |
| 相邻元素圆角一致 | ✅ Card(8px) 内控件使用 4-6px，不出现 16px 以上大圆角 |
| 表单以 onChange 触发校验 | ✅ `validateTrigger="onChange"`，关键字段增加 `onBlur` |
| 弹窗/抽屉使用标准 shadow | ✅ 通过 token 而非自定义 box-shadow |

### 6.4 自定义主题扩展方案

```typescript
// ConfigProvider 主题配置（在 App.tsx 根组件应用）
// 注意: NetLab 当前已启用 cssVar: { prefix: 'ant' }，所有令牌同时生成为 CSS 变量
import { ConfigProvider } from 'antd';

const theme = {
  // 启用 CSS 变量（已配置）
  cssVar: { prefix: 'ant' },

  token: {
    // 主色（保持与现有配置一致）
    colorPrimary: '#1677FF',
    colorSuccess: '#52C41A',
    colorWarning: '#FAAD14',
    colorError: '#FF4D4F',
    colorInfo: '#1677FF',

    // 中性色
    colorText: '#1F1F1F',
    colorTextSecondary: '#595959',
    colorTextDisabled: '#BFBFBF',
    colorBorder: '#D9D9D9',
    colorBorderSecondary: '#F0F0F0',

    // 背景色（三层表面模型）
    colorBgLayout: '#F5F5F5',
    colorBgContainer: '#FAFAFA',
    colorBgElevated: '#FFFFFF',

    // 字体
    fontSize: 14,                // 基础字号（与现有配置一致）
    fontSizeHeading1: 38,
    fontSizeHeading2: 30,
    fontSizeHeading3: 24,
    fontSizeHeading4: 20,
    fontSizeHeading5: 16,
    fontFamily: `-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif`,

    // 间距
    paddingXS: 4,
    paddingSM: 8,
    padding: 16,
    paddingMD: 16,
    paddingLG: 24,
    paddingXL: 32,

    // 圆角（与现有配置一致）
    borderRadius: 6,
    borderRadiusLG: 8,
    borderRadiusSM: 4,

    // 动效
    motionDurationFast: '0.1s',
    motionDurationMid: '0.2s',
    motionDurationSlow: '0.3s',

    // 控件高度
    controlHeight: 32,
  },

  components: {
    Layout: {
      headerHeight: 64,
      headerBg: '#FFFFFF',       // 现有配置：浅色 Header
      siderBg: '#FFFFFF',        // 现有配置：浅色 Sider
      bodyBg: '#F5F5F5',        // 现有配置
    },
    Menu: {
      darkItemBg: '#001529',
      darkItemSelectedBg: '#1677FF',
      itemBg: '#FFFFFF',
      itemSelectedBg: '#E6F4FF',     // 现有配置
      itemSelectedColor: '#1677FF',  // 现有配置
    },
    Table: {
      headerBg: '#FAFAFA',          // 现有配置
      rowHoverBg: '#F5F5F5',
    },
    Card: {
      headerFontSize: 16,           // 现有配置
      bodyPadding: 24,              // 现有配置 (paddingLG)
    },
    Drawer: {
      paddingLG: 24,
    },
    Tag: {
      defaultBg: '#FAFAFA',
      defaultColor: '#595959',
    },
    Segmented: {
      // 设备分类切换控件
      itemSelectedBg: '#FFFFFF',
      trackBg: '#F0F0F0',
    },
  },
};

// 暗色主题（Phase 4 启用）
// import { darkAlgorithm } from 'antd';
// const darkTheme = { ...theme, algorithm: darkAlgorithm };
```

---

## 七、附录

### 7.1 关键页面路由结构

```
/                          → 重定向至 /dashboard
/dashboard                 → 仪表盘（实验室列表）
/lab/:labId                → 实验室拓扑编辑器
/lab/:labId/monitor        → 实验室运行监控
/templates                 → 模板市场首页
/templates/:templateId     → 模板详情
/device-library            → 设备库管理
/settings                  → 全局设置
/settings/profile          → 个人设置
/settings/license          → 许可证管理
/help                      → 帮助中心
```

### 7.2 技术栈建议

| 层级 | 技术选择 | 当前状态 |
|------|---------|---------|
| 框架 | React 19 + TypeScript 6 | ✅ 已使用 |
| UI 组件库 | Ant Design 6.x | ✅ 已使用 |
| 拓扑渲染 | Konva.js (react-konva) 或自研 Canvas/SVG 层 | 🔲 待引入 |
| 状态管理 | Zustand 5（轻量） | ✅ 已使用，需新增 labStore / topologyStore |
| 路由 | React Router 7 | ✅ 已使用，需新增拓扑相关路由 |
| 图表 | @ant-design/charts 或 ReCharts | 🔲 待引入（监控面板需要） |
| 构建 | Vite 8 | ✅ 已使用 |
| 代码规范 | oxlint | ✅ 已配置 |
| 测试 | Vitest + React Testing Library | 🔲 待引入 |
| HTTP | Axios | ✅ 已使用，含令牌刷新队列 |

### 7.3 待修正项（从现有代码中发现）

| 项目 | 当前值 | 建议修改 | 原因 |
|------|--------|---------|------|
| PWA manifest `theme_color` | `#667eea` | `#1677FF` | 与 antd 主色统一，保持品牌一致性 |
| PWA manifest `background_color` | `#667eea` | `#F5F5F5` | 使用页面背景色，减少启动闪白 |
| SideMenu 菜单项 | 仪表盘/系统(用户/角色/权限) | 重构为网络模拟器导航（见 4.5 节） | 当前为通用管理系统菜单模板 |

### 7.4 设计交付物清单

- [x] 本文档（界面改造设计方案）
- [ ] Figma/蓝湖高保真原型（基于本文档制作）
- [ ] 组件库映射表（业务组件 → antd 组件对照）
- [ ] 交互流程图（核心用户旅程）
- [ ] 设计标注稿（标注间距、字号、颜色）
- [ ] 主题配置文件（扩展 `App.tsx` 中的 theme 对象）

### 7.5 与现有代码的增量改造路径

本设计方案遵循**最小破坏、增量改造**原则，充分复用现有基础设施：

1. **保留** `MainLayout` 的布局骨架（Header + Sider + Content），仅替换 Sider 菜单内容
2. **保留** `ConfigProvider` 主题配置，按本方案扩展组件令牌
3. **保留** `netlab-` 前缀 CSS 工具类体系，新增拓扑画布专用样式
4. **保留** `useToken()` 内联样式模式，确保新组件与现有组件视觉一致
5. **保留** i18n 框架（i18next），扩展命名空间而非重构
6. **保留** Zustand store 结构，新增领域 store 独立管理拓扑状态
7. **新增** `src/pages/lab/`、`src/pages/topology/`、`src/pages/templates/` 等页面目录
8. **新增** `src/components/topology/`、`src/components/device-panel/` 等业务组件目录

---

> **设计依据**: 本文档所有设计决策均基于 [Ant Design 官方设计文档](https://ant.design/design.md) 的设计价值观、设计令牌、组件规范和交互模式。四大核心价值观（Natural、Certain、Meaningful、Growing）贯穿每个设计环节，确保最终产品在视觉与交互层面与 Ant Design 生态无缝衔接。
