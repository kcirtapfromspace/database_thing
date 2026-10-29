with sessions as (
    select
        started_at::date as date,
        count(*) as sessions,
        count(distinct anonymous_id) as visitors,
        count(*) filter (where purchased) as converting_sessions,
        count(*) filter (where bounced) as bounced_sessions
    from {{ ref('int_sessions') }}
    group by 1
),

orders as (
    select
        order_date as date,
        count(*) filter (where is_successful) as orders,
        sum(total) filter (where is_successful) as gmv,
        sum(net_revenue) as net_revenue,
        sum(gross_margin) filter (where is_successful) as gross_margin,
        count(*) filter (where is_first_order) as new_customer_orders,
        count(*) filter (where status = 'payment_failed') as failed_checkouts,
        count(*) filter (where used_promo and is_successful) as promo_orders,
        sum(refunded_amount) as refunded_amount
    from {{ ref('fct_orders') }}
    group by 1
),

payments as (
    select
        attempted_at::date as date,
        count(*) as payment_attempts,
        count(*) filter (where status = 'declined') as payment_declines
    from {{ ref('stg_shop__payments') }}
    group by 1
)

select
    s.date,
    s.sessions,
    s.visitors,
    coalesce(o.orders, 0) as orders,
    coalesce(o.gmv, 0) as gmv,
    coalesce(o.net_revenue, 0) as net_revenue,
    coalesce(o.gross_margin, 0) as gross_margin,
    o.gmv / nullif(o.orders, 0) as avg_order_value,
    s.converting_sessions / s.sessions as conversion_rate,
    s.bounced_sessions / s.sessions as bounce_rate,
    coalesce(o.new_customer_orders, 0) as new_customer_orders,
    coalesce(o.orders - o.new_customer_orders, 0) as returning_customer_orders,
    coalesce(o.promo_orders, 0) as promo_orders,
    coalesce(o.failed_checkouts, 0) as failed_checkouts,
    coalesce(o.refunded_amount, 0) as refunded_amount,
    p.payment_declines / nullif(p.payment_attempts, 0) as payment_decline_rate
from sessions as s
left join orders as o using (date)
left join payments as p using (date)
order by s.date
