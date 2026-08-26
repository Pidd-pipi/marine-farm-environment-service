# 海洋牧场养殖环境监测服务（marine-farm-environment-service）

## 一、项目概述

基于 Go 实现的海洋牧场养殖环境 Web 项目，一款后端服务，完成养殖区划分、水质数据采集、溶解氧/温度异常预警、增氧设备联动与养殖日志管理。

项目类型：**全栈 Web 应用**（Go 后端服务 + `go:embed` 内嵌前端页面）。

## 二、业务背景与领域规则

海洋牧场（近海养殖）需要持续监测养殖区水质：溶解氧、水温、盐度、pH、氨氮。溶解氧过低会导致鱼类缺氧死亡，系统要实时预警并联动增氧机；水温骤变提示应激风险。每个养殖区有多个监测浮标，同一浮标数据异常要和相邻浮标交叉验证避免误报。养殖员每日记录投喂、死亡、病害情况。

关键领域规则（这些规则是后续埋 bug 验证跨文件改动的核心约束，必须真实实现）：

1. 养殖区状态机：正常(normal) → 预警(warning) → 危险(danger) → 应急增氧(aerating) → 恢复确认(restored)；溶解氧低于 4mg/L 预警、低于 3mg/L 危险。
2. 浮标上报：浮标按 5 分钟周期上报（buoy_id + 溶解氧 + 水温 + 盐度 + pH + 氨氮 + 时间戳），越限判定按养殖区配置。
3. 交叉验证：单浮标溶解氧危险时，若相邻浮标（同养殖区其他浮标）15 分钟内数据正常，则该次危险标记为「待核实」不直接触发增氧，人工确认后处置。
4. 增氧联动：确认危险后自动下发增氧机启动指令，记录联动日志；溶解氧恢复至 5mg/L 以上持续 30 分钟才允许人工确认恢复。
5. 增氧机状态机：停止(stopped) → 启动中(starting) → 运行(running) → 停止中(stopping)；启动/停止指令超时无反馈按故障处理并告警。
6. 养殖日志：每日投喂/死亡/病害记录，死亡数量异常（单日超存塘量 1%）自动提示。

## 三、核心实体（≥3 个，必须贯穿全栈）

每个实体必须贯穿「数据库/存储表 → domain model → repository → service → handler → 前端 API 层 → 前端页面/组件」全链路。

| 实体 | 关键字段 | 业务动作 |
|---|---|---|
| 养殖区 FarmZone | id、名称、面积、存塘量、溶解氧阈值、状态 | 维护、状态查询 |
| 监测浮标 Buoy | id、养殖区id、位置、状态 | 维护、上报 |
| 水质数据 WaterSample | id、浮标id、溶解氧、水温、盐度、pH、氨氮、时间戳、是否越限 | 采集、判定 |
| 预警记录 WarningRecord | id、养殖区id、类型、级别、状态(待核实/已确认/已解除) | 预警、核实 |
| 增氧联动 AerationLog | id、养殖区id、增氧机id、动作、下发时间、反馈 | 联动、反馈 |
| 养殖日志 FarmLog | id、养殖区id、日期、投喂量、死亡数、病害备注 | 记录、异常提示 |

## 四、核心页面与 API

### 前端页面（≥4 个路由，至少 2 个页面共用同一个业务组件）

| 项目 | 说明 |
|---|---|
| / 牧场总览 | 养殖区状态卡片 + 溶解氧实时值 + 预警计数 | FarmZone、WaterSample |
| /zones/{id} 养殖区详情 | 水质趋势 + 预警记录 + 增氧联动 | WaterSample、WarningRecord |
| /warnings 预警台 | 预警列表 + 核实/确认 | WarningRecord |
| /aeration 增氧控制 | 增氧机状态 + 手动启停 | AerationLog |
| /logs 养殖日志 | 投喂/死亡记录 + 异常提示 | FarmLog |

### 后端 REST API（与页面一一对应，命中真实业务链路）

| 项目 | 说明 |
|---|---|
| POST /api/buoys/{id}/samples | 水质上报（越限判定 + 交叉验证） |
| POST /api/warnings/{id}/verify | 核实待核实预警 |
| POST /api/zones/{id}/aerate | 启动增氧联动 |
| POST /api/zones/{id}/restore | 恢复确认 |
| POST /api/logs | 养殖日志记录 |
| GET /api/zones/{id}/samples | 水质趋势 |
| GET /api/warnings | 预警列表 |
| GET /api/aeration | 增氧记录 |
| GET /api/overview | 总览聚合 |
| GET /api/healthz | 健康检查 |

## 五、横切关注点（≥2 个）

1. 操作审计日志：预警核实、增氧联动、恢复确认、日志录入全部留痕；触达 handler → service → audit store。
2. 溶解氧恢复检测定时任务：每 5 分钟检查恢复条件并提示可恢复；触达 service → store → 养殖区详情。
3. 全局错误处理与统一响应格式。

## 六、共享枚举/常量（≥2 组）

枚举/常量要求前后端各自定义且保持一致，README 中列出所有出现位置。

1. 养殖区状态 ZoneStatus：normal / warning / danger / aerating / restored。
2. 预警类型 WarningType：do_low / temp_shock / ph_abnormal / ammonia_high。
3. 预警状态 WarningStatus：pending / confirmed / resolved；增氧机状态 AeratorStatus：stopped / starting / running / stopping / fault。

