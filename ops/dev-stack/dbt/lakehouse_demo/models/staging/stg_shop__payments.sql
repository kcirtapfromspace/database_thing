select
    payment_id,
    order_id,
    customer_id,
    method,
    provider,
    status,
    decline_reason,
    amount,
    card_country,
    risk_score,
    attempted_at::timestamp as attempted_at,
    updated_at::timestamp as updated_at
from ({{ cdc_current_state(source('cdc', 'payments'), 'payment_id') }})
