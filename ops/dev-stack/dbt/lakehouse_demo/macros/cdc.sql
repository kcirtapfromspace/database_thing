{% macro cdc_current_state(source_relation, key) %}
{#- Collapse a Debezium change stream to the latest image of each row.
    Rows whose most recent change is a delete are dropped. -#}
with ranked as (
    select
        value,
        row_number() over (
            partition by coalesce(value.after.{{ key }}, value.before.{{ key }})
            order by "offset" desc
        ) as _change_rank
    from {{ source_relation }}
)
select unnest(value.after), value.ts_ms as _cdc_ts_ms
from ranked
where _change_rank = 1 and value.op != 'd'
{% endmacro %}
