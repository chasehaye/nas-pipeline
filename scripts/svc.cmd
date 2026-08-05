@echo off
setlocal
rem Start or stop a single nas-pipeline service in its own window.
rem Usage: svc.cmd {start|stop} {bridge|processor|filter}
rem
rem This is the single place the three services are defined; run-all.cmd and
rem stop-all.cmd just call it for each service.
cd /d "%~dp0.."

set "ACTION=%~1"
set "SVC=%~2"

if /I "%SVC%"=="bridge"    ( set "TITLE=nas-bridge"    & set "DIR=bridge"    & set "RUN=mvnw.cmd spring-boot:run" )
if /I "%SVC%"=="processor" ( set "TITLE=nas-processor" & set "DIR=processor" & set "RUN=go run ./cmd/processor" )
if /I "%SVC%"=="filter"    ( set "TITLE=nas-filter"    & set "DIR=filter"    & set "RUN=go run ./cmd/filter" )

if not defined TITLE goto :badsvc
if /I "%ACTION%"=="start" goto :start
if /I "%ACTION%"=="stop"  goto :stop
echo Unknown action "%ACTION%". Use start or stop.
exit /b 1

:start
rem /k keeps the window open so a startup error stays readable.
start "%TITLE%" cmd /k "cd %DIR% && %RUN%"
exit /b 0

:stop
rem /T kills the whole tree (cmd -> mvnw -> java, cmd -> go -> binary); output
rem suppressed so a service that isn't running is not reported as an error.
taskkill /FI "WINDOWTITLE eq %TITLE%*" /T /F >nul 2>&1
echo Stopped %TITLE%.
exit /b 0

:badsvc
echo Unknown service "%SVC%". Use bridge, processor, or filter.
exit /b 1
