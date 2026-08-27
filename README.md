# 海洋牧场养殖环境监测服务（marine-farm-environment-service）

基于 Go 实现的海洋牧场养殖环境监测 **全栈 Web 项目**（Go 后端服务 + `go:embed`
内嵌原生前端），完成养殖区划分、水质数据采集、溶解氧/温度异常预警、增氧设备联动
与养殖日志管理。离线可运行，无任何外部服务依赖。

- 项目类型：全栈 Web 应用（Go 后端 + 原生 HTML/CSS/JS 前端）
- 模块路径：`example.com/marine-farm-environment-service`
- Go 版本：`go 1.23`（构建时建议使用 Go 1.23+）
- 存储：内存仓储 + JSON 文件持久化（默认 `data/marine_data.json`，可用
  `DATA_FILE=` 关闭持久化）
- 默认端口：`8080`（可用 `PORT` 覆盖）

## 一、业务规则

1. **养殖区状态机**：`normal → warning → danger → aerating → restored`；
   溶解氧低于 4mg/L 预警、低于 3mg/L 危险。合法迁移表见 `domain/zone.go`
   （`zoneTransitionTable`），越限判定见 `domain/sample.go`。
2. **浮标上报**：按 5 分钟周期上报（浮标 id + 溶解氧 + 水温 + 盐度 + pH + 氨氮 +
   时间戳），越限判定按养殖区配置阈值；乱序/过于频繁上报会被拒绝
   （`service/ingest_service.go`）。
3. **交叉验证**：单浮标溶解氧危险时，若同养殖区其他浮标 15 分钟内有正常数据，该次
   危险标记为「待核实（pending）」，不直接触发增氧，人工核实（`POST
   /api/warnings/{id}/verify`）后处置（`domain/warning.go`、
   `service/ingest_service.go`）。
4. **增氧联动**：确认危险后自动下发增氧机启动指令并记录联动日志
   （`service/aeration_service.go`）；溶解氧恢复至 5mg/L 以上并持续 30 分钟后，
   后台恢复检测器标记「可恢复」，人工确认恢复（`POST /api/zones/{id}/restore`）。
5. **增氧机状态机**：`stopped → starting → running → stopping`；启动/停止指令在
   超时（默认 2 分钟）内无终态反馈按故障处理并告警（`domain/aeration.go`、
   `service/restore_checker.go`）。
6. **养殖日志**：每日投喂/死亡/病害记录，单日死亡数超存塘量 1% 自动提示
   （`domain/farmlog.go`）。

## 二、架构与分层

```
HTTP 请求
   │  middleware：requestID → panic recovery → security headers → slog 访问日志
   ▼
httpapi/   REST 路由 + 处理器（薄层，只做解析/校验/响应封装）
   ▼
service/   业务用例编排（采集判定、预警核实、增氧联动、恢复检测、日志、审计、总览）
   ▼
domain/    实体 + 状态机 + 领域规则（纯 Go，不依赖存储/HTTP）
   ▼
store/     内存仓储 + JSON 文件原子持久化（深拷贝读接口，读写锁保护）
```

- **domain**：`FarmZone/Buoy/WaterSample/WarningRecord/AerationLog/FarmLog`、
  状态机、越限判定、交叉验证，零外部依赖。
- **store**：各实体仓储 + `Store` 聚合器。所有写操作串行化；读接口返回深拷贝，
  杜绝调用方原地修改引发的数据竞争。
- **service**：编排用例并写审计；后台恢复检测定时任务。
- **httpapi**：Go 1.22+ `http.ServeMux` 路由，统一响应信封
  `{"code":0,"message":"ok","data":...}`。
- **middleware**：requestID / panic recovery / security headers / slog 访问日志 /
  审计日志。
- **web**：原生 HTML/CSS/JS SPA，`go:embed` 内嵌，无外部 CDN。

## 三、目录结构

