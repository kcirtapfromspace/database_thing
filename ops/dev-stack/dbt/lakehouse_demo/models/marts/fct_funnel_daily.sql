select
    started_at::date as date,
    device,
    channel,
    count(*) as sessions,
    count(*) filter (where viewed_product) as viewed_product,
    count(*) filter (where added_to_cart) as added_to_cart,
    count(*) filter (where started_checkout) as started_checkout,
    count(*) filter (where purchased) as purchased
from {{ ref('int_sessions') }}
group by all
