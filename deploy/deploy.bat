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

:: 配置热点计划任务自动启动
echo "正在配置移动热点..."
netsh wlan set hostednetwork mode=allow ssid=Qingling key=Qingling@123
if not "%ERRORLEVEL%"=="0" (
    echo "警告：配置移动热点失败，但不会中断安装"
)

echo "正在配置热点计划任务自动启动..."
schtasks /create /tn "启动移动热点" /tr "powershell -WindowStyle Hidden -ExecutionPolicy Bypass -File \"%~dp0tools\hotspot.ps1\" 1" /sc onlogon /ru admin /rl HIGHEST /f
if not "%ERRORLEVEL%"=="0" (
    echo "警告：配置热点计划任务失败，但不会中断安装"
)

:: 关闭系统休眠
echo "正在关闭系统休眠..."
powercfg -h off
if not "%ERRORLEVEL%"=="0" (
    echo "警告：关闭系统休眠失败，但不会中断安装"
)

:: 设置高性能电源计划
echo "正在设置高性能电源计划..."
powercfg -setactive 8c5e7fda-e8bf-4a96-9a85-a6e23a8c635c
if not "%ERRORLEVEL%"=="0" (
    echo "警告：设置高性能电源计划失败，尝试创建新的高性能计划..."
    powercfg -duplicatescheme 8c5e7fda-e8bf-4a96-9a85-a6e23a8c635c
)

:: 禁用显示器自动关闭和屏幕保护
echo "正在禁用显示器自动关闭..."
powercfg -change -monitor-timeout-ac 0
powercfg -change -monitor-timeout-dc 0
reg add "HKCU\Control Panel\Desktop" /v ScreenSaveActive /t REG_SZ /d "0" /f
if not "%ERRORLEVEL%"=="0" (
    echo "警告：禁用屏幕保护失败，但不会中断安装"
)

:: 禁用系统待机
echo "正在禁用系统待机..."
powercfg -change -standby-timeout-ac 0
powercfg -change -standby-timeout-dc 0

:: 配置系统崩溃自动重启
echo "正在配置系统崩溃自动重启..."
reg add "HKLM\SYSTEM\CurrentControlSet\Control\CrashControl" /v AutoReboot /t REG_DWORD /d 1 /f
if not "%ERRORLEVEL%"=="0" (
    echo "警告：配置系统崩溃自动重启失败，但不会中断安装"
)

:: 禁用错误报告
echo "正在禁用错误报告..."
reg add "HKLM\SOFTWARE\Microsoft\Windows\Windows Error Reporting" /v Disabled /t REG_DWORD /d 1 /f
if not "%ERRORLEVEL%"=="0" (
    echo "警告：禁用错误报告失败，但不会中断安装"
)

:: 创建磁盘清理脚本和计划任务
echo "正在创建磁盘清理脚本..."
if not exist "C:\ProgramData\ServerMaintenance" (
    mkdir "C:\ProgramData\ServerMaintenance"
    if not "%ERRORLEVEL%"=="0" (
        echo "警告：创建服务器维护目录失败，但不会中断安装"
        goto :skip_disk_cleanup
    )
)

echo @echo off> "C:\ProgramData\ServerMaintenance\DiskCleanup.bat"
echo cleanmgr /sagerun:1 >> "C:\ProgramData\ServerMaintenance\DiskCleanup.bat"

:: 设置磁盘清理选项
echo "正在设置磁盘清理选项..."
reg add "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer\VolumeCaches\Temporary Files" /v StateFlags0001 /t REG_DWORD /d 2 /f
reg add "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer\VolumeCaches\Recycle Bin" /v StateFlags0001 /t REG_DWORD /d 2 /f
reg add "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer\VolumeCaches\Downloaded Program Files" /v StateFlags0001 /t REG_DWORD /d 2 /f

:: 创建磁盘清理计划任务
echo "正在创建磁盘清理计划任务..."
schtasks /create /tn "DiskCleanup" /tr "C:\ProgramData\ServerMaintenance\DiskCleanup.bat" /sc WEEKLY /d SUN /st 02:00 /ru "SYSTEM" /f
if not "%ERRORLEVEL%"=="0" (
    echo "警告：创建磁盘清理计划任务失败，但不会中断安装"
)

:skip_disk_cleanup

:: 配置系统日志清理
echo "正在配置系统日志清理..."
wevtutil sl Application /ms:20971520 /rt:false
wevtutil sl System /ms:20971520 /rt:false
wevtutil sl Security /ms:20971520 /rt:false

:: 启用远程桌面
echo "正在启用远程桌面..."
reg add "HKLM\SYSTEM\CurrentControlSet\Control\Terminal Server" /v fDenyTSConnections /t REG_DWORD /d 0 /f
if not "%ERRORLEVEL%"=="0" (
    echo "警告：启用远程桌面失败，但不会中断安装"
)

:: 配置防火墙允许远程桌面
netsh advfirewall firewall set rule group="远程桌面" new enable=yes
if not "%ERRORLEVEL%"=="0" (
    echo "警告：配置防火墙规则失败，但不会中断安装"
)

:: 优化网络设置
echo "正在优化网络设置..."
netsh int tcp set global autotuninglevel=normal
netsh int tcp set global congestionprovider=ctcp

:: 禁用不必要的服务
echo "正在禁用不必要的服务..."
sc config "DiagTrack" start= disabled
sc config "dmwappushservice" start= disabled
sc config "WSearch" start= disabled
sc config "wuauserv" start= disabled

