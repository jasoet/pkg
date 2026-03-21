# Security Fixes Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix all 8 critical and 24 high-severity findings from the code review audit across 10 packages.

**Architecture:** Each package gets its own branch (`fix/<pkg>-*`), merged to `main` in dependency order. Fixes are small, targeted changes — guard checks, config defaults, unexport methods, fix append patterns. TDD: write failing test first, then fix.

**Tech Stack:** Go 1.26+, testify, `task test`, `task lint`

---

### Task 1: otel — Fix slice aliasing in Start* methods (H11)

**Files:**
- Modify: `otel/instrumentation.go:355,388,421,464,500`
- Test: `otel/instrumentation_test.go`

**Step 1: Create branch**
```bash
git checkout -b fix/otel-safety main
```

**Step 2: Write failing test**

Add to `otel/instrumentation_test.go`:
```go
func TestStartHandler_NoSliceAliasing(t *testing.T) {
	cfg := NewConfig("test-service")
	ctx := ContextWithConfig(context.Background(), cfg)

	baseFields := make([]Field, 2, 5)
	baseFields[0] = F("key1", "val1")
	baseFields[1] = F("key2", "val2")

	subFields := baseFields[:2]
	lc := Layers.StartHandler(ctx, "TestOp", subFields...)
	defer lc.End()

	assert.Equal(t, "val1", baseFields[0].Value)
	assert.Equal(t, "val2", baseFields[1].Value)
}
```

**Step 3: Run test to verify it fails**
```bash
go test ./otel/ -run TestStartHandler_NoSliceAliasing -v
```

**Step 4: Fix all five Start* methods**

In `otel/instrumentation.go`, at lines 355, 388, 421, 464, 500, change FROM:
```go
allFields := append([]Field{F("layer", "handler")}, fields...)
```
Change TO:
```go
allFields := make([]Field, 0, 1+len(fields))
allFields = append(allFields, F("layer", "handler"))
allFields = append(allFields, fields...)
```

Apply with respective layer names: `"handler"`, `"service"`, `"operations"`, `"middleware"`, `"repository"`.

**Step 5: Run test to verify it passes**
```bash
go test ./otel/ -run TestStartHandler_NoSliceAliasing -v -race
```

**Step 6: Run full otel test suite**
```bash
go test ./otel/ -v -race
```

---

### Task 2: otel — Add no-op provider singletons (H12)

**Files:**
- Modify: `otel/config.go:15,162,222,231,240`

**Step 1: Add package-level singletons (around line 15)**
```go
var (
	noopTracerProvider = noopt.NewTracerProvider()
	noopMeterProvider  = noopm.NewMeterProvider()
	noopLoggerProvider = noopl.NewLoggerProvider()
)
```

**Step 2: Update GetTracer (line 222)**
```go
return noopTracerProvider.Tracer(scopeName, opts...)
```

**Step 3: Update GetMeter (line 231)**
```go
return noopMeterProvider.Meter(scopeName, opts...)
```

**Step 4: Update GetLogger (line 240)**
```go
return noopLoggerProvider.Logger(scopeName, opts...)
```

**Step 5: Update defaultLoggerProvider fallback (line 162)**
```go
return noopLoggerProvider
```

**Step 6: Run full test suite + lint**
```bash
go test ./otel/ -v -race && task lint
```

**Step 7: Commit and merge**
```bash
git add otel/instrumentation.go otel/config.go otel/instrumentation_test.go
git commit -m "fix(otel): fix slice aliasing in Start* methods and add no-op singletons"
git checkout main && git merge fix/otel-safety && git push origin main
git branch -d fix/otel-safety
```

---

### Task 3: config — Fix NestedEnvVars panics and truncation (C7, H20, H21)

**Files:**
- Modify: `config/config.go:65-78`
- Test: `config/config_test.go`

**Step 1: Create branch**
```bash
git checkout -b fix/config-safety main
```

**Step 2: Write failing tests**

Add to `config/config_test.go`:
```go
func TestNestedEnvVars_NegativeKeyDepth(t *testing.T) {
	v := viper.New()
	t.Setenv("TEST_APP_DB_HOST", "localhost")
	assert.NotPanics(t, func() {
		NestedEnvVars("TEST_APP_", -1, "app", v)
	})
}

func TestNestedEnvVars_MultiSegmentFieldName(t *testing.T) {
	v := viper.New()
	t.Setenv("TEST_APP_DB_CONNECTION_TIMEOUT", "30")
	NestedEnvVars("TEST_APP_", 2, "app", v)
	assert.Equal(t, "30", v.GetString("app.db.connection_timeout"))
}
```

