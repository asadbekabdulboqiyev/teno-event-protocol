# tep-go

TEP (Teno Event Protocol) — Go implementatsiyasi.
**stress-strike** (Go load-tester) bilan integratsiya va standart ish uchun.
HTTP/REST transport binding: `spec/transport-http.md`.

## Paket

`github.com/asadbekabdulboqiyev/teno-event-protocol/packages/tep-go/tep`

## Izazlash

```bash
go test ./...        # + -race bilan ham
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

`consumer.WithStore(store)` orqali Redis/Pg store almashtirish mumkin
(`tep.Store` interfeysi).

## stress-strike uyg'unligi

Go kutubxonasi hech qanday tashqi bog'liqliksiz, stress-strike'ning
`api` paketiga oson ulanishadi yoki TEP server'ni yuk-test qilishda
odatiy HTTP target sifatida ishlatiladi.