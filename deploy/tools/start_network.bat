@echo off
:: 切换到脚本所在目录
cd /d "%~dp0"
:: 设置代码页为UTF-8
chcp 65001

.\NetworkConfig.exe
