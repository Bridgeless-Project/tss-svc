module github.com/Bridgeless-Project/tss-svc

go 1.23.1

replace (
	// tss-lib fix ed25519
	github.com/agl/ed25519 => github.com/bnb-chain/edwards25519 v0.0.0-20200305024217-f36fc4b53d43
	// rosetta rebranding
	github.com/coinbase/rosetta-sdk-go => github.com/coinbase/mesh-sdk-go v0.7.9
	// bridge core sdk
	github.com/cosmos/cosmos-sdk => github.com/Bridgeless-Project/cosmos-sdk v0.46.33
	github.com/cosmos/iavl => github.com/cosmos/iavl v0.19.6
	// bridge cosmos sdk protobuf fix
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
	github.com/hyle-team/bridgeless-core/v12 => github.com/Bridgeless-Project/bridgeless-core/v12 v12.1.19
	// bridge cosmos sdk comebft change
	github.com/tendermint/tendermint => github.com/cometbft/cometbft v0.34.29
	github.com/tidwall/btree => github.com/tidwall/btree v1.5.0
	// deleted library fix that is used in other libraries
	github.com/tyler-smith/go-bip39 => github.com/distributed-lab/go-bip39 v1.1.0
)

require (
	cosmossdk.io/math v1.0.1
	github.com/Masterminds/squirrel v1.5.4
	github.com/bnb-chain/tss-lib/v2 v2.0.2
	github.com/btcsuite/btcd v0.24.2
	github.com/btcsuite/btcd/btcec/v2 v2.3.2
	github.com/btcsuite/btcd/btcutil v1.1.5
	github.com/btcsuite/btcd/chaincfg/chainhash v1.1.0
	github.com/cosmos/cosmos-sdk v0.46.13
	github.com/ethereum/go-ethereum v1.15.5
	github.com/go-chi/chi/v5 v5.0.7
	github.com/go-ozzo/ozzo-validation/v4 v4.2.1
	github.com/gogo/protobuf v1.3.3
	github.com/gorilla/websocket v1.5.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.25.1
	github.com/hashicorp/vault/api v1.15.0
	github.com/hyle-team/bridgeless-core/v12 v12.1.16-rc1
	github.com/ignite/cli v0.26.1
	github.com/pkg/errors v0.9.1
	github.com/rubenv/sql-migrate v1.7.0
	github.com/spf13/cobra v1.8.1
	github.com/stretchr/testify v1.10.0
	github.com/tendermint/tendermint v0.34.28
	github.com/xssnick/tonutils-go v1.13.0
	gitlab.com/distributed_lab/ape v1.7.2
	gitlab.com/distributed_lab/figure/v3 v3.1.4
	gitlab.com/distributed_lab/kit v1.11.4
	gitlab.com/distributed_lab/logan v3.8.1+incompatible
	go.uber.org/atomic v1.10.0
	golang.org/x/sync v0.14.0
	google.golang.org/genproto/googleapis/api v0.0.0-20241219192143-6b3ec007d9bb
	google.golang.org/grpc v1.69.2
	google.golang.org/protobuf v1.36.0
)

