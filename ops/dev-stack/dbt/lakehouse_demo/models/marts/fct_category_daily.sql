select
    o.order_date as date,
    p.category,
    count(distinct o.order_id) as orders,
    sum(i.quantity) as units,
    sum(i.line_total) as revenue,
    sum(i.quantity * (i.unit_price - p.unit_cost)) as gross_margin,
    count(distinct o.order_id) filter (where o.status = 'refunded') as refunded_orders
from {{ ref('fct_orders') }} as o
inner join {{ ref('stg_shop__order_items') }} as i using (order_id)
inner join {{ ref('stg_shop__products') }} as p using (product_id)
where o.is_successful
group by all
