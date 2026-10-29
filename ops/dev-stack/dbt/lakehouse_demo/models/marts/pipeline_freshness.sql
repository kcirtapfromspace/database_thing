-- One row describing this refresh: when it ran, what triggered it, and how current the data is.
select
    now()::timestamp as refreshed_at,
    '{{ var("triggered_by") }}' as triggered_by,
    (select max(event_time) from {{ ref('stg_events__clickstream') }}) as latest_event_at,
    (select max(updated_at) from {{ ref('stg_shop__orders') }}) as latest_order_change_at,
    (select count(*) from {{ ref('stg_events__clickstream') }}) as clickstream_events,
    (select count(*) from {{ ref('stg_shop__orders') }}) as orders,
    (select count(*) from {{ ref('stg_shop__customers') }}) as customers