```
.
├── go.mod                     # module example.com/marine-farm-environment-service, go 1.23
├── main.go                    # 入口：配置 → slog → 存储加载 → 引导数据 → 路由 → 优雅关闭
├── Dockerfile                 # 多阶段镜像（golang:1.23-alpine → alpine:3.20，非 root）
├── Makefile                   # build/vet/fmt/test/race/run/docker-build/docker-run
├── .dockerignore
├── runtime_smoke.json         # 冒烟配置（mode/start/ready_url/project_intro）
├── config/
│   └── config.go              # 全部可调参数 + 环境变量覆盖 + Validate()
├── domain/                    # 领域层：实体 + 状态机 + 业务规则
│   ├── zone.go / buoy.go / sample.go / warning.go / aeration.go / farmlog.go
│   ├── audit.go / types.go / errors.go
├── store/                     # 存储层：内存仓储 + JSON 文件原子持久化
│   ├── store.go / json_persist.go / clone.go
│   └── *_store.go             # zone/buoy/sample/warning/aeration/farmlog/audit 仓储
├── service/                   # 服务层：业务用例编排
│   ├── ingest_service.go      # 采集 + 越限 + 交叉验证 + 自动增氧
│   ├── warning_service.go     # 核实/解除
│   ├── aeration_service.go    # 启停 + 反馈 + 恢复确认
│   ├── restore_checker.go     # 5 分钟恢复检测定时任务
│   ├── farmlog_service.go     # 养殖日志
│   ├── audit_service.go       # 审计
│   ├── overview_service.go    # 总览聚合
│   ├── bootstrap.go           # 演示数据（幂等）
│   └── service.go             # 服务装配 + 后台任务启动
├── httpapi/                   # HTTP 层：REST 路由 + 处理器 + 统一响应
│   ├── router.go / respond.go / helpers.go
│   └── *_handler.go           # zone/buoy/sample/warning/aeration/farmlog/overview/health
├── middleware/                # requestID / recover / security / logging / audit
└── web/                       # 原生前端（go:embed 内嵌）
    ├── index.html / app.js / api.js / style.css / enums.js
    ├── components/            # ZoneCard / WaterTrend / WarningTable
    ├── hooks/                 # useZones / useWarnings
    └── pages/                 # overview / zone_detail / warnings / aeration / logs
```

## 四、运行

### 本地运行

```bash
# 默认监听 8080，使用 data/marine_data.json 持久化
go run .

# 指定端口 / 关闭持久化 / 调整日志级别
PORT=19010 LOG_LEVEL=debug DATA_FILE= go run .
```

启动后：

- 页面：`http://localhost:8080/`
- 存活探针：`GET /healthz`（200）
- 就绪探针：`GET /readyz`（200）

### 测试与静态检查

```bash
make build      # go build ./...
make vet        # go vet ./...
make fmt        # gofmt -w .
make test       # go test ./...
make race       # go test -race ./...
```

### Docker 部署

```bash
make docker-build
docker run --rm -p 8080:8080 marine-farm-environment-service:latest
# 覆盖端口与数据文件
docker run --rm -p 19010:19010 \
  -e PORT=19010 -e DATA_FILE=/app/data/marine_data.json \
  -v "$PWD/data:/app/data" \
  marine-farm-environment-service:latest
```

镜像说明：多阶段构建（`golang:1.23-alpine` → `alpine:3.20`），
`CGO_ENABLED=0`、非 root 用户 `app`、`EXPOSE 8080`、尊重 `PORT`、
自带 `HEALTHCHECK`（wget 探测 `/healthz`）。

## 五、REST API

统一响应信封：`{"code":0,"message":"ok","data":...}`；错误时
`{"code":404,"message":"...","error":"not_found","request_id":"..."}`。

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/healthz`、`/api/healthz` | 存活探针 |
| `GET` | `/readyz`、`/api/readyz` | 就绪探针 |
| `GET` | `/api/overview` | 总览聚合 |
| `GET` | `/api/audit` | 操作审计（`target_type`/`target_id`/`limit`/`offset`） |
| `GET` | `/api/zones`、`POST /api/zones`、`GET /api/zones/{id}` | 养殖区维护（分页） |
| `GET` | `/api/buoys`、`POST /api/buoys`、`GET /api/buoys/{id}` | 浮标维护（分页） |
| `POST` | `/api/buoys/{id}/samples` | 水质上报（越限判定 + 交叉验证） |
| `GET` | `/api/zones/{id}/samples` | 水质趋势（`buoy_id`/`limit`/`offset`） |
| `GET` | `/api/warnings` | 预警列表（`status`/`zone_id`/`type`/`limit`/`offset`） |
| `POST` | `/api/warnings/{id}/verify` | 核实待核实预警 |
| `POST` | `/api/warnings/{id}/resolve` | 解除预警 |
| `GET` | `/api/aeration` | 增氧记录（分页） |
| `POST` | `/api/zones/{id}/aerate` | 启动增氧联动 |
| `POST` | `/api/zones/{id}/stop-aeration` | 停止增氧 |
| `POST` | `/api/zones/{id}/restore` | 恢复确认 |
| `POST` | `/api/aeration/{id}/feedback` | 增氧机设备反馈（acknowledged/started/stopped/fault） |
| `GET` | `/api/logs`、`POST /api/logs`、`GET /api/logs/{id}` | 养殖日志（分页） |

### 分页约定

列表接口统一支持 `limit` 与 `offset`（均为非负整数）：

- 默认 `limit`：100（samples 为 200），最大上限 1000（samples 为 2000）；
- 响应体仍为纯数组（保持前端兼容），分页元数据放响应头：
  `X-Total-Count` / `X-Limit` / `X-Offset`。

### 输入校验

- 请求体上限 1 MiB，超限返回 400；
- JSON 仅允许一个值，尾随数据返回 400；
- `NaN` / `Infinity` / 溢出数字无法通过 JSON 解析，负值/越界在服务层返回 400；
- 非法路径、状态机冲突、未满足恢复条件等返回 404/409，不会 panic。

### 可复现的主业务链路（供验证）

```bash
BASE=http://localhost:8080

