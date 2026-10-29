select
    order_id,
    customer_id,
    session_id,
    status,
    currency,
    channel,
    device,
    ship_country,
    item_count,
    subtotal,
    discount,
    shipping_fee,
    tax,
    total,
    promo_code,
    created_at::timestamp as created_at,
    updated_at::timestamp as updated_at
from ({{ cdc_current_state(source('cdc', 'orders'), 'order_id') }})
