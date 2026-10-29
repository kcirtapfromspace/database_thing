---
title: Customers & retention
---

Customer state comes from the Postgres `customers` table via Debezium CDC, so profile edits,
VIP upgrades and account deletions all flow through. Segments use recency, frequency and monetary scores.

```sql segments
INSTALL httpfs; LOAD httpfs;
SET s3_endpoint='minio.minio.svc.cluster.local:9000'; SET s3_url_style='path'; SET s3_use_ssl=false;
SET s3_access_key_id='minio-sa'; SET s3_secret_access_key='minio123';
select rfm_segment,
       count(*) as customers_num0,
       sum(lifetime_revenue) as revenue_usd0,
       avg(lifetime_revenue) as avg_lifetime_value_usd,
       avg(orders) as avg_orders_num1
from read_parquet('s3://lakehouse/marts/dim_customers.parquet')
group by rfm_segment
order by revenue_usd0 desc
```


<BarChart data={segments} x=rfm_segment y=revenue_usd0 title="Lifetime revenue by RFM segment" swapXY=true/>

<DataTable data={segments}/>

```sql cohorts
INSTALL httpfs; LOAD httpfs;
SET s3_endpoint='minio.minio.svc.cluster.local:9000'; SET s3_url_style='path'; SET s3_use_ssl=false;
SET s3_access_key_id='minio-sa'; SET s3_secret_access_key='minio123';
select strftime(cohort_week, '%Y-%m-%d') as cohort_week,
       max(cohort_size) as customers_num0,
       max(retention_rate) filter (where week_number = 1) as week_1_pct1,
       max(retention_rate) filter (where week_number = 2) as week_2_pct1,
       max(retention_rate) filter (where week_number = 3) as week_3_pct1,
       sum(revenue) / max(cohort_size) as revenue_per_buyer_usd
from read_parquet('s3://lakehouse/marts/fct_cohort_retention.parquet')
group by cohort_week
order by cohort_week
```


### Weekly cohorts by first purchase

Share of each cohort that bought again N weeks after their first order.

<DataTable data={cohorts}/>

```sql curves
INSTALL httpfs; LOAD httpfs;
SET s3_endpoint='minio.minio.svc.cluster.local:9000'; SET s3_url_style='path'; SET s3_use_ssl=false;
SET s3_access_key_id='minio-sa'; SET s3_secret_access_key='minio123';
select week_number, strftime(cohort_week, '%b %d') as cohort,
       sum(revenue_per_customer) over (partition by cohort_week order by week_number) as cumulative_revenue_per_customer_usd
from read_parquet('s3://lakehouse/marts/fct_cohort_retention.parquet')
order by cohort_week, week_number
```


<LineChart data={curves} x=week_number y=cumulative_revenue_per_customer_usd series=cohort title="Cumulative revenue per customer, by cohort"/>

```sql acquisition
INSTALL httpfs; LOAD httpfs;
SET s3_endpoint='minio.minio.svc.cluster.local:9000'; SET s3_url_style='path'; SET s3_use_ssl=false;
SET s3_access_key_id='minio-sa'; SET s3_secret_access_key='minio123';
select acquisition_channel,
       count(*) as customers_num0,
       count(*) filter (where orders > 0) / count(*) as buyer_rate_pct1,
       avg(lifetime_revenue) filter (where orders > 0) as avg_lifetime_value_usd
from read_parquet('s3://lakehouse/marts/dim_customers.parquet')
group by 1 order by avg_lifetime_value_usd desc
```


<BarChart data={acquisition} x=acquisition_channel y=avg_lifetime_value_usd title="Lifetime value of buyers by acquisition channel" swapXY=true/>

```sql top_customers
INSTALL httpfs; LOAD httpfs;
SET s3_endpoint='minio.minio.svc.cluster.local:9000'; SET s3_url_style='path'; SET s3_use_ssl=false;
SET s3_access_key_id='minio-sa'; SET s3_secret_access_key='minio123';
select full_name, country, rfm_segment, orders as orders_num0,
       lifetime_revenue as lifetime_revenue_usd, strftime(last_order_at, '%Y-%m-%d') as last_order
from read_parquet('s3://lakehouse/marts/dim_customers.parquet')
order by lifetime_revenue desc limit 25
```


### Top customers

<DataTable data={top_customers} search=true/>
