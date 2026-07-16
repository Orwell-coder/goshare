# Winget 发布说明

## 自动生成

每次推送 `v*` 标签时，Release 工作流会自动：

1. 编译 + UPX 压缩 `goshare.exe`
2. 生成 3 个 winget manifest 文件（多文件格式）：
   - `Orwell-coder.goshare.yaml`（version）
   - `Orwell-coder.goshare.installer.yaml`（installer）
   - `Orwell-coder.goshare.locale.en-US.yaml`（defaultLocale）
3. 随 GitHub Release 一起发布

> winget-pkgs 已废弃 singleton（单文件）格式，必须使用上述多文件格式，否则会触发
> `Manifest-Validation-Error`。详见 [docs/winget.md](../docs/winget.md)。

## 提交到 winget-pkgs

### 方式一：手动提交 PR

1. Fork [microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs)
2. 下载 Release 中的 3 个 manifest 文件
3. 放到 `manifests/o/Orwell-coder/goshare/<version>/` 目录
4. 提交 PR

### 方式二：使用 wingetcreate 工具

```powershell
wingetcreate install

# 用 Release 中生成的 3 个 manifest 文件提交（指向目录）
wingetcreate submit winget_out/
```

### 方式三：用户手动添加源

用户在未收录前可直接安装：

```powershell
# 从 GitHub Release 下载
Invoke-WebRequest -Uri "https://github.com/Orwell-coder/goshare/releases/latest/download/goshare.exe" -OutFile goshare.exe
```
