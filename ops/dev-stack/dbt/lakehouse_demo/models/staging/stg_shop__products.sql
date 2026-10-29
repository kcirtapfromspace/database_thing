select
    product_id,
    sku,
    name as product_name,
    category,
    brand,
    price,
    unit_cost,
    is_active,
    updated_at::timestamp as updated_at
from ({{ cdc_current_state(source('cdc', 'products'), 'product_id') }})
