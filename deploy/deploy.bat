@echo off
:: 检查管理员权限
NET SESSION >nul 2>&1
IF %ERRORLEVEL% NEQ 0 (
    echo 请以管理员身份运行此脚本
    pause
    exit /b
)

echo 开始自动部署...

:: 配置自动登录
echo 正在配置自动登录...
reg add "HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon" /v DefaultUserName /t REG_SZ /d admin /f
reg add "HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon" /v DefaultPassword /t REG_SZ /d Admin@123 /f
reg add "HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon" /v AutoAdminLogon /t REG_SZ /d 1 /f

:: 配置热点自动启动
echo 正在配置热点自动启动...
reg add "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Run" /v HotSpot /t REG_SZ /d "cmd /c start netsh wlan set hostednetwork mode=allow ssid=Qingling key=Qingling@123 && netsh wlan start hostednetwork" /f

:: 安装软件
echo 正在检查安装文件...
if not exist ".\tools\ollama.exe" (
    echo 错误：未找到ollama安装文件
    goto :ERROR
)
if not exist ".\tools\DockerDesktop.exe" (
    echo 错误：未找到Docker Desktop安装文件
    goto :ERROR
)

:: 安装ollama
echo 正在安装ollama...
start /wait .\tools\ollama.exe /S
if %ERRORLEVEL% NEQ 0 (
    echo 错误：ollama安装失败
    goto :ERROR
)

:: 安装docker desktop
echo 正在安装Docker Desktop...
start /wait .\tools\DockerDesktop.exe install -quiet
if %ERRORLEVEL% NEQ 0 (
    echo 错误：Docker Desktop安装失败
    goto :ERROR
)

:: 创建Qingling用户账号
echo 正在创建Qingling用户账号...
net user Qingling Qingling@123 /add
if %ERRORLEVEL% NEQ 0 (
    echo 错误：创建用户账号失败
    goto :ERROR
)

echo 自动部署完成！
echo 请重启计算机以使所有配置生效。
goto :END

:ERROR
echo 部署过程中出现错误，请检查以上错误信息。
pause
exit /b 1

:END
pause
exit /b 0