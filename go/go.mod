module gitlab.com/crankykernel/maker/go

require (
	github.com/coreos/etcd v3.3.11+incompatible // indirect
	github.com/crankykernel/binanceapi-go v0.0.0-20190224053504-f3a270d2d894
	github.com/gobuffalo/buffalo-plugins v1.13.0 // indirect
	github.com/gobuffalo/packr v1.22.0
	github.com/gorilla/context v1.1.1 // indirect
	github.com/gorilla/mux v1.6.2
	github.com/gorilla/websocket v1.4.0
	github.com/inconshreveable/mousetrap v1.0.0
	github.com/markbates/going v1.0.3 // indirect
	github.com/mattn/go-sqlite3 v1.9.0
	github.com/mitchellh/go-homedir v1.0.0 // indirect
	github.com/oklog/ulid v0.3.0
	github.com/sirupsen/logrus v1.3.0
	github.com/spf13/afero v1.2.1 // indirect
	github.com/spf13/cobra v0.0.3
	github.com/spf13/viper v1.3.1
	github.com/ugorji/go/codec v0.0.0-20190204201341-e444a5086c43 // indirect
	gitlab.com/crankykernel/cryptotrader v0.0.0-20190118193049-f5c4978e61cb
	golang.org/x/crypto v0.0.0-20190131182504-b8fe1690c613
	golang.org/x/net v0.0.0-20190125091013-d26f9f9a57f3 // indirect
	golang.org/x/sys v0.0.0-20190204203706-41f3e6584952 // indirect
	golang.org/x/tools v0.0.0-20190205181801-90c8b4f75bb8 // indirect
	gopkg.in/airbrake/gobrake.v2 v2.0.9 // indirect
	gopkg.in/gemnasium/logrus-airbrake-hook.v2 v2.1.2 // indirect
	gopkg.in/yaml.v2 v2.2.2
)

//replace gitlab.com/crankykernel/cryptotrader => ../../cryptotrader
//replace github.com/crankykernel/binanceapi-go => ../../binanceapi-go
