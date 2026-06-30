---
name: release
description: 发布 tke-extend-network-controller 新版本。当用户提到"发布版本"、"release 新版本"、"发个新版本"、"publish/release vX.Y.Z"、或要求跑 `make release` 创建 GitHub Release 时，使用此 skill。它串联 CHANGELOG 版本号确认 → make release（更新 Chart.yaml + 打 tag + buildx 多架构镜像推送）→ git push → gh release create 全流程。务必在用户提出发布新版本时主动触发，即使用户只说"发一下 v2.5.0"。
---

# 发布新版本

本 skill 串联 tke-extend-network-controller 的完整发布流程。发布是不可逆的对外操作（推送公网镜像、打 git tag、创建 GitHub Release），执行前务必完成预检，执行中逐步说明。

## 流程总览

```
检查 CHANGELOG 确定版本号
        │
        ▼
发布前预检（git 干净 / tag 不存在 / gh 已登录 / buildx 与 dockerhub 凭证就绪）
        │
        ▼
更新 CHANGELOG 发布日期 → git add（进 release commit）
        │
        ▼
make release VERSION=x.y.z（无 v 前缀）
  ├─ 更新 Chart.yaml (version & appVersion)
  ├─ git commit "release vX.Y.Z"（含 CHANGELOG 日期更新）
  ├─ git tag vX.Y.Z
  └─ docker buildx 构建并推送 4 架构镜像（耗时主体，建议后台运行）
        │
        ▼
git push origin main && git push origin vX.Y.Z
        │
        ▼
gh release create vX.Y.Z --title "vX.Y.Z Release" --notes-file <从 CHANGELOG 提取>
```

## Step 1：确定版本号

读取 `CHANGELOG.md` 顶部。待发布版本通常标注 `(发布时间待定)`，例如：

```markdown
## v2.4.2 (发布时间待定)
```

- 版本号即 `2.4.2`（无 v 前缀，用于 `make release VERSION=`）。
- 若用户已显式指定版本（如"发 v2.5.0"），以用户指定为准，但仍要核对 CHANGELOG 是否已包含该版本条目——若 CHANGELOG 顶部没有对应版本块，先与用户确认。

## Step 2：发布前预检（不可跳过）

发布是对外不可逆操作，预检能避免中途失败留下半成品状态。逐项检查：

1. **工作区干净**：`git status --short` 应无输出。release commit 会把已 stage 的改动一并提交，若有未预期的改动会混入发布提交。
2. **tag 不存在**：`git tag -l vX.Y.Z` 应为空；本地与远程都要确认（`git ls-remote --tags origin vX.Y.Z`）。重复打 tag 会失败。
3. **Chart.yaml 当前版本**：确认 `charts/tke-extend-network-controller/Chart.yaml` 的 `version`/`appVersion` 仍是上一版本，`make release` 会把它改成新版本。
4. **gh 已登录且有 repo scope**：`gh auth status`。本仓库 remote 是 `tkestack/tke-extend-network-controller`（org 仓库），确认 gh 能访问。
5. **docker buildx 可用且 dockerhub 已登录**：`docker buildx version` + 检查 `~/.docker/config.json` 含 `https://index.docker.io/v1/`。镜像推送到 `imroc/tke-extend-network-controller`，需要 dockerhub 推送凭证。

任一不满足则停下，向用户报告缺什么、该如何补救（如 `gh auth login`、`docker login`），不要自行尝试登录。

## Step 3：更新 CHANGELOG 发布日期

待发布版本块从 `(发布时间待定)` 改为实际日期。**用系统真实日期，不盲信上下文里的 currentDate**：

```bash
date "+%Y-%m-%d"
```

然后用 Edit 把 `## vX.Y.Z (发布时间待定)` 改为 `## vX.Y.Z (YYYY-MM-DD)`。

**关键：随后必须 `git add CHANGELOG.md`**。原因——`make release` 只 stage `Chart.yaml`，但它的 `git commit` 会提交所有已 stage 的改动。先 stage CHANGELOG，日期更新就会和 Chart.yaml 版本更新一起进入同一个 `release vX.Y.Z` commit，发布历史才干净一致。否则 CHANGELOG 日期更新会变成游离改动，需要额外提交。

## Step 4：执行 make release

```bash
make release VERSION=x.y.z   # 注意：无 v 前缀
```

它做的事（见 Makefile `release` target）：

