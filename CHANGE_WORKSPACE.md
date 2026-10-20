# ChangePilot 功能与接口

ChangePilot 基于 WeKnora 现有知识工作空间，从“研发变更”创建一项需求，填写确定的代码 commit，选择版本化规则，再确认检查计划。Codex / Claude Code 仍由开发者操作；工作台不修改业务仓库。

## 本地源码与上游

- 上游：Tencent/WeKnora。
- 固定基线：`737d237283ee1a8abd19a4755ad3197f79ce1303`。
- 本地分支：`feature/change-workspace`，保留上游 Git 历史和 LICENSE。
- 此二开新增 change 的类型、GORM repository、application service、handler、迁移、Vue 页面、client/CLI 读入口、两个只读 workspace MCP 工具。
- 身份、知识解析、检索、模型、权限、工作空间 MCP 传输与前端设计系统来自上游。

TaskFlow 客户端采用用户自有源码的同步副本，见 `internal/integration/taskflowclient/README.md`。没有引入指向兄弟目录的 Go module replace。

## 运行入口

沿用上游 `./cmd/server` 和 `frontend` Vite 入口。必须保留完整上游配置，并合入 `config/change-workspace.example.yaml` 的新增段。工作空间 RBAC 必须处于强制模式，才能使用新增写操作。

```text
go build -o tmp/weknora-server-linux ./cmd/server
cd frontend
npm ci
npm run dev -- --host 127.0.0.1 --port 15173
```

后端示例地址为 `127.0.0.1:18088`，前端通过 `VITE_DEV_PROXY_TARGET=http://127.0.0.1:18088` 连接。独立 clone 的数据库、Redis、构建、密钥、tenant 映射和模型配置见 [源码启动手册](./docs/CHANGE_PILOT_QUICKSTART.md)。TaskFlow 使用 `taskflow_token_env` 指定的环境变量取得服务凭据。

前端入口：`/platform/changes`。现有登录、工作空间切换、知识库、模型与 MCP 集成设置继续使用原入口。

## 用同一核心适配两个业务

本地样例已完成商品预售与 SaaS 工单两条闭环。商品关注旧版 API、null 日期和时区；工单关注持久化、筛选/计数、分页和团队权限。业务包清单分别位于兄弟样例仓库的 `packages/catalog-presale/1.0.0/package.json` 与 `packages/saas-ticket-priority/1.0.0/package.json`。

包清单记录资料、模块、验收项和检查 ID，不是可执行插件。操作者同时配置工作台的项目/工作空间/业务包/必要引用映射与 TaskFlow 的可信仓库、主体和固定命令。两个业务复用同一全仓测试/构建模板，领域规则与断言不同；工作台核心没有商品或工单分支。新可信业务按这一映射接入，不通过上传包获取任意命令执行能力。

仓库内的 [验证记录](./docs/CHANGE_WORKSPACE_VALIDATION_20261020.md) 区分模块回归与真实集成结果。原始运行日志保存在验收者本机档案，不作为独立 clone 的运行依赖。本地模型临时入口已依照授权在验收后关闭，保存的建议/报告仍可查看；后续推理须配置获准的可达模型地址。

## 三个页面完成什么

1. **变更单**：保存目标、代码版本和协作信息。每次 PATCH 使用 `expected_revision`；冲突不覆盖浏览器输入。
2. **上下文证据**：复用现有知识检索发现候选，人工提供确定的仓库路径和 commit；服务端从 Git 对象重新读取正式依据。模型生成影响建议，开发者可以编辑保存；建议不会自动改动依据、检查计划或确认状态。
3. **验证与交接**：确认不可变快照，用稳定 run UUID 提交 TaskFlow；查询实际 attempt、逐项结果和日志。必要检查在当前 generation 全部通过，且快照仍对应当前输入，才能人工接收关闭。

模型未配置或调用失败时保留明确错误。新建议调用显式关闭 thinking，沿用上游模型适配，不将推理片段当作完成的评审结论。真实模型/检索与浏览器验收结果由总集成证据记录。

建议使用既定 2048-token 输出预算，提示最多六条简明意见，覆盖前后端、兼容性、缺失事实与注册检查，并最多列出两项 `path@commit` 依据。模型返回 `finish_reason=length` 时明确报错，不保存截断正文、不覆盖原草稿、不增加 revision，也不自动重试；正常 `stop` 继续保存。

## 状态与数据归属

