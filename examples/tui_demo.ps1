# Auto Claude Code TUI Demo Script
# This script demonstrates how to use the TUI interface for task monitoring

Write-Host "=== Auto Claude Code TUI Demo ===" -ForegroundColor Cyan
Write-Host ""

# Check if program exists
$programPath = ".\bin\auto-claude-code.exe"
if (-not (Test-Path $programPath)) {
    Write-Host "Error: Program file not found: $programPath" -ForegroundColor Red
    Write-Host "Please run: go build -o bin/auto-claude-code.exe cmd/auto-claude-code/main.go" -ForegroundColor Yellow
    exit 1
}

Write-Host "1. Checking program version..." -ForegroundColor Green
& $programPath --version
Write-Host ""

Write-Host "2. Viewing TUI command help..." -ForegroundColor Green
& $programPath task tui --help
Write-Host ""

Write-Host "3. Available TUI startup command examples:" -ForegroundColor Green
Write-Host "   # Basic startup (default 2-second refresh)" -ForegroundColor Gray
Write-Host "   auto-claude-code task tui" -ForegroundColor White
Write-Host ""
Write-Host "   # Custom refresh interval (5 seconds)" -ForegroundColor Gray
Write-Host "   auto-claude-code task tui -i 5" -ForegroundColor White
Write-Host ""
Write-Host "   # Specify server address" -ForegroundColor Gray
Write-Host "   auto-claude-code task tui -s http://192.168.1.100:8080" -ForegroundColor White
Write-Host ""
Write-Host "   # Combined parameters" -ForegroundColor Gray
Write-Host "   auto-claude-code task tui -s http://localhost:8080 -i 3" -ForegroundColor White
Write-Host ""

Write-Host "4. TUI interface shortcuts:" -ForegroundColor Green
Write-Host "   Up/Down  - Select task" -ForegroundColor White
Write-Host "   Enter    - View details" -ForegroundColor White
Write-Host "   c        - Cancel task" -ForegroundColor White
Write-Host "   r        - Manual refresh" -ForegroundColor White
Write-Host "   q/Ctrl+C - Exit" -ForegroundColor White
Write-Host ""

Write-Host "5. Interface layout description:" -ForegroundColor Green
Write-Host "   +-- Header Information -------------------------+" -ForegroundColor Cyan
Write-Host "   | Server address, last update time             |" -ForegroundColor Cyan
Write-Host "   +-- System Overview ----+-- Task Details -----+" -ForegroundColor Green
Write-Host "   | Task statistics       | Selected task info   |" -ForegroundColor Green
Write-Host "   +-- Task List ------------------------------- +" -ForegroundColor Yellow
Write-Host "   | ID | Status | Project | Desc | Priority ... |" -ForegroundColor Yellow
Write-Host "   +-- Shortcut Help --------------------------- +" -ForegroundColor White
Write-Host "   | Operation tips                               |" -ForegroundColor White
Write-Host "   +---------------------------------------------+" -ForegroundColor White
Write-Host ""

Write-Host "6. Status icon descriptions:" -ForegroundColor Green
Write-Host "   Hourglass pending   - Waiting for execution" -ForegroundColor White
Write-Host "   Arrows    running   - Currently executing" -ForegroundColor White
Write-Host "   Check     completed - Successfully executed" -ForegroundColor White
Write-Host "   X         failed    - Execution failed" -ForegroundColor White
Write-Host "   Stop      cancelled - Cancelled" -ForegroundColor White
Write-Host "   Clock     timeout   - Execution timeout" -ForegroundColor White
Write-Host ""

Write-Host "Notes:" -ForegroundColor Yellow
Write-Host "- TUI interface requires MCP server to be running to display actual data" -ForegroundColor Gray
Write-Host "- If server is not running, interface will show connection error but won't crash" -ForegroundColor Gray
Write-Host "- Supports terminal window resizing, interface adapts automatically" -ForegroundColor Gray
Write-Host "- Works normally in Windows Terminal, PowerShell, and CMD" -ForegroundColor Gray
Write-Host ""

Write-Host "To start MCP server, run:" -ForegroundColor Yellow
Write-Host "auto-claude-code mcp-server" -ForegroundColor White
Write-Host ""

Write-Host "Demo complete! You can now try starting the TUI interface." -ForegroundColor Cyan 