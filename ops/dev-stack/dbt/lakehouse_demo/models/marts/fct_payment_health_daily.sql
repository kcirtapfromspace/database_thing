select
    attempted_at::date as date,
    provider,
    method,
    count(*) as attempts,
    count(*) filter (where status != 'declined') as approved,
    count(*) filter (where status = 'declined') as declined,
    count(*) filter (where status = 'declined') / count(*) as decline_rate,
    count(*) filter (where decline_reason = 'processor_unavailable') as processor_errors,
    count(*) filter (where decline_reason = 'fraud_suspected') as fraud_declines,
    count(*) filter (where status = 'refunded') as refunded,
    count(*) filter (where status = 'chargeback') as chargebacks,
    mode(decline_reason) filter (where status = 'declined') as top_decline_reason,
    avg(risk_score) as avg_risk_score,
    sum(amount) filter (where status != 'declined') as approved_amount
from {{ ref('stg_shop__payments') }}
group by all