## 七、共享前端组件与 hooks（组件 ≥3 个、hooks ≥2 个）

### 共享组件（放 `web/components/`）

1. ZoneCard：养殖区状态卡片，被总览与详情共用。
2. WaterTrend：水质趋势组件，被详情与预警台共用。
3. WarningTable：预警表格，被预警台与总览共用。

### 自定义 hooks（放 `web/hooks/`）

1. useZones(poll)：养殖区数据轮询，被总览与详情共用。
2. useWarnings(filter)：预警列表，被预警台与详情共用。

## 八、后端中间件（≥2 个）

1. auditLogger：审计日志中间件。
2. errorHandler：统一错误/panic 处理中间件。
3. requestID：trace id 注入中间件。

## 九、技术要求

- 语言：**Go 1.23**（go.mod 声明 `go 1.23`，module 路径 `example.com/marine-farm-environment-service`）
- 运行：`go run .` 默认监听 `8080`，支持 `PORT` 环境变量覆盖
- 存储：SQLite（`modernc.org/sqlite` 纯 Go 驱动，CGO 关闭）或内置内存仓储 + JSON 文件持久化，二选一，必须可重复构建、无外部服务依赖
- 前端：纯原生 HTML/CSS/JS，`go:embed` 内嵌 `web/` 静态资源，禁止引入外部 CDN 依赖（离线可跑）
- 服务入口：`GET /healthz` 返回 200；页面 `GET /` 可访问
- 根目录必须包含 `runtime_smoke.json`：`mode: service` + `start: go run .` + `ready_url: /healthz`；`project_intro` 一句话简介必须包含项目类型（如「基于 Go 实现的XXX Web 项目，一款后端服务，完成……」）
- 根目录必须包含 `README.md`：项目说明、目录结构、运行与测试命令、环境变量说明
- 构建：`go build ./...` 与 `go test ./...` 必须全部通过（基线干净、无 bug）

## 十、文件结构强制清单（规模目标：≥2000 行 Go 功能代码、≥20 个 `.go` 文件）

```
backend/
├── go.mod
├── main.go
├── config/
│   └── config.go            # 溶解氧阈值、交叉验证窗口、恢复条件
├── domain/
│   ├── zone.go              # 养殖区 + 状态机
│   ├── buoy.go              # 监测浮标
│   ├── sample.go            # 水质数据 + 越限判定
│   ├── warning.go           # 预警 + 交叉验证
│   ├── aeration.go          # 增氧联动
│   └── farmlog.go           # 养殖日志
├── store/
│   ├── zone_store.go
│   ├── buoy_store.go
│   ├── sample_store.go
│   ├── warning_store.go
│   ├── aeration_store.go
│   ├── farmlog_store.go
│   └── audit_store.go
├── service/
│   ├── ingest_service.go    # 采集 + 越限 + 交叉验证
│   ├── warning_service.go   # 核实/确认
│   ├── aeration_service.go  # 增氧联动 + 反馈
│   ├── restore_checker.go   # 恢复检测
│   ├── farmlog_service.go
│   └── audit_service.go
├── httpapi/
│   ├── router.go
│   ├── zone_handler.go
│   ├── buoy_handler.go
│   ├── warning_handler.go
│   ├── aeration_handler.go
│   ├── farmlog_handler.go
│   └── health_handler.go
├── middleware/
│   ├── audit.go
│   ├── error_handler.go
│   └── request_id.go
└── web/
    ├── index.html
    ├── app.js
    ├── style.css
    ├── components/
    └── hooks/
```

**严禁合并职责到单一文件**：handler、service、repository、domain 必须分层；禁止把所有逻辑塞进 `main.go` 或一个 `handlers.go`。目标规模下限 2000 行 / 20 个 `.go` 文件，实际建议做到 3000 行以上 / 30 个文件以上，保证每个业务模块（实体、状态机、联动、报表）都有独立文件。

## 十一、运行、测试与交付要求

1. `go build ./...` 通过；`go test ./...` 全绿（含各业务模块的单元测试，测试文件不计入规模）。
2. `go run .` 后 `GET /healthz` 返回 200，前端页面 `GET /` 可打开且核心接口可用。
3. 每个核心业务动作都要有可复现的输入（API 请求/页面操作），方便后续构造缺陷与验证命令。
4. 代码中不得出现任何「故意埋错」「TODO bug」类注释；交付为干净基线。

## 十二、质量红线

1. **天然多文件、多层耦合**：任何一个小改动（如给某状态新增一个合法迁移）都应触达 3-5 个文件（domain + repository + service + handler + 前端组件 + 枚举定义）。
2. 业务规则必须具体、可验证：状态机迁移表、联动逻辑、校验边界、生命周期管理必须真实存在，禁止空壳 CRUD。
3. 本项目用于评测跨文件协同改动能力，禁止做成本目录、对账/财务、库存盘点、电商订单、预约挂号、工单客服、数据可视化报表类业务。
4. 前端页面必须真实消费后端接口，禁止纯静态假页面。

---
*生成说明：本提示词面向 Go 标注数据流水线 2000 行档位，主题已对照禁选题材清单核验。*
