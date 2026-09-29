# test.ps1
# A simple PowerShell script to run tests and linters locally on Windows.

Write-Host "Running go fmt..." -ForegroundColor Cyan
go fmt ./...

Write-Host "Running go vet..." -ForegroundColor Cyan
go vet ./...

Write-Host "Running go test..." -ForegroundColor Cyan
go test ./...

if (Get-Command golangci-lint -ErrorAction SilentlyContinue) {
    Write-Host "Running golangci-lint..." -ForegroundColor Cyan
    golangci-lint run
} else {
    Write-Host "golangci-lint is not installed locally." -ForegroundColor Yellow
    Write-Host "To install run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" -ForegroundColor Yellow
}

Write-Host "Done!" -ForegroundColor Green
