select
    customer_id,
    email,
    full_name,
    country,
    city,
    segment,
    acquisition_channel,
    marketing_opt_in,
    signup_at::timestamp as signup_at,
    updated_at::timestamp as updated_at
from ({{ cdc_current_state(source('cdc', 'customers'), 'customer_id') }})
