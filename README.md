# Introduce
what's Panda agent?  

Panda agent is a high performance agent programm with high security, which will help you download the sectors in our Sealing as a service Platform.
What you need to do is just download and run the agent, then no other action required.

# Download
 git clone https://github.com/bitrainforest/PandaAgent.git

# Build
In your project's root path, exec:  
```
make build
```
Then the panda executable file will exist in the ./output/bin directory.  

You can use Panda -help for more information.
```
./Panda -help
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
# Run
```
Panda --conf-dir /your/config --log-dir /your/log -run 
```

# Config

This is example config below:
```
Product: ## this key should be consistent with '--env', default is 'Testing'
 Transformer:
   MaxDownloader: 5  ## max downloader exist in agent
   MaxDownloadRetry: 3 ## max retry numbers for one download task
   TransformPartSize: 5242880 ## when download big file, the single part size
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