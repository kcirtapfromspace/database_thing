-- Every status transition an order went through, reconstructed from the CDC update history.
select
    coalesce(value.after.order_id, value.before.order_id) as order_id,
    value.op as operation,
    value.before.status as from_status,
    value.after.status as to_status,
    value.after.updated_at::timestamp as changed_at,
    "offset" as change_offset
from {{ source('cdc', 'orders') }}
where value.op in ('c', 'r')
   or (value.op = 'u' and value.before.status is distinct from value.after.status)
