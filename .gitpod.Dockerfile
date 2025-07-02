FROM gitpod/workspace-go:latest

USER root

# Install additional system packages
RUN apt-get update && apt-get install -y \
    postgresql-client \
    redis-tools \
    jq \
    tree \
    htop \
    curl \
    wget \
    vim \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Install Docker Compose
RUN curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose \
    && chmod +x /usr/local/bin/docker-compose

USER gitpod

# Set up Go environment
ENV GO111MODULE=on \
    GOPROXY=https://proxy.golang.org,direct \
    GOSUMDB=sum.golang.org \
    CGO_ENABLED=0

# Install Go development tools
RUN go install github.com/magefile/mage@latest && \
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest && \
    go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest && \
    go install gotest.tools/gotestsum@latest && \
    go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest && \
    go install github.com/swaggo/swag/cmd/swag@latest && \
    go install github.com/golang/mock/mockgen@latest

# Set up shell aliases and environment
RUN echo 'alias ll="ls -la"' >> ~/.bashrc && \
    echo 'alias got="go test"' >> ~/.bashrc && \
    echo 'alias gob="go build"' >> ~/.bashrc && \
    echo 'alias mt="mage test"' >> ~/.bashrc && \
    echo 'alias ml="mage lint"' >> ~/.bashrc && \
    echo 'alias mi="mage integrationTest"' >> ~/.bashrc && \
    echo 'export DEBUG=true' >> ~/.bashrc && \
    echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.bashrc

# Create workspace directories
RUN mkdir -p /home/gitpod/workspace/logs && \
    mkdir -p /home/gitpod/workspace/scripts