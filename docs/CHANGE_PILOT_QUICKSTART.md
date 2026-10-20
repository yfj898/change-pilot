# 从源码运行 ChangePilot

完成后可以在 `/platform/changes` 创建变更，引用业务仓库的确定版本，确认快照并提交真实检查。建议将三个仓库 clone 在同一开发目录，分别保留自己的运行数据。

## 1. 准备 Linux 开发环境

已验收环境为 WSL Ubuntu-24.04、Go 1.26.4、Node.js 24、PostgreSQL 17/ParadeDB 0.25.11 与 Redis 7。后端包含 CGo 依赖，Windows 原生构建在上游 `pg_query` 组合下受限，使用 Linux/WSL 构建完整服务。

```bash
# Ubuntu，已有依赖则无需重复安装
sudo apt-get update
sudo apt-get install -y build-essential libsqlite3-dev git curl openssl

# 安装 Go 1.26 和 Node.js 24 后检查
go version
node --version
npm --version

git clone https://github.com/yfj898/change-pilot.git
git clone https://github.com/yfj898/taskflow-go-next.git
git clone https://github.com/yfj898/change-workspace-demo.git
cd change-pilot
```

## 2. 启动 PostgreSQL 与 Redis

复用仓库现有开发基础设施，不需要先构建上游 Docker 应用镜像。

**启动前先限制数据库端口为回环地址。** 在 `docker-compose.dev.yml` 中，将 `postgres.ports` 改为 `127.0.0.1:${DB_PORT:-5432}:5432`，将 `redis.ports` 改为 `127.0.0.1:${REDIS_PORT:-6379}:6379`；原文件默认绑定全部宿主机接口。

```bash
cp .env.example .env
# 编辑 .env：设置本机未占用的 DB_PORT 与 REDIS_PORT、DB_USER、DB_NAME，
# 并为 DB_PASSWORD、REDIS_PASSWORD 设置自己的强密码。
# 按上文完成 docker-compose.dev.yml 的两个回环端口映射后，再启动：
docker compose -f docker-compose.dev.yml up -d postgres redis
docker compose -f docker-compose.dev.yml ps
```

保留 named volume，普通停止用 `down`，不要加 `-v`。

后端裸二进制读取**进程环境**，不会自动加载 `.env`。在运行后端的终端中设置以下值；数据库名称、用户名、密码必须与刚创建的容器一致。

```bash
export DB_DRIVER=postgres
export DB_HOST=127.0.0.1
export DB_PORT=5432                 # 使用 .env 中实际映射端口
export DB_USER=changepilot          # 使用 .env 中实际用户名
export DB_NAME=changepilot          # 使用 .env 中实际数据库名
read -r -s -p 'PostgreSQL password: ' DB_PASSWORD; echo; export DB_PASSWORD

export REDIS_ADDR=127.0.0.1:6379     # 使用实际映射端口
read -r -s -p 'Redis password: ' REDIS_PASSWORD; echo; export REDIS_PASSWORD
export STREAM_MANAGER_TYPE=redis
export RETRIEVE_DRIVER=postgres
export STORAGE_TYPE=local
export LOCAL_STORAGE_BASE_DIR="$PWD/data/files"
export WEKNORA_TENANT_ENABLE_RBAC=true
export AUTO_MIGRATE=true
export OLLAMA_OPTIONAL=true
export GIN_MODE=release
mkdir -p tmp data/files
```

初次运行时生成并保存 JWT/AES 密钥，重启继续使用同一份。下面文件仅含本机生成的十六进制值，位于被 Git 忽略的 `tmp/`：

```bash
umask 077
if [ ! -e tmp/change-pilot.keys.env ]; then
  printf 'JWT_SECRET=%s\nSYSTEM_AES_KEY=%s\n' \
    "$(openssl rand -hex 32)" "$(openssl rand -hex 16)" > tmp/change-pilot.keys.env
fi
set -a
. tmp/change-pilot.keys.env
set +a
```

保留密钥才能继续解密已有模型/API 凭据。不要将 `.env`、此密钥文件或数据库上传 Git。

## 3. 构建并启动本仓库后端与前端

在 `config/config.yaml` 中保留完整上游配置，将 `server.host` 设为 `127.0.0.1`、`server.port` 设为 `18088`。新功能暂不启用，先完成注册。

```bash
go mod download
go build -p=2 -o tmp/weknora-server-linux ./cmd/server
# 保持 cwd 为仓库根目录，供 config/、migrations/ 和 prompt_templates/ 解析
./tmp/weknora-server-linux
```

启动会按上游机制执行版本化迁移。已有数据库升级前先备份；新增 PostgreSQL 迁移为 125–127，SQLite 为 42–44。不要对共享或生产库用测试数据清理、force 或 down 迁移。

