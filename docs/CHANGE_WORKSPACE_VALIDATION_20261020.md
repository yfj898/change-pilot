# ChangePilot 验证记录

日期：2026-10-20。上游基线为 `737d237283ee1a8abd19a4755ad3197f79ce1303`，本地分支为 `feature/change-workspace`。

这里记录模块修复、真实服务集成与检查范围。下文 `tmp/` 日志、运行二进制及父工作区档案保存在验收机器，不随源码发布，也不是独立 clone 的启动依赖；源码启动见 [CHANGE_PILOT_QUICKSTART.md](./CHANGE_PILOT_QUICKSTART.md)。

## 用于总集成的产物

最终完整服务二进制位于 `tmp/weknora-server-linux-final`，由 Linux Go 1.26.4 对 `./cmd/server` 构建，2026-10-20 12:37:20 完成，文件为 358,033,072 字节，包含模型截断修复。构建没有删除上游数据库、模型或文档能力。

初次集成所用 `tmp/weknora-server-linux` 保留原文件，358,028,472 字节，时间仍为 12:14:18；它不包含之后的截断修复。新文件采用不同路径，交由总集成负责人决定切换，不覆盖执行中的程序。

前端最终产物位于 `frontend/dist`。总集成负责人继续使用已启动的 Vite 15173 和后端 18088 配置，统一管理进程、凭据及临时模型入口。本次模块交付未启动服务。

## 本轮修复的三个交互与授权问题

### 旧建议的文档失效后，仍可修复变更

`Get`、`List`、`Patch` 与建议写入响应只返回元数据。建议正文只通过 `Context` 返回，并重新校验该建议的原始来源。Repository 列表原有的 `Omit("draft")` 保持不变。

创建者或管理员不再因为历史建议的来源被删除、撤权而失去所有编辑入口。未改变验证输入的元数据编辑不重复解析旧来源；修改后的输入仍正常校验。上下文页面先加载元数据，正文失败时保留引用编辑和清空草稿操作。

`POST :id/context-draft` 的 `content: ""` 显式清空草稿，并增加编辑 revision；input_revision 不变。生成新建议只读取当前已保存依据，不再通过旧建议正文。修正引用后，可清空或重新生成旧建议，恢复上下文读取。

### 待核对运行不会占用新快照

浏览器用一条存储记录保存待核对的 `run_id` 与 `snapshot_id` 对，并保留多个未决提交。旧的两键存储格式按原配对读取。

“核对并恢复原提交”使用原有配对；“提交当前快照的新运行”生成新的 UUID。恢复历史 A 时若 TaskFlow 已有记录，读取其实际结果；若历史 A 尚未被受理，不启动其旧输入。当前确认的 B 可以独立提交，A 的关联仍保留。

### 运行报告使用其关联快照的来源权限

`Run` 在调用 TaskFlow 前重新授权关联快照的全部来源；`Artifact` 经同一路径后才获取日志；提交恢复与交接也不能绕过该边界。KB 受限 API Key 不能以当前草稿的 kbA 范围读取历史 kbB 快照的报告，缺少 knowledge 映射的来源也不会被视为无范围资料。

直接无权读取返回明确错误。运行列表保留本工作空间的关联元数据；不可读的历史项带 `execution_error`，不附带运行正文，从而不阻塞其他合法报告的显示。

## 已完成的检查及其范围

### 完整 Linux 服务构建通过

构建命令为：

```sh
go build -p=2 -o tmp/weknora-server-linux ./cmd/server
```

使用现有 WSL Ubuntu-24.04、CGo、独立 Linux GOCACHE 和现有 module 源码缓存。命令级 GOPROXY 为 `https://goproxy.cn`，没有关闭 Go module 校验。

首轮构建实际报 `sqlite3.h: No such file or directory`。总集成负责人安装 `libsqlite3-dev` 后，同一完整入口构建通过。保留的真实日志：

