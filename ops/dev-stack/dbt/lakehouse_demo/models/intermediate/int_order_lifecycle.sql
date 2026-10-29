-- Orders enriched with when each status was reached, margin, payment, shipping and refunds.
with status_times as (
    select
        order_id,
        min(changed_at) filter (where to_status = 'paid') as paid_at,
        min(changed_at) filter (where to_status = 'shipped') as shipped_at,
        min(changed_at) filter (where to_status = 'delivered') as delivered_at,
        min(changed_at) filter (where to_status = 'cancelled') as cancelled_at,
        min(changed_at) filter (where to_status = 'refunded') as refunded_at,
        count(*) as status_changes
    from {{ ref('stg_shop__order_status_changes') }}
    group by order_id
),

items as (
    select
        i.order_id,
        sum(i.quantity) as units,
        sum(i.quantity * (i.unit_price - p.unit_cost)) as gross_margin,
        arg_max(p.category, i.line_total) as primary_category
    from {{ ref('stg_shop__order_items') }} as i
    inner join {{ ref('stg_shop__products') }} as p using (product_id)
    group by i.order_id
),

payment_attempts as (
    select
        order_id,
        count(*) as payment_attempts,
        arg_max(method, attempted_at) as payment_method,
        arg_max(provider, attempted_at) as payment_provider,
        arg_max(status, attempted_at) as payment_status,
        max(risk_score) as max_risk_score
    from {{ ref('stg_shop__payments') }}
    group by order_id
),

refunds as (
    select order_id, sum(amount) as refunded_amount, any_value(reason) as refund_reason
    from {{ ref('stg_shop__refunds') }}
    group by order_id
)

select
    o.*,
    st.paid_at,
    st.shipped_at,
    st.delivered_at,
    st.cancelled_at,
    st.refunded_at,
    st.status_changes,
    i.units,
    i.gross_margin,
    i.primary_category,
    pa.payment_attempts,
    pa.payment_method,
    pa.payment_provider,
    pa.payment_status,
    pa.max_risk_score,
    s.carrier,
    s.service_level,
    date_diff('minute', o.created_at, s.shipped_at) / 60.0 as hours_to_ship,
    date_diff('minute', o.created_at, s.delivered_at) / 1440.0 as days_to_deliver,
    coalesce(r.refunded_amount, 0) as refunded_amount,
    r.refund_reason
from {{ ref('stg_shop__orders') }} as o
left join status_times as st using (order_id)
left join items as i using (order_id)
left join payment_attempts as pa using (order_id)
left join {{ ref('stg_shop__shipments') }} as s using (order_id)
left join refunds as r using (order_id)
