select
    order_item_id,
    order_id,
    product_id,
    quantity,
    unit_price,
    quantity * unit_price as line_total,
    created_at::timestamp as created_at
from ({{ cdc_current_state(source('cdc', 'order_items'), 'order_item_id') }})
