module gitlab.com/crankykernel/maker/go

require (
	github.com/crankykernel/binanceapi-go v0.0.0-20190307162445-c030c1a8c110
	github.com/gobuffalo/packr v1.22.0
	github.com/gorilla/context v1.1.1 // indirect
	github.com/gorilla/mux v1.6.2
	github.com/gorilla/websocket v1.4.0
	github.com/inconshreveable/mousetrap v1.0.0
	github.com/mattn/go-sqlite3 v1.9.0
	github.com/oklog/ulid v0.3.0
	github.com/sirupsen/logrus v1.3.0
	github.com/spf13/afero v1.2.1 // indirect
	github.com/spf13/cobra v0.0.3
	github.com/spf13/viper v1.3.1
	github.com/stretchr/testify v1.3.0
	golang.org/x/crypto v0.0.0-20190131182504-b8fe1690c613
	golang.org/x/sys v0.0.0-20190204203706-41f3e6584952 // indirect
	gopkg.in/yaml.v2 v2.2.2
)

//replace github.com/crankykernel/binanceapi-go => ../../binanceapi-go
