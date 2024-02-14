```go
	tmp := &Service{}
	server := NewServer()

	_ = server.Register("service", tmp)

	server.Handler([]byte(`{"jsonrpc": "2.0", "method": "service.Method2", "params": [1], "id":"1"}`))
	server.Handler([]byte(`{"jsonrpc": "2.0", "method": "servic1e.Method3", "params": [[1,2,7]], "id":"1"}`))
	server.Handler([]byte(`{"jsonrpc": "2.0", "method": "service.Method4", "params": [[1,2]], "id":"1"}`))
	server.Handler([]byte(`{"jsonrpc": "2.0", "method": "service.Method5", "params": [{"name":"maxim", "phones":[123,456]}], "id":"1"}`))

```