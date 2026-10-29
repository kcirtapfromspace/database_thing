---
title: Products & categories
---

```sql categories
INSTALL httpfs; LOAD httpfs;
SET s3_endpoint='minio.minio.svc.cluster.local:9000'; SET s3_url_style='path'; SET s3_use_ssl=false;
SET s3_access_key_id='minio-sa'; SET s3_secret_access_key='minio123';
select category, sum(revenue) as revenue_usd0, sum(gross_margin) / sum(revenue) as margin_pct1,
       sum(units) as units_num0, sum(refunded_orders) / sum(orders) as refund_rate_pct1
from read_parquet('s3://lakehouse/marts/fct_category_daily.parquet')
group by category order by revenue_usd0 desc
```


<BarChart data={categories} x=category y=revenue_usd0 title="Revenue by category" swapXY=true/>

<DataTable data={categories}/>

```sql category_daily
INSTALL httpfs; LOAD httpfs;
SET s3_endpoint='minio.minio.svc.cluster.local:9000'; SET s3_url_style='path'; SET s3_use_ssl=false;
SET s3_access_key_id='minio-sa'; SET s3_secret_access_key='minio123';
select date, category, revenue as revenue_usd0 from read_parquet('s3://lakehouse/marts/fct_category_daily.parquet')
where date < (select max(date) from read_parquet('s3://lakehouse/marts/fct_category_daily.parquet'))
order by date
```


<AreaChart data={category_daily} x=date y=revenue_usd0 series=category title="Daily revenue by category"/>

```sql products
INSTALL httpfs; LOAD httpfs;
SET s3_endpoint='minio.minio.svc.cluster.local:9000'; SET s3_url_style='path'; SET s3_use_ssl=false;
SET s3_access_key_id='minio-sa'; SET s3_secret_access_key='minio123';
select product_name, category, price as price_usd, views as views_num0, cart_adds as cart_adds_num0,
       units_sold as units_num0, revenue as revenue_usd0,
       view_to_cart_rate as view_to_cart_pct1, view_to_purchase_rate as view_to_purchase_pct2
from read_parquet('s3://lakehouse/marts/fct_product_performance.parquet')
order by revenue desc limit 50
```


### Top products

Views and cart adds come from the clickstream; units and revenue from orders.

<DataTable data={products} search=true rows=15/>
