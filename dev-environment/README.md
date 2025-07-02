# Development Environment Setup

This directory contains configuration files and guides for setting up an optimal development environment for the github.com/jasoet/pkg utility library.

## Available Environments

### 🔧 Local Development

#### VS Code Configuration
The `.vscode/` directory contains optimized settings for Go development:

- **settings.json**: Go-specific editor settings, linting, formatting
- **launch.json**: Debug configurations for main app, tests, templates, and examples
- **tasks.json**: Build tasks for testing, linting, and Docker operations
- **extensions.json**: Recommended VS Code extensions

#### Key Features:
- Auto-formatting on save with gofmt
- Integrated testing and debugging
- Docker integration
- Git integration with GitLens
- SQL tools for database work
- REST client for API testing

#### Setup:
```bash
# Open in VS Code
code .

# Install recommended extensions (VS Code will prompt)
# Or manually: Ctrl+Shift+P -> "Extensions: Show Recommended Extensions"
```

### 🐳 Dev Containers

#### Development Container
The `.devcontainer/` directory provides a complete containerized development environment:

- **devcontainer.json**: Container configuration with Go 1.23, Docker, and tools
- **setup.sh**: Automated setup script for development tools and aliases

#### Features:
- Pre-configured Go development environment
- All development tools pre-installed
- Docker-in-Docker support
- Port forwarding for web services and databases
- Integrated terminal with helpful aliases

#### Setup:
```bash
# Open in VS Code with Dev Containers extension
# Command palette: "Dev Containers: Reopen in Container"
```

### ☁️ Cloud Development

#### Gitpod Integration
The `.gitpod.yml` and `.gitpod.Dockerfile` provide cloud-based development:

- **Automatic environment setup** on workspace creation
- **Pre-built development services** (PostgreSQL, Redis)
- **Integrated VS Code** in the browser
- **GitHub integration** with prebuild support

#### Features:
- Zero-setup development environment
- Automatic service startup
- Pre-installed Go tools and extensions
- Browser-based VS Code experience

#### Setup:
```bash
# Visit: https://gitpod.io/#https://github.com/jasoet/pkg
# Or add "gitpod.io/#" prefix to any GitHub URL
```

### 📝 Editor Configuration

#### EditorConfig
The `.editorconfig` file ensures consistent coding styles across different editors:

- **Go files**: Tab indentation (4 spaces)
- **YAML/JSON**: Space indentation (2 spaces)
- **Markdown**: Preserve trailing whitespace for line breaks
- **Shell scripts**: Space indentation (2 spaces)

## Development Tools

### Pre-installed Tools

All environments include these essential Go development tools:

```bash
# Build automation
mage

# Linting and static analysis
golangci-lint
gosec

# Testing
gotestsum

# Database migrations
migrate

# Documentation
swag

# Mock generation
mockgen

# Security scanning
nancy
```

### Quick Commands

Common development commands available in all environments:

```bash
# Testing
mage test                    # Run unit tests
mage integrationTest         # Run integration tests with Docker
mage checkAll               # Run all quality checks

# Code Quality
mage lint                   # Run linter
mage security               # Run security analysis
mage coverage               # Generate coverage report

# Development Services
mage docker:up              # Start PostgreSQL and other services
mage docker:down            # Stop services
mage docker:logs            # View service logs

# Tools
mage tools                  # Install all development tools
mage clean                  # Clean build artifacts
```

### Helpful Aliases

All environments include these shell aliases:

```bash
# Go commands
got="go test"
gob="go build"
gom="go mod"
gor="go run"

# Mage shortcuts
mt="mage test"
ml="mage lint"
mi="mage integrationTest"
mca="mage checkAll"

# Docker shortcuts
dcu="docker-compose up -d"
dcd="docker-compose down"
dcl="docker-compose logs -f"
dcr="docker-compose restart"

# Git shortcuts
gs="git status"
ga="git add"
gc="git commit"
gp="git push"
gl="git log --oneline"
gd="git diff"
```

## Environment-Specific Features

### VS Code Local Development

