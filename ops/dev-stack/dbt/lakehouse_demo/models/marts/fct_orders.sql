with orders as (
    select
        *,
        status in ('paid', 'shipped', 'delivered', 'refunded') as is_successful
    from {{ ref('int_order_lifecycle') }}
),

first_orders as (
    select customer_id, min(created_at) as first_order_at
    from orders
    where is_successful
    group by customer_id
)

select
    o.order_id,
    o.customer_id,
    o.session_id,
    o.created_at,
    o.created_at::date as order_date,
    o.status,
    o.is_successful,
    o.is_successful and o.created_at = f.first_order_at as is_first_order,
    o.channel,
    o.device,
    o.ship_country,
    o.item_count,
    o.units,
    o.subtotal,
    o.discount,
    o.shipping_fee,
    o.tax,
    o.total,
    o.promo_code,
    o.promo_code is not null as used_promo,
    case when o.is_successful then o.total - o.refunded_amount else 0 end as net_revenue,
    o.gross_margin,
    o.primary_category,
    o.payment_attempts,
    o.payment_method,
    o.payment_provider,
    o.payment_status,
    o.max_risk_score,
    o.carrier,
    o.service_level,
    o.hours_to_ship,
    o.days_to_deliver,
    o.refunded_amount,
    o.refund_reason,
    o.paid_at,
    o.shipped_at,
    o.delivered_at,
    o.refunded_at
from orders as o
left join first_orders as f using (customer_id)
