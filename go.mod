module github.com/luxfi/geth

go 1.26.4

require (
	github.com/Azure/azure-sdk-for-go/sdk/storage/azblob v1.6.4
	github.com/Microsoft/go-winio v0.6.2
	github.com/aws/aws-sdk-go-v2 v1.41.5
	github.com/aws/aws-sdk-go-v2/config v1.32.13
	github.com/aws/aws-sdk-go-v2/credentials v1.19.13
	github.com/aws/aws-sdk-go-v2/service/route53 v1.62.5
	github.com/cespare/cp v0.1.0
	github.com/cloudflare/cloudflare-go v0.116.0
	github.com/cockroachdb/pebble v1.1.5
	github.com/consensys/gnark-crypto v0.20.1
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc
	github.com/dchest/siphash v1.2.3
	github.com/deckarep/golang-set/v2 v2.8.0
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.1
	github.com/donovanhide/eventsource v0.0.0-20210830082556-c59027999da0
	github.com/dop251/goja v0.0.0-20260311135729-065cd970411c
	github.com/fatih/color v1.19.0
	github.com/ferranbt/fastssz v1.0.0
	github.com/fsnotify/fsnotify v1.9.0
	github.com/gballet/go-libpcsclite v0.0.0-20250918194357-1ec6f2e601c6
	github.com/gofrs/flock v0.13.0
	github.com/golang-jwt/jwt/v4 v4.5.2
	github.com/golang/snappy v1.0.0
	github.com/google/gofuzz v1.2.0
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.4-0.20250319132907-e064f32e3674
	github.com/graph-gophers/graphql-go v1.9.0
	github.com/hashicorp/go-bexpr v0.1.16
	github.com/holiman/billy v0.0.0-20250707135307-f2f9b9aae7db
	github.com/holiman/bloomfilter/v2 v2.0.3
	github.com/holiman/uint256 v1.3.2
	github.com/huin/goupnp v1.3.0
	github.com/influxdata/influxdb-client-go/v2 v2.14.0
	github.com/influxdata/influxdb1-client v0.0.0-20220302092344-a9ab5670611c
	github.com/jackpal/go-nat-pmp v1.0.2
	github.com/jedisct1/go-minisign v0.0.0-20241212093149-d2f9f49435c7
	github.com/kylelemons/godebug v1.1.0
	github.com/luxfi/cache v1.3.1
	github.com/luxfi/consensus v1.36.81
	github.com/luxfi/crypto v1.20.2
	github.com/luxfi/database v1.21.5
	github.com/luxfi/hid v0.9.3
	github.com/luxfi/ids v1.3.2
	github.com/luxfi/log v1.4.3
	github.com/luxfi/math/big v0.1.0
	github.com/luxfi/pq v1.1.0
	github.com/luxfi/version v1.0.1
	github.com/luxfi/zapdb v1.10.6
	github.com/mattn/go-colorable v0.1.15
	github.com/mattn/go-isatty v0.0.22
	github.com/naoina/toml v0.1.2-0.20170918210437-9fafd6967416
	github.com/peterh/liner v1.2.2
	github.com/pion/stun/v2 v2.0.0
	github.com/protolambda/bls12-381-util v0.1.0
	github.com/protolambda/zrnt v0.34.1
	github.com/protolambda/ztyp v0.2.2
	github.com/rs/cors v1.11.1
	github.com/shirou/gopsutil v3.21.11+incompatible
	github.com/status-im/keycard-go v0.3.3
	github.com/stretchr/testify v1.11.1
	github.com/supranational/blst v0.3.16
	github.com/syndtr/goleveldb v1.0.1-0.20220721030215-126854af5e6d
	github.com/urfave/cli/v2 v2.27.7
	go.uber.org/automaxprocs v1.6.0
	go.uber.org/goleak v1.3.0
	golang.org/x/crypto v0.52.0
	golang.org/x/sync v0.20.0
	golang.org/x/sys v0.45.0
	golang.org/x/text v0.37.0
	golang.org/x/time v0.15.0
	golang.org/x/tools v0.45.0
	google.golang.org/protobuf v1.36.11
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	filippo.io/hpke v0.4.0 // indirect
	github.com/ALTree/bigfloat v0.2.0 // indirect
	github.com/Masterminds/semver/v3 v3.4.0 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.8 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.22 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.21 // indirect
	github.com/aws/aws-sdk-go-v2/service/s3 v1.97.3 // indirect
	github.com/cenkalti/backoff v2.2.1+incompatible // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/cockroachdb/errors v1.12.0 // indirect
	github.com/ethereum/go-bigmodexpfix v0.0.0-20250911101455-f9e208c548ab // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/gorilla/rpc v1.2.1 // indirect
	github.com/grandcat/zeroconf v1.0.0 // indirect
	github.com/hanzoai/vfs v0.4.3 // indirect
	github.com/hanzos3/crc64nvme v1.1.2 // indirect
	github.com/hanzos3/go v1.0.2 // indirect
	github.com/hanzos3/md5-simd v1.1.3 // indirect
	github.com/klauspost/crc32 v1.3.0 // indirect
	github.com/luxfi/accel v1.2.4 // indirect
	github.com/luxfi/age v1.6.0 // indirect
	github.com/luxfi/atomic v1.0.0 // indirect
	github.com/luxfi/compress v0.1.1 // indirect
	github.com/luxfi/concurrent v0.1.1 // indirect
	github.com/luxfi/constants v1.6.2 // indirect
	github.com/luxfi/container v0.2.2 // indirect
	github.com/luxfi/corona v0.10.4 // indirect
	github.com/luxfi/crypto/ipa v1.2.4 // indirect
	github.com/luxfi/lattice/v7 v7.1.4 // indirect
	github.com/luxfi/mdns v0.1.1 // indirect
	github.com/luxfi/mlwe v0.3.0 // indirect
	github.com/luxfi/mock v0.1.1 // indirect
	github.com/luxfi/runtime v1.3.1 // indirect
	github.com/luxfi/utils v1.3.1 // indirect
	github.com/luxfi/validators v1.3.1 // indirect
	github.com/luxfi/zap v1.2.6 // indirect
	github.com/miekg/dns v1.1.72 // indirect
	github.com/montanaflynn/stats v0.9.0 // indirect
	github.com/philhofer/fwd v1.2.0 // indirect
	github.com/rs/xid v1.6.0 // indirect
	github.com/tinylib/msgp v1.6.4 // indirect
	github.com/wlynxg/anet v0.0.5 // indirect
	github.com/zeebo/blake3 v0.2.4 // indirect
	go.uber.org/mock v0.6.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/exp v0.0.0-20260529124908-c761662dc8c9 // indirect
)

