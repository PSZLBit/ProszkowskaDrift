<#
.SYNOPSIS
    Runs the current Proszkowska Drift slice with one command.

.DESCRIPTION
    Starts the Go server, four simulated controllers, and the Godot game in the
    correct order. The Godot window opens fullscreen and auto-starts the round.
    Everything is stopped when the game window closes, when Q is pressed, or
    when -DurationSeconds expires.

    This is a development demo, not the festival launcher. The current slice
    shows four slot-keyed cars and the server input path. Parking tasks,
    scoring, HUD, and the Android controller are not on main yet.

.PARAMETER GodotPath
    Path to the Godot 4 executable. Use this when Godot is not on PATH.

.PARAMETER DurationSeconds
    Stop the demo after this many seconds. 0 (the default) runs until the game
    window closes.

.EXAMPLE
    .\scripts\demo.ps1

.EXAMPLE
    .\scripts\demo.ps1 -GodotPath 'C:\Tools\Godot\Godot_v4.7.2-stable_win64.exe'

.EXAMPLE
    .\scripts\demo.ps1 -DurationSeconds 30

.NOTES
    Owner: team. Local development demo only.
#>
[CmdletBinding()]
param(
    [string]$GodotPath,
    [ValidateRange(0, 86400)]
    [int]$DurationSeconds = 0
)

$ErrorActionPreference = 'Stop'

$repoRoot = Split-Path -Parent $PSScriptRoot
$gameDir = Join-Path $repoRoot 'game'
$serverDir = Join-Path $repoRoot 'server'
$port = 8080
$gameToken = 'dev-token'

$tempRoot = if ($env:TEMP) { $env:TEMP } else { [IO.Path]::GetTempPath() }
$runDir = Join-Path $tempRoot ("proszkowska-drift-demo-" + (Get-Date -Format 'yyyyMMdd-HHmmss'))
$binDir = Join-Path $runDir 'bin'
$logDir = Join-Path $runDir 'logs'

$serverOut = Join-Path $logDir 'server.stdout.log'
$serverErr = Join-Path $logDir 'server.stderr.log'
$controllerOut = Join-Path $logDir 'controllers.stdout.log'
$controllerErr = Join-Path $logDir 'controllers.stderr.log'
$gameOut = Join-Path $logDir 'game.stdout.log'
$gameErr = Join-Path $logDir 'game.stderr.log'
$importOut = Join-Path $logDir 'godot-import.stdout.log'
$importErr = Join-Path $logDir 'godot-import.stderr.log'

$serverProcess = $null
$controllerProcess = $null
$gameProcess = $null
$exitCode = 0

function Write-Step {
    param([string]$Message)
    Write-Host "[demo] $Message" -ForegroundColor Cyan
}

function Write-Ok {
    param([string]$Message)
    Write-Host "[ ok ] $Message" -ForegroundColor Green
}

function Write-Warn {
    param([string]$Message)
    Write-Host "[warn] $Message" -ForegroundColor Yellow
}

function Get-Health {
    try {
        return Invoke-RestMethod -Uri "http://127.0.0.1:$port/healthz" -TimeoutSec 2
    } catch {
        return $null
    }
}

function Wait-ForHealth {
    param([int]$TimeoutSeconds)

    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    while ((Get-Date) -lt $deadline) {
        $health = Get-Health
        if ($null -ne $health) {
            return $health
        }
        Start-Sleep -Milliseconds 250
    }
    return $null
}

function Wait-ForControllers {
    param([int]$Expected, [int]$TimeoutSeconds)

    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    $health = $null
    while ((Get-Date) -lt $deadline) {
        $health = Get-Health
        if ($null -ne $health -and [int]$health.controllers -ge $Expected) {
            return $health
        }
        Start-Sleep -Milliseconds 250
    }
    return $health
}

function Wait-ForGameReady {
    param([int]$TimeoutSeconds)

    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    $health = $null
    while ((Get-Date) -lt $deadline) {
        $health = Get-Health
        if ($null -ne $health -and [bool]$health.gameReady) {
            return $health
        }
        Start-Sleep -Milliseconds 250
    }
    return $health
}

function Wait-ForPhase {
    param([string]$Phase, [int]$TimeoutSeconds)

    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    $health = $null
    while ((Get-Date) -lt $deadline) {
        $health = Get-Health
        if ($null -ne $health -and [string]$health.phase -eq $Phase) {
            return $health
        }
        Start-Sleep -Milliseconds 250
    }
    return $health
}

