# Opens one PowerShell window per app service (plus the web dev server), each
# cd'd into its module. Called by `make services` on Windows. Infra
# (Kafka/Redis/Postgres) comes from `make up` first.

$root = Split-Path -Parent $PSScriptRoot

$services = @(
    @{ name = 'bridge';       cmd = '.\mvnw.cmd spring-boot:run' },
    @{ name = 'normalizer';   cmd = 'go run ./cmd/normalizer' },
    @{ name = 'filter';       cmd = 'go run ./cmd/filter' },
    @{ name = 'cache-writer'; cmd = 'go run ./cmd/cache-writer' },
    @{ name = 'database-writer'; cmd = 'go run ./cmd/database-writer' },
    @{ name = 'api';          cmd = 'go run ./cmd/api' },
    @{ name = 'web';          cmd = 'npm run dev' }
)

foreach ($s in $services) {
    $dir = Join-Path $root $s.name
    $title = $s.name
    Start-Process powershell -ArgumentList @(
        '-NoExit',
        '-Command',
        "`$Host.UI.RawUI.WindowTitle = '$title'; Set-Location '$dir'; $($s.cmd)"
    )
}
