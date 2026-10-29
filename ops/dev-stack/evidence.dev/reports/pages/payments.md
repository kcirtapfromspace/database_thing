---
title: Payments & risk
---

Every payment attempt, including declines and retries, captured from Postgres by Debezium. Today's
row includes any outages or fraud bursts injected from the Tilt UI; RisingWave catches those live and
Argo opens an incident for each (see Grafana).

```sql providers
INSTALL httpfs; LOAD httpfs;
SET s3_endpoint='minio.minio.svc.cluster.local:9000'; SET s3_url_style='path'; SET s3_use_ssl=false;
SET s3_access_key_id='minio-sa'; SET s3_secret_access_key='minio123';
select date, provider, sum(approved) / sum(attempts) as approval_rate_pct1
from read_parquet('s3://lakehouse/marts/fct_payment_health_daily.parquet')
group by 1, 2 order by 1
```


<LineChart data={providers} x=date y=approval_rate_pct1 series=provider yMin=0.5 yMax=1 title="Approval rate by provider"/>

```sql decline_mix
INSTALL httpfs; LOAD httpfs;
SET s3_endpoint='minio.minio.svc.cluster.local:9000'; SET s3_url_style='path'; SET s3_use_ssl=false;
SET s3_access_key_id='minio-sa'; SET s3_secret_access_key='minio123';
select date, 'Processor unavailable' as decline_type, sum(processor_errors) / sum(attempts) as decline_rate_pct1 from read_parquet('s3://lakehouse/marts/fct_payment_health_daily.parquet') group by 1, 2 union all
select date, 'Fraud suspected', sum(fraud_declines) / sum(attempts) from read_parquet('s3://lakehouse/marts/fct_payment_health_daily.parquet') group by 1, 2 union all
select date, 'Issuer decline', sum(declined - processor_errors - fraud_declines) / sum(attempts) from read_parquet('s3://lakehouse/marts/fct_payment_health_daily.parquet') group by 1, 2
order by 1
```


<BarChart data={decline_mix} x=date y=decline_rate_pct1 series=decline_type title="Decline rate by type" subtitle="Share of payment attempts; today includes any injected outage or fraud burst"/>

```sql methods
INSTALL httpfs; LOAD httpfs;
SET s3_endpoint='minio.minio.svc.cluster.local:9000'; SET s3_url_style='path'; SET s3_use_ssl=false;
SET s3_access_key_id='minio-sa'; SET s3_secret_access_key='minio123';
select method,
       sum(attempts) as attempts_num0,
       sum(approved) / sum(attempts) as approval_rate_pct1,
       sum(approved_amount) as approved_amount_usd0,
       sum(chargebacks) as chargebacks_num0
from read_parquet('s3://lakehouse/marts/fct_payment_health_daily.parquet')
group by method order by attempts_num0 desc
```


### Payment methods

<DataTable data={methods}/>

```sql provider_totals
INSTALL httpfs; LOAD httpfs;
SET s3_endpoint='minio.minio.svc.cluster.local:9000'; SET s3_url_style='path'; SET s3_use_ssl=false;
SET s3_access_key_id='minio-sa'; SET s3_secret_access_key='minio123';
select provider, sum(attempts) as attempts_num0, sum(declined) / sum(attempts) as decline_rate_pct1,
       mode(top_decline_reason) as most_common_decline
from read_parquet('s3://lakehouse/marts/fct_payment_health_daily.parquet')
group by provider order by attempts_num0 desc
```


<DataTable data={provider_totals}/>
