@echo off
set "GOROOT=D:\长风破浪会有时\MyTools\go_sdk\go"
set "GOPATH=%USERPROFILE%\go"
set "PATH=%GOROOT%\bin;%PATH%"

echo Setting environment variables permanently for the current user...
setx GOROOT "%GOROOT%"
setx GOPATH "%GOPATH%"
setx Path "%GOROOT%\bin;%Path%"

echo.
echo Environment variables set! You may need to restart your terminal.
echo GOROOT: %GOROOT%
echo GOPATH: %GOPATH%
echo.
pause
