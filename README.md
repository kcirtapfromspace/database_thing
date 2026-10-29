# tributary

tributary is a data platform for a local Kubernetes cluster.
A simulated online shop sends events into streams.
The streams go to live dashboards and to a lakehouse.

Use tributary to learn and show these patterns:

- Change data capture (CDC) from Postgres to Kafka
- Stream processing with materialized views
- Event-driven workflows
- A lakehouse with dbt and DuckDB

## Architecture

```
shopgen ──orders, payments, shipments──▶ Postgres ──Debezium CDC──▶ Redpanda (cdc.shop.*)
   └────────────clickstream (Avro)─────────────────────────────────▶ Redpanda (shop.clickstream)

Redpanda ──▶ RisingWave materialized views ──▶ Grafana "Streaming Operations"
                 └─ anomaly signals ──▶ Redpanda (alerts.anomalies) ──▶ Argo Events ──▶ incident-response workflow

Redpanda ──S3 sink──▶ MinIO lakehouse raw/ ──object event──▶ Argo Events ──▶ dbt + DuckDB workflow ──▶ marts/ ──▶ Evidence
```

The platform has two data paths: a stream path and a batch path.

### Stream path

1. `shopgen` simulates shoppers. It writes orders, payments, shipments and refunds to Postgres.
2. `shopgen` also sends clickstream events to the Redpanda topic `shop.clickstream`.
3. Debezium reads all changes in the Postgres `shop` schema. It sends them to the `cdc.shop.*` topics.
4. RisingWave reads the topics and keeps its materialized views up to date.
5. When RisingWave finds an anomaly, it sends a signal to the `alerts.anomalies` topic.
6. Argo Events reads the signal and starts an `incident-response` workflow.
7. The workflow records the incident in RisingWave. It also writes evidence to MinIO.

### Batch path

1. A Kafka Connect sink writes all topics to MinIO as Parquet files in `lakehouse/raw/`.
2. When MinIO receives a new orders file, it sends an event to Argo Events.
3. Argo Events starts the `dbt-lakehouse` workflow. The limit is one workflow each minute.
4. dbt and DuckDB read the raw files. They write the marts to `lakehouse/marts/`.
5. Evidence reads the marts and shows the reports.

### Components

