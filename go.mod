module github.com/jasoet/pkg/v2

go 1.24.7

require (
	github.com/argoproj/argo-workflows/v3 v3.7.2
	github.com/docker/docker v28.4.0+incompatible
	github.com/docker/go-connections v0.6.0
	github.com/go-playground/validator/v10 v10.27.0
	github.com/go-resty/resty/v2 v2.16.5
	github.com/golang-migrate/migrate/v4 v4.19.0
	github.com/google/uuid v1.6.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.2
	github.com/labstack/echo/v4 v4.13.4
	github.com/lib/pq v1.10.9
	github.com/prometheus/client_golang v1.23.2
	github.com/rs/zerolog v1.34.0
	github.com/spf13/viper v1.21.0
	github.com/stretchr/testify v1.11.1
	github.com/testcontainers/testcontainers-go v0.39.0
	github.com/testcontainers/testcontainers-go/modules/mssql v0.39.0
	github.com/testcontainers/testcontainers-go/modules/mysql v0.39.0
	github.com/testcontainers/testcontainers-go/modules/postgres v0.39.0
	github.com/uber-go/tally/v4 v4.1.17
	go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho v0.63.0
	go.opentelemetry.io/otel v1.38.0
	go.opentelemetry.io/otel/log v0.14.0
	go.opentelemetry.io/otel/metric v1.38.0
	go.opentelemetry.io/otel/sdk v1.38.0
	go.opentelemetry.io/otel/sdk/log v0.14.0
	go.opentelemetry.io/otel/sdk/metric v1.38.0
	go.opentelemetry.io/otel/trace v1.38.0
	go.temporal.io/api v1.53.0
	go.temporal.io/sdk v1.36.0
	go.temporal.io/sdk/contrib/tally v0.2.0
	golang.org/x/crypto v0.42.0
	golang.org/x/net v0.44.0
	golang.org/x/time v0.13.0
	google.golang.org/grpc v1.75.1
	google.golang.org/protobuf v1.36.9
	gopkg.in/yaml.v3 v3.0.1
	gorm.io/driver/mysql v1.6.0
	gorm.io/driver/postgres v1.6.0
	gorm.io/driver/sqlserver v1.6.1
	gorm.io/gorm v1.31.0
	k8s.io/api v0.33.1
	k8s.io/apimachinery v0.33.1
	k8s.io/client-go v0.33.1
)

