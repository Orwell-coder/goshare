# Winget 发布说明

## 自动生成

每次推送 `v*` 标签时，Release 工作流会自动：

1. 编译 + UPX 压缩 `goshare.exe`
2. 生成 winget manifest (`Orwell-coder.goshare.yaml`)
3. 随 GitHub Release 一起发布

## 提交到 winget-pkgs

### 方式一：手动提交 PR

1. Fork [microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs)
2. 下载 Release 中生成的 `Orwell-coder.goshare.yaml`
3. 放到 `manifests/o/Orwell-coder/goshare/<version>/` 目录
4. 提交 PR

### 方式二：使用 wingetcreate 工具

```powershell
wingetcreate install

# 用 Release 中生成的 manifest 提交
wingetcreate submit winget_out/Orwell-coder.goshare.yaml
```

### 方式三：用户手动添加源

用户在未收录前可直接安装：

```powershell
# 从 GitHub Release 下载
Invoke-WebRequest -Uri "https://github.com/Orwell-coder/goshare/releases/latest/download/goshare.exe" -OutFile goshare.exe
```
