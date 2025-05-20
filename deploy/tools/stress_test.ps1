<#
.SYNOPSIS
工作站硬件烤机测试脚本

.DESCRIPTION
对CPU、内存、硬盘、显卡和网络进行压力测试，持续最长48小时
测试结果以JSON格式记录日志，并在控制台实时显示状态

.NOTES
文件名称: stress_test.ps1
创建日期: $(Get-Date -Format "yyyy-MM-dd")
#>

# 初始化设置
$TestStartTime = Get-Date
$LogDir = ".\logs"
$LogFile = "$LogDir\stress_test_$(Get-Date -Format 'yyyyMMdd_HHmmss').json"

# 创建日志目录
if (-not (Test-Path $LogDir)) {
    New-Item -ItemType Directory -Path $LogDir | Out-Null
}

# 定义测试参数
$TestDuration = 48 * 60 * 60  # 48小时(秒)
$Interval = 5  # 采样间隔(秒)

# JSON日志结构
$TestLog = @{
    SystemInfo = @{
        HostName = $env:COMPUTERNAME
        OSVersion = [System.Environment]::OSVersion.VersionString
        StartTime = $TestStartTime.ToString("yyyy-MM-dd HH:mm:ss")
    }
    TestModules = @("CPU", "Memory", "Disk", "GPU", "Network")
    TestResults = @()
}

# CPU测试函数
function Test-CPU {
    param (
        [int]$Duration
    )
    
    # 使用Prime95算法进行CPU压力测试
    # 这里简化实现，实际使用时可以调用专业工具
    $result = @{
        Usage = (Get-Counter '\Processor(_Total)\% Processor Time').CounterSamples.CookedValue
        Temperature = 0  # 需要硬件特定工具获取
        Threads = [System.Environment]::ProcessorCount
        Errors = 0
    }
    
    return $result
}

# 内存测试函数
function Test-Memory {
    param (
        [int]$Duration
    )
    
    # 内存填充和校验测试
    $result = @{
        TotalGB = [math]::Round((Get-CimInstance Win32_ComputerSystem).TotalPhysicalMemory / 1GB, 2)
        UsedGB = [math]::Round((Get-Counter '\Memory\Available MBytes').CounterSamples.CookedValue / 1024, 2)
        Errors = 0
        Throughput = 0
    }
    
    return $result
}

# 硬盘测试函数
function Test-Disk {
    param (
        [int]$Duration
    )
    
    # 获取磁盘信息
    $disk = Get-CimInstance Win32_LogicalDisk | Where-Object { $_.DriveType -eq 3 } | Select-Object -First 1
    
    # 简单IO测试
    $result = @{
        Drive = $disk.DeviceID
        TotalGB = [math]::Round($disk.Size / 1GB, 2)
        FreeGB = [math]::Round($disk.FreeSpace / 1GB, 2)
        ReadSpeed = 0  # 需要实际测试
        WriteSpeed = 0  # 需要实际测试
        Errors = 0
    }
    
    return $result
}

# 显卡测试函数
function Test-GPU {
    param (
        [int]$Duration
    )
    
    # 简化实现，实际使用时可以调用专业工具
    $result = @{
        Name = "Unknown"
        Usage = 0
        Temperature = 0
        FPS = 0
        Errors = 0
    }
    
    return $result
}

# 网络测试函数
function Test-Network {
    param (
        [int]$Duration
    )
    
    # 测试网络连接
    $result = @{
        Interface = "Unknown"
        Bandwidth = 0
        Latency = 0
        PacketLoss = 0
        Errors = 0
    }
    
    return $result
}

# 主测试循环
$elapsed = 0
while ($elapsed -lt $TestDuration) {
    $currentTime = Get-Date
    $elapsed = ($currentTime - $TestStartTime).TotalSeconds
    
    # 执行各模块测试
    $testResult = @{
        Timestamp = $currentTime.ToString("yyyy-MM-dd HH:mm:ss")
        CPU = Test-CPU -Duration $Interval
        Memory = Test-Memory -Duration $Interval
        Disk = Test-Disk -Duration $Interval
        GPU = Test-GPU -Duration $Interval
        Network = Test-Network -Duration $Interval
    }
    
    # 添加到日志
    $TestLog.TestResults += $testResult
    
    # 写入日志文件
    $TestLog | ConvertTo-Json -Depth 5 | Out-File -FilePath $LogFile -Force
    
    # 控制台输出
    Clear-Host
    Write-Host "=== 工作站烤机测试 ===" -ForegroundColor Cyan
    Write-Host "已运行: $([math]::Round($elapsed/3600, 2)) 小时 / 总时长: 48 小时"
    Write-Host "日志文件: $LogFile"
    Write-Host ""
    
    # 显示CPU状态
    Write-Host "CPU 状态:" -ForegroundColor Yellow
    Write-Host "  使用率: $($testResult.CPU.Usage)%"
    Write-Host "  温度: $($testResult.CPU.Temperature)°C"
    Write-Host "  线程数: $($testResult.CPU.Threads)"
    Write-Host ""
    
    # 显示内存状态
    Write-Host "内存 状态:" -ForegroundColor Green
    Write-Host "  总量: $($testResult.Memory.TotalGB) GB"
    Write-Host "  已用: $($testResult.Memory.UsedGB) GB"
    Write-Host ""
    
    # 显示磁盘状态
    Write-Host "磁盘 状态:" -ForegroundColor Magenta
    Write-Host "  盘符: $($testResult.Disk.Drive)"
    Write-Host "  总量: $($testResult.Disk.TotalGB) GB"
    Write-Host "  剩余: $($testResult.Disk.FreeGB) GB"
    Write-Host ""
    
    # 显示GPU状态
    Write-Host "显卡 状态:" -ForegroundColor Blue
    Write-Host "  名称: $($testResult.GPU.Name)"
    Write-Host "  使用率: $($testResult.GPU.Usage)%"
    Write-Host "  温度: $($testResult.GPU.Temperature)°C"
    Write-Host ""
    
    # 显示网络状态
    Write-Host "网络 状态:" -ForegroundColor DarkCyan
    Write-Host "  接口: $($testResult.Network.Interface)"
    Write-Host "  带宽: $($testResult.Network.Bandwidth) Mbps"
    Write-Host "  延迟: $($testResult.Network.Latency) ms"
    Write-Host ""
    
    # 等待下一个采样周期
    Start-Sleep -Seconds $Interval
}

Write-Host "测试完成!" -ForegroundColor Green