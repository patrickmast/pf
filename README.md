# pf - folder picker

A fast terminal folder picker with fuzzy search.

## Installation

### Option 1: Homebrew (macOS)

```bash
brew tap patrickmast/pf
brew install pf
pf --install
source ~/.zshrc
```

### Option 2: Go install

```bash
go install github.com/patrickmast/pf@latest
pf --install
source ~/.zshrc
```

### Option 3: Build from source

```bash
git clone https://github.com/patrickmast/pf.git
cd pf
go build -o pf .
cp pf /usr/local/bin/
pf --install
source ~/.zshrc
```

## Usage

```bash
pf              # Start in current directory
pf ~/Projects   # Start in specific directory
```

## Keyboard shortcuts

| Key | Action |
|-----|--------|
| `↑` / `↓` | Navigate list |
| `Enter` | Open folder |
| `Tab` | Select folder & cd to it |
| `Esc` | Go to parent folder |
| `Backspace` | Clear filter character |
| `Ctrl+C` | Quit without selecting |
| `F1` | Show help |

## Filtering

Just start typing to filter folders.

Multiple words filter independently - typing `my proj` matches folders containing both "my" AND "proj" (e.g., "my-project", "project-my-app").

## CLI options

```bash
pf --help      # Show help
pf --install   # Install shell function
```

## Why a shell function?

A subprocess cannot change the parent shell's directory. The shell function captures `pf`'s output (the selected path) and runs `cd` in your current shell.

## License

MIT
