---
title: Customer Insights
sources:
    - my_query.sql
    - some_category/my_category_query.sql
---# {$page.params.customer}

{#each location_summary as location}

## {location.name}

<Value data={location.sales_usd}/> in sales at a <Value data={location.gross_margin_pct}/> gross margin.

{/each}


