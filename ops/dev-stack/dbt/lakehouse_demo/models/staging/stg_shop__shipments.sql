select
    shipment_id,
    order_id,
    carrier,
    service_level,
    status,
    created_at::timestamp as label_created_at,
    shipped_at::timestamp as shipped_at,
    delivered_at::timestamp as delivered_at
from ({{ cdc_current_state(source('cdc', 'shipments'), 'shipment_id') }})
