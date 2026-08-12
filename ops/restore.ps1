param(
    [Parameter(Mandatory = $true)]
    [string]$Backup,
    [switch]$ConfirmDatabaseReset
)

$ErrorActionPreference = "Stop"
if (-not $ConfirmDatabaseReset) {
    throw "Restore overwrites application data. Run again with -ConfirmDatabaseReset."
}

$workspace = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$backupDirectory = (Resolve-Path (Join-Path $workspace "backups")).Path
$backupPath = (Resolve-Path -LiteralPath $Backup).Path
if (-not $backupPath.StartsWith($backupDirectory + [IO.Path]::DirectorySeparatorChar)) {
    throw "Backup must be located inside $backupDirectory."
}
$name = [IO.Path]::GetFileName($backupPath)

Push-Location $workspace
try {
    docker-compose stop backend web
    $containerCommand = 'pg_restore --clean --if-exists --no-owner --no-privileges --username="$POSTGRES_USER" --dbname="$POSTGRES_DB" "/backups/' + $name + '"'
    docker-compose exec -T postgres sh -c $containerCommand
    if ($LASTEXITCODE -ne 0) {
        throw "pg_restore failed with exit code $LASTEXITCODE."
    }
    docker-compose up -d backend web
    Write-Output "Database restored from $backupPath"
} finally {
    Pop-Location
}