function Show-LogTail {
    param(
        [string]$Path,
        [string]$Label,
        [int]$Lines = 20
    )

    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) {
        return
    }

    $content = @(Get-Content -LiteralPath $Path -Tail $Lines -ErrorAction SilentlyContinue)
    if ($content.Count -eq 0) {
        return
    }

    Write-Host "--- $Label ---" -ForegroundColor DarkYellow
    foreach ($line in $content) {
        Write-Host "  $line" -ForegroundColor DarkGray
    }
}

function Stop-DemoProcess {
    param(
        $Process,
        [string]$Name
    )

    if ($null -eq $Process) {
        return
    }

    try {
        if (-not $Process.HasExited) {
            Write-Step "Stopping $Name (PID $($Process.Id))"
            Stop-Process -Id $Process.Id -Force -ErrorAction SilentlyContinue
        }
    } catch {
        # Cleanup must not hide the original demo error.
    }
}

function Resolve-GodotExecutable {
    param([string]$ExplicitPath)

    if (-not [string]::IsNullOrWhiteSpace($ExplicitPath)) {
        if (-not (Test-Path -LiteralPath $ExplicitPath -PathType Leaf)) {
            throw "Godot executable not found at '$ExplicitPath'."
        }
        return (Resolve-Path -LiteralPath $ExplicitPath).Path
    }

    $candidates = New-Object System.Collections.Generic.List[string]

    foreach ($envName in @('GODOT_BIN', 'GODOT4', 'GODOT')) {
        $value = [Environment]::GetEnvironmentVariable($envName)
        if (-not [string]::IsNullOrWhiteSpace($value)) {
            $candidates.Add($value)
        }
    }

    foreach ($commandName in @('godot', 'godot4', 'godot.exe', 'godot4.exe')) {
        $command = Get-Command $commandName -ErrorAction SilentlyContinue
        if ($null -ne $command -and $command.CommandType -eq 'Application' -and $command.Source) {
            $candidates.Add($command.Source)
        }
    }

    $searchPatterns = @()
    if ($env:LOCALAPPDATA) {
        $searchPatterns += (Join-Path $env:LOCALAPPDATA 'Programs\Godot\Godot_v*_win64.exe')
        $searchPatterns += (Join-Path $env:LOCALAPPDATA 'Godot\Godot_v*_win64.exe')
    }
    if ($env:ProgramFiles) {
        $searchPatterns += (Join-Path $env:ProgramFiles 'Godot\Godot_v*_win64.exe')
    }
    if (${env:ProgramFiles(x86)}) {
        $searchPatterns += (Join-Path ${env:ProgramFiles(x86)} 'Godot\Godot_v*_win64.exe')
    }
    if ($env:TEMP) {
        $searchPatterns += (Join-Path $env:TEMP 'opencode\godot\Godot_v*_win64.exe')
        $searchPatterns += (Join-Path $env:TEMP 'Godot_v*_win64.exe')
    }
    if ($env:USERPROFILE) {
        $searchPatterns += (Join-Path $env:USERPROFILE 'Downloads\Godot_v*_win64.exe')
        $searchPatterns += (Join-Path $env:USERPROFILE 'scoop\apps\godot\current\godot.exe')
    }
    $searchPatterns += (Join-Path $repoRoot '.tools\Godot_v*_win64.exe')

    foreach ($pattern in $searchPatterns) {
        $items = @(Get-ChildItem -Path $pattern -File -ErrorAction SilentlyContinue |
            Sort-Object LastWriteTime -Descending)
        foreach ($item in $items) {
            $candidates.Add($item.FullName)
        }
    }

    foreach ($candidate in $candidates) {
        if (Test-Path -LiteralPath $candidate -PathType Leaf) {
            return (Resolve-Path -LiteralPath $candidate).Path
        }
    }

    throw "Godot 4 was not found. Install Godot 4.7.2 and add it to PATH, or pass -GodotPath 'C:\path\to\Godot_v4.7.2-stable_win64.exe'."
}

function Get-GodotConsolePath {
    param([string]$GodotExe)

    $directory = [IO.Path]::GetDirectoryName($GodotExe)
    $baseName = [IO.Path]::GetFileNameWithoutExtension($GodotExe)
    $consolePath = Join-Path $directory ($baseName + '_console.exe')
    if (Test-Path -LiteralPath $consolePath -PathType Leaf) {
        return $consolePath
    }
    return $GodotExe
}

function Test-PortInUse {
    param([int]$Port)

    try {
        $listeners = @(Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction Stop)
        if ($listeners.Count -eq 0) {
            return $null
        }
        return ($listeners | Select-Object -ExpandProperty OwningProcess -Unique) -join ', '
    } catch {
        return $null
    }
}

