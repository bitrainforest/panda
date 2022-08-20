<h1 align="center">panda</h1>

## What's panda?

panda is a high performance agent build with high security in mind, the functionality is download the sectors from Pandarua Platform.

## Getting start

You need to make sure `go` is in your PATH. The version must `>= 1.18.0`.

```shell
# Clone the repo
git clone https://github.com/bitrainforest/PandaAgent.git

# Build the agent
cd PandaAgent && make build
```

The executable file is in `./output/bin` and we will copy it into `/usr/local/bin/`.

## Usage

You can use `panda help` to see all available commands.

```shell
panda -help
NAME:
   github.com/bitrainforest/PandaAgent - A new cli application
USAGE:
   github.com/bitrainforest/PandaAgent [global options] command [command options] [arguments...]
VERSION:
   No version
COMMANDS:
   run      run service
   help, h  Shows a list of commands or help for one command
GLOBAL OPTIONS:
   --conf-dir value  configuration file directory [$CONFDIR]
   --env value       enviroment, can be Develop, Product, or some other cumtomize value [$ENV]
   --help, -h        show help (default: false)
   --log-dir value   the log dir [$LOGDIR]
   --version, -v     print the version (default: false)
```

### Run the agnet

```shell
panda --conf-dir [your config path] --log-dir [your log path] --env [Default, Testing, Product...] run
```

## Config Example

```shell
Default: ## This key should be consistent with '--env', default is 'Default'
 Transformer:
   MaxDownloader: 5  ## Max downloader exist in agent
   MaxDownloadRetry: 3 ## Max retries for one download task
   TransformPartSize: 5242880 ## The single part size when download large file
   SingleDownloadMaxWorkers: 5  ## Parallel workers in multipart download
 Miner:
   MinerSealedPath: '/your/sealed/path'
   MinerSealedCachePath: '/your/cache/path'
   APIToken: 'yourMinerAPIToken'
   ID: 'txxxx'
   StorageID: 'yourSealedStorageID'
   Address: 'localhost:1234'
Log:
   Level: 'Debug'
   Dir: '/your/log/path'
Platform:
   Token: 'pandaToken'
   Timeout: 5
   CheckFrequency: '10s'
   HeartFrequency: '5s'
   QueryURL: "https://xxx.com/sector/downloadable/list"
   CallBack: "https://xxx.com/sector/downloaded/callback"
   DownloadURL: "https://xxx.com/"
   HeartURL: "https://xxx.com/agent/heartbeat"
```
