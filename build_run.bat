@echo off
echo Building Linux binary...
set CGO_ENABLED=0
set GOOS=linux
set GOARCH=amd64
go build -o bluebell-linux .

if %errorlevel% neq 0 (
    echo Build failed!
    exit /b %errorlevel%
)

echo Build success! Starting Docker...
docker-compose up -d --build

echo Done! Application is running at http://localhost:8080
