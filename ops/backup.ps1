param(
    [string]$Name = ("shortli-" + (Get-Date -Format "yyyyMMdd-HHmmss") + ".dump")
)

$ErrorActionPreference = "Stop"
if ([IO.Path]::GetFileName($Name) -ne $Name -or -not $Name.EndsWith(".dump")) {
    throw "Backup name must be a plain .dump filename."
}

$workspace = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$backupDirectory = Join-Path $workspace "backups"
New-Item -ItemType Directory -Force -Path $backupDirectory | Out-Null

Push-Location $workspace
try {
    $containerCommand = 'pg_dump --format=custom --no-owner --no-privileges --username="$POSTGRES_USER" --dbname="$POSTGRES_DB" --file="/backups/' + $Name + '"'
    docker-compose exec -T postgres sh -c $containerCommand
    if ($LASTEXITCODE -ne 0) {
        throw "pg_dump failed with exit code $LASTEXITCODE."
    }
    $backupPath = Join-Path $backupDirectory $Name
    if (-not (Test-Path -LiteralPath $backupPath)) {
        throw "Backup file was not created."
    }
    Write-Output "Backup created: $backupPath"
} finally {
    Pop-Location
}