# 1. 创建养殖区
curl -s -X POST $BASE/api/zones -H 'Content-Type: application/json' \
  -d '{"name":"测试区","area":100,"stock":50000}'

# 2. 创建两个浮标（用于交叉验证）
curl -s -X POST $BASE/api/buoys -H 'Content-Type: application/json' \
  -d '{"zone_id":"zone_N","name":"A浮标"}'
curl -s -X POST $BASE/api/buoys -H 'Content-Type: application/json' \
  -d '{"zone_id":"zone_N","name":"B浮标"}'

# 3. 邻居浮标 2 分钟前上报正常溶解氧
curl -s -X POST $BASE/api/buoys/buoy_B/samples -H 'Content-Type: application/json' \
  -d '{"do":6.1,"temperature":24,"salinity":31,"ph":8.1,"ammonia":0.05,"timestamp":"<now-2m RFC3339>"}'

# 4. 危险上报 -> 交叉验证存疑，预警待核实，不触发增氧
curl -s -X POST $BASE/api/buoys/buoy_A/samples -H 'Content-Type: application/json' \
  -d '{"do":2.4,"temperature":24,"salinity":31,"ph":8.1,"ammonia":0.05,"timestamp":"<now RFC3339>"}'

# 5. 核实预警 -> 确认危险 + 自动增氧 + 养殖区进入应急增氧
curl -s -X POST $BASE/api/warnings/warning_N/verify

# 6. 恢复条件未满足时确认恢复 -> 409
curl -s -X POST $BASE/api/zones/zone_N/restore

# 7. 养殖日志（800 死亡 / 50000 存塘 = 1.6% -> 自动异常提示）
curl -s -X POST $BASE/api/logs -H 'Content-Type: application/json' \
  -d '{"zone_id":"zone_N","date":"2026-08-25","feed_amount":500,"death_count":800}'
