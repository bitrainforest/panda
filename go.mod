module pandarua-agent

go 1.17

require (
	github.com/gorilla/websocket v1.5.0
	github.com/imdario/mergo v0.3.13
	github.com/bitrainforest/PandaAgent v0.0.1
	github.com/rs/zerolog v1.27.0
	github.com/urfave/cli/v2 v2.11.1
	gopkg.in/yaml.v2 v2.4.0
)

require (
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673 // indirect
	golang.org/x/sys v0.0.0-20220520151302-bc2c85ada10a // indirect
)


replace github.com/prometheus/client_golang => github.com/prometheus/client_golang v1.13.0
