@echo off
echo Initializing MS SQL database schema...

echo Waiting for MS SQL to be ready...
timeout /t 30 /nobreak > nul

echo Copying schema file to container...
docker cp sql/schema.sql mssql:/tmp/schema.sql

echo Executing database schema...
docker exec -it mssql /opt/mssql-tools18/bin/sqlcmd -S localhost -U sa -P "Your_strong_pwd1" -C -i /tmp/schema.sql

echo Database schema initialized successfully!
echo You can now test the pipeline using the sample_curls.http file
pause
