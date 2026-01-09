# Integration Havochvatten

## Regler för utveckling

- **Dokumentation:** Uppdatera alltid README.md vid förändringar som påverkar arkitektur, API-anrop, konfiguration eller dataflöde.
- **Loggning:** Använd `github.com/diwise/service-chassis/pkg/infrastructure/o11y` för logging med OpenTelemetry-integration.
- **Loggning av fel:** Vid loggning av errors, använd alltid `"err", err.Error()` som parameter. Exempel: `logger.Error("failed to do something", "err", err.Error())`
- **Projektstruktur:** Följ Standard Go Project Layout.

## Projektbeskrivning

En integrationstjänst som körs som ett CronJob i Kubernetes. Tjänsten hämtar information om badplatser från Havs- och vattenmyndighetens API, tolkar data och transformerar det till SenML+JSON format för LwM2M Temperature-objektet.

## Tekniska specifikationer

- **Språk:** Go 1.25
- **Körning:** Kubernetes CronJob
- **Datakälla:** Havs- och vattenmyndighetens API
- **Utdataformat:** SenML+JSON (RFC 8428)
- **LwM2M-modell:** Temperature (Object ID 3303)
- **Loggning & Observability:** OpenTelemetry via [github.com/diwise/service-chassis](https://github.com/diwise/service-chassis)

## Output-format

### SenML+JSON

Tjänsten genererar SenML+JSON (Sensor Measurement Lists) enligt [RFC 8428](https://datatracker.ietf.org/doc/html/rfc8428).

**Beroende:** [github.com/diwise/senml](https://github.com/diwise/senml)

### LwM2M Temperature Object (3303)

Temperaturobjektet (IPSO Smart Object) - vi använder resurs 0 för URN och 5700 för temperaturvärde:

| Resurs ID | Namn | Typ | Beskrivning |
|-----------|------|-----|-------------|
| 0 | URN | String | LwM2M objekt-URN (`urn:oma:lwm2m:ext:3303`) |
| 5700 | Sensor Value | Float | Aktuellt temperaturvärde |

### Exempel på SenML+JSON output

```json
[
  {
    "bn": "se0441264000000306/3303/",
    "bt": 1736438400,
    "n": "0",
    "vs": "urn:oma:lwm2m:ext:3303"
  },
  {
    "n": "5700",
    "v": 18.5,
    "u": "Cel"
  }
]
```

Där:
- `bn` (base name): `<nutsCode>/3303/` - NUTS-kod (lowercase) följt av LwM2M Object ID
- `bt` (base time): Unix timestamp (int64)
- `n` (name): Resurs-ID (0 för URN, 5700 för temperaturvärde)
- `vs` (string value): LwM2M objekt-URN
- `v` (value): Temperaturvärde i grader Celsius
- `u` (unit): Enhet ("Cel" för Celsius)

## Konfiguration

### Miljövariabler

| Miljövariabel | Beskrivning | Default |
|---------------|-------------|--------|
| `NUTS_CODES` | Kommaseparerad lista av bathing water IDs | (obligatorisk) |
| `HOV_BADPLATSEN_URL` | Bas-URL för Havs- och vattenmyndighetens API | `https://gw.havochvatten.se/external-public/bathing-waters/v2` |
| `LWM2M_ENDPOINT_URL` | URL för att POST:a SenML-meddelanden till IoT Agent | `http://iot-agent/api/v0/messages/lwm2m` |
| `INCLUDE_FUTURE_FORECASTS` | Inkludera prognoser för framtida timmar | `false` |

Exempel:
```
NUTS_CODES=SE0A21480000004452,SE0441273000000001
LWM2M_ENDPOINT_URL=http://iot-agent:8080/api/v0/messages/lwm2m
```

Motivering: Miljövariabler är enklast att hantera i Kubernetes CronJobs via ConfigMaps eller direkt i job-specifikationen.

## API-källa

Havs- och vattenmyndigheten tillhandahåller API för badvatteninformation:
- Bas-URL: `https://badplatsen.havochvatten.se/badplatsen/api/`

## Projektstruktur

Projektet följer [Standard Go Project Layout](https://github.com/golang-standards/project-layout):

```
.
├── AGENTS.md                 # Denna fil - projektdokumentation
├── cmd/
│   └── integration-havochvatten/
│       └── main.go           # Applikationens entrypoint
├── internal/                 # Privat applikationskod (kan ej importeras externt)
│   ├── config/               # Konfigurationshantering
│   ├── client/               # HTTP-klient för API-anrop
│   ├── models/               # Datamodeller
│   └── senml/                # SenML+JSON transformering
├── pkg/                      # Bibliotekskod som kan användas av externa projekt
├── api/                      # API-definitioner (OpenAPI, Protobuf etc.)
├── configs/                  # Konfigurationsmallar
├── deployments/              # Kubernetes, Docker Compose etc.
│   ├── Dockerfile
│   └── k8s/
│       └── cronjob.yaml      # Kubernetes CronJob-manifest
├── scripts/                  # Bygg- och hjälpskript
├── build/                    # Packaging och CI
├── go.mod
└── go.sum
```

### Katalogbeskrivning

| Katalog | Beskrivning |
|---------|-------------|
| `cmd/` | Huvudapplikationer. Varje underkatalog är en separat körbar fil. |
| `internal/` | Privat kod som inte kan importeras av andra projekt. |
| `pkg/` | Bibliotekskod avsedd för återanvändning i andra projekt. |
| `api/` | API-specifikationer och kontraktdefinitioner. |
| `configs/` | Standardkonfigurationer och mallar. |
| `deployments/` | IaaS, PaaS, container-orchestration (Kubernetes, Docker Compose). |
| `scripts/` | Skript för bygg, installation, analys etc. |
| `build/` | Dockerfile och CI-konfiguration. |

## Utveckling

### Bygga applikationen

```bash
go build -o integration-havochvatten ./cmd/integration-havochvatten
```

### Köra lokalt

```bash
NUTS_CODES=SE110,SE121 ./integration-havochvatten
```

### Docker

```bash
docker build -f deployments/Dockerfile -t integration-havochvatten:latest .
docker run -e NUTS_CODES=SE110,SE121 integration-havochvatten:latest
```

---

# API-dokumentation (Havs- och vattenmyndigheten)

> Denna sektion innehåller dokumentation från Havs- och vattenmyndigheten för deras Bathing Waters API.

## Utvecklarguide

Denna dokumentation vänder sig till dig som utvecklare som ska anpassa din applikation för att kunna använda detta API.

### API-specifikation

För att utforma REST API specifikationer använder myndigheten standarden [OpenAPI version 3](https://github.com/OAI/OpenAPI-Specification).
API-specifikationen kan användas för att generera kod till din applikation. Tips på verktyg för kodgenerering finns [här](https://openapi.tools).

### Förändringar

Ett API ska vara stabilt och alla ändringar som görs ska vara bakåtkompatibla. Om detta inte kan uppfyllas kommer en ny version av detta API att publiceras. Den tidigare versionen kommer sedan under kontrollerade former att avvecklas.

#### Förbered applikationen på att kunna hantera bakåtkompatibla utökningar

Applikationer som använder ett API ska vara robusta och vara förberedda för att kunna hantera bakåtkompatibla ändringar av ett API. Det innebär att:

- Vara konservativa i anrop av ett API. Exempelvis ska inte data sättas utan att ha kontroll på dess längd.
- Vara toleranta med okända egenskaper i meddelandet som kan behövas i ett efterföljande PUT-anrop.
- Vara förberedd på att x-extensible-enum kan leverera nya värden.
- Vara förberedd att HTTP-statuskoder kan returneras som inte finns angivna i API-specifikationen. Standardhantering av koder ska bygga på exempelvis 1XX, 2XX, 3XX och 4XX enligt [RFC7231 sektion 6](https://www.rfc-editor.org/rfc/rfc7231#section-6).
- Följ omdirigeringen när HTTP-statuskod 301 returneras.

#### Utfasning

När ett API fasas ut kommer detta att synliggöras i API-specifikationen samt att HTTP-rubriker kommer att finnas med i svaret.

Exempel på utfasningsinformation i API-specifikationen:
```yaml
/resurs:
  get:
    deprecated: true
    description: This operation is deprecated and will be retired 2023-01-01. Instead use the operation getResourcById.
```

Exempel på utfasningsinformation i HTTP-svaret:
```
X-API-Deprecated: true
X-API-Retire-Time: 2023-01-01T12:00:00
```

### Tekniska begränsningar

API:et är konfigurerat för att tillåta maximalt **1000 anrop per minut**. När denna nivå har uppnåts returneras HTTP-statuskod `429 Too many requests`.

### Säkerhet

Detta API är en REST-tjänst baserad på HTTP och JSON. Kommunikation mot tjänsten krypteras via HTTPS.

## Miljöer

### Test

| Funktion | URL |
|----------|-----|
| Gateway | https://gw-test.havochvatten.se |
| API | https://gw-test.havochvatten.se/external-public/bathing-waters/v2 |
| Health check | https://gw-test.havochvatten.se/external-public/bathing-waters/v2/operations/health-checks/readiness |

### Produktion

| Funktion | URL |
|----------|-----|
| Gateway | https://gw.havochvatten.se |
| API | https://gw.havochvatten.se/external-public/bathing-waters/v2 |
| Health check | https://gw.havochvatten.se/external-public/bathing-waters/v2/operations/health-checks/readiness |

## Övervakning

**Sökväg:** `/operations/health-checks/readiness`

För att säkerställa att API:et är tillgängligt finns det en resurs att anropa. Förväntat värde är "up" eller "down".

Exempel på svar:
```json
{
  "readiness": {
    "description": "Service is ready",
    "state": "up"
  }
}
```

## Bathing-waters

**Sökväg:** `/bathing-waters?filter=(hasAdviceAgainstBathing eq 'true')`

För anropet finns valfri frågeparametern `filter` som accepterar följande parameter:

| Parameter | Tillåtna operationer | Tillåtna värden | Default | Beskrivning |
|-----------|---------------------|-----------------|---------|-------------|
| `hasAdviceAgainstBathing` | eq, ne | true, false | - | Utan filter hämtas badplatser med och utan pågående avrådan. `eq 'true'` för endast badplatser med avrådan, `eq 'false'` för endast utan. |

## Changelog

**Sökväg:** `/bathing-waters/changelogs?filter=(...)`

För anropet finns frågeparametern `filter` som accepterar följande parametrar:

| Parameter | Operationer | Tillåtna värden | Default | Beskrivning |
|-----------|-------------|-----------------|---------|-------------|
| `changedAt` | ge | RFC3339 timestamp | Dagens datum 00:00 (CET/CEST) | Datumet av första ändring som ska hämtas |
| `isNew` | eq, ne | true, false | true | `eq 'true'` för att hämta nya badplatser |
| `hasUpdatedCoords` | eq, ne | true, false | true | `eq 'true'` för badplatser med uppdaterade koordinater |
| `hasUpdatedName` | eq, ne | true, false | true | `eq 'true'` för badplatser med uppdaterade namn |
| `onlyLatest` | eq, ne | true, false | true | `eq 'true'` för att hämta bara den senaste ändringen |
| `statusToFind` | eq | active, inactive, all, none | all | Filtrera på statusändringar |

## Kontakt och support

Om du har ytterligare frågor eller behöver hjälp är du välkommen att kontakta Havs- och vattenmyndigheten via mejl på [itsupport@havochvatten.se](mailto:itsupport@havochvatten.se).

**Källa:** [Havs- och vattenmyndigheten](https://www.havochvatten.se)

# API dokumentation

Denna dokumentation vänder sig till dig som arbetar i verksamheten som vill förstå hur detta API kan användas och till vad.

## Vad kan jag använda detta API till?

Vårt API för Badplatsen är till för dig som vill hämta information om badplatser i Sverige.

## Vilken information kan jag få via detta API?

Du ställer en fråga via API:et genom att fråga efter en badplats BathingWater id eller relaterad data. Som svar får du information om badplatsen som finns registrerad hos Havs- och vattenmyndigheten.

| Information om en badvatten ändring innehåller | Exempel | Beskrivning |
|-----------------------------------------------|---------|-------------|
| changedAt | 2023-04-26T11:04:24Z | När ändringen skedde i UTC |
| bathingWaterId | SE0441278000000146 | Badvattnets unika id |
| difference | newWater | Kort beskrivning av ändringen |

## Hur får jag tillgång till detta API?

- Du ansluter och testar. I testmiljön kan du säkerställa att du kan koppla upp dina applikationer mot API:et och att din applikation fungerar som det är tänkt.
- När testerna är avklarade och verifierade får du därefter tillgång till produktionsmiljön.

## Vilken teknik är detta API baserat på?

Detta API är en REST-tjänst baserad på HTTP och JSON. Kommunikation mot tjänsten krypteras via HTTPS.

## Finns det en testmiljö?

Det finns en testmiljö där du kan testa din integration och säkerställa att den fungerar innan det är dags för produktion. Testdatat är oftast statiskt vilket innebär att det alltid är samma svar som returneras.

## Vilken teknisk tillgänglighet har detta API?

API:et är tillgängligt dygnet runt men det kan förekomma kortare avbrott vid servicefönster. Vi har support för dig under helgfria vardagar mellan 9–15.

## Hur många anrop kan jag göra mot detta API?

API:et är konfigurerat för att tillåta maximalt 1000 anrop per minut.

## Hur kommer jag i kontakt med er om jag har frågor eller behöver hjälp?

Om du har ytterligare frågor eller behöver hjälp är du välkommen att kontakta oss. Du når oss via mejl på [itsupport@havochvatten.se](mailto:itsupport@havochvatten.se).

---

**Källa:** [Havs- och vattenmyndigheten](https://www.havochvatten.se)