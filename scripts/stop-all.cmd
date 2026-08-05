@echo off
rem Stop every service window started by services-up, one after another with a
rem 1s gap. Infra is left running -- use `make infra-down` (or `make down`) to
rem stop Kafka/Redis/Postgres too.
cd /d "%~dp0.."
call scripts\svc.cmd stop bridge
timeout /t 1 /nobreak >nul
call scripts\svc.cmd stop processor
timeout /t 1 /nobreak >nul
call scripts\svc.cmd stop filter
