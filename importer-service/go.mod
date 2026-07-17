module github.com/minhnbnt/uptime-monitor-microservices/importer-service

go 1.26.5

tool github.com/ogen-go/ogen/cmd/ogen

require (
	github.com/go-faster/errors v0.7.1
	github.com/go-faster/jx v1.2.0
	github.com/minhnbnt/uptime-monitor-microservices/common/authclient v0.0.0-00010101000000-000000000000
	github.com/minhnbnt/uptime-monitor-microservices/common/proto v0.0.0-00010101000000-000000000000
	github.com/ogen-go/ogen v1.23.0
	github.com/samber/do/v2 v2.0.0
	github.com/samber/lo v1.53.0
	github.com/spf13/viper v1.21.0
	github.com/xuri/excelize/v2 v2.9.0
	google.golang.org/grpc v1.82.0
)

require (
	github.com/dlclark/regexp2 v1.12.0 // indirect
	github.com/fatih/color v1.19.0 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-faster/yaml v0.4.6 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.22 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/richardlehane/mscfb v1.0.4 // indirect
	github.com/richardlehane/msoleps v1.0.4 // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/samber/go-type-to-string v1.8.0 // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/xuri/efp v0.0.0-20240408161823-9ad904a10d6d // indirect
	github.com/xuri/nfp v0.0.0-20240318013403-ab9948c2c4a7 // indirect
	github.com/yuin/goldmark v1.8.2 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.28.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.54.0 // indirect
	golang.org/x/exp v0.0.0-20230725093048-515e97ebf090 // indirect
	golang.org/x/mod v0.38.0 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	golang.org/x/tools v0.48.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260414002931-afd174a4e478 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

replace github.com/minhnbnt/uptime-monitor-microservices/common/proto => ../common/proto

replace github.com/minhnbnt/uptime-monitor-microservices/common/authclient => ../common/authclient
