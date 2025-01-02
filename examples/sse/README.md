# SSE Transport example


Run the server:
```
 go run ./server/...
server: 2025/01/02 15:08:40 starting up...
server: 2025/01/02 15:08:40 server connected and ready at [::]:8090 - waiting for requests
```

# Go client

Runt the client:
```
$ go run ./client/...
client: 2025/01/02 15:19:51 starting up...
client: 2025/01/02 15:19:51 connected to server
client: 2025/01/02 15:19:51 sending ping request...
client: 2025/01/02 15:19:51 received ping response
client: 2025/01/02 15:19:51 requesting tools...
client: 2025/01/02 15:19:51 list tools error: ID: {3}, Error: error -32601: Method not found
client: 2025/01/02 15:19:51 waiting for messages to be processed...
client: 2025/01/02 15:19:52 client finished
```

# cURL

Create a session in one terminal window:
```
curl -N http://localhost:8090/sse

event: endpoint
data: /message?sessionId=af2a5089-37e2-4231-aebc-5dd87a451fd0

## THIS WILL FIRE ONCE YOU'VE FETCHED THE MESSAGES IN ANOTHER TERMINAL WINDOW SHOWN BELOW
event: message
data: {"result":{},"id":1,"jsonrpc":"2.0"}
```

Fetch events for that session in another window:
```
curl -X POST -H "Content-Type: application/json" "http://localhost:8090/message?sessionId=af2a5089-37e2-4231-aebc-5dd87a451fd0" -d '{"jsonrpc":"2.0","method":"ping","id":1}'


```
