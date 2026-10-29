---
title: Fulfillment
---

Shipment milestones (label created, in transit, delivered) are CDC updates on the `shipments` table.

```sql carriers
INSTALL httpfs; LOAD httpfs;
SET s3_endpoint='minio.minio.svc.cluster.local:9000'; SET s3_url_style='path'; SET s3_use_ssl=false;
SET s3_access_key_id='minio-sa'; SET s3_secret_access_key='minio123';
select carrier || ' ' || service_level as service, ship_country,
       shipments as shipments_num0, avg_hours_to_ship as hours_to_ship_num1,
       p50_days_to_deliver as median_days_num1, p90_days_to_deliver as p90_days_num1,
       refund_rate as refund_rate_pct1
from read_parquet('s3://lakehouse/marts/fct_fulfillment.parquet')
order by shipments desc
```


```sql speed
INSTALL httpfs; LOAD httpfs;
SET s3_endpoint='minio.minio.svc.cluster.local:9000'; SET s3_url_style='path'; SET s3_use_ssl=false;
SET s3_access_key_id='minio-sa'; SET s3_secret_access_key='minio123';
select carrier || ' ' || service_level as service,
       sum(p50_days_to_deliver * delivered) / sum(delivered) as median_days_num1
from read_parquet('s3://lakehouse/marts/fct_fulfillment.parquet')
group by 1 order by 2
```


<BarChart data={speed} x=service y=median_days_num1 title="Typical delivery time by service" swapXY=true/>

<DataTable data={carriers} search=true rows=12/>

```sql refunds
INSTALL httpfs; LOAD httpfs;
SET s3_endpoint='minio.minio.svc.cluster.local:9000'; SET s3_url_style='path'; SET s3_use_ssl=false;
SET s3_access_key_id='minio-sa'; SET s3_secret_access_key='minio123';
select refund_reason, count(*) as orders_num0, sum(refunded_amount) as refunded_usd0
from read_parquet('s3://lakehouse/marts/fct_orders.parquet')
where refund_reason is not null
group by 1 order by 2 desc
```


<BarChart data={refunds} x=refund_reason y=refunded_usd0 title="Refunds by reason" swapXY=true/>

```sql status_mix
INSTALL httpfs; LOAD httpfs;
SET s3_endpoint='minio.minio.svc.cluster.local:9000'; SET s3_url_style='path'; SET s3_use_ssl=false;
SET s3_access_key_id='minio-sa'; SET s3_secret_access_key='minio123';
select status, count(*) as orders_num0, sum(total) as value_usd0 from read_parquet('s3://lakehouse/marts/fct_orders.parquet') group by 1 order by 2 desc
```


<DataTable data={status_mix}/>
