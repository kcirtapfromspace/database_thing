select
    carrier,
    service_level,
    ship_country,
    count(*) as shipments,
    count(*) filter (where delivered_at is not null) as delivered,
    avg(hours_to_ship) as avg_hours_to_ship,
    quantile_cont(days_to_deliver, 0.5) as p50_days_to_deliver,
    quantile_cont(days_to_deliver, 0.9) as p90_days_to_deliver,
    count(*) filter (where status = 'refunded') / count(*) as refund_rate
from {{ ref('fct_orders') }}
where carrier is not null
group by all
