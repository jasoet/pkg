#!/bin/bash

# Development container setup script
set -e

echo "🚀 Setting up Go development environment..."

# Update system packages
echo "📦 Updating system packages..."
sudo apt-get update

# Install additional tools
echo "🔧 Installing development tools..."
sudo apt-get install -y \
    curl \
    wget \
    jq \
    vim \
    tree \
    htop \
    postgresql-client \
    redis-tools

# Install Go tools
echo "🔨 Installing Go development tools..."

# Install mage (build automation)
go install github.com/magefile/mage@latest

# Install linting tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Install security tools
go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest

# Install testing tools
go install gotest.tools/gotestsum@latest

# Install migration tools
go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Install documentation tools
go install github.com/swaggo/swag/cmd/swag@latest

# Install mock generation
go install github.com/golang/mock/mockgen@latest

# Install dependency scanner
go install github.com/sonatypecommunity/nancy@latest

# Install code generation tools
go install github.com/deepmap/oapi-codegen/cmd/oapi-codegen@latest

echo "📥 Installing project dependencies..."
go mod download
go mod tidy

# Install development tools via mage
echo "🛠️ Installing additional tools via mage..."
mage tools || echo "⚠️ Some tools may have failed to install (this is often expected)"

# Set up Git configuration (if not already set)
if [ -z "$(git config --global user.name)" ]; then
    echo "⚙️ Setting up Git configuration..."
    echo "Please configure Git:"
    echo "git config --global user.name 'Your Name'"
    echo "git config --global user.email 'your.email@example.com'"
fi

# Create useful aliases
echo "🔗 Setting up useful aliases..."
cat >> ~/.bashrc << 'EOF'

# Go development aliases
alias ll='ls -la'
alias la='ls -A'
alias l='ls -CF'
alias ..='cd ..'
alias ...='cd ../..'

# Go aliases
alias got='go test'
alias gob='go build'
alias gom='go mod'
alias gor='go run'

# Mage aliases
alias mt='mage test'
alias ml='mage lint'
alias mi='mage integrationTest'
alias mca='mage checkAll'

# Docker aliases
alias dcu='docker-compose up -d'
alias dcd='docker-compose down'
alias dcl='docker-compose logs -f'
alias dcr='docker-compose restart'

# Git aliases
alias gs='git status'
alias ga='git add'
alias gc='git commit'
alias gp='git push'
alias gl='git log --oneline'
alias gd='git diff'

EOF

# Set up shell prompt with Go info
cat >> ~/.bashrc << 'EOF'
# Custom prompt with Go version
parse_git_branch() {
    git branch 2> /dev/null | sed -e '/^[^*]/d' -e 's/* \(.*\)/ (\1)/'
}

export PS1="\[\033[32m\]\u@\h\[\033[00m\]:\[\033[34m\]\w\[\033[31m\]\$(parse_git_branch)\[\033[00m\]$ "
EOF

# Create development directories
echo "📁 Creating development directories..."
mkdir -p ~/workspace
mkdir -p ~/logs
mkdir -p ~/scripts

# Set up environment variables
echo "🌍 Setting up environment variables..."
cat >> ~/.bashrc << 'EOF'
# Go development environment
export GOPROXY=https://proxy.golang.org,direct
export GOSUMDB=sum.golang.org
export GOPRIVATE=""
export CGO_ENABLED=0

# Development settings
export DEBUG=true
export LOG_LEVEL=debug

# Add Go bin to PATH
export PATH=$PATH:$(go env GOPATH)/bin
EOF

# Create development scripts
echo "📜 Creating development helper scripts..."

# Quick test script
cat > ~/scripts/quick-test.sh << 'EOF'
#!/bin/bash
echo "🧪 Running quick tests..."
mage test || go test -short ./...
EOF

# Quick setup script for new projects
cat > ~/scripts/setup-project.sh << 'EOF'
#!/bin/bash
echo "🏗️ Setting up new project with github.com/jasoet/pkg..."
if [ -z "$1" ]; then
    echo "Usage: $0 <project-name>"
    exit 1
fi

PROJECT_NAME=$1
mkdir -p "$PROJECT_NAME"
cd "$PROJECT_NAME"

go mod init "$PROJECT_NAME"
go get github.com/jasoet/pkg@latest

echo "✅ Project $PROJECT_NAME set up successfully!"
echo "💡 Check the templates directory for starting points:"
echo "   - templates/web-service/"
echo "   - templates/worker/"
echo "   - templates/cli-app/"
EOF

chmod +x ~/scripts/*.sh

# Add scripts to PATH
echo 'export PATH=$PATH:~/scripts' >> ~/.bashrc

echo "✅ Development environment setup complete!"
echo ""
echo "🎯 Quick start commands:"
echo "  mage test           - Run tests"
echo "  mage lint           - Run linter"
echo "  mage integrationTest - Run integration tests"
echo "  mage checkAll       - Run all quality checks"
echo "  mage docker:up      - Start development services"
echo ""
echo "📁 Useful directories:"
echo "  ~/workspace         - For your projects"
echo "  ~/logs             - For log files"
echo "  ~/scripts          - Helper scripts"
echo ""
echo "🔗 Helper scripts:"
echo "  quick-test.sh      - Run quick tests"
echo "  setup-project.sh   - Set up new project"
echo ""
echo "📖 Documentation:"
echo "  .claude/           - Claude Code integration guides"
echo "  templates/         - Project templates"
echo "  integration-examples/ - Complete working examples"
echo ""
echo "🔄 Reload your shell: source ~/.bashrc"