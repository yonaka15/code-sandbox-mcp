# Function to check if running in a terminal that supports colors
function Test-ColorSupport {
    # Check if we're in a terminal that supports VirtualTerminalLevel
    $supportsVT = $false
    try {
        $supportsVT = [Console]::IsOutputRedirected -eq $false -and 
                      [Console]::IsErrorRedirected -eq $false -and
                      [Environment]::GetEnvironmentVariable("TERM") -ne $null
    } catch {
        $supportsVT = $false
    }
    return $supportsVT
}

# Function to write colored output
function Write-ColoredMessage {
    param(
        [string]$Message,
        [System.ConsoleColor]$Color = [System.ConsoleColor]::White
    )
    
    if (Test-ColorSupport) {
        $originalColor = [Console]::ForegroundColor
        [Console]::ForegroundColor = $Color
        Write-Host $Message
        [Console]::ForegroundColor = $originalColor
    } else {
        Write-Host $Message
    }
}

# Check if Docker is installed
if (-not (Get-Command "docker" -ErrorAction SilentlyContinue)) {
    Write-ColoredMessage "Error: Docker is not installed" -Color Red
    Write-ColoredMessage "Please install Docker Desktop for Windows:" -Color Yellow
    Write-Host "  https://docs.docker.com/desktop/install/windows-install/"
    exit 1
}

# Check if Docker daemon is running
try {
    docker info | Out-Null
} catch {
    Write-ColoredMessage "Error: Docker daemon is not running" -Color Red
    Write-ColoredMessage "Please start Docker Desktop and try again" -Color Yellow
    exit 1
}

Write-ColoredMessage "Downloading latest release..." -Color Green

# Determine architecture
$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }

# Get the latest release URL
$apiResponse = Invoke-RestMethod -Uri "https://api.github.com/repos/Automata-Labs-team/code-sandbox-mcp/releases/latest"
$asset = $apiResponse.assets | Where-Object { $_.name -like "code-sandbox-mcp-windows-$arch.exe" }

if (-not $asset) {
    Write-ColoredMessage "Error: Could not find release for windows-$arch" -Color Red
    exit 1
}

# Create installation directory
$installDir = "$env:LOCALAPPDATA\code-sandbox-mcp"
New-Item -ItemType Directory -Force -Path $installDir | Out-Null

# Download and install the binary
Write-ColoredMessage "Installing to $installDir\code-sandbox-mcp.exe..." -Color Green
Invoke-WebRequest -Uri $asset.browser_download_url -OutFile "$installDir\code-sandbox-mcp.exe"

# Add to Claude Desktop config
Write-ColoredMessage "Adding to Claude Desktop configuration..." -Color Green
& "$installDir\code-sandbox-mcp.exe" --install

Write-ColoredMessage "Installation complete!" -Color Green
Write-Host "You can now use code-sandbox-mcp with Claude Desktop or other AI applications." 