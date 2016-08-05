# ShareGo
A clipboard sharing tool, only working for linux distros.

## Usage

### Start the tool
```
sharego start
```
Ideally run as a background process.

### Get contents of clipshare buffer (received from other peers)
```
sharego get
```

### Send clipboard contents to other peers
```
sharego set 127.0.0.1 10.0.x.x
```

### Stop clipshare
```
sharego stop
```


###Todo
* Queued Clipshare buffer
* Daemon mode
* Security
* Refactor
