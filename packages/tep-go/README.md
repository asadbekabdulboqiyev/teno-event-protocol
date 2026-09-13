# tep-go

Go implementation of TEP (Teno Event Protocol).
For integration with **stress-strike** (Go load-tester) and standard use.
HTTP/REST transport binding: `spec/transport-http.md`.

## Package

`github.com/asadbekabdulboqiyev/teno-event-protocol/packages/tep-go/tep`

## Install

```bash
go test ./...        # run with -race too
go vet ./...
```

## Producer

```go
import "github.com/asadbekabdulboqiyev/teno-event-protocol/packages/tep-go/tep"

client, _ := tep.NewClient(tep.ClientOptions{
    URL:    "https://consumer.example.com/v1/events/push",
    Secret: []byte(os.Getenv("TEP_SECRET")),
    Key:    "nescom-key",
    Source: "nescom",
})
env := tep.NewEnvelope("loadtest.finished", "stress-strike", map[string]any{"rps": 1200})
res, err := client.Push(context.Background(), env)
```

## Consumer (net/http)

```go
consumer := tep.NewConsumer(secret, func(e *tep.Envelope) (tep.TepCode, error) {
    log.Printf("handling %s", e.Type)
    return tep.CodeOK, nil
})
mux.HandleFunc("/v1/events/push", consumer.PushHandler())
mux.HandleFunc("/v1/events/{id}/status", consumer.StatusHandler())
```

Swap the store via `consumer.WithStore(store)` for Redis/Pg (the `tep.Store`
interface).

## stress-strike compatibility

The Go library has no external dependencies, plugs into stress-strike's
`api` package easily, or can be used as a regular HTTP target when load
testing a TEP server.