**Step 3: Run tests to verify they fail**
```bash
go test ./config/ -run "TestNestedEnvVars_NegativeKeyDepth|TestNestedEnvVars_MultiSegmentFieldName" -v
```

**Step 4: Fix NestedEnvVars in `config/config.go`**

Add guard after function signature (line 65):
```go
if keyDepth < 0 {
	return
}
```

Update doc comment to add thread-safety note:
```go
// NestedEnvVars scans environment variables with the given prefix and injects
// matched keys into the Viper instance as nested map values.
//
// Note: This function is NOT goroutine-safe when called with a shared *viper.Viper
// instance. Callers must serialize access if calling concurrently.
```

At line 78, change FROM:
```go
fieldName := strings.ToLower(keyParts[keyDepth+1])
```
Change TO:
```go
fieldName := strings.ToLower(strings.Join(keyParts[keyDepth+1:], "_"))
```

**Step 5: Run tests**
```bash
go test ./config/ -v
```

**Step 6: Commit and merge**
```bash
git add config/
git commit -m "fix(config): guard negative keyDepth panic and fix multi-segment field truncation"
git checkout main && git merge fix/config-safety && git push origin main
git branch -d fix/config-safety
```

---

### Task 4: compress — Add sentinel errors and fix decompression bomb (C5)

**Files:**
- Create: `compress/errors.go`
- Modify: `compress/gz.go:54`
- Modify: `compress/tar.go:168`
- Test: `compress/security_test.go`

**Step 1: Create branch**
```bash
git checkout -b fix/compress-security main
```

**Step 2: Create sentinel errors file `compress/errors.go`**
```go
package compress

import "errors"

var (
	ErrSizeLimitExceeded = errors.New("size limit exceeded")
	ErrPathTraversal     = errors.New("path traversal detected")
)
```

**Step 3: Fix `extractTarFile` in `tar.go:168`**

After `io.Copy` with `LimitReader`, add:
```go
if written >= maxFileSize {
	probe := make([]byte, 1)
	if n, _ := tarReader.Read(probe); n > 0 {
		return totalWritten, fmt.Errorf("file %s: %w (max %d bytes)", header.Name, ErrSizeLimitExceeded, maxFileSize)
	}
}
```

**Step 4: Fix `UnGz` in `gz.go:54`**

After `io.Copy` with `LimitReader`, add:
```go
if written >= maxSize {
	probe := make([]byte, 1)
	if n, _ := zipReader.Read(probe); n > 0 {
		return written, fmt.Errorf("%w: file exceeds maximum size of %d bytes", ErrSizeLimitExceeded, maxSize)
	}
}
```

**Step 5: Update tests in `compress/security_test.go`**

Change decompression bomb tests from `assert.NoError` to `assert.ErrorIs(t, err, ErrSizeLimitExceeded)`.

**Step 6: Run tests**
```bash
go test ./compress/ -v
```

---

### Task 5: compress — Fix path traversal and path stripping (C6, H18, H19)

**Files:**
- Modify: `compress/gz.go:35`
- Modify: `compress/tar.go:41,243`
- Test: `compress/security_test.go`

**Step 1: Fix `UnGz` path traversal check (`gz.go:35`)**

Change FROM:
```go
cleanDst := filepath.Clean(dst)
if strings.Contains(cleanDst, "..") {
	return 0, fmt.Errorf("invalid destination path: %s", dst)
}
```
Change TO:
```go
if !filepath.IsAbs(dst) {
	return 0, fmt.Errorf("%w: destination must be an absolute path: %s", ErrPathTraversal, dst)
}
```

**Step 2: Fix `Tar` path stripping (`tar.go:41`)**

Change FROM:
```go
localDirectory := strings.Replace(file, sourceDirectory, "", -1)
header.Name = strings.TrimPrefix(localDirectory, string(filepath.Separator))
```
Change TO:
```go
relPath, err := filepath.Rel(sourceDirectory, file)
if err != nil {
	return err
}
header.Name = relPath
```

**Step 3: Add symlink TOCTOU defense in `UnTar` (after line 243)**

After the `strings.HasPrefix` path check, add:
```go
parentDir := filepath.Dir(target)
resolvedParent, err := filepath.EvalSymlinks(parentDir)
if err == nil {
	cleanDest := filepath.Clean(destinationDir)
	if !strings.HasPrefix(resolvedParent, cleanDest+string(os.PathSeparator)) && resolvedParent != cleanDest {
		return totalWritten, fmt.Errorf("%w: symlink resolves outside destination: %s", ErrPathTraversal, header.Name)
	}
}
```

**Step 4: Add tests**