require (
	dario.cat/mergo v1.0.2 // indirect
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20250102033503-faa5f7b0171c // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.3.1 // indirect
	github.com/Masterminds/sprig/v3 v3.3.0 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/argoproj/argo-events v1.9.6 // indirect
	github.com/argoproj/pkg v0.13.7-0.20250123033407-65f2d4777bfd // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cenkalti/backoff/v5 v5.0.2 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/colinmarc/hdfs/v2 v2.4.0 // indirect
	github.com/containerd/errdefs v1.0.0 // indirect
	github.com/containerd/errdefs/pkg v0.3.0 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/containerd/platforms v0.2.1 // indirect
	github.com/coreos/go-oidc/v3 v3.14.1 // indirect
	github.com/cpuguy83/dockercfg v0.3.2 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/doublerebel/bellows v0.0.0-20160303004610-f177d92a03d3 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/ebitengine/purego v0.9.0 // indirect
	github.com/emicklei/go-restful/v3 v3.12.2 // indirect
	github.com/evilmonkeyinc/jsonpath v0.8.1 // indirect
	github.com/expr-lang/expr v1.17.5 // indirect
	github.com/facebookgo/clock v0.0.0-20150410010913-600d898af40a // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/fxamacker/cbor/v2 v2.8.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.10 // indirect
	github.com/go-jose/go-jose/v3 v3.0.4 // indirect
	github.com/go-jose/go-jose/v4 v4.1.1 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/go-openapi/jsonpointer v0.21.1 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.1 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-sql-driver/mysql v1.9.3 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-sql/civil v0.0.0-20220223132316-b832511892a9 // indirect
	github.com/golang-sql/sqlexp v0.1.0 // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/gnostic-models v0.6.9 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/gorilla/websocket v1.5.4-0.20250319132907-e064f32e3674 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.3.2 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/huandu/xstrings v1.5.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jackc/pgio v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgtype v1.14.4 // indirect
	github.com/jackc/pgx/v5 v5.7.6 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jcmturner/aescts/v2 v2.0.0 // indirect
	github.com/jcmturner/dnsutils/v2 v2.0.0 // indirect
	github.com/jcmturner/gofork v1.7.6 // indirect
	github.com/jcmturner/goidentity/v6 v6.0.1 // indirect
	github.com/jcmturner/gokrb5/v8 v8.4.4 // indirect
	github.com/jcmturner/rpc/v2 v2.0.3 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/klauspost/pgzip v1.2.6 // indirect
	github.com/labstack/gommon v0.4.2 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20250827001030-24949be3fa54 // indirect
	github.com/magiconair/properties v1.8.10 // indirect
	github.com/mailru/easyjson v0.9.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-sqlite3 v1.14.28 // indirect
	github.com/microsoft/go-mssqldb v1.9.3 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/go-archive v0.1.0 // indirect
	github.com/moby/patternmatcher v0.6.0 // indirect
	github.com/moby/spdystream v0.5.0 // indirect
	github.com/moby/sys/sequential v0.6.0 // indirect
	github.com/moby/sys/user v0.4.0 // indirect
	github.com/moby/sys/userns v0.1.0 // indirect
	github.com/moby/term v0.5.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/mxk/go-flowrate v0.0.0-20140419014527-cca7078d478f // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/nexus-rpc/sdk-go v0.4.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/procfs v0.17.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/robfig/cron v1.2.0 // indirect
	github.com/robfig/cron/v3 v3.0.1 // indirect
	github.com/sagikazarmark/locafero v0.12.0 // indirect
	github.com/segmentio/fasthash v1.0.3 // indirect
	github.com/sethvargo/go-limiter v1.0.0 // indirect
	github.com/shirou/gopsutil/v4 v4.25.9 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/cobra v1.9.1 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/tklauser/go-sysconf v0.3.15 // indirect
	github.com/tklauser/numcpus v0.10.0 // indirect
	github.com/twmb/murmur3 v1.1.8 // indirect
	github.com/upper/db/v4 v4.10.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.63.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/runtime v0.61.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.36.0 // indirect
	go.opentelemetry.io/otel/exporters/prometheus v0.58.0 // indirect
	go.opentelemetry.io/proto/otlp v1.7.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.yaml.in/yaml/v2 v2.4.3 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/exp v0.0.0-20250531010427-b6e5de432a8b // indirect
	golang.org/x/oauth2 v0.30.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/sys v0.36.0 // indirect
	golang.org/x/term v0.35.0 // indirect
	golang.org/x/text v0.29.0 // indirect
	google.golang.org/genproto v0.0.0-20250603155806-513f23925822 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250929231259-57b25ae835d4 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250929231259-57b25ae835d4 // indirect
	gopkg.in/evanphx/json-patch.v4 v4.12.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
	k8s.io/kube-openapi v0.0.0-20250318190949-c8a335a9a2ff // indirect
	k8s.io/utils v0.0.0-20250502105355-0f33e8f1c979 // indirect
	modernc.org/libc v1.65.8 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
	modernc.org/sqlite v1.37.1 // indirect
	sigs.k8s.io/json v0.0.0-20241014173422-cfa47c3a1cc8 // indirect
	sigs.k8s.io/randfill v1.0.0 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.7.0 // indirect
	sigs.k8s.io/yaml v1.4.0 // indirect
	zombiezen.com/go/sqlite v1.4.2 // indirect
)
