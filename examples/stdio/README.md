# HOWTO

Build the server and the client

In an ideal world we could just do this
```shell
client | server
```

But we don't live in an ideal world. In the example above, pipe (`|`) doesnt do biderectional communication.
But we can use something like [`socat`](https://linux.die.net/man/1/socat): on macOS you can install it with brew.

Then, in _two separate shells_:

Server:
```
socat -d -d UNIX-LISTEN:/tmp/mcp.sock,fork EXEC:"./server/server"

2025/01/01 23:14:18 socat[60395] N listening on LEN=15 AF=1 "/tmp/mcp.sock"
2025/01/01 23:14:21 socat[60395] N accepting connection from LEN=16 AF=1 "" on LEN=15 AF=1 "/tmp/mcp.sock"
2025/01/01 23:14:21 socat[60395] N forked off child process 60398
2025/01/01 23:14:21 socat[60395] N listening on LEN=15 AF=1 "/tmp/mcp.sock"
2025/01/01 23:14:21 socat[60398] N forking off child, using socket for reading and writing
2025/01/01 23:14:21 socat[60398] N forked off child process 60399
2025/01/01 23:14:21 socat[60399] N execvp'ing "./server/server"
2025/01/01 23:14:21 socat[60398] N forked off child process 60399
2025/01/01 23:14:21 socat[60398] N starting data transfer loop with FDs [6,6] and [5,5]
2025/01/01 23:14:21 socat[60398] N write(5, 0x154010000, 192) completed
server: 2025/01/01 23:14:21 starting up...
server: 2025/01/01 23:14:21 server connected and ready - waiting for requests
{"id":1,"jsonrpc":"2.0","method":"initialize","params":{"capabilities":{"roots":{}},"clientInfo":{"name":"githuh.com/milosgajdos/go-mcp","version":"v1.alpha"},"protocolVersion":"2024-11-05"}}
2025/01/01 23:14:21 socat[60398] N write(6, 0x154010000, 198) completed
{"jsonrpc":"2.0","method":"notifications/initialized"}
{"id":2,"jsonrpc":"2.0","method":"ping"}
2025/01/01 23:14:21 socat[60398] N write(5, 0x154010000, 96) completed
2025/01/01 23:14:21 socat[60398] N write(6, 0x154010000, 37) completed
{"id":3,"jsonrpc":"2.0","method":"tools/list"}
request handler not found
2025/01/01 23:14:21 socat[60398] N write(5, 0x154010000, 47) completed
2025/01/01 23:14:21 socat[60398] N write(6, 0x154010000, 78) completed
2025/01/01 23:14:22 socat[60398] N socket 1 (fd 6) is at EOF
server: 2025/01/01 23:14:23 shutting down...
2025/01/01 23:14:23 socat[60398] N exiting with status 0
2025/01/01 23:14:23 socat[60395] N childdied(): handling signal 20
^C2025/01/01 23:14:53 socat[60395] N socat_signal(): handling signal 2
2025/01/01 23:14:53 socat[60395] N exiting on signal 2
2025/01/01 23:14:53 socat[60395] N socat_signal(): finishing signal 2
2025/01/01 23:14:53 socat[60395] N exit(130)
```

Client
```
$ socat -d -d EXEC:"./client/client" UNIX-CONNECT:/tmp/mcp.sock

2025/01/01 23:14:21 socat[60396] N forking off child, using socket for reading and writing
2025/01/01 23:14:21 socat[60396] N forked off child process 60397
2025/01/01 23:14:21 socat[60396] N forked off child process 60397
2025/01/01 23:14:21 socat[60396] N opening connection to LEN=15 AF=1 "/tmp/mcp.sock"
2025/01/01 23:14:21 socat[60396] N successfully connected from local address LEN=16 AF=1 ""
2025/01/01 23:14:21 socat[60396] N starting data transfer loop with FDs [5,5] and [6,6]
2025/01/01 23:14:21 socat[60397] N execvp'ing "./client/client"
client: 2025/01/01 23:14:21 starting up...
2025/01/01 23:14:21 socat[60396] N write(6, 0x14b814000, 192) completed
2025/01/01 23:14:21 socat[60396] N write(5, 0x14b814000, 198) completed
{"result":{"capabilities":{"prompts":{},"resources":{},"tools":{}},"protocolVersion":"2024-11-05","serverInfo":{"name":"githuh.com/milosgajdos/go-mcp","version":"v1.alpha"}},"id":1,"jsonrpc":"2.0"}
client: 2025/01/01 23:14:21 connected to server
client: 2025/01/01 23:14:21 sending ping request...
2025/01/01 23:14:21 socat[60396] N write(6, 0x14b814000, 96) completed
2025/01/01 23:14:21 socat[60396] N write(5, 0x14b814000, 37) completed
{"result":{},"id":2,"jsonrpc":"2.0"}
client: 2025/01/01 23:14:21 received ping response
client: 2025/01/01 23:14:21 requesting tools...
2025/01/01 23:14:21 socat[60396] N write(6, 0x14b814000, 47) completed
2025/01/01 23:14:21 socat[60396] N write(5, 0x14b814000, 78) completed
{"id":3,"jsonrpc":"2.0","error":{"code":-32601,"message":"Method not found"}}
client: 2025/01/01 23:14:21 list tools error: ID: {3}, Error: error -32601: Method not found
client: 2025/01/01 23:14:21 waiting for messages to be processed...
client: 2025/01/01 23:14:22 client finished
2025/01/01 23:14:22 socat[60396] N socket 1 (fd 5) is at EOF
2025/01/01 23:14:22 socat[60396] N childdied(): handling signal 20
2025/01/01 23:14:23 socat[60396] N exiting with status 0
```
