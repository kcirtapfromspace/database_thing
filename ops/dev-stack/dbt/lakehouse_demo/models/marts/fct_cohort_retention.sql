-- Weekly cohorts by first purchase: how many come back, and what they spend.
with orders as (
    select customer_id, created_at, net_revenue
    from {{ ref('fct_orders') }}
    where is_successful
),

cohorts as (
    select customer_id, date_trunc('week', min(created_at))::date as cohort_week
    from orders
    group by customer_id
),

cohort_sizes as (
    select cohort_week, count(*) as cohort_size
    from cohorts
    group by cohort_week
)

select
    c.cohort_week,
    date_diff('week', c.cohort_week, date_trunc('week', o.created_at)::date) as week_number,
    cs.cohort_size,
    count(distinct o.customer_id) as active_customers,
    count(distinct o.customer_id) / cs.cohort_size as retention_rate,
    sum(o.net_revenue) as revenue,
    sum(o.net_revenue) / cs.cohort_size as revenue_per_customer
from orders as o
inner join cohorts as c using (customer_id)
inner join cohort_sizes as cs using (cohort_week)
group by 1, 2, 3
