# HR Multi-Agent System Demo

$ErrorActionPreference = "Stop"

$baseUrl = "http://localhost:8000"

Write-Host "HR Multi-Agent System Demo" -ForegroundColor Cyan
Write-Host "=========================" -ForegroundColor Cyan
Write-Host ""

function Test-Endpoint {
    param($name, $url)
    try {
        $response = Invoke-WebRequest -Uri "$baseUrl$url" -UseBasicParsing -TimeoutSec 5
        if ($response.StatusCode -eq 200) {
            Write-Host "[OK] $name is healthy" -ForegroundColor Green
            return $true
        }
    } catch {
        Write-Host "[FAIL] $name is not responding" -ForegroundColor Red
    }
    return $false
}

Write-Host "Checking system health..." -ForegroundColor Yellow
Write-Host ""

$healthOk = Test-Endpoint "Health" "/health"

if (-not $healthOk) {
    Write-Host "System is not ready. Make sure to run: docker compose up --build" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "Running demo HR pipeline..." -ForegroundColor Yellow
Write-Host ""

$payload = @{
    candidate_name = "Ivan Petrov"
    raw_resume = @"
Name: Ivan Petrov
Email: ivan.petrov@example.com
Phone: +7-999-123-4567

Skills:
- Go programming (5 years)
- Python development (3 years)
- Docker and Kubernetes
- PostgreSQL and Redis
- Microservices architecture
- CI/CD with Jenkins

Experience:
- Senior Backend Developer at TechCorp (2020-2024)
- Software Engineer at StartupXYZ (2017-2020)

Education:
- MSc Computer Science, Moscow State University (2015-2017)
"@
    vacancy_title = "Go Backend Developer"
}

try {
    Write-Host "Submitting candidate: $($payload.candidate_name)" -ForegroundColor Cyan
    $response = Invoke-RestMethod -Method Post -Uri "$baseUrl/pipeline/hr" `
        -ContentType "application/json" `
        -Body ($payload | ConvertTo-Json -Depth 10)
    
    Write-Host ""
    Write-Host "Pipeline Result:" -ForegroundColor Green
    $response | ConvertTo-Json -Depth 10 | Write-Host
    
    Start-Sleep -Seconds 2
    
    Write-Host ""
    Write-Host "Fetching statistics..." -ForegroundColor Yellow
    $stats = Invoke-RestMethod -Uri "$baseUrl/stats"
    Write-Host "Processed: $($stats.processed_tasks)" -ForegroundColor White
    Write-Host "Failed: $($stats.failed_tasks)" -ForegroundColor White
    Write-Host "Success Rate: $($stats.success_rate)%" -ForegroundColor White
    
} catch {
    Write-Host "Error: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host ""
Write-Host "Demo complete!" -ForegroundColor Cyan