```go
func TestUnGz_RejectsRelativePath(t *testing.T) {
	_, err := UnGz("testdata/test.gz", "../../../tmp/evil")
	assert.ErrorIs(t, err, ErrPathTraversal)
}
```

**Step 5: Run tests + lint**
```bash
go test ./compress/ -v && task lint
```

**Step 6: Commit and merge**
```bash
git add compress/
git commit -m "fix(compress): fix decompression bomb detection, path traversal, and symlink TOCTOU"
git checkout main && git merge fix/compress-security && git push origin main
git branch -d fix/compress-security
```

---

### Task 6: db — Unexport Dsn, wrap errors, change SSL default (H1, H2, H3)

**Files:**
- Modify: `db/pool.go:79,116,160,167,182`
- Test: `db/pool_test.go`

**Step 1: Create branch**
```bash
git checkout -b fix/db-security main
```

**Step 2: Write failing tests**
```go
func TestRedactedDsn_MasksPassword(t *testing.T) {
	cfg := ConnectionConfig{
		Host: "localhost", Port: 5432, Username: "admin",
		Password: "supersecret", Database: "testdb", DbType: Postgresql,
	}
	redacted := cfg.RedactedDsn()
	assert.Contains(t, redacted, "***")
	assert.NotContains(t, redacted, "supersecret")
}

func TestEffectiveSSLMode_DefaultsToRequire(t *testing.T) {
	cfg := ConnectionConfig{}
	assert.Equal(t, "require", cfg.effectiveSSLMode())
}
```

**Step 3: Rename `Dsn()` to `dsn()`, add `RedactedDsn()` in `pool.go`**

Line 116: `func (c *ConnectionConfig) Dsn()` -> `func (c *ConnectionConfig) dsn()`

Update all internal callers. Add:
```go
func (c *ConnectionConfig) RedactedDsn() string {
	original := c.dsn()
	if c.Password != "" {
		return strings.ReplaceAll(original, c.Password, "***")
	}
	return original
}
```

**Step 4: Wrap connection errors (lines 160, 167, 182)**
```go
return nil, fmt.Errorf("failed to open database connection to %s:%d/%s: %w", c.Host, c.Port, c.Database, err)
return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
return nil, fmt.Errorf("failed to ping database at %s:%d/%s: %w", c.Host, c.Port, c.Database, err)
```

**Step 5: Change SSL default (`pool.go:79`)**
```go
return "require"  // was "disable"
```

**Step 6: Update existing tests, run tests + lint**
```bash
go test ./db/ -v && task lint
```

**Step 7: Commit and merge**
```bash
git add db/
git commit -m "fix(db): unexport Dsn, wrap connection errors, default SSL to require"
git checkout main && git merge fix/db-security && git push origin main
git branch -d fix/db-security
```

---

### Task 7: docker — Fix regex panic, double-start, format injection (H6, H7, H8)

**Files:**
- Modify: `docker/wait.go:31-33`
- Modify: `docker/executor.go:108`
- Modify: `docker/network.go:189`
- Test: `docker/wait_test.go`, `docker/executor_test.go`, `docker/network_test.go`

**Step 1: Create branch**
```bash
git checkout -b fix/docker-safety main
```

**Step 2: Fix `WaitForLog` (`wait.go:31-33`)**

Add `compileErr error` field to `waitForLog` struct. Change constructor:
```go
func WaitForLog(pattern string) *waitForLog {
	compiled, err := regexp.Compile(pattern)
	return &waitForLog{
		pattern:    compiled,
		compileErr: err,
	}
}
```

In `WaitUntilReady`, add at top:
```go
if w.compileErr != nil {
	return fmt.Errorf("invalid regex pattern: %w", w.compileErr)
}
```

**Step 3: Fix `Start()` idempotency (`executor.go:108`)**

Add after lock acquisition:
```go
if e.containerID != "" {
	e.mu.Unlock()
	return fmt.Errorf("container already started: %s", e.containerID)
}
```

**Step 4: Fix `ConnectionString` (`network.go:189`)**

Change FROM:
```go
return fmt.Sprintf(template, endpoint), nil
```
Change TO:
```go
return strings.ReplaceAll(template, "{{endpoint}}", endpoint), nil
```

Update doc comment and any tests using `%s` placeholder to use `{{endpoint}}`.

**Step 5: Run tests + lint**
```bash
go test ./docker/ -v && task lint
```

**Step 6: Commit and merge**
```bash
git add docker/
git commit -m "fix(docker): prevent regex panic, double-start leak, and format injection"
git checkout main && git merge fix/docker-safety && git push origin main
git branch -d fix/docker-safety
```

