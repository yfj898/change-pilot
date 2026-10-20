# ChangePilot 公开交付与验证范围

三个仓库以独立职责公开维护：

- [ChangePilot](https://github.com/yfj898/change-pilot)：需求、版本化依据、建议、确认快照、检查报告展示与人工交接。
- [TaskFlow-Go vNext](https://github.com/yfj898/taskflow-go-next)：注册可信仓库与固定检查，管理 run/Task/Outbox、generation 和原始日志。
- [Go/Vue 业务样例](https://github.com/yfj898/change-workspace-demo)：商品预售和 SaaS 工单，SQLite 持久化、业务规则与受控故障/修复提交。

## 有效证据对应什么

业务与协议的实测范围见 [验证记录](CHANGE_WORKSPACE_VALIDATION_20261020.md)，源码配置见 [独立启动说明](CHANGE_PILOT_QUICKSTART.md)。模型建议、真实检查和人工接收分别记录，不能相互代替。

TaskFlow 的 [verification](https://github.com/yfj898/taskflow-go-next/actions/runs/39405966732) 和 [Supply Chain](https://github.com/yfj898/taskflow-go-next/actions/runs/39405966688)，以及业务样例的 [business-checks](https://github.com/yfj898/change-workspace-demo/actions/runs/39405422643) 均已在 GitHub 实际通过。工作台前端的当前检查结果从 [Frontend 工作流](https://github.com/yfj898/change-pilot/actions/workflows/frontend.yml) 查看；它保留单元测试、样式规范、类型检查、构建、Docker UI 和 embed/MCP 代理路由检查。

首轮前端 CI 发现设计 token fallback 与字号字面量超过现有基线两项错误，使用项目现有 token 修正源码，不放宽 style guard。代码相关检查和仅文档更新分开判断，文档提交不代表重新执行全部业务用例。

本轮 Windows 原 `npm test` 全套为 1875 项：1866 通过、8 失败、1 跳过；8 项继承用例涉及 CRLF、路径分隔符和 POSIX shell，在首轮 Linux CI 中均已通过。Windows 的样式规范 10/10 与生产构建通过；完整前端测试以 Linux 工作流为验收平台，不把 Windows 全套记为通过。

## 可选 DSH 适配器尚有上游兼容问题

继承的 `packages/dsh-weknora` 是 DeepSeek Harness 适配器，与本次新增的工作空间 HTTP MCP 入口分开。

[首轮 DSH 工作流](https://github.com/yfj898/change-pilot/actions/runs/39407857108) 的 TypeScript、单元测试、Go 类型契约与打包清单检查通过；真实 Harness E2E 在固定 `@deepseek-ai/dsh@0.1.0-rc.8` 启动时失败，原因是 `user patch-layer watching requires the Cordis HMR service`。该运行还未进入工具调用断言，不能作为 DSH 可用性的验证结果。

需要 DSH 时，应先解决固定 Harness 的启动兼容性，再执行该适配器的 E2E。当前可复现的工具入口是已验收的 CLI 与工作空间 HTTP MCP，后者明确选择 `get_change_context`、`get_verification_run` 两个只读工具。DSH 失败没有改写这些入口的实测结果，原失败工作流与门禁也保留。

## 公开源码与本机运行配置分开

保留 Tencent/WeKnora 的固定基线历史、LICENSE、第三方声明和原版 README。ChangePilot 是展示品牌，Go module、`weknora` CLI、API、存储键和环境变量维持兼容。

公开仓库只包含源码、说明、受控 fixture 和脱敏证据；本机数据库、登录会话、服务令牌、模型、缓存及个人简历均不在源码交付范围内。软件包与镜像的上游发布 job 受仓库守卫限制，本次源码公开不会发布到上游账号。
