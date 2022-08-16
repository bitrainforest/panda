<h1 align="center">Panda</h1>

## What's Panda?

Panda is a high performance agent build with high security in mind, the functionality is download the sectors from Pandarua Platform.

## Getting start

You need to make sure `go` is in your PATH. The version must `>= 1.18.0`.

```shell
# Clone the repo
git clone https://github.com/bitrainforest/PandaAgent.git

# Build the agent
cd PandaAgent && make build
```

The executable file is in `./output/bin`.

## Usage

You can use `Panda help` to see all available commands.

```shell
Panda -help
NAME:
   github.com/pandarua-agent - A new cli application
USAGE:
   github.com/pandarua-agent [global options] command [command options] [arguments...]
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
Panda --conf-dir [your config path] --log-dir [your log path] -run
```

## Config Example

```shell
Product: ## This key should be consistent with '--env', default is 'Testing'
 Transformer:
   MaxDownloader: 5  ## Max downloader exist in agent
   MaxDownloadRetry: 3 ## Max retries for one download task
   TransformPartSize: 5242880 ## The single part size when download large file
   SingleDownloadMaxWorkers: 5  ## Parallel workers in multipart download
 Miner:
   MinerSealedPath: "/your/sealed/path"
   MinerSealedCachePath: "/your/cache/path"
   APIToken: "yourMinerAPIToken"
   ID: "youreMinerID"
   StorageID: "yourSealedStorageID"
 Log:
   Level: "Debug"
   Dir: "/your/log/path"
 PandaRemote:
   Address: "Address"
   QueryURL: "QueryURL"
   CallBack: "CallBack"
   DownloadURL: "DownloadURL"
   Timeout: 5
   HeartURL: "HeartURL"
   CheckFrequency: "10s"
   HeartFrequency: "5s"
   Token: "PandaToken"
```