1. 校验 VERSION 格式 `^[0-9]+\.[0-9]+\.[0-9]+$`
2. `sed` 更新 `Chart.yaml` 的 `version` 和 `appVersion`
3. `git add Chart.yaml` + `git commit -m "release vX.Y.Z"`（含你 stage 的 CHANGELOG）
4. `git tag vX.Y.Z`
5. 调 `make docker-buildx`：构建 `linux/arm64,linux/amd64,linux/s390x,linux/ppc64le` 4 架构并 `--push` 到 `imroc/tke-extend-network-controller:X.Y.Z`

### 后台运行 buildx

第 5 步多架构构建+推送是耗时主体（首次拉基础镜像+交叉编译+推 4 架构，通常数分钟到十几分钟）。**用 `run_in_background: true` 后台执行**，避免阻塞。完成后 harness 会通知。

后台执行后，读一次早期输出确认前半段（commit + tag）成功、buildx 已开始构建——若前半段就出错（如 tag 已存在），不必空等 buildx。

### 镜像 tag 由 git tag 决定

`make release` 不传 IMG，子 make 的 `docker-buildx` 通过 `git describe --tags --exact-match` 取当前 git tag（去掉 v 前缀）作为镜像 tag。所以 Step 4 打的 `vX.Y.Z` tag 直接决定镜像 tag 为 `X.Y.Z`。这也是为什么 tag 必须在 buildx 之前打好——`make release` 已保证这个顺序。

### buildx 完成判定

成功标志（输出尾部）：

- `pushing manifest for docker.io/imroc/tke-extend-network-controller:X.Y.Z@sha256:...`
- `>> Released vX.Y.Z. Remember to push: git push && git push origin vX.Y.Z`
- `docker buildx rm` 清理 builder、`rm Dockerfile.cross`、工作区恢复干净

exit code 0 即成功。若失败，**不要 push**——镜像未推成功就 push tag/创建 release 会导致用户拉到不存在的镜像。先排查构建日志。

## Step 5：推送 git 提交与 tag

镜像推送成功后：

```bash
git push origin main
git push origin vX.Y.Z
```

顺序：先推 main（release commit），再推 tag。两者都成功才算代码对外发布完成。

## Step 6：创建 GitHub Release

### 提取 release notes

**用 awk 从 CHANGELOG 精确提取对应版本块，不要手动转录**（手抄易漏字、易改写，且与 CHANGELOG 不一致）。提取 `## vX.Y.Z` 到下一个 `## ` 之间的非空行：

```bash
awk '/^## vX\.Y\.Z/{flag=1;next} /^## /{flag=0} flag' CHANGELOG.md | awk 'NF' > /tmp/vX.Y.Z-release-notes.md
```

注意 awk 正则里的 `.` 要转义为 `\.`，版本号点号是字面量。提取后 `cat` 核对条数与 CHANGELOG 一致。

本仓库 release notes 风格是 CHANGELOG 条目直接列表，不加额外标题或包装（参考历史 release `v2.4.1 Release` 的 body）。

### 创建 release

```bash
gh release create vX.Y.Z \
  --repo tkestack/tke-extend-network-controller \
  --title "vX.Y.Z Release" \
  --target main \
  --notes-file /tmp/vX.Y.Z-release-notes.md
```

- 标题固定格式 `vX.Y.Z Release`（带 v 前缀，与历史 release 一致）。
- `--target main` 指向 release commit 所在分支。
- 不加 `--draft` / `--prerelease`（正式版本直接发布）。

### 核验

```bash
gh release view vX.Y.Z --repo tkestack/tke-extend-network-controller
```

确认 title、tag、非 draft/prerelease、notes 内容与 CHANGELOG 一致。返回 release URL 给用户。

## 注意事项

- **版本号前缀**：`make release VERSION=` 用无 v 前缀（`2.4.2`）；git tag、gh release tag、release 标题用带 v 前缀（`v2.4.2`、`v2.4.2 Release`）。两处不要搞混。
- **本仓库存在两类 release**：`vX.Y.Z Release`（控制器版本，tag `vX.Y.Z`，本 skill 产出）和 `tke-extend-network-controller-X.Y.Z`（chart 版本）。本 skill 创建的是前者。
- **不要并发发布**：buildx 运行期间不要重复执行 `make release`，会重复创建 builder、打 tag 冲突。
- **失败回滚**：若 buildx 失败，本地已有 commit 和 tag 但未 push。可 `git tag -d vX.Y.Z` 删本地 tag、`git reset --hard HEAD^` 撤 release commit（仅当确认未 push 时），修正后重跑。已 push 的 tag/commit 不要轻易回滚，宁可发补丁版本。
