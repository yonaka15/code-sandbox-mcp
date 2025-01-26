# Check if Docker is installed
if (-not (Get-Command "docker" -ErrorAction SilentlyContinue)) {
    Write-Host "Error: Docker is not installed" -ForegroundColor Red
    Write-Host "Please install Docker Desktop for Windows:" -ForegroundColor Yellow
    Write-Host "  https://docs.docker.com/desktop/install/windows-install/"
    exit 1
}

# Check if Docker daemon is running
try {
    docker info | Out-Null
} catch {
    Write-Host "Error: Docker daemon is not running" -ForegroundColor Red
    Write-Host "Please start Docker Desktop and try again" -ForegroundColor Yellow
    exit 1
}

Write-Host "Downloading latest release..." -ForegroundColor Green

# Determine architecture
$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }

# Get the latest release URL
$apiResponse = Invoke-RestMethod -Uri "https://api.github.com/repos/Automata-Labs-team/docker-sandbox-mcp/releases/latest"
$asset = $apiResponse.assets | Where-Object { $_.name -like "docker-sandbox-mcp-windows-$arch.exe" }

if (-not $asset) {
    Write-Host "Error: Could not find release for windows-$arch" -ForegroundColor Red
    exit 1
}

# Create installation directory
$installDir = "$env:LOCALAPPDATA\docker-sandbox-mcp"
New-Item -ItemType Directory -Force -Path $installDir | Out-Null

# Download and install the binary
Write-Host "Installing to $installDir\docker-sandbox-mcp.exe..." -ForegroundColor Green
Invoke-WebRequest -Uri $asset.browser_download_url -OutFile "$installDir\docker-sandbox-mcp.exe"

# Add to Claude Desktop config
Write-Host "Adding to Claude Desktop configuration..." -ForegroundColor Green
& "$installDir\docker-sandbox-mcp.exe" --install

Write-Host "Installation complete!" -ForegroundColor Green
Write-Host "You can now use docker-sandbox-mcp with Claude Desktop or other AI applications." 