---

### Task 8: server — Add HTTP timeouts and body limit (H4, H5)

**Files:**
- Modify: `server/server.go:113`
- Test: `server/server_test.go`

**Step 1: Create branch**
```bash
git checkout -b fix/server-security main
```

**Step 2: Add timeouts after `echo.New()` (line 113)**
```go
e.Server.ReadHeaderTimeout = 5 * time.Second
e.Server.ReadTimeout = 30 * time.Second
e.Server.WriteTimeout = 30 * time.Second
e.Server.IdleTimeout = 120 * time.Second
```

**Step 3: Add body limit middleware in `setupEcho`**
```go
e.Use(middleware.BodyLimit("4M"))
```

**Step 4: Write test**
```go
func TestSetupEcho_HasTimeouts(t *testing.T) {
	// Verify e.Server has non-zero timeouts after setup
}
```

**Step 5: Run tests + lint**
```bash
go test ./server/ -v && task lint
```

**Step 6: Commit and merge**
```bash
git add server/
git commit -m "fix(server): add HTTP timeouts and default body size limit"
git checkout main && git merge fix/server-security && git push origin main
git branch -d fix/server-security
```

---

### Task 9: rest — Truncate response body, add nil guard (H16, H17)

**Files:**
- Modify: `rest/config.go:10`
- Modify: `rest/client.go:56,238`
- Test: `rest/client_test.go`

**Step 1: Create branch**
```bash
git checkout -b fix/rest-security main
```

**Step 2: Add `MaxResponseBodyLog` to Config (`config.go`)**
```go
MaxResponseBodyLog int `yaml:"maxResponseBodyLog" mapstructure:"maxResponseBodyLog"`
```
Default: `1024`

**Step 3: Add truncation helper in `client.go`**
```go
func truncateBody(body string, maxLen int) string {
	if maxLen > 0 && len(body) > maxLen {
		return body[:maxLen] + "...(truncated)"
	}
	return body
}
```

**Step 4: Apply truncation at line 238 and in HandleResponse**

**Step 5: Add nil guard in `WithOTelConfig` (line 56)**
```go
if client.restConfig != nil {
	client.restConfig.OTelConfig = cfg
}
```

**Step 6: Run tests + lint**
```bash
go test ./rest/ -v && task lint
```

**Step 7: Commit and merge**
```bash
git add rest/
git commit -m "fix(rest): truncate response body in errors and add nil guard for OTelConfig"
git checkout main && git merge fix/rest-security && git push origin main
git branch -d fix/rest-security
```

---

### Task 10: grpc — Remove TLS stub, fix reflection default (C3, C4)

**Files:**
- Modify: `grpc/config.go:66-68,98,354-360`
- Modify: `grpc/echo_gateway.go:56,65-76`
- Modify: `grpc/server.go:272`
- Test: `grpc/server_test.go`

**Step 1: Create branch**
```bash
git checkout -b fix/grpc-security main
```

**Step 2: Remove `WithTLS` and fields (`config.go`)**

Delete `enableTLS`, `certFile`, `keyFile` fields (lines 66-68). Delete `WithTLS` function (lines 354-360).

**Step 3: Change reflection default (`config.go:98`)**
```go
enableReflection: false,  // was true
```

**Step 4: Fix `waitForGRPCServer` (`echo_gateway.go:65-76`)**

Replace with `net.DialTimeout`:
```go
func waitForGRPCServer(endpoint string, maxRetries int) error {
	var err error
	for i := 0; i < maxRetries; i++ {
		conn, dialErr := net.DialTimeout("tcp", endpoint, 1*time.Second)
		if dialErr == nil {
			conn.Close()
			return nil
		}
		err = dialErr
		time.Sleep(time.Duration(100*(1<<uint(i))) * time.Millisecond)
	}
	return fmt.Errorf("gRPC server at %s not ready after %d retries: %w", endpoint, maxRetries, err)
}
```

**Step 5: Accept optional dial options in `SetupGatewayForSeparate` (line 56)**
```go
func (s *Server) SetupGatewayForSeparate(e *echo.Echo, grpcPort string, dialOpts ...grpc.DialOption) error {
	opts := dialOpts
	if len(opts) == 0 {
		opts = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	}
```

**Step 6: Add `ReadHeaderTimeout` to H2C server (`server.go:272`)**
```go
ReadHeaderTimeout: 5 * time.Second,
```

**Step 7: Update tests, run tests + lint**
```bash
go test ./grpc/ -v && task lint
```

