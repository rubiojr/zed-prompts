# Zed Prompts Importer Exporter

A command-line utility for importing and exporting prompts from Zed's prompt library.

## Overview

This tool allows you to:
- Export your Zed prompts to JSON format for backup or sharing
- Import prompts from JSON into your Zed prompt library
- List available prompts in your library

## Installation

```bash
go install github.com/rubiojr/zed-prompts@latest
```

## Usage

> [!NOTE]
> Running the tool when Zed is running generally works, though it may cause database inconsistencies, if the prompt library is being modified concurrently.
> It's recommended to stop Zed before running the tool.

### Export Prompts

Export your prompts from Zed's LMDB database to a JSON file:

```bash
zed-prompts export --output prompts.json
```

Export to stdout:

```bash
zed-prompts export --output -
```

Importing from a remote computer (the remote computer must have the tool installed):

```bash
ssh <my-host> zed-prompts export --output - | zed-prompts import --input -
```

Use a custom database path:

```bash
zed-prompts export --db /path/to/prompts-library-db.0.mdb --output prompts.json
```

### Import Prompts

Import prompts from a JSON file into your Zed prompt library:

```bash
zed-prompts import --input prompts.json
```

Import from stdin:

```bash
cat prompts.json | zed-prompts import --input -
```

Use a custom database path:

```bash
zed-prompts import --input prompts.json --db /path/to/prompts-library-db.0.mdb
```

### List Available Prompts

List metadata for all available prompts:

```bash
zed-prompts list
```

## Default Database Location

By default, the tool looks for Zed's prompt database at the following locations.

Linux:

```
~/.local/share/zed/prompts/prompts-library-db.0.mdb
```

macOS:

```
~/.config/zed/prompts
```

## JSON Format

The JSON file format for import/export is an array of prompt objects:

```json
[
  {
    "metadata": {
      "id": {
        "kind": "User",
        "uuid": "12345678-1234-1234-1234-123456789012"
      },
      "title": "My Prompt",
      "default": false,
      "saved_at": "2023-01-01T12:00:00Z"
    },
    "content": "This is the text of the prompt"
  }
]
```

## Use Cases

- Backup your prompt library
- Share prompts with colleagues
- Migrate prompts between different machines
- Version control your prompts collection

## Requirements

- Go 1.24 or higher
- Zed editor installed (for accessing the prompts database)

## Note

This tool directly manipulates the LMDB database used by Zed. It's recommended to close Zed before importing prompts to prevent potential conflicts.