- 首轮原因：`tmp/linux-build-20261020.log`。
- 修复后通过：`tmp/linux-build-20261020-retry.log`。
- 依赖准备：`tmp/linux-deps-20261020.log`。

Windows 原生定向服务测试仍受 `pg_query.Parse/Deparse undefined` 限制；该平台结果不记为通过，Linux 结果提供本次服务验收基础。

### 完整 service 包中的定向回归通过

```sh
go test -p=2 -count=1 -v ./internal/application/service -run '^TestChange'
```

10 个 Change Workspace 顶层测试通过，其中包含 8 个撤权恢复、当前/历史快照与 knowledge 映射组合子场景。这个筛选另命中 3 个上游改密码测试，也均通过。日志为 `tmp/linux-change-tests-20261020.log`。

新增用例覆盖：

- 来源撤权或删除后的元数据读取、带原输入的标题修改、移除失效关联、显式清空、重新生成。
- GET、列表、PATCH 与建议写入响应不返回未经正文授权的草稿。
- 无 Draft 的运行，以及当前 Draft 属于 kbA、历史快照属于 kbB 的运行权限。
- KB 受限读取缺少 knowledge 映射时被拒绝；无权限请求不调用 TaskFlow 服务凭据传输。
- 文档删除后的报告、日志和接收交接被拒绝。
- A 提交结果不明后，B 独立提交；A 恢复不重启历史输入、不产生额外执行。

这些 service 测试使用内存 repository、确定性模型和 verifier，验证应用规则及调用边界；没有把它们计为真实 PostgreSQL、TaskFlow 或 Ollama 调用。

### 前端状态、类型与构建通过

```sh
node node_modules/tsx/dist/cli.mjs --test src/views/changes/verificationIntent.test.ts
npm run type-check
npm run check-i18n
npm run build
```

- 提交意图状态函数：6/6 通过。
- Vue TypeScript 检查通过。
- i18n 审核：15/15 通过。
- 最终 Vite 构建通过，日志为 `tmp/frontend-build-20261020.log`。上游大型资源 chunk 警告仍存在；本次未扩展为全站打包优化任务。

状态函数测试验证配对不重绑、A/B 并存、旧格式恢复和仅以实际结果消解未知提交；它们没有代替真实浏览器点击。

## T17：真实 PostgreSQL 基线 124 → 增量 127

仅在同一隔离容器新建的 `weknora_change_migration_test` 上执行，没有在 `weknora_change` 应用库运行 DDL。实际版本为 PostgreSQL 17.11，ParadeDB/pg_search 0.25.11；容器使用 `paradedb/paradedb:v0.25.11-pg17`。

先通过上游使用的 golang-migrate 对完整版本化迁移源码执行到 124，确认 `schema_migrations=(124,false)`。再直接调用服务启动所用 `database.RunMigrationsWithOptions`，禁用自动 dirty 恢复，实际升级至 `(127,false)`。重复调用该接口仍为 127，模式无变化，没有 force 迁移版本。

基线清单与增量后的实际数据库定义逐项比较：

- 78 张基线表、1108 列、334 个索引、113 个约束原定义保留。78 张表含新数据库模板继承的 PostGIS `spatial_ref_sys`，不是 78 张新增业务表。
- 两条既有 tenant/model 测试夹具的字段值保留。
- 新增表恰为 `change_requests`、`change_snapshots`、`change_run_links`、`change_handoffs`；`change_requests.draft` 列存在。
- 实际写入四张新表，并分别验证快照输入版本唯一约束、变更 stage、提交 state、交接 conclusion 的 CHECK 约束，以及 snapshot 外键拒绝无效值。SQLSTATE 分别为 23505、23514、23503。

证据位于 `tmp/t17-postgres-migration.log` 与 `tmp/t17-postgres-result.json`，后者包含完整前后清单。一次性数据库保留供总集成检查。

