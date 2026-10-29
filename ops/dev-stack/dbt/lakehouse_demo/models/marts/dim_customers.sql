-- Customers with lifetime value and RFM (recency, frequency, monetary) segmentation.
with orders as (
    select * from {{ ref('fct_orders') }} where is_successful
),

as_of as (
    select max(created_at) as as_of from {{ ref('stg_shop__orders') }}
),

buyers as (
    select
        customer_id,
        count(*) as orders,
        sum(net_revenue) as lifetime_revenue,
        avg(total) as avg_order_value,
        min(created_at) as first_order_at,
        max(created_at) as last_order_at
    from orders
    group by customer_id
),

scored as (
    select
        b.*,
        date_diff('day', b.last_order_at, a.as_of) as days_since_last_order,
        ntile(5) over (order by b.last_order_at) as recency_score,
        case when b.orders >= 3 then 5 when b.orders = 2 then 3 else 1 end as frequency_score,
        ntile(5) over (order by b.lifetime_revenue) as monetary_score
    from buyers as b
    cross join as_of as a
)

select
    c.*,
    coalesce(s.orders, 0) as orders,
    coalesce(s.lifetime_revenue, 0) as lifetime_revenue,
    s.avg_order_value,
    s.first_order_at,
    s.last_order_at,
    s.days_since_last_order,
    s.recency_score,
    s.frequency_score,
    s.monetary_score,
    case
        when s.customer_id is null then 'Prospect'
        when s.recency_score >= 4 and s.frequency_score >= 3 then 'Champion'
        when s.frequency_score >= 3 then 'Loyal'
        when s.recency_score >= 4 and s.monetary_score >= 4 then 'Big Spender'
        when s.recency_score >= 4 then 'New'
        when s.recency_score <= 2 and s.monetary_score >= 4 then 'At Risk'
        when s.recency_score <= 2 then 'Hibernating'
        else 'Needs Attention'
    end as rfm_segment
from {{ ref('stg_shop__customers') }} as c
left join scored as s using (customer_id)
