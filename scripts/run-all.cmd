@echo off
rem Start every nas-pipeline service, each in its own window, one after another
rem with a 1s gap so startup logs don't race and each has a moment to settle.
rem Infra should be up already (the Makefile's `up` target ensures it). Service
rem definitions live in svc.cmd, so this just calls it for each.
cd /d "%~dp0.."
call scripts\svc.cmd start bridge
timeout /t 1 /nobreak >nul
call scripts\svc.cmd start processor
timeout /t 1 /nobreak >nul
call scripts\svc.cmd start filter