require (
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.21.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.12.0 // indirect
	github.com/DataDog/zstd v1.5.7 // indirect
	github.com/apapsch/go-jsonmerge/v2 v2.0.0 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.21 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.21 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.21 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.21 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.0.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.30.14 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.18 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.41.10 // indirect
	github.com/aws/smithy-go v1.24.2 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bits-and-blooms/bitset v1.24.4 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cloudflare/circl v1.6.3 // indirect
	github.com/cockroachdb/fifo v0.0.0-20240816210425-c5d0cb0b6fc0 // indirect
	github.com/cockroachdb/logtags v0.0.0-20241215232642-bb51bb14a506 // indirect
	github.com/cockroachdb/redact v1.1.8 // indirect
	github.com/cockroachdb/tokenbucket v0.0.0-20250429170803-42689b6311bb // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.7 // indirect
	github.com/crate-crypto/go-eth-kzg v1.5.0 // indirect
	github.com/dgraph-io/ristretto/v2 v2.4.0 // indirect
	github.com/dlclark/regexp2 v1.11.5 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/emicklei/dot v1.11.0 // indirect
	github.com/ethereum/c-kzg-4844/v2 v2.1.7 // indirect
	github.com/fjl/gencodec v0.1.1 // indirect
	github.com/garslo/gogen v0.0.0-20170306192744-1d203ffc1f61 // indirect
	github.com/getsentry/sentry-go v0.44.1 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/go-sourcemap/sourcemap v2.1.4+incompatible // indirect
	github.com/goccy/go-json v0.10.6 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/flatbuffers v25.12.19+incompatible // indirect
	github.com/google/go-querystring v1.2.0 // indirect
	github.com/google/pprof v0.0.0-20260302011040-a15ffb7f9dcc // indirect
	github.com/influxdata/line-protocol v0.0.0-20210922203350-b1ad95c89adf // indirect
	github.com/kilic/bls12-381 v0.1.0 // indirect
	github.com/klauspost/compress v1.18.6 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/luxfi/math v1.5.1
	github.com/luxfi/metric v1.8.1 // indirect
	github.com/luxfi/p2p v1.22.1 // indirect
	github.com/luxfi/precompile v0.19.3
	github.com/luxfi/sampler v1.1.0 // indirect
	github.com/luxfi/vm v1.3.16
	github.com/luxfi/warp v1.24.1 // indirect
	github.com/mattn/go-runewidth v0.0.21 // indirect
	github.com/minio/sha256-simd v1.0.1 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/pointerstructure v1.2.1 // indirect
	github.com/mr-tron/base58 v1.3.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/naoina/go-stringutil v0.1.0 // indirect
	github.com/oapi-codegen/runtime v1.3.1 // indirect
	github.com/pion/dtls/v2 v2.2.12 // indirect
	github.com/pion/logging v0.2.4 // indirect
	github.com/pion/transport/v2 v2.2.10 // indirect
	github.com/pion/transport/v3 v3.1.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_golang v1.23.2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.67.5 // indirect
	github.com/prometheus/procfs v0.20.1 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/tklauser/go-sysconf v0.4.0 // indirect
	github.com/tklauser/numcpus v0.12.0 // indirect
	github.com/xrash/smetrics v0.0.0-20250705151800-55b8f293f342 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.44.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	go.yaml.in/yaml/v2 v2.4.4 // indirect
	golang.org/x/mod v0.36.0 // indirect
	golang.org/x/net v0.55.0 // indirect
	gonum.org/v1/gonum v0.17.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

tool (
	github.com/fjl/gencodec
	golang.org/x/tools/cmd/stringer
	google.golang.org/protobuf/cmd/protoc-gen-go
)

// Exclude old monolithic genproto to avoid conflicts
exclude google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1

// Unify HID libraries - karalabe/hid and luxfi/hid have same CGO code
replace github.com/karalabe/hid => github.com/luxfi/hid v0.9.2
