# Documentation

Welcome to the MCP Server documentation. This directory contains comprehensive guides and references for using the MCP Server package.

## Getting Started

- **[Quick Start Guide](../README.md#quick-start)** - Get up and running in minutes
- **[API Reference](../README.md#api-reference)** - Programmatic API documentation

## Features

### File-Based Tool Registration

- **[File-Based Tool Registration](./file-based-registration.md)** - Declaratively define tools and handlers using JSON/YAML files
  - Tools configuration format
  - Handlers configuration format
  - Path and query parameter substitution
  - Environment variable support
  - Best practices

### Configuration Validation

- **[Configuration Validation](./configuration-validation.md)** - Automatic validation of configuration files
  - Current validation features
  - How validation works
  - Common validation errors and fixes
  - Validation rules and best practices
  - Troubleshooting guide

### MCP Schema Versions

- **[MCP Schema Version Support](./mcp-schema-versions.md)** - Guide to MCP schema version support
  - Supported versions (2024-11-05, 2025-03-26, 2025-06-18)
  - How to specify a version
  - Version compatibility and migration guide
  - Best practices

### Authorization

- **[Authorization Strategies](./auth/README.md)** - Comprehensive guide to all authorization strategies
  - [Pass-Through](./auth/pass-through.md) - Forward client authorization as-is
  - [Transform](./auth/transform.md) - Modify authorization header format
  - [Static](./auth/static.md) - Use fixed authorization values
  - [Basic](./auth/basic.md) - Basic Authentication encoding
  - [None](./auth/none.md) - Explicitly disable authorization

## Examples

- **[HTTP Server Example](../examples/simple/)** - Complete HTTP server with file-based registration
- **[Stdio Server Example](../examples/stdio-server/)** - Stdio transport example

## Additional Resources

- **[Main README](../README.md)** - Overview and quick start
- **[GitHub Repository](https://github.com/vubon/go-mcp-server)** - Source code and issues

