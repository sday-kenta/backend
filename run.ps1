Get-Content .env | ForEach-Object {
  if ($_ -match '^\s*([^#][^=]+)=(.*)$') {
    [Environment]::SetEnvironmentVariable($matches[1].Trim(), $matches[2].Trim(), 'Process')
  }
}

go mod download
$env:CGO_ENABLED = "0"
go run -tags migrate ./cmd/app

#Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser - для запуска скрипта

Get-Content .env | ForEach-Object {
  if ($_ -match '^\s*([^#][^=]+)=(.*)$') {
    [Environment]::SetEnvironmentVariable($matches[1].Trim(), $matches[2].Trim(), 'Process')
  }
}
go mod download
$env:CGO_ENABLED = "0"
go run -tags migrate ./cmd/app

#Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser - для запуска скрипта
