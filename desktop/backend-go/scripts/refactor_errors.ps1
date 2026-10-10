# Error Refactoring Script
# Refactors error handling in Go handlers to use centralized error builder

$handlersPath = "C:\Users\Pichau\Desktop\BusinessOS-main-dev\desktop\backend-go\internal\handlers"
$targetFiles = @(
    "tables.go",
    "nodes.go",
    "clients.go",
    "memory.go",
    "crm.go",
    "contexts.go",
    "workspace_handlers.go",
    "filesystem.go",
    "comment_handlers.go",
    "projects.go"
)

Write-Host "Starting error handling refactoring..." -ForegroundColor Green

foreach ($file in $targetFiles) {
    $filePath = Join-Path $handlersPath $file

    if (-not (Test-Path $filePath)) {
        Write-Host "Skipping $file (not found)" -ForegroundColor Yellow
        continue
    }

    Write-Host "`nProcessing $file..." -ForegroundColor Cyan

    # Read file content
    $content = Get-Content $filePath -Raw

    # Backup original
    $backupPath = "$filePath.backup"
    $content | Set-Content $backupPath -NoNewline
    Write-Host "  - Backup created: $backupPath" -ForegroundColor Gray

    # 1. Add imports if needed
    if ($content -notmatch '"github.com/rhl/businessos-backend/internal/utils"') {
        Write-Host "  - Adding utils import" -ForegroundColor Gray
        $content = $content -replace '(import \([^)]*"github.com/rhl/businessos-backend/internal/services")', "`$1`n`t`"github.com/rhl/businessos-backend/internal/utils`""
    }

    if ($content -notmatch '"log/slog"') {
        Write-Host "  - Adding slog import" -ForegroundColor Gray
        $content = $content -replace '(import \([^)]*"log")', "`$1`n`t`"log/slog`""
    }

    # Count original errors
    $originalErrors = ([regex]::Matches($content, 'gin\.H\{"error":')).Count
    Write-Host "  - Original error responses: $originalErrors" -ForegroundColor Gray

    # 2. Replace "Not authenticated"
    $content = $content -replace '(?s)if user == nil \{\s+c\.JSON\(http\.StatusUnauthorized, gin\.H\{"error": "Not authenticated"\}\)\s+return\s+\}',
        "if user == nil {`n`t`tutils.RespondUnauthorized(c, slog.Default())`n`t`treturn`n`t}"

    # 3. Replace "Invalid X ID" patterns
    $content = $content -replace 'c\.JSON\(http\.StatusBadRequest, gin\.H\{"error": "Invalid ([a-z_]+) ID"\}\)',
        'utils.RespondInvalidID(c, slog.Default(), "$1_id")'

    # 4. Replace "X not found" patterns
    $content = $content -replace 'c\.JSON\(http\.StatusNotFound, gin\.H\{"error": "([A-Z][a-z]+) not found"\}\)',
        'utils.RespondNotFound(c, slog.Default(), "$1")'

    # 5. Replace simple "Failed to X" patterns
    $content = $content -replace 'c\.JSON\(http\.StatusInternalServerError, gin\.H\{"error": "Failed to ([^"]+)"\}\)',
        'utils.RespondInternalError(c, slog.Default(), "$1", nil)'

    # 6. Replace "Failed to X: " + err.Error()
    $content = $content -replace 'c\.JSON\(http\.StatusInternalServerError, gin\.H\{"error": "Failed to ([^"]+): " \+ err\.Error\(\)\}\)',
        'utils.RespondInternalError(c, slog.Default(), "$1", err)'

    # 7. Replace simple err.Error() bad requests
    $content = $content -replace '(?s)(if err := c\.ShouldBindJSON\([^)]+\); err != nil \{\s+)c\.JSON\(http\.StatusBadRequest, gin\.H\{"error": err\.Error\(\)\}\)',
        '$1utils.RespondInvalidRequest(c, slog.Default(), err)'

    # Count new errors
    $newErrors = ([regex]::Matches($content, 'gin\.H\{"error":')).Count
    $reduced = $originalErrors - $newErrors

    Write-Host "  - New error responses: $newErrors" -ForegroundColor Gray
    Write-Host "  - Reduced: $reduced error patterns" -ForegroundColor Green

    # Save modified content
    $content | Set-Content $filePath -NoNewline

    Write-Host "  - File updated successfully" -ForegroundColor Green
}

Write-Host "`n==================================" -ForegroundColor Green
Write-Host "Refactoring complete!" -ForegroundColor Green
Write-Host "==================================`n" -ForegroundColor Green

Write-Host "Next steps:" -ForegroundColor Yellow
Write-Host "1. Run: go build ./..." -ForegroundColor White
Write-Host "2. Run: go test ./..." -ForegroundColor White
Write-Host "3. Review changes: git diff" -ForegroundColor White
Write-Host "4. Remove backups if satisfied: rm *.backup" -ForegroundColor White
