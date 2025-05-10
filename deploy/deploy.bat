@echo off
:: 切换到脚本所在目录
cd /d "%~dp0"
:: 设置代码页为UTF-8
chcp 65001
:: 检查管理员权限
NET SESSION >nul 2>&1
if not "%ERRORLEVEL%"=="0" (
    echo "请以管理员身份运行此脚本"
    pause
    exit /b
)

echo "开始自动部署..."

:: 检查并创建admin账号
echo "正在检查admin账号..."
net user admin >nul 2>nul
if not "%ERRORLEVEL%"=="0" (
    echo "admin账号不存在，正在创建..."
    net user admin Admin@123 /add
    net localgroup administrators admin /add
)

:: 验证admin账号设置
net user admin >nul 2>nul
if not "%ERRORLEVEL%"=="0" goto :admin_error
net localgroup administrators | find "admin" >nul
if not "%ERRORLEVEL%"=="0" goto :admin_error
echo "admin账号配置成功"
goto :admin_done

:admin_error
echo "错误：admin账号配置失败"
goto :ERROR

:admin_done

:: 配置自动登录
echo "正在配置自动登录..."
reg add "HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon" /v DefaultUserName /t REG_SZ /d admin /f
if not "%ERRORLEVEL%"=="0" (
    echo "错误：设置默认用户名失败"
    goto :ERROR
)
reg add "HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon" /v DefaultPassword /t REG_SZ /d Admin@123 /f
if not "%ERRORLEVEL%"=="0" (
    echo "错误：设置默认密码失败"
    goto :ERROR
)
reg add "HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon" /v AutoAdminLogon /t REG_SZ /d 1 /f
if not "%ERRORLEVEL%"=="0" (
    echo "错误：启用自动登录失败"
    goto :ERROR
)

:: 配置热点自动启动
echo "正在配置热点自动启动..."
reg add "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Run" /v HotSpot /t REG_SZ /d "cmd /c start netsh wlan set hostednetwork mode=allow ssid=Qingling key=Qingling@123 && netsh wlan start hostednetwork" /f
if not "%ERRORLEVEL%"=="0" (
    echo "警告：配置热点自动启动失败，但不会中断安装"
)

:: 安装软件
echo "正在检查安装文件..."
if not exist ".\tools\OllamaSetup.exe" (
    echo "错误：未找到ollama安装文件"
    goto :ERROR
)
if not exist ".\tools\Docker Desktop Installer.exe" (
    echo "错误：未找到Docker Desktop安装文件"
    goto :ERROR
)

:: 安装ollama
echo "正在安装ollama..."
start /wait .\tools\OllamaSetup.exe /VERYSILENT /DIR=C:\ollama /NORESTART
if not "%ERRORLEVEL%"=="0" (
    echo "错误：ollama安装失败：%ERRORLEVEL%"
    goto :ERROR
)

:: 设置OLLAMA环境变量
echo "正在设置OLLAMA环境变量..."
setx OLLAMA_KEEP_ALIVE "-1" /M
if not "%ERRORLEVEL%"=="0" (
    echo "警告：设置环境变量失败，但不会中断安装"
)
setx OLLAMA_ORIGINS "*" /M
if not "%ERRORLEVEL%"=="0" (
    echo "警告：设置环境变量失败，但不会中断安装"
)

:: 配置ollama开机自启动
echo "正在配置ollama开机自启动..."
reg add "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Run" /v Ollama /t REG_SZ /d "C:\ollama\ollama.exe serve" /f
if not "%ERRORLEVEL%"=="0" (
    echo "警告：配置ollama开机自启动失败，但不会中断安装"
)

:: 检查Docker Desktop是否已安装
echo "正在检查Docker Desktop..."
if exist "C:\Program Files\Docker Desktop\Docker Desktop.exe" (
    echo "Docker Desktop已安装，跳过安装步骤"
) else (
    echo "Docker Desktop未安装，开始安装..."
    start /wait "" ".\tools\Docker Desktop Installer.exe" install --quiet --accept-license --always-run-service --backend=wsl-2 --installation-dir="C:\Program Files\Docker Desktop" --wsl-default-data-root="C:\ProgramData\DockerDesktop" 
    :: 指定docker额外安装参数 --allowed-org=docker --admin-settings="{'configurationFileVersion': 2, 'enhancedContainerIsolation': {'value': true, 'locked': false}}"
    if not "%ERRORLEVEL%"=="0" (
        echo "错误：Docker Desktop安装失败：%ERRORLEVEL%"
        goto :ERROR
    )
    echo "Docker Desktop安装成功"
    net localgroup docker-users admin /add
)

:: 配置Docker Desktop开机自启动
echo "正在配置Docker Desktop开机自启动..."
reg add "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Run" /v DockerDesktop /t REG_SZ /d "\"C:\Program Files\Docker Desktop\Docker Desktop.exe\"" /f
if not "%ERRORLEVEL%"=="0" (
    echo "警告：配置Docker Desktop开机自启动失败，但不会中断安装"
)

:: 创建Qingling用户账号
echo "正在创建Qingling用户账号..."
net user Qingling >nul 2>nul
if not "%ERRORLEVEL%"=="0" (
    echo "Qingling账号不存在，正在创建..."
    net user Qingling Qingling@123 /add
)

:: 验证Qingling账号设置
net user Qingling >nul 2>nul
if not "%ERRORLEVEL%"=="0" (
    echo "错误：Qingling账号创建失败"
    goto :ERROR
)
echo "Qingling账号配置成功"

echo "自动部署完成！"
echo "请重启计算机以使所有配置生效。"
goto :END

:ERROR
echo "部署过程中出现错误，请检查以上错误信息。"
pause
exit /b 1

:END
pause
exit /b 0