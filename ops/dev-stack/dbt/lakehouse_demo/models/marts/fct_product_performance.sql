with views as (
    select product_id, count(*) as views, count(distinct session_id) as viewing_sessions
    from {{ ref('stg_events__clickstream') }}
    where event_type = 'product_view'
    group by product_id
),

cart_adds as (
    select product_id, count(*) as cart_adds
    from {{ ref('stg_events__clickstream') }}
    where event_type = 'add_to_cart'
    group by product_id
),

sales as (
    select
        i.product_id,
        count(distinct i.order_id) as orders,
        sum(i.quantity) as units_sold,
        sum(i.line_total) as revenue
    from {{ ref('stg_shop__order_items') }} as i
    inner join {{ ref('fct_orders') }} as o using (order_id)
    where o.is_successful
    group by i.product_id
)

select
    p.product_id,
    p.sku,
    p.product_name,
    p.category,
    p.brand,
    p.price,
    p.price - p.unit_cost as unit_margin,
    coalesce(v.views, 0) as views,
    coalesce(c.cart_adds, 0) as cart_adds,
    coalesce(s.orders, 0) as orders,
    coalesce(s.units_sold, 0) as units_sold,
    coalesce(s.revenue, 0) as revenue,
    c.cart_adds / nullif(v.views, 0) as view_to_cart_rate,
    s.orders / nullif(v.viewing_sessions, 0) as view_to_purchase_rate
from {{ ref('stg_shop__products') }} as p
left join views as v using (product_id)
left join cart_adds as c using (product_id)
left join sales as s using (product_id)
