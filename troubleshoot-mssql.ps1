# PowerShell script to troubleshoot MS SQL container issues
Write-Host "=== MS SQL Container Troubleshooting ===" -ForegroundColor Green

# Check if container is running
Write-Host "`n1. Checking container status..." -ForegroundColor Yellow
docker ps -a --filter "name=mssql"

# Check container logs
Write-Host "`n2. Checking MS SQL container logs..." -ForegroundColor Yellow
docker logs mssql --tail 50

# Check if MS SQL is accepting connections
Write-Host "`n3. Testing MS SQL connection..." -ForegroundColor Yellow
try {
    docker exec -it mssql /opt/mssql-tools18/bin/sqlcmd -S localhost -U sa -P "Your_strong_pwd1" -Q "SELECT 1" 2>&1
    Write-Host "MS SQL connection successful!" -ForegroundColor Green
} catch {
    Write-Host "MS SQL connection failed. Trying alternative path..." -ForegroundColor Red
    try {
        docker exec -it mssql /opt/mssql-tools/bin/sqlcmd -S localhost -U sa -P "Your_strong_pwd1" -Q "SELECT 1" 2>&1
        Write-Host "MS SQL connection successful with alternative path!" -ForegroundColor Green
    } catch {
        Write-Host "MS SQL connection failed with both paths." -ForegroundColor Red
    }
}

# Check available tools in container
Write-Host "`n4. Checking available SQL tools..." -ForegroundColor Yellow
docker exec -it mssql find /opt -name "sqlcmd" 2>/dev/null

# Check container health
Write-Host "`n5. Checking container health..." -ForegroundColor Yellow
docker inspect mssql --format='{{.State.Health.Status}}' 2>/dev/null

Write-Host "`n=== Troubleshooting Complete ===" -ForegroundColor Green
Write-Host "If MS SQL is still not working, try:" -ForegroundColor Cyan
Write-Host "1. docker compose down -v  (removes volumes)" -ForegroundColor White
Write-Host "2. docker compose up --build mssql  (rebuilds MS SQL)" -ForegroundColor White
Write-Host "3. Wait 2-3 minutes for MS SQL to fully initialize" -ForegroundColor White
