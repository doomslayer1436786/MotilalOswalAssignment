# PowerShell script to initialize MS SQL database schema
# Run this after docker-compose up --build

Write-Host "Initializing MS SQL database schema..." -ForegroundColor Green

# Wait for MS SQL to be ready
Write-Host "Waiting for MS SQL to be ready..." -ForegroundColor Yellow
Start-Sleep -Seconds 30

# Copy schema file to container
Write-Host "Copying schema file to container..." -ForegroundColor Yellow
docker cp sql/schema.sql mssql:/tmp/schema.sql

# Execute schema
Write-Host "Executing database schema..." -ForegroundColor Yellow
docker exec -it mssql /opt/mssql-tools18/bin/sqlcmd -S localhost -U sa -P "Your_strong_pwd1" -C -i /tmp/schema.sql

Write-Host "Database schema initialized successfully!" -ForegroundColor Green
Write-Host "You can now test the pipeline using the sample_curls.http file" -ForegroundColor Cyan
