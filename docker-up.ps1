# Run full stack: db, rabbitmq, nats, app, nginx
# Requires Docker Desktop

Write-Host "Starting Docker Compose (db, rabbitmq, nats, app, nginx)..." -ForegroundColor Cyan
docker compose -f docker-compose.yml up -d --build

if ($LASTEXITCODE -ne 0) {
    Write-Host "Docker Compose failed." -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "Services started. Waiting for app to be ready..." -ForegroundColor Cyan
Start-Sleep -Seconds 5

Write-Host ""
Write-Host "API:  http://localhost:8080" -ForegroundColor Green
Write-Host "Swagger: http://localhost:8080/swagger/index.html" -ForegroundColor Green
Write-Host "Health: http://localhost:8080/healthz" -ForegroundColor Green
Write-Host ""
Write-Host "To view logs: docker compose logs -f app" -ForegroundColor Yellow
Write-Host "To stop: docker compose down" -ForegroundColor Yellow
