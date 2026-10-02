# Использование: .\make.ps1 run | build | test
param(
    [Parameter(Position = 0)]
    [ValidateSet("run", "build", "test")]
    [string]$Target = "run"
)

switch ($Target) {
    "run"   { go run ./cmd/server }
    "build" { go build -o .\bin\server.exe .\cmd\server }
    "test"  { go test ./... -v }
}
