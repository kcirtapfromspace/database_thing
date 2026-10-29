---
title: Shop performance
---

Batch analytics over the lakehouse. dbt rebuilds these marts every time new order changes land in
MinIO: the S3 sink writes Parquet, MinIO raises an event, and Argo Events submits the dbt workflow.
For second-by-second numbers see the **Streaming Operations** dashboard in Grafana.

```sql freshness
INSTALL httpfs; LOAD httpfs;
SET s3_endpoint='minio.minio.svc.cluster.local:9000'; SET s3_url_style='path'; SET s3_use_ssl=false;
SET s3_access_key_id='minio-sa'; SET s3_secret_access_key='minio123';
select
    strftime(refreshed_at, '%Y-%m-%d %H:%M:%S') as refreshed_at,
    triggered_by,
    strftime(latest_order_change_at, '%H:%M:%S') as latest_order_change_at
from read_parquet('s3://lakehouse/marts/pipeline_freshness.parquet')
```


<Alert status="info">
Marts refreshed <b>{freshness[0]?.refreshed_at ?? 'unknown'} UTC</b>, triggered by <code>{freshness[0]?.triggered_by ?? 'unknown'}</code>.
Latest order change included: {freshness[0]?.latest_order_change_at ?? 'unknown'} UTC.
</Alert>

```sql kpis
INSTALL httpfs; LOAD httpfs;
SET s3_endpoint='minio.minio.svc.cluster.local:9000'; SET s3_url_style='path'; SET s3_use_ssl=false;
SET s3_access_key_id='minio-sa'; SET s3_secret_access_key='minio123';
with d as (select * from read_parquet('s3://lakehouse/marts/fct_daily_kpis.parquet')),
bounds as (select max(date) - interval 1 day as last_full_day from d),
cur as (select d.* from d, bounds where date > last_full_day - interval 7 day and date <= last_full_day),
prev as (select d.* from d, bounds where date > last_full_day - interval 14 day and date <= last_full_day - interval 7 day)
select
    (select sum(gmv) from cur) as revenue_usd0k,
    (select sum(gmv) from cur) / (select sum(gmv) from prev) - 1 as revenue_change_pct1,
    (select sum(orders) from cur) as orders_num0,
    (select sum(orders) from cur) / (select sum(orders) from prev) - 1 as orders_change_pct1,
    (select sum(gmv) / sum(orders) from cur) as aov_usd,
    (select sum(gmv) / sum(orders) from cur) / (select sum(gmv) / sum(orders) from prev) - 1 as aov_change_pct1,
    (select sum(conversion_rate * sessions) / sum(sessions) from cur) as conversion_pct2,
    (select sum(conversion_rate * sessions) / sum(sessions) from cur)
        / (select sum(conversion_rate * sessions) / sum(sessions) from prev) - 1 as conversion_change_pct1
```


### Last 7 full days vs the 7 before

<BigValue data={kpis} value=revenue_usd0k comparison=revenue_change_pct1 title="Revenue" comparisonTitle="vs prior week"/>
<BigValue data={kpis} value=orders_num0 comparison=orders_change_pct1 title="Orders" comparisonTitle="vs prior week"/>
<BigValue data={kpis} value=aov_usd comparison=aov_change_pct1 title="Average order value" comparisonTitle="vs prior week"/>
<BigValue data={kpis} value=conversion_pct2 comparison=conversion_change_pct1 title="Session conversion" comparisonTitle="vs prior week"/>

```sql daily
INSTALL httpfs; LOAD httpfs;
SET s3_endpoint='minio.minio.svc.cluster.local:9000'; SET s3_url_style='path'; SET s3_use_ssl=false;
SET s3_access_key_id='minio-sa'; SET s3_secret_access_key='minio123';
select date, gmv as gross_revenue_usd0, net_revenue as net_revenue_usd0, orders as orders_num0,
       conversion_rate as conversion_pct1, payment_decline_rate as payment_decline_rate_pct1,
       promo_orders as promo_orders_num0
from read_parquet('s3://lakehouse/marts/fct_daily_kpis.parquet')
where date < (select max(date) from read_parquet('s3://lakehouse/marts/fct_daily_kpis.parquet'))
order by date
```


Trend charts run through yesterday: today's live traffic is generated at an accelerated demo rate,
so it lives in Grafana instead.

<AreaChart data={daily} x=date y={['gross_revenue_usd0', 'net_revenue_usd0']} title="Daily revenue" subtitle="Gross vs net of refunds; the two spikes are promo days"/>

```sql orders_by_type
INSTALL httpfs; LOAD httpfs;
SET s3_endpoint='minio.minio.svc.cluster.local:9000'; SET s3_url_style='path'; SET s3_use_ssl=false;
SET s3_access_key_id='minio-sa'; SET s3_secret_access_key='minio123';
select date, 'New customers' as customer_type, new_customer_orders as orders_num0 from read_parquet('s3://lakehouse/marts/fct_daily_kpis.parquet') where date < (select max(date) from read_parquet('s3://lakehouse/marts/fct_daily_kpis.parquet'))
union all
select date, 'Returning customers', returning_customer_orders from read_parquet('s3://lakehouse/marts/fct_daily_kpis.parquet') where date < (select max(date) from read_parquet('s3://lakehouse/marts/fct_daily_kpis.parquet'))
order by date
```


<BarChart data={orders_by_type} x=date y=orders_num0 series=customer_type title="Orders by customer type"/>

<LineChart data={daily} x=date y={['conversion_pct1', 'payment_decline_rate_pct1']} title="Session conversion and payment decline rate"/>

### Explore

- [Conversion funnel](/funnel): where sessions drop off, by device and channel
- [Customers & retention](/customers): RFM segments, weekly cohorts, lifetime value
- [Payments & risk](/payments): approval rates by provider, decline mix, chargebacks
- [Products & categories](/merchandising): category revenue and margin, product conversion
- [Fulfillment](/operations): carrier speed and refunds
- [How the data gets here](/pipeline): the streaming and event-driven pipeline behind this site
