# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Macbat** is a macOS battery monitoring utility that helps extend MacBook battery life by tracking charge levels and sending system notifications when charging/discharging thresholds are reached (typically 20-80%). The application runs as a background `launchd` agent with a system tray icon interface.

- **Language**: Go 1.24.4
- **Platform**: macOS (supports both Apple Silicon and Intel)
- **Architecture**: CLI application with system tray GUI + background monitoring service

## Build and Run Commands

### Development
```bash
# Build the binary
make build

# Build and run with log output
make run

# Clean build artifacts
make clean

# Quick check (format + vet + test)
make quick
```

### Release and Distribution
```bash
# Build release binary
make release

# Full production release (creates GitHub release + updates Homebrew formula)
make prod

# Create and push new version tag
make next-tag

# Delete a specific tag
make del-tag TAG=v2.1.1
```

### Testing
```bash
# Run all tests
make test

# Run tests with coverage report
make test-coverage

# Run race detector
make test-race

# Run specific test
make test-specific TEST=TestName
```

## High-Level Architecture

### Core Components

1. **Entry Point** ([cmd/macbat/main.go](cmd/macbat/main.go))
   - Initializes the application via `NewApp()`
   - Uses `urfave/cli/v3` for command-line interface
   - Supports multiple execution modes via flags

2. **Execution Modes**
   - **Default mode**: Auto-installs if not installed, then launches system tray
   - **`--background`**: Runs battery monitoring loop (called by launchd)
   - **`--gui-agent`**: Runs system tray interface
   - **`install`**: Sets up launchd agents and config files
   - **`uninstall`**: Removes launchd agents and optionally preserves config/logs
   - **`config`**: Opens config.json in nano editor
   - **`log`**: Displays or follows log file

3. **Battery Monitoring** ([internal/monitor/monitor.go](internal/monitor/monitor.go))
   - Core monitoring logic in `Monitor` struct
   - Continuously checks battery state via `battery.GetBatteryInfo()`
   - Sends notifications when thresholds are crossed
   - Uses dynamic check intervals: faster when charging (`CheckIntervalWhenCharging`), slower when discharging (`CheckIntervalWhenDischarging`)
   - Tracks state changes to avoid duplicate notifications

4. **System Tray Interface** ([internal/tray/tray.go](internal/tray/tray.go))
   - Uses `getlantern/systray` for macOS menu bar icon
   - Displays real-time battery info (charge %, cycles, health, time remaining)
   - Allows user to modify thresholds and check intervals via dialogs
   - Updates menu every 5 seconds
   - Provides access to config.json and logs

5. **Configuration Management** ([internal/config/config.go](internal/config/config.go))
   - `Config` struct holds all settings (thresholds, intervals, logging)
   - `Manager` handles loading/saving JSON config (~/.config/macbat/config.json)
   - Auto-creates config with defaults if missing
   - Config watcher ([internal/config/watcher.go](internal/config/watcher.go)) detects file changes via `fsnotify`

6. **Battery Information** ([internal/battery/battery_info.go](internal/battery/battery_info.go))
   - Retrieves macOS battery data via IOKit (likely using system commands or CGO)
   - Returns `BatteryInfo` struct with: charge level, charging state, cycles, health, time estimates

### Key Directories

- `cmd/macbat/` - Main application entry point and CLI command handlers
- `internal/battery/` - Battery information retrieval
- `internal/monitor/` - Core monitoring logic
- `internal/tray/` - System tray GUI implementation
- `internal/config/` - Configuration management and file watching
- `internal/logger/` - Logging infrastructure
- `internal/paths/` - Path resolution for config/logs
- `internal/dialog/` - System notifications
- `internal/background/` - Background process management
- `internal/version/` - Version info injected at build time

### Build System Details

The [Makefile](Makefile) includes sophisticated build automation:
- Version information is injected via `-ldflags` from git tags, commit hash, and build date
- Build number auto-increments via `scripts/update_build_number.sh`
- `make prod` performs full release cycle: build → tag → GitHub release → update Homebrew formula with architecture-specific binaries (arm64/amd64)

## Configuration File

Default location: `~/Library/Application Support/macbat/config.json`

Key settings:
- `min_threshold` / `max_threshold`: Battery charge thresholds (%)
- `check_interval_charging` / `check_interval_discharging`: Monitoring intervals (seconds)
- `debug_enabled`: Enable detailed logging
- `log_file_path`: Log file location
- `run_at_load`: Auto-start with macOS

## Important Implementation Notes

1. **macOS-Specific**: Uses IOKit for battery info, launchd for background service, system notifications for alerts
2. **CGO Required**: `CGO_ENABLED=1` is necessary for system tray and dialogs
3. **Thread Safety**: System tray updates use `runtime.LockOSThread()` for IOKit compatibility
4. **Dual Process Model**: Background monitoring process + GUI agent process, coordinated via launchd
5. **State Management**: Monitor tracks `lastLevel` and `lastKnownCharging` to detect changes and reset state on mode transitions
6. **Version Injection**: Version, commit hash, and build number are injected at compile time via ldflags

## Common Development Workflows

### Modifying Monitoring Logic
1. Edit [internal/monitor/monitor.go](internal/monitor/monitor.go)
2. Key methods: `Check()`, `checkChargingState()`, `checkDischargingState()`
3. Test locally with `make run`
4. Check logs: `./macbat --log`

### Changing Thresholds or Intervals
- Programmatically: Edit [internal/config/config.go](internal/config/config.go) `Default()` function
- At runtime: Use system tray menu or `./macbat config`

### Adding New CLI Commands
1. Add command handler in [cmd/macbat/commands.go](cmd/macbat/commands.go)
2. Register in `App.setupCLI()` (likely in [cmd/macbat/init.go](cmd/macbat/init.go))
3. Follow existing pattern with `*cli.Command` structure

### Releasing New Version
```bash
make prod  # Full automated release pipeline
```
This will build, create git tag, publish GitHub release with binaries, and update Homebrew tap.