```

完整恢复链路（溶解氧 ≥5mg/L 持续 30 分钟）可用短周期参数快速演示：

```bash
PORT=18018 RESTORE_SUSTAINED=2s RESTORE_CHECK_INTERVAL=1s SAMPLE_PERIOD=1s go run .
# 上报 3 条间隔 ≥2s 的 DO=5.8 样本后，等待恢复检测器运行，
# 再 POST /api/zones/{id}/restore 即成功（zone -> restored）。
```

## 六、环境变量

| 变量 | 默认值 | 说明 |
|---|---|---|
| `PORT` | `8080` | HTTP 监听端口 |
| `DATA_FILE` | `data/marine_data.json` | JSON 持久化文件；空字符串关闭持久化 |
| `LOG_LEVEL` | `info` | 日志级别：`debug` / `info` / `warn` / `error` |
| `DO_WARN_THRESHOLD` | `4.0` | 溶解氧预警阈值（mg/L） |
| `DO_DANGER_THRESHOLD` | `3.0` | 溶解氧危险阈值（mg/L） |
| `DO_RESTORE_THRESHOLD` | `5.0` | 溶解氧恢复阈值（mg/L） |
| `RESTORE_SUSTAINED` | `30m` | 恢复条件持续时长 |
| `CROSS_CHECK_WINDOW` | `15m` | 交叉验证时间窗口 |
| `SAMPLE_PERIOD` | `5m` | 浮标上报周期 |
| `SAMPLE_PERIOD_TOLERANCE` | `60s` | 上报周期容差（防止时钟漂移误拒） |
| `AERATOR_TIMEOUT` | `2m` | 增氧机反馈超时（超时按故障） |
| `RESTORE_CHECK_INTERVAL` | `5m` | 恢复检测器轮询周期 |
| `TEMP_MIN` / `TEMP_MAX` | `10` / `32` | 水温正常范围（°C） |
| `SALINITY_MIN` / `SALINITY_MAX` | `25` / `35` | 盐度正常范围（‰） |
| `PH_MIN` / `PH_MAX` | `7.0` / `8.8` | pH 正常范围 |
| `AMMONIA_MAX` | `0.2` | 氨氮上限（mg/L） |
| `DEATH_ABNORMAL_RATIO` | `0.01` | 单日死亡异常比例（> 存塘量该比例） |
| `MAX_SAMPLES_PER_BUOY` | `2000` | 每个浮标保留的水质样本上限 |

时长环境变量使用 Go `time.ParseDuration` 格式（如 `90s`、`5m`、`1h`）。

## 七、共享枚举 / 常量前后端位置

| 枚举 | 后端 | 前端 | 说明 |
|---|---|---|---|
| 养殖区状态 `ZoneStatus` | `domain/types.go`（`ZoneStatusNormal…`）+ 状态机 `domain/zone.go` | `web/enums.js`（`ZoneStatus`/`ZONE_STATUS_LABEL`/`ZONE_STATUS_CLASS`） | normal/warning/danger/aerating/restored |
| 预警类型 `WarningType` | `domain/types.go` | `web/enums.js`（`WarningType`/`WARNING_TYPE_LABEL`） | do_low/temp_shock/ph_abnormal/ammonia_high |
| 预警状态 `WarningStatus` | `domain/types.go` | `web/enums.js`（`WarningStatus`/`WARNING_STATUS_LABEL`） | pending/confirmed/resolved |
| 增氧机状态 `AeratorStatus` | `domain/types.go` + 状态机 `domain/aeration.go` | `web/enums.js`（`AeratorStatus`/`AERATOR_STATUS_LABEL`/`AERATOR_STATUS_CLASS`） | stopped/starting/running/stopping/fault |

其余共享常量：反馈状态 `FeedbackStatus`、动作 `AerationAction`、触发源
`AerationTrigger`、浮标状态 `BuoyStatus`，均位于 `domain/types.go` ↔
`web/enums.js`。

## 八、横切关注点

1. **操作审计日志**：预警核实、增氧联动、恢复确认、日志录入全部留痕，链路为
   handler → service → audit store（`service/audit_service.go`）；另有一层 HTTP
   请求审计中间件（`middleware/audit.go`），`GET /api/audit` 可查。
2. **溶解氧恢复检测定时任务**：每 5 分钟检查恢复条件并标记「可恢复」，链路为
   service → store → 养殖区详情页（`service/restore_checker.go`）。
3. **全局错误处理与统一响应格式**：`middleware/error_handler.go` +
   `httpapi/respond.go`，所有错误携带稳定错误码
   （not_found/invalid_input/conflict/internal）。
4. **结构化日志与访问日志**：`log/slog`，`LOG_LEVEL` 控制级别；每次 HTTP 请求
   输出一条 `slog` 访问日志，包含 method/path/status/duration/request_id。
5. **安全基线**：requestID、panic recovery、security headers
   （nosniff/DENY/no-referrer/Permissions-Policy、API no-store）。

## 九、健康检查

- `GET /healthz`：存活探针，返回 200 与服务状态摘要。
- `GET /readyz`：就绪探针，返回 200 表示服务已就绪可接流量。
- Docker 镜像已内置 `HEALTHCHECK`（每 30s 探测 `/healthz`）。

## 十、故障排查

- **启动即退出**：检查环境变量是否非法（如 `LOG_LEVEL` 不在允许集合、
  `DO_WARN_THRESHOLD <= DO_DANGER_THRESHOLD`）；启动日志会给出 `config:` 前缀错误。
- **持久化文件损坏**：服务会把损坏快照备份为 `data/marine_data.json.bak`，
  降级为空库启动并在日志输出 `WARN`；检查 `.bak` 后可删除或修复。
- **端口被占用**：设置 `PORT=其他端口 go run .`。
- **前端页面空白**：确认 `/api/healthz` 返回 200，浏览器控制台查看
  `/api/overview` 等接口是否 200；本项目禁止外部 CDN，需在线内打开或本机部署。
- **增氧机不联动**：单浮标危险读数会被相邻浮标交叉验证置为 `pending`，需先
  `POST /api/warnings/{id}/verify` 人工核实。
