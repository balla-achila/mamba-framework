# Config Package

## Overview
The config package handles application configuration loading and management.

## Features
- JSON configuration file support
- Environment-based configuration
- Default values for missing fields
- Type-safe configuration access

## Usage
```go
cfg, err := config.Load("config/config.json")
if err != nil {
    log.Fatal(err)
}
