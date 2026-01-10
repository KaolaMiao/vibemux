@echo off
chcp 65001 >nul 2>&1

REM 配置 Go 环境 (使用上级目录 SDK，避免污染项目)
set "PROJECT_ROOT=%~dp0"
REM 回退一级目录查找 go_sdk
set "GOROOT=%PROJECT_ROOT%..\go_sdk\go"
set "GOPATH=%PROJECT_ROOT%.go_cache"
set "GOCACHE=%GOPATH%\cache"
set "GOPROXY=https://goproxy.cn,direct"
set "GOTOOLCHAIN=local"

REM 添加 PATH
set "PATH=%GOROOT%\bin;%PATH%"

echo ==========================================
echo        VibeMux 自动编译脚本 (Clean)
echo ==========================================
echo.
echo [环境配置]
echo GOROOT: %GOROOT%
echo GOPATH: %GOPATH%
echo.

REM 检查 Go 是否可用
if not exist "%GOROOT%\bin\go.exe" (
    echo [错误] 找不到 Go SDK。
    echo 期望路径: %GOROOT%\bin\go.exe
    pause
    exit /b 1
)

echo [当前版本]
"%GOROOT%\bin\go.exe" version

echo.
echo [1/2] 整理依赖 (go mod tidy)...
"%GOROOT%\bin\go.exe" mod tidy
if %ERRORLEVEL% NEQ 0 (
    echo [错误] 依赖下载失败
    pause
    exit /b 1
)

echo.
echo [2/2] 编译项目 (go build)...
"%GOROOT%\bin\go.exe" build -o bin\vibemux.exe .

if %ERRORLEVEL% EQU 0 (
    echo.
    echo ═══════════════════════════════════════
    echo ✓ 编译成功！
    echo   文件路径: bin\vibemux.exe
    echo ═══════════════════════════════════════
    echo.
) else (
    echo.
    echo ✗ 编译失败，错误代码: %ERRORLEVEL%
)

echo 按任意键退出...
pause