**Step 8: Commit and merge**
```bash
git add grpc/
git commit -m "fix(grpc): remove TLS stub, disable reflection by default, fix server probe"
git checkout main && git merge fix/grpc-security && git push origin main
git branch -d fix/grpc-security
```

---

### Task 11: ssh — Fix data race, shutdown, credentials (C1, C2, H9, H10)

**Files:**
- Modify: `ssh/tunnel.go:23,49,121,168-177,183,213`
- Test: `ssh/tunnel_test.go`

**Step 1: Create branch**
```bash
git checkout -b fix/ssh-security main
```

**Step 2: Write tests**
```go
func TestTunnel_PasswordNotInYAML(t *testing.T) {
	cfg := Config{Password: "secret123"}
	data, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "secret123")
}

func TestTunnel_DoubleStartReturnsError(t *testing.T) {
	tunnel := &Tunnel{client: &ssh.Client{}}
	err := tunnel.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already started")
}
```

**Step 3: Fix `Password` YAML tag (line 23)**
```go
Password string `yaml:"-" mapstructure:"-"`
```

**Step 4: Add `stopCh` and `wg` to Tunnel struct (line 49)**
```go
stopCh chan struct{}
wg     sync.WaitGroup
```

**Step 5: Fix `Start()` — add guard, init stopCh, protect client (line 121)**

Guard at top:
```go
t.mu.Lock()
if t.client != nil {
	t.mu.Unlock()
	return fmt.Errorf("tunnel already started")
}
t.stopCh = make(chan struct{})
t.mu.Unlock()
```

Assign client under lock after dial:
```go
t.mu.Lock()
t.client = client
t.mu.Unlock()
```

**Step 6: Fix accept loop (lines 168-177)**

Check `stopCh` on accept error:
```go
select {
case <-t.stopCh:
	return
default:
}
```

Track forward goroutines with `t.wg`.

**Step 7: Fix `forward()` (line 183)**

Read client under lock:
```go
t.mu.Lock()
client := t.client
t.mu.Unlock()
if client == nil { return }
```

**Step 8: Fix `Close()` (line 213)**

Signal stop, close listener, wait for goroutines, close client under lock, nil out client.

**Step 9: Run tests with race detector**
```bash
go test ./ssh/ -v -race
```

**Step 10: Commit and merge**
```bash
git add ssh/
git commit -m "fix(ssh): fix data race, add shutdown signal, protect credentials"
git checkout main && git merge fix/ssh-security && git push origin main
git branch -d fix/ssh-security
```

---

### Task 12: argo — Fix mustParseQuantity panic and BuildWithEntrypoint mutation (C8, H24)

**Files:**
- Modify: `argo/builder/template/script.go:321-346`
- Modify: `argo/builder/builder.go:524`
- Test: `argo/builder/template/script_test.go`, `argo/builder/builder_test.go`

**Step 1: Create branch**
```bash
git checkout -b fix/argo-safety main
```

**Step 2: Write failing tests**
```go
func TestScript_InvalidCPUQuantity(t *testing.T) {
	script := NewScript("test", "python:3.9").CPU("not-valid-cpu")
	_, err := script.Templates()
	assert.Error(t, err)
}

func TestBuildWithEntrypoint_Idempotent(t *testing.T) {
	builder := New("test-workflow")
	builder.AddExitHandler(someExitStep)
	wf1, _ := builder.BuildWithEntrypoint("entry1")
	wf2, _ := builder.BuildWithEntrypoint("entry2")
	assert.Equal(t, len(wf1.Spec.Templates), len(wf2.Spec.Templates))
}
```

**Step 3: Replace `mustParseQuantity` with `resource.ParseQuantity` (`script.go:344`)**

Delete `mustParseQuantity`. Update `buildResourceRequirements` to return `(corev1.ResourceRequirements, error)`. Propagate error through `Templates()`.

**Step 4: Fix `BuildWithEntrypoint` (`builder.go:524`)**
```go
templates := make([]v1alpha1.Template, len(b.templates), len(b.templates)+1)
copy(templates, b.templates)
templates = append(templates, exitHandler)
```

**Step 5: Run tests + lint**
```bash
go test ./argo/... -v && task lint
```

**Step 6: Commit and merge**
```bash
git add argo/
git commit -m "fix(argo): replace panicking mustParseQuantity and fix BuildWithEntrypoint mutation"
git checkout main && git merge fix/argo-safety && git push origin main
git branch -d fix/argo-safety
```

---

### Task 13: Final verification

**Step 1: Run full test suite**
```bash
task test
```

**Step 2: Run linter**
```bash
task lint
```

**Step 3: Verify git log**
```bash
git log --oneline -12
```
Expected: 10 fix commits on main.