require (
	cosmossdk.io/errors v1.0.0-beta.7 // indirect
	filippo.io/edwards25519 v1.0.0 // indirect
	github.com/99designs/go-keychain v0.0.0-20191008050251-8e49817e8af4 // indirect
	github.com/99designs/keyring v1.2.1 // indirect
	github.com/ChainSafe/go-schnorrkel v1.0.0 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/StackExchange/wmi v1.2.1 // indirect
	github.com/agl/ed25519 v0.0.0-20200225211852-fd4d107ace12 // indirect
	github.com/armon/go-metrics v0.4.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bgentry/speakeasy v0.1.1-0.20220910012023-760eaf8b6816 // indirect
	github.com/bits-and-blooms/bitset v1.17.0 // indirect
	github.com/btcsuite/btclog v0.0.0-20170628155309-84c8d2346e9f // indirect
	github.com/btcsuite/btcutil v1.0.3-0.20201208143702-a53e38424cce // indirect
	github.com/btcsuite/go-socks v0.0.0-20170105172521-4720035b7bfd // indirect
	github.com/btcsuite/websocket v0.0.0-20150119174127-31079b680792 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/certifi/gocertifi v0.0.0-20200211180108-c7c1fbc02894 // indirect
	github.com/cespare/xxhash v1.1.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/confio/ics23/go v0.9.0 // indirect
	github.com/consensys/bavard v0.1.22 // indirect
	github.com/consensys/gnark-crypto v0.14.0 // indirect
	github.com/cosmos/btcutil v1.0.5 // indirect
	github.com/cosmos/cosmos-proto v1.0.0-beta.3 // indirect
	github.com/cosmos/go-bip39 v1.0.0 // indirect
	github.com/cosmos/gorocksdb v1.2.0 // indirect
	github.com/cosmos/iavl v0.20.0 // indirect
	github.com/cosmos/ledger-cosmos-go v0.12.2 // indirect
	github.com/crate-crypto/go-ipa v0.0.0-20240724233137-53bbb0ceb27a // indirect
	github.com/crate-crypto/go-kzg-4844 v1.1.0 // indirect
	github.com/danieljoos/wincred v1.1.2 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/deckarep/golang-set/v2 v2.6.0 // indirect
	github.com/decred/dcrd/crypto/blake256 v1.0.1 // indirect
	github.com/decred/dcrd/dcrec/edwards/v2 v2.0.3 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.2.0 // indirect
	github.com/dgraph-io/badger/v2 v2.2007.4 // indirect
	github.com/dgraph-io/ristretto v0.1.1 // indirect
	github.com/dgryski/go-farm v0.0.0-20200201041132-a6ae2369ad13 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/dvsekhvalnov/jose2go v1.5.0 // indirect
	github.com/ethereum/c-kzg-4844 v1.0.0 // indirect
	github.com/ethereum/go-verkle v0.2.2 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/getsentry/raven-go v0.2.0 // indirect
	github.com/getsentry/sentry-go v0.27.0 // indirect
	github.com/go-chi/chi v4.1.2+incompatible // indirect
	github.com/go-gorp/gorp/v3 v3.1.0 // indirect
	github.com/go-jose/go-jose/v4 v4.0.1 // indirect
	github.com/go-kit/kit v0.12.0 // indirect
	github.com/go-kit/log v0.2.1 // indirect
	github.com/go-logfmt/logfmt v0.6.0 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/godbus/dbus v0.0.0-20190726142602-4481cbc300e2 // indirect
	github.com/golang/glog v1.2.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/golang/snappy v0.0.5-0.20220116011046-fa5810519dcb // indirect
	github.com/google/btree v1.1.2 // indirect
	github.com/google/jsonapi v0.0.0-20200226002910-c8283f632fb7 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/gsterjov/go-libsecret v0.0.0-20161001094733-a6f4afe4910c // indirect
	github.com/gtank/merlin v0.1.1 // indirect
	github.com/gtank/ristretto255 v0.1.2 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.7 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-secure-stdlib/parseutil v0.1.6 // indirect
	github.com/hashicorp/go-secure-stdlib/strutil v0.1.2 // indirect
	github.com/hashicorp/go-sockaddr v1.0.2 // indirect
	github.com/hashicorp/golang-lru v0.5.5-0.20210104140557-80c98217689d // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/hdevalence/ed25519consensus v0.1.0 // indirect
	github.com/holiman/uint256 v1.3.2 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/ipfs/go-log v1.0.5 // indirect
	github.com/ipfs/go-log/v2 v2.1.3 // indirect
	github.com/jmhodges/levigo v1.0.0 // indirect
	github.com/jmoiron/sqlx v1.3.5 // indirect
	github.com/klauspost/compress v1.17.0 // indirect
	github.com/lann/builder v0.0.0-20180802200727-47ae307949d0 // indirect
	github.com/lann/ps v0.0.0-20150810152359-62de8c46ede0 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/libp2p/go-buffer-pool v0.1.0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/mimoo/StrobeGo v0.0.0-20210601165009-122bf33a46e0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mmcloughlin/addchain v0.4.0 // indirect
	github.com/mtibben/percent v0.2.1 // indirect
	github.com/oasisprotocol/curve25519-voi v0.0.0-20220328075252-7dd334e3daae // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/otiai10/primes v0.0.0-20210501021515-f1b2be525a11 // indirect
	github.com/pelletier/go-toml/v2 v2.1.0 // indirect
	github.com/petermattis/goid v0.0.0-20230317030725-371a4b8eda08 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_golang v1.14.0 // indirect
	github.com/prometheus/client_model v0.4.0 // indirect
	github.com/prometheus/common v0.42.0 // indirect
	github.com/prometheus/procfs v0.9.0 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475 // indirect
	github.com/regen-network/cosmos-proto v0.3.1 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/sagikazarmark/locafero v0.4.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/sasha-s/go-deadlock v0.3.1 // indirect
	github.com/shirou/gopsutil v3.21.4-0.20210419000835-c7a38de76ee5+incompatible // indirect
	github.com/sigurn/crc16 v0.0.0-20211026045750-20ab5afb07e3 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spf13/viper v1.18.2 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/supranational/blst v0.3.14 // indirect
	github.com/syndtr/goleveldb v1.0.1-0.20220721030215-126854af5e6d // indirect
	github.com/tendermint/go-amino v0.16.0 // indirect
	github.com/tendermint/tm-db v0.6.7 // indirect
	github.com/tidwall/btree v1.6.0 // indirect
	github.com/tidwall/gjson v1.14.4 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	github.com/tklauser/go-sysconf v0.3.12 // indirect
	github.com/tklauser/numcpus v0.6.1 // indirect
	github.com/zondax/hid v0.9.1 // indirect
	github.com/zondax/ledger-go v0.14.1 // indirect
	gitlab.com/distributed_lab/figure v2.1.2+incompatible // indirect
	gitlab.com/distributed_lab/lorem v0.2.0 // indirect
	gitlab.com/distributed_lab/running v1.6.0 // indirect
	go.etcd.io/bbolt v1.3.7 // indirect
	go.uber.org/multierr v1.9.0 // indirect
	go.uber.org/zap v1.23.0 // indirect
	golang.org/x/crypto v0.38.0 // indirect
	golang.org/x/exp v0.0.0-20231006140011-7918f672742d // indirect
	golang.org/x/net v0.34.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/term v0.32.0 // indirect
	golang.org/x/text v0.25.0 // indirect
	golang.org/x/time v0.9.0 // indirect
	google.golang.org/genproto v0.0.0-20240213162025-012b6fc9bca9 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241219192143-6b3ec007d9bb // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	rsc.io/tmplfunc v0.0.3 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)