基线准备记录在 `tmp/t17-postgres-baseline124.log`：0→124 已成功，随后测试夹具误用 124 已删除的 `tenants.api_key` 列而中断。依据 124 的实际字段修正夹具后，在同一干净 124 基线上完成增量；迁移源码未因此调整。此次验证覆盖全新上游基线和两条保留夹具，不覆盖生产数据规模、并发迁移或 PostgreSQL down 回退。已有 SQLite 证据继续复用。

## 模型输出截断：红灯复现与定向修复

真实建议曾返回非空但截断的正文。新增回归先在原实现复现：`finish_reason=length` 分支得到 nil 错误，测试失败；同一轮 `stop` 分支通过。红灯日志为 `tmp/advice-length-red.log`。

修复后的 `PrepareDraft` 在 `length` 时返回 HTTP 503，错误详情包含 `finish_reason` 和 `max_tokens=2048`，原 Draft、revision 和 input_revision 完全不变，模型只调用一次。提示收敛为最多六条简明意见及两项必要依据，保持原输出预算。没有改写上游 Ollama provider。

修复后，相关 Change 源文件和测试的定向运行全部通过：11 个顶层测试，包含上述 `length` 与 `stop` 两分支，以及此前撤权、历史快照和 unknown 恢复用例。日志为 `tmp/advice-length-green.log`。该检查编译和运行列出的真实应用方法，使用确定性模型响应；真实模型再验收由总集成执行。

定向回归后，完整 `./cmd/server` 构建到独立的 `tmp/weknora-server-linux-final`，退出码为 0，日志为 `tmp/linux-build-final-20261020.log`。该后续修复没有前端变化，复用此前有效的前端证据。

## 总集成已消费最终服务

2026-10-20，总集成负责人已切换并运行包含截断修复的最终二进制。真实 `qwen3.5:4b` 简明建议生成完成，开发者在页面修正了 null 展示的歧义措辞，随后独立确认快照；模型没有自动采用依据或接收交接。

商品变更实际经历故障提交失败、页面拒收、修复 commit 三项必要检查通过与页面人工接收；SaaS 工单使用同一核心完成第二业务交接。真实 HTTP、CLI、HTTP MCP 协议客户端读取同一工单 context/run 完全一致，实际浏览器另行核对同一记录。三份中文文档已真实入库，两条向量查询首位命中对应规则；这两条样例不是检索准确率基准。

新增功能关闭冒烟已完成：旧知识库和已解析文档读取均为 200，changes 明确 503；随后已恢复新功能、TaskFlow 直连并停止故障代理。用户批准的临时模型入口已关闭，原 Windows 回环 Ollama 保留。完整运行与浏览器原始档案保存在验收者本机的 `docs/resume_jd_20261019/acceptance_results_20261020.md`，不作为仓库内链接。

## ChangePilot 展示更名与 fork 工作流

产品页面标题、favicon、导航、登录和应用助手等产品名称统一为 ChangePilot。品牌与仓库链接集中于 `frontend/src/config/branding.ts`。登录页面描述版本化依据、注册检查和人工交接；上游官网入口明确标注“基于 WeKnora”。

Go module、`weknora` CLI 命令、API、存储键、环境变量、WeKnora Cloud 供应商与上游标准沙箱模板名称保持兼容。原 README 全文保存到 `README.upstream.md`；首页单列上游能力、本项目新增与 TaskFlow 客户端来源。

更名后的 Vue TypeScript 检查、15/15 i18n 审核和 Vite 生产构建通过。完整服务源码没有行为变化，复用前述 Linux 构建与集成证据；本轮没有重编后端。大型上游 chunk 警告仍存在。

GitHub 的 13 份 workflow YAML 已实际解析。10 个上游 Docker、PyPI、npm、Lite Release/Homebrew 与定时 drift job 设置 `github.repository == 'Tencent/WeKnora'` 精确守卫，个人仓库 main 推送不会向这些上游发布目标外发；DSH 定时测试也只在上游自动运行。App、Frontend、CLI、Helm、Anydoc、DocReader 等常规检查保留。外部服务器 CLI E2E 在个人仓库仅可显式手动触发，仍需操作者配置测试凭据。