- `revision` 每次实际编辑递增，用于 CAS；`input_revision` 仅在目标、业务包、代码、引用、检查计划变化时递增。
- 标题、备注、负责人及建议草稿不使已验证输入失效。
- 快照保存确认时的两种 revision、完整 input、实际确认人，不能覆盖。
- `change_run_links` 只保存提交意图、错误与关联；任务状态和检查报告以 TaskFlow 为准。
- 网络返回不明时保存 `unknown`，使用同一 UUID 查询/恢复。明确拒绝保存 `rejected` 并返回错误；人工重跑创建另一个 run，历史结果保留。
- 浏览器把待核对的 UUID 与提交快照成对保存。“核对并恢复原提交”固定使用原快照；“提交当前快照的新运行”创建新 UUID，不让旧的 unknown 阻塞当前输入。历史输入尚未被 TaskFlow 受理时，只允许核对，不再启动该旧输入的检查。
- `GET` 和 MCP 读取不提交、重试或修改执行任务。MCP 读报告不会把 TaskFlow 的结果改写到另一套状态机。
- 接收交接在 CAS 事务中关闭当前 active 变更。拒收追加说明、保留 active。

## 权限与执行范围

- 所有仓库、工作空间、业务包和检查映射由服务端注册，请求不提供仓库路径、命令或 Token。
- Viewer 可读同空间记录；Contributor 可创建并修改自己创建的变更；Admin/Owner 可管理空间记录；接收/拒收由有读取权限的 Contributor 以上成员执行，记录真实操作者。
- API Key 读取必须声明 `read_changes` 或具有既有 full-access；原 `retrieve` 不附带新增记录权限。机器写操作不可用。
- 两个 MCP 工具是 `get_change_context`、`get_verification_run`，需 endpoint 显式选择，不加入默认工具白名单。
- KB 受限消费者的每个来源必须关联获授权 knowledge 文档；没有映射时拒绝读取，而不是把 Git 原文当成无范围资料。
- 文档权限在返回引用、模型草稿及上下文时重新校验，不从缓存正文绕过。v1 不聚合跨工作空间知识。
- 变更 GET、列表、PATCH 和建议写入响应仅返回元数据，不附带建议正文。Context 返回建议时检查其原始依据；原始文档删除或撤权后，创建者仍可修改元数据、移除失效引用，或发送 `content: ""` 清空旧建议再恢复读取。清空与重新生成只增加编辑 revision，不改变 input_revision。
- 验证报告和日志按其关联的确认快照重校验每项来源，不能用当前草稿的知识范围代替历史快照权限。直接越界读取返回权限错误；运行列表保留关联元数据，并为不可读的报告显示 `execution_error`，不附带报告正文。
- 只运行注册的可信仓库、确定 commit 和固定检查模板；不新增任意 shell、生产发布或自动代码提交。

## 新增 HTTP 与工具

新增接口位于 `/api/v1/changes`：创建/列表/读取/PATCH；`:id/context`；`:id/context-draft`；`:id/snapshots`；`:id/runs`、`:id/runs/:run_id` 及受保护的 artifact；`:id/handoffs`。

确认快照需要 `expected_revision`。提交检查需要 `snapshot_id`、调用方预先生成的 `run_id` 和 `expected_revision`。交接需要 `run_id`、`conclusion=accepted|rejected`、`notes` 和 `expected_revision`。

`POST :id/context-draft` 的 `content` 字段为空字符串时显式清空草稿；非空时保存人工编辑，未提供 `content` 时才使用 `model_id` 生成建议。读建议正文使用 `GET :id/context`。

```text
weknora changes context <change-id> --format json
weknora changes runs get <change-id> <run-id> --format json
```

CLI 沿用上游 `{ok,data,meta,profile}` 成功 envelope、stderr 错误和 auth/profile 行为；失败结果仍以原 status/verdict 返回，不改成成功。

## 数据库迁移与回退

PostgreSQL 新迁移为 `000125`–`000127`，SQLite 为 `000042`–`000044`。沿用上游版本化迁移，不用启动时 AutoMigrate 改造旧知识表。关闭 `change_workspace.enabled` 后旧知识功能继续可用。

回退时先停止新提交并处理/保留已有 run，停止本次开发进程，保留数据库与报告。这里提供的 down 迁移仅供明确清退新增数据时使用，不能在普通应用回退中自动执行。

## 定向检查

```text
python scripts/test-change-sqlite.py
go test ./internal/integration/taskflowclient
go test ./internal/application/service -run '^TestChange'
go test ./internal/application/repository -run '^TestChange'
go test ./internal/types -run 'MCPEndpoint|Change'
go test ./internal/mcpserver
cd client && go test ./...
cd ../cli && go test ./... && go vet ./...
cd ../frontend && npm run type-check && npm run check-i18n && npm run build
```

上游服务需要 Linux/CGo。Windows 原生出现的 `pg_query.Parse/Deparse undefined` 是该依赖组合的实际构建限制，本次使用独立 WSL Linux 缓存构建，不删除上游功能来绕过。真实服务、PostgreSQL、TaskFlow、模型、MCP 协议与浏览器操作必须分别记录，不用确定性替身测试冒充这些结果。