try {
    Write-Host ''
    Write-Host '  Proszkowska Drift - current main-branch demo' -ForegroundColor White
    Write-Host '  Go server + four simulated controllers + Godot display' -ForegroundColor Gray
    Write-Host ''
    Write-Host '  Visible now: session slots, validated input forwarding, four moving cars.' -ForegroundColor Gray
    Write-Host '  Not on main yet: parking tasks, scoring, HUD, Android controller.' -ForegroundColor DarkYellow
    Write-Host ''

    Write-Step 'Checking prerequisites'
    $goCommand = Get-Command go -ErrorAction SilentlyContinue
    if ($null -eq $goCommand) {
        throw 'Go was not found on PATH. Install Go 1.23 or newer.'
    }

    if (-not (Test-Path -LiteralPath (Join-Path $serverDir 'go.mod') -PathType Leaf)) {
        throw "Server module not found at '$serverDir'. Run this script from the repository scripts folder."
    }
    if (-not (Test-Path -LiteralPath (Join-Path $gameDir 'project.godot') -PathType Leaf)) {
        throw "Godot project not found at '$gameDir'. Run this script from the repository scripts folder."
    }

    $godotExe = Resolve-GodotExecutable -ExplicitPath $GodotPath
    $godotConsole = Get-GodotConsolePath -GodotExe $godotExe
    Write-Ok "Godot: $godotExe"

    $portOwner = Test-PortInUse -Port $port
    if ($null -ne $portOwner) {
        throw "Port $port is already in use (PID $portOwner). Stop that process and run the demo again."
    }

    New-Item -ItemType Directory -Path $runDir -Force | Out-Null
    New-Item -ItemType Directory -Path $binDir -Force | Out-Null
    New-Item -ItemType Directory -Path $logDir -Force | Out-Null

    Write-Step 'Building the Go server and fake controller'
    $serverExe = Join-Path $binDir 'server.exe'
    $controllerExe = Join-Path $binDir 'fakecontroller.exe'
    Push-Location -LiteralPath $serverDir
    try {
        & go build -o $serverExe ./cmd/server
        if ($LASTEXITCODE -ne 0) {
            throw 'go build ./cmd/server failed.'
        }
        & go build -o $controllerExe ./cmd/fakecontroller
        if ($LASTEXITCODE -ne 0) {
            throw 'go build ./cmd/fakecontroller failed.'
        }
    } finally {
        Pop-Location
    }
    Write-Ok 'Go tools built.'

    Write-Step 'Importing the Godot project (headless)'
    $importArgs = '--headless --path "{0}" --import' -f $gameDir
    $importProcess = Start-Process -FilePath $godotConsole -ArgumentList $importArgs `
        -WorkingDirectory $repoRoot -NoNewWindow -Wait -PassThru `
        -RedirectStandardOutput $importOut -RedirectStandardError $importErr
    if ($importProcess.ExitCode -ne 0) {
        Show-LogTail -Path $importOut -Label 'Godot import output'
        Show-LogTail -Path $importErr -Label 'Godot import errors'
        throw "Godot import failed with exit code $($importProcess.ExitCode)."
    }
    Write-Ok 'Godot project imported.'

    Write-Step "Starting the Go server on port $port"
    $serverArgs = @('-addr', ":$port", '-game-token', $gameToken, '-log-level', 'info')
    $serverProcess = Start-Process -FilePath $serverExe -ArgumentList $serverArgs `
        -WorkingDirectory $serverDir -NoNewWindow -PassThru `
        -RedirectStandardOutput $serverOut -RedirectStandardError $serverErr
    $health = Wait-ForHealth -TimeoutSeconds 15
    if ($null -eq $health) {
        Show-LogTail -Path $serverErr -Label 'Server errors'
        Show-LogTail -Path $serverOut -Label 'Server output'
        throw "The server did not become healthy at http://127.0.0.1:$port/healthz."
    }
    Write-Ok "Server healthy: phase=$($health.phase), controllers=$($health.controllers), gameReady=$($health.gameReady)"

    Write-Step 'Connecting four simulated controllers'
    $controllerArgs = @('-clients', '4', '-duration', '24h', '-token-prefix', 'demo')
    $controllerProcess = Start-Process -FilePath $controllerExe -ArgumentList $controllerArgs `
        -WorkingDirectory $serverDir -NoNewWindow -PassThru `
        -RedirectStandardOutput $controllerOut -RedirectStandardError $controllerErr
    $health = Wait-ForControllers -Expected 4 -TimeoutSeconds 15
    if ($null -eq $health -or [int]$health.controllers -lt 4) {
        if ($null -ne $health) {
            Write-Warn "Only $($health.controllers) controllers connected."
        }
        Show-LogTail -Path $controllerOut -Label 'Controller output'
        Show-LogTail -Path $controllerErr -Label 'Controller errors'
        throw 'Four simulated controllers did not connect.'
    }
    Write-Ok 'Four simulated controllers connected (slots P1-P4).'

    Write-Step 'Starting the Godot game'
    $env:PD_GAME_TOKEN = $gameToken
    $gameArgs = '--path "{0}" -- --dev-auto-start' -f $gameDir
    $gameProcess = Start-Process -FilePath $godotConsole -ArgumentList $gameArgs `
        -WorkingDirectory $repoRoot -NoNewWindow -PassThru `
        -RedirectStandardOutput $gameOut -RedirectStandardError $gameErr

    $health = Wait-ForGameReady -TimeoutSeconds 20
    if ($null -eq $health -or -not [bool]$health.gameReady) {
        if ($gameProcess.HasExited) {
            Write-Warn "Godot exited with code $($gameProcess.ExitCode)."
        }
        Show-LogTail -Path $gameOut -Label 'Game output'
        Show-LogTail -Path $gameErr -Label 'Game errors'
        throw 'The Godot game did not connect to the server.'
    }
    Write-Ok 'Game connected and sent game_ready.'

    $health = Wait-ForPhase -Phase 'playing' -TimeoutSeconds 20
    if ($null -eq $health -or [string]$health.phase -ne 'playing') {
        Show-LogTail -Path $gameOut -Label 'Game output'
        Show-LogTail -Path $gameErr -Label 'Game errors'
        throw 'The round did not start. Press Enter in the game window to retry, or check the logs.'
    }
    Write-Ok 'Round started.'

    Write-Host ''
    Write-Host 'Demo is running. Keep this window open.' -ForegroundColor Green
    Write-Host '  * The Go server owns the session, slots, and input validation.'
    Write-Host '  * Four simulated controllers are driving the four cars.'
    Write-Host '  * The Godot window shows the current placeholder map and cars.'
    Write-Host ''
    Write-Host 'Game window keys: Enter start/resume, P pause/resume, R reset, Q quit.'
    Write-Host 'Press Q in the game window or close it to stop everything.'
    if ($DurationSeconds -gt 0) {
        Write-Host "This run stops automatically after $DurationSeconds seconds."
    }
    Write-Host "Logs: $logDir"
    Write-Host ''

    $startedAt = Get-Date
    $lastStatusAt = $startedAt

    while ($true) {
        if ($gameProcess.HasExited) {
            break
        }

        if ($DurationSeconds -gt 0 -and ((Get-Date) - $startedAt).TotalSeconds -ge $DurationSeconds) {
            Write-Step "Duration reached ($DurationSeconds s); stopping the demo."
            break
        }

        if ($controllerProcess.HasExited) {
            Show-LogTail -Path $controllerOut -Label 'Controller output'
            Show-LogTail -Path $controllerErr -Label 'Controller errors'
            throw 'The simulated controllers exited unexpectedly.'
        }

        if ($serverProcess.HasExited) {
            Show-LogTail -Path $serverErr -Label 'Server errors'
            throw 'The server exited unexpectedly.'
        }

        if (((Get-Date) - $lastStatusAt).TotalSeconds -ge 10) {
            $health = Get-Health
            if ($null -ne $health) {
                Write-Host ("  [status] phase={0} controllers={1} gameReady={2}" -f `
                    $health.phase, $health.controllers, $health.gameReady) -ForegroundColor DarkGray
            }
            $lastStatusAt = Get-Date
        }

        Start-Sleep -Milliseconds 500
    }

    if ($gameProcess.HasExited -and $gameProcess.ExitCode -ne 0) {
        Show-LogTail -Path $gameOut -Label 'Game output'
        Show-LogTail -Path $gameErr -Label 'Game errors'
        throw "The Godot game exited with code $($gameProcess.ExitCode)."
    }

    Write-Ok 'Godot closed cleanly.'
} catch {
    $exitCode = 1
    Write-Host ''
    Write-Host "Demo failed: $($_.Exception.Message)" -ForegroundColor Red

    if ($null -ne $serverProcess) {
        Show-LogTail -Path $serverErr -Label 'Server errors'
    }
    if ($null -ne $controllerProcess) {
        Show-LogTail -Path $controllerOut -Label 'Controller output'
    }
    if ($null -ne $gameProcess) {
        Show-LogTail -Path $gameOut -Label 'Game output'
        Show-LogTail -Path $gameErr -Label 'Game errors'
    }
} finally {
    Stop-DemoProcess -Process $gameProcess -Name 'Godot'
    Stop-DemoProcess -Process $controllerProcess -Name 'simulated controllers'
    Stop-DemoProcess -Process $serverProcess -Name 'server'
    Write-Host "Logs kept in: $logDir" -ForegroundColor DarkGray
}

exit $exitCode