| Component | Function | Location |
|---|---|---|
| [shopgen](ops/dev-stack/shopgen) | Writes 30 days of history one time. Then it simulates live shoppers: the funnel, payments with declines and retries, fulfillment, refunds, chargebacks, VIP upgrades and account deletions. | http://localhost:8090/status |
| [Postgres](https://www.postgresql.org/) 12 | Keeps the shop data in the `shop` schema. | `localhost:5432` |
| [Debezium](https://debezium.io/) `shop-cdc` | Sends all inserts, updates and deletes in the `shop` schema to the `cdc.shop.*` topics. | Kafka Connect, http://localhost:8083 |
| `lakehouse-sink` | Writes all CDC and clickstream topics to MinIO as Parquet files. | Kafka Connect |
| [Redpanda](https://redpanda.com/) | Kafka-compatible event broker. It also carries the Argo Events event bus. | Console, http://localhost:8080 |
| [RisingWave](https://risingwave.com/) | Keeps live views of the streams: the funnel, revenue for each minute and payment health. It finds anomalies in sliding windows. | `psql -h localhost -p 4566 -U root dev` |
| [Argo Events](https://argoproj.github.io/argo-events/) | Starts workflows from MinIO events and from anomaly signals. | `ops/dev-stack/event-driven` |
| [Argo Workflows](https://argoproj.github.io/workflows/) | Runs the `dbt-lakehouse` and `incident-response` workflows. | http://localhost:2746 |
| [MinIO](https://min.io/) | S3-compatible object storage for the lakehouse. | http://localhost:9001 |
| [dbt](https://www.getdbt.com/) and [DuckDB](https://duckdb.org/) | Change the raw CDC data into staging, intermediate and mart models. | `ops/dev-stack/dbt/lakehouse_demo` |
| [Grafana](https://grafana.com/) and Prometheus | Show the "Streaming Operations" dashboard: RisingWave views and platform metrics. | http://localhost:3000 |
| [Evidence](https://evidence.dev/) | Shows batch reports from the marts. | http://localhost:4001 |

## Requirements

### Hardware

- Set the Docker memory limit to 16 GB or more. We tested the platform with 16 GB.
- We tested the platform on macOS with Apple silicon (arm64).

### Software

Install these tools:

| Tool | Purpose |
|---|---|
| [Docker](https://docs.docker.com/get-docker/) | Runs the cluster node and builds the images. |
| [kind](https://kind.sigs.k8s.io/) | Runs Kubernetes in a Docker container. |
| [ctlptl](https://github.com/tilt-dev/ctlptl) | Creates the kind cluster and the local image registry. |
| [kubectl](https://kubernetes.io/docs/tasks/tools/) | Connects to the Kubernetes cluster. |
| [Helm](https://helm.sh/docs/intro/install/) | Installs the charts. Tilt uses Helm. |
| [Tilt](https://docs.tilt.dev/install.html) | Builds, deploys and monitors the platform. |
| `psql` (optional) | Connects to RisingWave and Postgres. |
| [Argo CLI](https://argo-workflows.readthedocs.io/en/latest/walk-through/argo-cli/) (optional) | Submits workflows from the command line. |

On macOS, you can use Homebrew:

```sh
brew install kind ctlptl kubectl helm tilt
```

## Start the platform

1. Make sure that Docker operates.
2. Create the local image registry:

   ```sh
   ctlptl create registry ctlptl-registry --port=5005
   ```

3. Create a kind cluster that uses the registry:

   ```sh
   ctlptl create cluster kind --registry=ctlptl-registry
   ```

4. Make sure that kubectl uses the kind cluster:

   ```sh
   kubectl config use-context kind-kind
   ```

5. Go to the root directory of the repository.
6. Start Tilt:

   ```sh
   tilt up
   ```

7. Open the Tilt UI at http://localhost:10350.
8. Wait until all resources are green.

> **NOTE:** The first start takes more time than later starts.
> Tilt builds the images, and `shopgen` writes 30 days of history.

## Open the services

| Service | URL | Login |
|---|---|---|
| Tilt | http://localhost:10350 | None |
| Grafana | http://localhost:3000 | None. You are an anonymous administrator. The password for the `admin` user is `prom-operator`. |
| Redpanda Console | http://localhost:8080 | None |
| Kafka Connect REST API | http://localhost:8083 | None |
| MinIO console | http://localhost:9001 | `rootuser` / `rootpass123` |
| Argo Workflows | http://localhost:2746 | None |
| Evidence | http://localhost:4001 | None |
| Prometheus | http://localhost:9090 | None |
| shopgen status | http://localhost:8090/status | None |
| go-loader | http://localhost:8000 | None |
| RisingWave | `psql -h localhost -p 4566 -U root dev` | User `root`, no password |
| Postgres | `localhost:5432` | Refer to `ops/dev-stack/postgres_db/config/postgres.env`. |

> **CAUTION:** These logins are for local development only.
> Do not open these ports to a shared network.

In Grafana, open the **Streaming Operations** dashboard.

## Start an incident

You can start a simulated incident. The platform finds the incident and starts a workflow.

1. Open the Tilt UI.
2. Find the **shop** group.
3. Click the trigger button of one of these resources:

   | Resource | Incident |
   |---|---|
   | `incident-payment-outage` | The `adyen` payment provider declines most payments. |
   | `incident-fraud-burst` | Fraud accounts make many small orders in a short time. |
   | `incident-traffic-spike` | Traffic increases to four times the usual rate. Most new sessions use the `FLASH25` promotion. |

You can also send a request to `shopgen`:

```sh
curl -X POST 'localhost:8090/incident/payment_outage?minutes=5'
```

Results:

- RisingWave finds a payment outage or a fraud burst in approximately 1 minute.
- Argo Events then starts an `incident-*` workflow in Argo Workflows.
- The incident shows in the Grafana **Streaming Operations** dashboard.
- A traffic spike does not start a workflow. It shows in the traffic panels.

> **NOTE:** RisingWave sends one signal for each alert type and subject in a 10-minute window.
> A second incident of the same type in the same window does not start a new workflow.

## Stop the platform

To stop Tilt and keep the cluster and the data:

1. Push **Ctrl+C** in the terminal that runs `tilt up`.

To remove the platform from the cluster:

1. Run `tilt down`.

To delete the cluster and the registry:

1. Run these commands:

   ```sh
   ctlptl delete cluster kind-kind
   ctlptl delete registry ctlptl-registry
   ```

> **CAUTION:** `tilt down` deletes most of the platform data.
> When you delete the cluster, you delete all data in Postgres, MinIO and RisingWave.

## Data and reports

### dbt project

The dbt project is in `ops/dev-stack/dbt/lakehouse_demo`. It has three layers:

| Layer | Models | Output |
|---|---|---|
| Staging | `stg_shop__*` and `stg_events__clickstream`. They change the CDC history into the current state of each row. | DuckDB tables |
| Intermediate | `int_sessions` and `int_order_lifecycle` | DuckDB tables |
| Marts | `fct_orders`, `dim_customers` (RFM segments), `fct_daily_kpis`, `fct_funnel_daily`, `fct_cohort_retention`, `fct_payment_health_daily`, `fct_category_daily`, `fct_product_performance`, `fct_fulfillment` and `pipeline_freshness` | Parquet files in `s3://lakehouse/marts` |

### Evidence reports

| Page | Content |
|---|---|
| Shop performance | Revenue, orders, order value and conversion for each day |
| Funnel | Sessions, steps of the funnel, channels and devices |
| Customers | RFM segments and cohort retention |
| Payments | Declines, retries, payment providers and risk |
| Merchandising | Products and categories |
| Operations | Fulfillment and shipments |
| Pipeline | Data freshness and the file that started the last refresh |

### Grafana dashboard

The **Streaming Operations** dashboard has three rows:

| Row | Content |
|---|---|
| Live storefront | Sessions, conversion, orders and revenue from RisingWave |
| Payments and risk | Decline rates for each provider, live alerts and incidents |
| Event backbone | Redpanda throughput, consumer lag, Argo Events triggers and `shopgen` metrics |

## Repository layout

| Path | Content |
|---|---|
| `tiltfile` | The Tilt configuration. It builds the images and deploys all components. |
| `Dockerfiles/` | The image build files |
| `ops/dev-stack/shopgen/` | The data generator |
| `ops/dev-stack/postgres_db/` | Postgres |
| `ops/dev-stack/debezium/` | Kafka Connect and the connector configuration |
| `ops/dev-stack/redpanda/` | Redpanda and Redpanda Console |
| `ops/dev-stack/risingwave/` | RisingWave and the streaming SQL |
| `ops/dev-stack/event-driven/` | The Argo Events bus, event sources, sensors and workflow templates |
| `ops/dev-stack/argo/`, `ops/dev-stack/argo-events/` | Argo Workflows and Argo Events chart values, and older workflow templates |
| `ops/dev-stack/minio/` | MinIO |
| `ops/dev-stack/dbt/lakehouse_demo/` | The dbt project |
| `ops/dev-stack/evidence.dev/` | The Evidence reports |
| `ops/dev-stack/monitoring/` | Prometheus and Grafana values, PodMonitors and the dashboard |
| `ops/dev-stack/go_loader/` | A Go service that loads files into Postgres |

The other directories in `ops/dev-stack/` are older experiments. Tilt does not deploy them.

## Known limits

- Live traffic is faster than real time: 150 sessions each minute.
  Thus, the Evidence charts show complete days only. Grafana shows the current day.
- The sink writes one small Parquet file for each topic each minute.
  For long runs, add compaction or incremental dbt models.
- The `data-workflow` template clones the dbt project from the `main` branch on GitHub.
  Push your model changes before you use it. The `dbt-lakehouse` template uses the project in the local image.
- The data quality prototypes in `ops/dev-stack/py_app/src/quality_checks` (deequ and Great Expectations) do not operate.
- Evidence uses port 4001 on the host, not the default port 4000.

## Troubleshooting

| Problem | Possible cause | Action |
|---|---|---|
| Tilt cannot connect to the cluster after Docker restarts. | Docker stopped the kind node. Docker Desktop can also change the kubectl context. | Run `docker start kind-control-plane`. Then run `kubectl config use-context kind-kind`. |
| A port forward fails. | Another process uses the port. | Stop the other process, or change the port in `tiltfile`. |
| Evidence shows old data. | Evidence keeps query results for one minute. | Wait one minute. Then load the page again. |
| No `dbt-lakehouse` workflows start. | MinIO receives no new orders files. | Make sure that the `shop-cdc` and `lakehouse-sink` connectors operate: `curl 'localhost:8083/connectors?expand=status'`. |

## Optional settings

### GitHub token

The `data-workflow` template uses a GitHub token to clone the repository.
Tilt reads the token from the `GITHUB_ACCESS_TOKEN` environment variable.

1. Create a GitHub personal access token with read access to the repository.
2. Set the variable before you start Tilt. This example uses the 1Password CLI:

   ```sh
   export GITHUB_ACCESS_TOKEN=op://development/github/personal_access_token_dataworkflow
   op run -- tilt up
   ```

### Slack notifications

The `data-workflow` template does not send Slack notifications by default.
To send notifications, submit the workflow with these parameters:

```sh
argo submit -n argo --from workflowtemplate/data-workflow \
  -p enableSlackNotifications=true \
  -p slack-webhook-url=<your-webhook-url>
```

> **CAUTION:** Do not write a webhook URL into a file in this repository.

## Background

This project started from the DuckDB article [Modern Data Stack in a Box](https://duckdb.org/2022/10/12/modern-data-stack-in-a-box.html).