:: 配置Windows更新延期计划任务
echo "正在配置Windows更新延期计划任务..."

:: 创建脚本目录
echo "创建脚本目录..."
if not exist "C:\ProgramData\WindowsUpdateDelay" (
    mkdir "C:\ProgramData\WindowsUpdateDelay"
    if not "%ERRORLEVEL%"=="0" (
        echo "警告：创建脚本目录失败，但不会中断安装"
        goto :skip_update_delay
    )
)

:: 创建PowerShell脚本
echo "创建Windows更新延期脚本..."
echo # DelayWindowsUpdate.ps1> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo # 脚本功能：将所有Windows更新推迟30天>> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo # 创建日期：%date%>> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo.>> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo # 设置日志文件路径>> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo $logFile = "$env:ProgramData\WindowsUpdateDelay\delay_update_log.txt">> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo.>> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo # 确保日志目录存在>> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo if (-not (Test-Path "$env:ProgramData\WindowsUpdateDelay")) {>> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo     New-Item -Path "$env:ProgramData\WindowsUpdateDelay" -ItemType Directory -Force ^| Out-Null>> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo }>> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo.>> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo # 记录日志函数>> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo function Write-Log {>> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo     param (>> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo         [string]$Message>> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo     )>> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo     >> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo     $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss">> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo     "$timestamp - $Message" ^| Out-File -FilePath $logFile -Append>> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo     Write-Host $Message>> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo }>> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo.>> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo # 记录脚本开始执行>> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo Write-Log "开始执行Windows更新延期脚本">> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo.>> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo try {>> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo     # 设置质量更新延期天数（安全更新等）>> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo     Write-Log "设置质量更新延期30天">> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo     Set-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\WindowsUpdate\UX\Settings" -Name "DeferQualityUpdatesPeriodInDays" -Value 30 -Type DWord -Force>> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo     >> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo     # 设置功能更新延期天数（Windows版本更新）>> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo     Write-Log "设置功能更新延期30天">> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo     Set-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\WindowsUpdate\UX\Settings" -Name "DeferFeatureUpdatesPeriodInDays" -Value 30 -Type DWord -Force>> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo     >> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo     # 暂停更新>> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo     Write-Log "暂停Windows更新30天">> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo     $pauseDate = (Get-Date).AddDays(30).ToString("yyyy-MM-dd")>> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo     Set-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\WindowsUpdate\UX\Settings" -Name "PauseUpdatesExpiryTime" -Value $pauseDate -Type String -Force>> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo     >> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo     # 禁用自动更新>> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo     Write-Log "禁用自动更新">> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo     if (-not (Test-Path "HKLM:\SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate\AU")) {>> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo         New-Item -Path "HKLM:\SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate\AU" -Force ^| Out-Null>> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo     }>> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo     Set-ItemProperty -Path "HKLM:\SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate\AU" -Name "NoAutoUpdate" -Value 1 -Type DWord -Force>> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo     >> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo     Write-Log "Windows更新已成功延期30天">> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo     exit 0>> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo }>> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo catch {>> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo     Write-Log "错误：延期Windows更新失败 - $_">> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo     exit 1>> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
echo }>> "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"

:: 创建计划任务
echo "创建Windows更新延期计划任务..."
schtasks /create /tn "DelayWindowsUpdate" /tr "powershell.exe -ExecutionPolicy Bypass -File C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1" /sc WEEKLY /mo 2 /d MON /st 03:00 /ru "SYSTEM" /rl HIGHEST /f
if not "%ERRORLEVEL%"=="0" (
    echo "警告：创建Windows更新延期计划任务失败，但不会中断安装"
) else (
    echo "Windows更新延期计划任务创建成功"
)

:: 立即执行一次脚本
echo "立即执行Windows更新延期..."
powershell.exe -ExecutionPolicy Bypass -File "C:\ProgramData\WindowsUpdateDelay\DelayWindowsUpdate.ps1"
if not "%ERRORLEVEL%"=="0" (
    echo "警告：Windows更新延期执行失败，但不会中断安装"
) else (
    echo "Windows更新已成功延期30天"
)

:skip_update_delay

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

:: 配置ollama计划任务启动
echo "正在配置ollama计划任务启动..."
schtasks /create /tn "启动Ollama" /tr "\"C:\ollama\ollama.exe\" serve" /sc onlogon /ru admin /rl HIGHEST /f
if not "%ERRORLEVEL%"=="0" (
    echo "警告：配置ollama计划任务失败，但不会中断安装"
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

:: 配置Docker Desktop计划任务启动
echo "正在配置Docker Desktop计划任务启动..."
schtasks /create /tn "启动Docker Desktop" /tr "\"C:\Program Files\Docker Desktop\Docker Desktop.exe\"" /sc onlogon /ru admin /rl HIGHEST /f
if not "%ERRORLEVEL%"=="0" (
    echo "警告：配置Docker Desktop计划任务失败，但不会中断安装"
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

:: 配置admin登录后30秒自动锁屏
echo "正在配置admin自动锁屏计划任务..."
schtasks /create /tn "Admin自动锁屏" /tr "timeout /t 30 && rundll32.exe user32.dll,LockWorkStation" /sc onlogon /ru admin /rl HIGHEST /f /delay 0000:00:30
if not "%ERRORLEVEL%"=="0" (
    echo "警告：配置自动锁屏计划任务失败，但不会中断安装"
)

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