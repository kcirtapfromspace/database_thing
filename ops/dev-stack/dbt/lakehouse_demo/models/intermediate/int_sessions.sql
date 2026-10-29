-- One row per storefront session with funnel progress and outcome.
select
    session_id,
    min(event_time) as started_at,
    max(event_time) as ended_at,
    date_diff('second', min(event_time), max(event_time)) as duration_seconds,
    any_value(anonymous_id) as anonymous_id,
    max(customer_id) as customer_id,
    any_value(device) as device,
    any_value(channel) as channel,
    any_value(country) as country,
    count(*) as events,
    count(*) filter (where event_type = 'product_view') as product_views,
    count(*) filter (where event_type = 'search') as searches,
    count(*) filter (where event_type = 'add_to_cart') as cart_adds,
    bool_or(event_type = 'product_view') as viewed_product,
    bool_or(event_type = 'add_to_cart') as added_to_cart,
    bool_or(event_type = 'checkout_start') as started_checkout,
    bool_or(event_type = 'purchase') as purchased,
    bool_or(event_type = 'checkout_error') as hit_checkout_error,
    coalesce(sum(event_value) filter (where event_type = 'purchase'), 0) as revenue,
    count(*) = 1 as bounced
from {{ ref('stg_events__clickstream') }}
group by session_id
