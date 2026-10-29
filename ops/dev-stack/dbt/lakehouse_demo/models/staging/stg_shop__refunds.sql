select
    refund_id,
    order_id,
    payment_id,
    amount,
    reason,
    created_at::timestamp as created_at
from ({{ cdc_current_state(source('cdc', 'refunds'), 'refund_id') }})