**Debugging Support:**
- Debug main application
- Debug tests with breakpoints
- Debug templates and integration examples
- Debug specific packages

**Integrated Testing:**
- Run tests from editor
- View coverage in editor
- Test explorer integration

**Database Integration:**
- SQL tools for database queries
- Connection management
- Schema visualization

### Dev Containers

**Isolated Environment:**
- Consistent development environment
- No local tool installation required
- Docker-in-Docker for full container development

**Port Forwarding:**
- Automatic port forwarding for web services (8080)
- Database access (PostgreSQL 5432, Redis 6379)
- Live development with hot reload

### Gitpod Cloud

**Zero Setup:**
- Instant development environment
- No local dependencies
- Pre-built workspace images

**GitHub Integration:**
- Automatic workspace creation from GitHub URLs
- Pull request development
- Branch-based workspaces

## Project Structure Integration

All development environments are configured to work optimally with the project structure:

```
github.com/jasoet/pkg/
├── .vscode/                 # VS Code configuration
├── .devcontainer/           # Dev container configuration
├── .gitpod.yml             # Gitpod configuration
├── .editorconfig           # Editor configuration
├── CLAUDE.md               # Claude Code integration guide
├── .claude/                # Claude Code metadata
├── templates/              # Project templates
├── integration-examples/   # Working examples
├── pkg/                    # Core library packages
└── scripts/               # Development scripts
```

### Template Development

Each development environment includes debug configurations for all templates:

- **Web Service Template**: Debug with proper environment variables
- **Worker Template**: Debug background processing
- **CLI App Template**: Debug with command-line arguments

### Integration Examples

Debug configurations for all integration examples:

- **E-commerce API**: Full-featured web application
- **Analytics Dashboard**: Real-time data processing
- **Data Pipeline**: ETL and batch processing

## Troubleshooting

### Common Issues

**Go tools not found:**
```bash
# Reinstall tools
mage tools

# Check PATH
echo $PATH | grep $(go env GOPATH)/bin
```

**Docker permission issues:**
```bash
# Add user to docker group (Linux)
sudo usermod -aG docker $USER
# Logout and login again
```

**Database connection issues:**
```bash
# Check services are running
mage docker:up
docker-compose ps

# Test connection
psql -h localhost -p 5432 -U jasoet -d pkg_db
```

**VS Code extensions not working:**
```bash
# Reload window
Ctrl+Shift+P -> "Developer: Reload Window"

# Check Go environment
Ctrl+Shift+P -> "Go: Environment"
```

### Environment Variables

Key environment variables for development:

```bash
# Go development
export GOPROXY=https://proxy.golang.org,direct
export GOSUMDB=sum.golang.org
export CGO_ENABLED=0

# Application debugging
export DEBUG=true
export LOG_LEVEL=debug

# Database (for integration tests)
export AUTOMATION=true
```

## IDE-Specific Guides

### IntelliJ IDEA / GoLand

While not included in the pre-configured environments, the project works well with JetBrains IDEs:

**Setup:**
1. Open project in GoLand
2. Configure Go SDK (1.23+)
3. Enable Go modules
4. Install recommended plugins

**Useful Plugins:**
- Docker integration
- Database tools
- YAML support
- Makefile support

### Vim/Neovim

For Vim users, recommended plugins:

```vim
" Go development
Plug 'fatih/vim-go'
Plug 'neoclide/coc.nvim'

" General development
Plug 'tpope/vim-fugitive'
Plug 'junegunn/fzf.vim'
```

### Emacs

For Emacs users, recommended packages:

```elisp
;; Go development
(use-package go-mode)
(use-package company-go)
(use-package flycheck-golangci-lint)

;; General development
(use-package magit)
(use-package projectile)
```

## Contributing

When contributing to this project:

1. **Use any of the provided development environments**
2. **Follow the EditorConfig settings** for consistent formatting
3. **Run quality checks** before submitting: `mage checkAll`
4. **Test your changes** with: `mage integrationTest`
5. **Update documentation** if adding new features

The development environments are designed to make contribution easy and consistent across different setups and platforms.