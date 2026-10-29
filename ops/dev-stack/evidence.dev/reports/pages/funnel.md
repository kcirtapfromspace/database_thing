---
title: Conversion funnel
---

Sessions are reconstructed from the clickstream that the storefront publishes straight to Redpanda.

```sql funnel
INSTALL httpfs; LOAD httpfs;
SET s3_endpoint='minio.minio.svc.cluster.local:9000'; SET s3_url_style='path'; SET s3_use_ssl=false;
SET s3_access_key_id='minio-sa'; SET s3_secret_access_key='minio123';
with f as (select * from read_parquet('s3://lakehouse/marts/fct_funnel_daily.parquet')
           where date >= (select max(date) from read_parquet('s3://lakehouse/marts/fct_funnel_daily.parquet')) - interval 7 day
             and date < (select max(date) from read_parquet('s3://lakehouse/marts/fct_funnel_daily.parquet')))
select * from (
    select 1 as ord, 'Sessions' as step, sum(sessions) as sessions from f union all
    select 2, 'Viewed a product', sum(viewed_product) from f union all
    select 3, 'Added to cart', sum(added_to_cart) from f union all
    select 4, 'Started checkout', sum(started_checkout) from f union all
    select 5, 'Purchased', sum(purchased) from f
) order by ord
```


<FunnelChart data={funnel} nameCol=step valueCol=sessions title="Last 7 full days"/>

```sql by_device
INSTALL httpfs; LOAD httpfs;
SET s3_endpoint='minio.minio.svc.cluster.local:9000'; SET s3_url_style='path'; SET s3_use_ssl=false;
SET s3_access_key_id='minio-sa'; SET s3_secret_access_key='minio123';
select device,
       sum(sessions) as sessions_num0,
       sum(added_to_cart) / sum(sessions) as cart_rate_pct1,
       sum(purchased) / sum(sessions) as conversion_pct2
from read_parquet('s3://lakehouse/marts/fct_funnel_daily.parquet')
group by device order by conversion_pct2 desc
```


```sql by_channel
INSTALL httpfs; LOAD httpfs;
SET s3_endpoint='minio.minio.svc.cluster.local:9000'; SET s3_url_style='path'; SET s3_use_ssl=false;
SET s3_access_key_id='minio-sa'; SET s3_secret_access_key='minio123';
select channel,
       sum(sessions) as sessions_num0,
       sum(added_to_cart) / sum(sessions) as cart_rate_pct1,
       sum(purchased) / sum(sessions) as conversion_pct2
from read_parquet('s3://lakehouse/marts/fct_funnel_daily.parquet')
group by channel order by conversion_pct2 desc
```


<BarChart data={by_device} x=device y=conversion_pct2 title="Conversion by device" swapXY=true/>
<BarChart data={by_channel} x=channel y=conversion_pct2 title="Conversion by acquisition channel" swapXY=true/>

<DataTable data={by_channel}/>

```sql step_rates
INSTALL httpfs; LOAD httpfs;
SET s3_endpoint='minio.minio.svc.cluster.local:9000'; SET s3_url_style='path'; SET s3_use_ssl=false;
SET s3_access_key_id='minio-sa'; SET s3_secret_access_key='minio123';
with d as (select date, sum(viewed_product) v, sum(added_to_cart) a, sum(started_checkout) c, sum(purchased) p
           from read_parquet('s3://lakehouse/marts/fct_funnel_daily.parquet') where date < (select max(date) from read_parquet('s3://lakehouse/marts/fct_funnel_daily.parquet')) group by date)
select date, 'View to cart' as step, a / v as rate_pct1 from d union all
select date, 'Cart to checkout', c / a from d union all
select date, 'Checkout to purchase', p / c from d
order by date
```


<LineChart data={step_rates} x=date y=rate_pct1 series=step title="Step conversion rates by day"/>
