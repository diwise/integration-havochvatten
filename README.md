# Integration Havochvatten

Integrationstjänst som hämtar vattentemperaturprognoser (från Copernicus) via Havs- och vattenmyndighetens API och transformerar till LwM2M/SenML-format.

## Översikt

Tjänsten körs som ett Kubernetes CronJob och utför följande:

1. Hämtar temperaturprognoser (forecasts) från Havs- och vattenmyndighetens API
2. Filtrerar prognoser baserat på konfigurerade bathing water IDs
3. Transformerar prognosdata till SenML+JSON format med LwM2M Temperature-objektet (3303)
4. POST:ar SenML-paketet till IoT Agent (`LWM2M_ENDPOINT_URL`)

**Datakälla:** Prognoserna kommer från [Copernicus Marine Service](https://marine.copernicus.eu/) via Havs- och vattenmyndighetens API.

## Arkitektur

```
┌─────────────────────────────────────────────────────────────────────┐
│                        CronJob Container                            │
│                                                                     │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────────────┐  │
│  │   config    │───▶│   client    │───▶│  Havochvatten API       │  │
│  │             │    │             │    │  (gw.havochvatten.se)   │  │
│  │ NUTS_CODES  │    │ HTTP Client │◀───│                         │  │
│  └─────────────┘    └─────────────┘    └─────────────────────────┘  │
│         │                 │                      │                  │
│         │                 │                      ▼                  │
│         │                 │              ┌──────────────┐           │
│         │                 │              │  Copernicus  │           │
│         │                 │              │  Marine Data │           │
│         │                 │              └──────────────┘           │
│         │                 ▼                                         │
│         │          ┌─────────────┐                                  │
│         │          │   models    │                                  │
│         │          │             │                                  │
│         │          │ Forecast    │                                  │
│         │          │ WaterTemp   │                                  │
│         │          └─────────────┘                                  │
│         │                 │                                         │
│         ▼                 ▼                                         │
│  ┌─────────────────────────────────┐                                │
│  │            senml                │                                │
│  │                                 │                                │
│  │  Transform → SenML+JSON         │                                │
│  │  LwM2M Temperature (3303/5700)  │                                │
│  └─────────────────────────────────┘                                │
│                 │                                                   │
│                 ▼                                                   │
│  ┌─────────────────────────────────┐                                │
│  │         HTTP POST               │                                │
│  │  Content-Type: application/json │──────▶ IoT Agent              │
│  │  LWM2M_ENDPOINT_URL             │       /api/v0/messages/lwm2m  │
│  └─────────────────────────────────┘                                │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

## API-anrop

Tjänsten anropar följande endpoint på Havs- och vattenmyndighetens API:

### GET /forecasts

Hämtar vattentemperaturprognoser från Copernicus för alla badplatser.

**URL:** `https://gw.havochvatten.se/external-public/bathing-waters/v2/forecasts`

**Response-struktur:**
```json
{
  "forecasts": [
    {
      "bathingWaterId": "SE0441273000000001",
      "waterForecasts": [
        { "waterTemp": "15", "measHour": "12" },
        { "waterTemp": "17", "measHour": "15" },
        { "waterTemp": "18", "measHour": "18" }
      ]
    }
  ]
}
```

## Datatransformation

### Indata (från Copernicus via API)

| Fält | Typ | Beskrivning |
|------|-----|-------------|
| `waterTemp` | string | Prognostiserad vattentemperatur i °C |
| `measHour` | string | Timme för prognosen (24h format) |

### Utdata (SenML+JSON)

Temperaturdata transformeras till LwM2M Temperature Object (3303):

```json
[
  {
    "bn": "se0441264000000306/3303/",
    "bt": 1736938800,
    "n": "0",
    "vs": "urn:oma:lwm2m:ext:3303"
  },
  {
    "n": "5700",
    "v": 21.0,
    "u": "Cel"
  }
]
```

| SenML-fält | Värde | Beskrivning |
|------------|-------|-------------|
| `bn` | `<nutsCode>/3303/` | Base name: NUTS-kod (lowercase) + LwM2M Object ID |
| `bt` | Unix timestamp | Base time: Tidpunkt för mätning (int64) |
| `n` | `0` eller `5700` | Name: Resurs-ID (0 för URN, 5700 för Sensor Value) |
| `vs` | `urn:oma:lwm2m:ext:3303` | String Value: LwM2M objekt-URN |
| `v` | float | Value: Temperatur i Celsius |
| `u` | `Cel` | Unit: Celsius |

## Paketstruktur

```
.
├── cmd/integration-havochvatten/
│   └── main.go              # Applikationens entrypoint
├── internal/
│   ├── client/
│   │   └── client.go        # HTTP-klient för API-anrop
│   ├── config/
│   │   └── config.go        # Konfigurationshantering (env vars)
│   ├── models/
│   │   └── bathing_water.go # Datamodeller för API-responses
│   └── senml/
│       └── transform.go     # SenML+JSON transformering
├── deployments/
│   ├── Dockerfile           # Multi-stage Docker build
│   └── k8s/
│       └── cronjob.yaml     # Kubernetes CronJob manifest
└── api/
    └── swagger.yaml         # OpenAPI-spec för Havochvatten API
```

## Konfiguration

| Miljövariabel | Beskrivning | Default |
|---------------|-------------|---------|
| `NUTS_CODES` | Kommaseparerad lista av bathing water IDs | (obligatorisk) |
| `HOV_BADPLATSEN_URL` | Bas-URL för API | `https://gw.havochvatten.se/external-public/bathing-waters/v2` |
| `LWM2M_ENDPOINT_URL` | URL för att POST:a SenML-meddelanden | `http://iot-agent/api/v0/messages/lwm2m` |
| `INCLUDE_FUTURE_FORECASTS` | Inkludera prognoser för framtida timmar | `false` |

## Användning

### Köra lokalt

```bash
NUTS_CODES=SE0A21480000004452 LWM2M_ENDPOINT_URL=http://localhost:8080/api/v0/messages/lwm2m go run ./cmd/integration-havochvatten
```

### Docker

```bash
docker build -f deployments/Dockerfile -t integration-havochvatten:latest .
docker run -e NUTS_CODES=SE0A21480000004452 -e LWM2M_ENDPOINT_URL=http://iot-agent:8080/api/v0/messages/lwm2m integration-havochvatten:latest
```

### Kubernetes

```bash
kubectl apply -f deployments/k8s/cronjob.yaml
```

## Beroenden

- [github.com/diwise/senml](https://github.com/diwise/senml) - SenML-serialisering enligt RFC 8428
- [github.com/diwise/service-chassis](https://github.com/diwise/service-chassis) - Observability (logging, tracing, metrics) med OpenTelemetry
- [go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp](https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp) - HTTP-klient tracing

## Licens

Se LICENSE-filen för licensinformation.
