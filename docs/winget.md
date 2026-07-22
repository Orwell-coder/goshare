# WinGet 自动发布流程

## 概述

WinGet（Windows Package Manager）是微软官方的 Windows 包管理器。本项目通过 GitHub Actions 自动化 workflow，在每次推送版本 tag 时自动向 [winget-pkgs](https://github.com/microsoft/winget-pkgs) 仓库提交 manifest PR，审核通过后用户即可通过 `winget install Orwell-coder.goshare` 安装。

## 架构

两个 workflow 文件协作完成自动发布：

```
推送 v* tag
    │
    ▼
release.yml ─── 构建 -> 创建 GitHub Release
    │                │
    │          outputs: version, installer_url, installer_sha256
    │                │
    ▼                ▼
winget.yml ◄────────┘
    │
    ├─ Clone 用户的 winget-pkgs fork
    ├─ 从上游 master 创建分支
    ├─ 生成 3 个多文件 manifest（version / installer / locale）
    ├─ Push 到 fork
    └─ 创建 PR 到 microsoft/winget-pkgs
```

| 文件            | 触发条件                                 | 职责                                                 |
| --------------- | ---------------------------------------- | ---------------------------------------------------- |
| `release.yml` | `push: tags: ["v*"]`                   | 构建 exe、压缩、生成 GitHub Release、调用 winget.yml |
| `winget.yml`  | `workflow_call`（被 release.yml 调用） | 生成 manifest、提交 PR 到 winget-pkgs                |

### 为什么是两层架构

- 早期尝试在 `winget.yml` 中直接使用 `on: release: [published]` + 第三方 action（`vedantmgoyal9/winget-releaser`），但该 action **要求包必须已存在于 winget-pkgs 中**，首次发布无法使用
- 改为直接脚本提交 PR 的方式，初次和后续版本都能正常处理
- 拆分为两个文件可独立修改 winget 提交逻辑而不影响构建流程

---

## 一次性准备工作

以下操作只需在项目中执行一次。

### 1. Fork winget-pkgs

前往 https://github.com/microsoft/winget-pkgs ，点击右上角 **Fork** 按钮，fork 到你的 GitHub 账号下。

> **为什么需要 fork？** winget-pkgs 由微软维护，必须有 fork 才能提交 PR。

### 2. 创建 Personal Access Token (PAT)

1. 前往 https://github.com/settings/tokens -> **Generate new token (classic)**
2. 勾选 `public_repo` 权限
3. 生成后将 token 复制下来

### 3. 配置仓库 Secret

1. 前往仓库 **Settings -> Secrets and variables -> Actions**
2. 点击 **New repository secret**
3. Name: `WINGET_TOKEN`
4. Value: 粘贴上一步生成的 PAT

---

## 发布流程（开发者操作）

### 触发方式

推送一个 `v` 开头的 tag 即可触发整个流程：

```powershell
# 1. 确保所有改动已提交
git add .
git commit -m "feat: some new feature"

# 2. 创建并推送 tag
git tag -a v0.4.0 -m "v0.4.0"
git push origin v0.4.0
```

推送后 GitHub Actions 自动执行 `release.yml`，无需其他手动操作。

### Workflow 执行步骤

#### 第 1 阶段：release job（构建 + 发布）

| 步骤                | 说明                                                                                  |
| ------------------- | ------------------------------------------------------------------------------------- |
| Checkout            | 拉取代码                                                                              |
| Setup Go            | 安装 Go 1.24                                                                          |
| Cross-compile       | 交叉编译`GOOS=windows GOARCH=amd64`，生成 `goshare.exe`（不压缩，避免 AV 误报） |
| Generate checksum   | 生成`sha256sum` 校验文件                                                            |
| Set winget metadata | 提取 version、installer_url、installer_sha256，设为 job outputs                       |
| Generate manifest   | 生成 3 个 winget manifest 文件，随 Release 附带                                        |
| Create Release      | 使用`softprops/action-gh-release` 创建 GitHub Release，附带 exe + sha256 + manifest |

#### 第 2 阶段：winget job（提交 winget-pkgs PR）

接收 release job 的三个 outputs 作为 inputs：

| Input                | 示例值                                                                           |
| -------------------- | -------------------------------------------------------------------------------- |
| `version`          | `0.4.0`（已去除 `v` 前缀）                                                   |
| `installer_url`    | `https://github.com/Orwell-coder/goshare/releases/download/v0.4.0/goshare.exe` |
| `installer_sha256` | `c2d35cd74de4df9e...`                                                          |

具体步骤：

1. **Clone fork** - 使用 PAT 克隆 `Orwell-coder/winget-pkgs`
2. **同步上游** - 添加 `microsoft/winget-pkgs` 为 upstream，从 `upstream/master` 创建分支 `goshare-{version}`
3. **生成 manifest** - 在路径 `manifests/o/Orwell-coder/goshare/{version}/` 下创建 3 个多文件 manifest（version / installer / locale）
4. **Push** - 推送到 fork 的分支
5. **创建 PR** - 使用 `gh pr create` 向 `microsoft/winget-pkgs` 发起 Pull Request

---

## Manifest 格式

生成的 manifest 采用 **多文件（multi-file）** 格式，路径结构：

```
winget-pkgs/
└── manifests/
    └── o/                          # Publisher 首字母小写
        └── Orwell-coder/
            └── goshare/
                └── 0.4.0/
                    ├── Orwell-coder.goshare.yaml                  # version 清单
                    ├── Orwell-coder.goshare.installer.yaml         # installer 清单
                    └── Orwell-coder.goshare.locale.en-US.yaml      # defaultLocale 清单
```

> **重要：** winget-pkgs 已废弃 singleton（单文件）格式。提交 singleton 清单会被打上
> `Manifest-Validation-Error` 标签并拒绝合并。必须使用上述 3 文件的多文件格式，
> 且 `ManifestVersion` 使用 `1.12.0`（当前推荐版本）。

### 1. version 清单（`Orwell-coder.goshare.yaml`）

```yaml
# yaml-language-server: $schema=https://aka.ms/winget-manifest.version.1.12.0.schema.json

PackageIdentifier: Orwell-coder.goshare
PackageVersion: 0.4.0
DefaultLocale: en-US
ManifestType: version
ManifestVersion: 1.12.0
```

### 2. installer 清单（`Orwell-coder.goshare.installer.yaml`）

```yaml
# yaml-language-server: $schema=https://aka.ms/winget-manifest.installer.1.12.0.schema.json

PackageIdentifier: Orwell-coder.goshare
PackageVersion: 0.4.0
InstallerType: portable
Installers:
- Architecture: x64
  InstallerUrl: https://github.com/Orwell-coder/goshare/releases/download/v0.4.0/goshare.exe
  InstallerSha256: c2d35cd74de4df9e988ed24b3c4faeb0dad120ff16af503a9547f734f76d4a47
ManifestType: installer
ManifestVersion: 1.12.0
ReleaseDate: 2026-07-15
```

### 3. defaultLocale 清单（`Orwell-coder.goshare.locale.en-US.yaml`）

```yaml
# yaml-language-server: $schema=https://aka.ms/winget-manifest.defaultLocale.1.12.0.schema.json

PackageIdentifier: Orwell-coder.goshare
PackageVersion: 0.4.0
PackageLocale: en-US
Publisher: Orwell-coder
PublisherUrl: https://github.com/Orwell-coder/goshare
PackageName: GoShare
PackageUrl: https://github.com/Orwell-coder/goshare
License: MIT
ShortDescription: High-performance LAN file sharing tool
Description: GoShare is a single-binary file sharing tool for LAN environments. Provides both a browser-based HTTP interface and a high-speed TCP CLI client for fast folder transfers.
Tags:
- file-sharing
- lan
- cli
- go
ManifestType: defaultLocale
ManifestVersion: 1.12.0
```

### 关键字段说明

| 字段                  | 说明                                          | 注意事项                              |
| --------------------- | --------------------------------------------- | ------------------------------------- |
| `PackageIdentifier` | 全局唯一标识符，格式`<Publisher>.<Package>` | 一经确定不可更改；文件名必须与之完全一致（大小写敏感） |
| `PackageVersion`    | 版本号，与 tag 一致（去掉`v` 前缀）         | 必须与 Release 版本匹配               |
| `DefaultLocale`     | version 清单中指向默认 locale 文件            | 值为 `en-US`，对应 `.locale.en-US.yaml` |
| `InstallerType`     | `portable`（便携版 exe）                    | 本项目不是安装包，用 portable          |
| `InstallerUrl`      | 下载链接                                      | 必须是直链，不可为网页                 |
| `InstallerSha256`   | 文件的 SHA256 校验值                          | winget 下载后会自动校验                |
| `ReleaseDate`       | 发布日期，格式`YYYY-MM-DD`                  | 由 workflow 自动生成；放在 installer 清单 |
| `Architecture`      | `x64`（64 位）                              | 如需 32 位，添加`x86` entry          |
| `ManifestType`      | `version` / `installer` / `defaultLocale`  | 每个文件对应一种类型，不可用 singleton |
| `ManifestVersion`   | `1.12.0`                                     | winget manifest schema 版本（1.10.0 也可接受） |

---

## PR 创建后：需要手动操作的事项

Workflow 自动创建 PR 后，还需要你手动完成以下操作才能被合并：

### 1. 签署 CLA（必须）

微软要求所有 winget-pkgs 贡献者签署 Contributor License Agreement。

在 PR 评论中回复：

```
@microsoft-github-policy-service agree
```

CLA 签署后，`license/cla` 检查变为绿色 ✅。**只需签署一次**，后续 PR 自动通过。

### 2. 等待验证

winget-bot 会自动运行验证流水线，检查 manifest 格式、URL 可访问性、SHA256 校验等。

如果验证失败，查看 bot 评论了解具体错误，修复后重新推送 tag。

### 3. 等待审核

微软审核团队会 review PR。审核通过后 PR 被合并，用户即可通过 winget 安装。

审核通常需要 **几小时到几天**。

---

## 故障排查

### 问题 1：PR 中的 ReleaseDate 显示为 $(date +%Y-%m-%d)

**原因：** winget.yml 中 heredoc 使用了单引号 `<< 'MANIFEST'`，bash 不会展开 `$(date)`。

**解决：** 已修复为无引号 `<< MANIFEST`，同时预先计算 `RELEASE_DATE=$(date +%Y-%m-%d)`。

### 问题 2：workflow 报 "not reusable"

**原因：** `winget.yml` 缺少 `on.workflow_call` 触发器。

**解决：** 已添加 `workflow_call` 并定义输入参数。

### 问题 3：action 找不到

**原因：** `wednesday-solutions/winget-releaser` 不存在，`vedantmgoyal9/winget-releaser` 要求包已存在。

**解决：** 改为脚本直接提交 PR。

### 问题 4：包不存在错误

`Package XXX does not exist in the winget-pkgs repository`

**原因：** 使用了 `winget-releaser` action，该 action 只能更新已存在的包。

**解决：** 当前方案直接提交 manifest，首次和后续版本都能处理。

### 问题 5：PR 创建失败

**常见原因：**

- fork 不存在 -> 检查 `https://github.com/<你的用户名>/winget-pkgs` 是否存在
- PAT 过期或无权限 -> 检查 Secrets 中的 `WINGET_TOKEN`
- 分支名冲突 -> 同版本无需重复操作，检查是否已有同名分支

### 问题 6：PR 被打上 `Manifest-Validation-Error` 标签

**原因：** 提交了 singleton（单文件）清单。winget-pkgs 已废弃 singleton 格式，只接受多文件清单。

**解决：** 使用上面「Manifest 格式」中的 3 文件结构（version / installer / defaultLocale），`ManifestVersion` 设为 `1.12.0`。参考 [ValidationFailureGuide](https://github.com/microsoft/winget-pkgs/blob/master/doc/ValidationFailureGuide.md)。

---

## 手动修复已打开的 PR

如果 workflow 自动创建的 PR 被打上 `Manifest-Validation-Error` 标签（如 [PR #402565](https://github.com/microsoft/winget-pkgs/pull/402565)），可以手动修复并更新，无需等待下一次发版。

### 问题原因
旧版 workflow 生成的 singleton（单文件）manifest 已被 winget-pkgs 废弃。PR 分支中只有 `Orwell-coder.goshare.yaml` 一个文件，需要替换为 3 个多文件格式。

### 修复步骤（命令行）

```bash
# 1. 浅克隆你的 winget-pkgs fork 的对应分支（避免下载整个巨型仓库）
#    把 Orwell-coder 换成你的 GitHub 用户名，版本号对应实际 PR
git clone --depth 1 --branch goshare-0.3.1 https://github.com/Orwell-coder/winget-pkgs.git
cd winget-pkgs

# 2. 删除旧的 singleton 文件
git rm manifests/o/Orwell-coder/goshare/0.3.1/Orwell-coder.goshare.yaml

# 3. 复制 3 个新的 manifest 文件（路径根据你本地 goshare 仓库位置调整）
#    正确的 manifest 文件已由当前版 workflow 预生成在 winget_out_v0.3.1/ 下
cp /c/Users/attem/D/presonal/goshare/winget_out_v0.3.1/*.yaml \
   manifests/o/Orwell-coder/goshare/0.3.1/

# 4. 本地验证（可选但推荐）
winget validate manifests\o\Orwell-coder\goshare\0.3.1\
# 预期输出: Manifest validation succeeded.

# 5. Commit and push
git add manifests/o/Orwell-coder/goshare/0.3.1/
git commit -m "Orwell-coder.goshare version 0.3.1

This updates the manifest from deprecated singleton format to the required
multi-file format (version / installer / defaultLocale). ManifestVersion
bumped from 1.6.0 to 1.12.0 with yaml-language-server schema comments."

git push origin goshare-0.3.1
```

### 修复后观察

推送到 fork 后，PR 会**自动更新**，winget-bot 重新运行验证流水线。在 PR 页面观察标签变化：

- ✅ `Manifest-Validation-Error` 消失
- ✅ 出现 `Azure-Pipeline-Passed`

等待微软审核人员合并即可。

### 也可以在 GitHub Web 界面操作

1. 打开你的 fork: `https://github.com/Orwell-coder/winget-pkgs/tree/goshare-0.3.1`
2. 导航到 `manifests/o/Orwell-coder/goshare/0.3.1/`
3. 删除旧的 singleton 文件
4. 点击 **Add file** → **Create new file**，依次创建 3 个新文件

---

## 本地验证

在推送 tag 前，可以本地验证 manifest 格式：

```powershell
# winget 内置验证命令（不需要额外安装），指向包含 3 个 manifest 文件的目录
winget validate .\winget_out\
```

> 注意：必须指向**目录**，不能指向单个 `.yaml` 文件；且目录内只能有 3 个 manifest 文件，不能有其他文件（否则 winget validate 会尝试解析所有文件并报错）。

---

## 相关链接

- [winget-pkgs 仓库](https://github.com/microsoft/winget-pkgs)
- [WinGet 清单规范](https://github.com/microsoft/winget-cli/blob/master/doc/ManifestSpecv1.6.md)
- [winget-create 工具](https://github.com/microsoft/winget-create)
- [winget-pkgs 贡献指南](https://github.com/microsoft/winget-pkgs/blob/master/CONTRIBUTING.md)
- [验证指南（ValidationFailureGuide）](https://github.com/microsoft/winget-pkgs/blob/master/doc/ValidationFailureGuide.md)