另开终端：

```bash
cd change-pilot/frontend
npm ci
VITE_DEV_PROXY_TARGET=http://127.0.0.1:18088 \
  npm run dev -- --host 127.0.0.1 --port 15173 --strictPort
```

打开 `http://127.0.0.1:15173` 注册本地账户，在工作空间设置中确认**实际 tenant ID**。注册产生的空间 ID 由数据库分配，不应固定假设为 `1`。

本路线使用手工 Markdown 业务文档，可使用上游内置解析路径；Office/PDF 等扩展解析能力按 [上游文档](https://weknora.weixin.qq.com/docs/) 配置相应引擎。完整 Docker 镜像默认还构建 BrowserSkill/Anydoc，首次构建成本高于此源码开发路线。

## 4. 配置可信执行端与业务映射

先在兄弟仓库按 [TaskFlow verification quickstart](https://github.com/yfj898/taskflow-go-next/blob/main/docs/VERIFICATION_QUICKSTART.md) 启动服务、配置可信项目与固定检查。确认 TaskFlow 能访问业务 Git 仓库，并为该注册项安装 Go/Node 与所需依赖。核对服务主体、profile ID/revision 与检查 ID。

停止本次 ChangePilot 开发进程，将 [`config/change-workspace.example.yaml`](../config/change-workspace.example.yaml) 的完整 `change_workspace` 段合入 `config/config.yaml`，按本机环境修改：

| 配置 | 应填内容 |
| --- | --- |
| `enabled` | `true` |
| `taskflow_endpoint` | 当前 ChangePilot 进程可访问的 TaskFlow HTTP 地址 |
| `taskflow_token_env` | `TASKFLOW_ACCESS_TOKEN`；字段保存环境变量名，不是 Token 本文 |
| 每项 `tenant_id` | 上一步登录账户当前工作空间的实际 ID |
| `repository_path` | 当前 Linux 进程可读的业务仓库绝对路径，例如 `/home/me/dev/change-workspace-demo` |
| `id`、业务包与检查字段 | 与业务包、TaskFlow 操作者注册值一致；示例包含商品与工单两项映射 |

每项映射使用 `change-workspace-demo` 项目、`verify-change` profile/revision `"1"` 及 `backend-test`、`frontend-test`、`frontend-build` 三项检查。TaskFlow 负责注册命令，ChangePilot 只发送这些固定 ID。两个进程可以位于不同目录；各自路径均由操作者配置，不能接受请求传入的目录或命令。

在此前后端终端保留数据库/密钥环境，输入执行端签发的凭据后重启同一二进制：

```bash
read -r -s -p 'TaskFlow service token: ' TASKFLOW_ACCESS_TOKEN; echo
export TASKFLOW_ACCESS_TOKEN
./tmp/weknora-server-linux
```

新页面位于 `http://127.0.0.1:15173/platform/changes`。代码和依据都使用完整 commit；ChangePilot 从 Git 对象重读来源，TaskFlow 在确定 commit 执行注册检查。人工接收前需要当前快照的所有必要检查通过。

## 5. 需要建议或知识检索时再配置模型

使用当前 Linux 后端**确实可达**的 Ollama 或企业 API。Windows 的 `127.0.0.1` 不是 WSL NAT 下的 Windows 宿主地址；不要通过暴露到物理网卡来代替部署网络规划。

本地验收使用 `qwen3.5:4b` 对话与 `qwen3-embedding:0.6b` 向量模型，后者实际输出 1024 维。在“设置 → 模型”中配置，或复用 [`config/builtin_models.yaml.example`](../config/builtin_models.yaml.example) 的声明式入口。使用 `source: local` 时，共用 `OLLAMA_BASE_URL`，模型 `name` 必须与 Ollama 安装名相同；embedding dimension 必须与模型输出一致。

生成建议需要可用的聊天模型；知识入库/向量检索需要 embedding 模型。模型调用失败保留明确错误，既有草稿不被错误结果覆盖。手工引用 Git 依据、确认快照与真实检查的链路可独立使用。

## 检查与更多接口

```bash
cd frontend
npm run type-check
npm run check-i18n
npm run build
```

后端定向回归、CLI/HTTP MCP 与权限边界见 [CHANGE_WORKSPACE.md](../CHANGE_WORKSPACE.md)。CLI 保留 `weknora` 命令名；新增变更读取命令必须从本仓库 `cli/` 构建，不能用上游二进制代替。

上游标准部署说明完整保留于 [README.upstream.md](../README.upstream.md)。其中 `docker compose pull` 拉取的是上游镜像，不能代替构建本二开的源码与配置。
