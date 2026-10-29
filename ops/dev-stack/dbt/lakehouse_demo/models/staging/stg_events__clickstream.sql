-- Kafka delivery is at-least-once, so keep the first copy of each event.
select
    value.event_id as event_id,
    value.event_type as event_type,
    value.event_time::timestamp as event_time,
    value.session_id as session_id,
    value.customer_id as customer_id,
    value.anonymous_id as anonymous_id,
    value.page as page,
    value.product_id as product_id,
    value.category as category,
    value.search_query as search_query,
    value.order_id as order_id,
    value.value as event_value,
    value.device as device,
    value.country as country,
    value.utm_source as channel,
    value.is_backfill as is_backfill
from {{ source('events', 'clickstream') }}
qualify row_number() over (partition by value.event_id order by "offset") = 1
