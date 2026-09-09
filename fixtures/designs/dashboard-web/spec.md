# Dashboard — Design Spec

## Overview

Web application dashboard for analytics, featuring charts, KPIs, and a data table.

## Screen layout

### Overview
- Top bar with date range picker
- 4 KPI cards: Active users, Revenue, Conversion, Churn
- KPI trend sparklines

### Charts
- Line chart: revenue over time
- Bar chart: signups by channel
- Donut chart: plan distribution

### Data table
- Columns: user, plan, status, last active, spend
- Sortable headers
- Search and filters
- Pagination (25/page)

## States

- Empty: no data in range
- Loading: skeleton cards
- Error: inline error with retry

## Design tokens

- Primary: #8B5CF6 (violet 500)
- Background: #F9FAFB
- Cards: white, 1px border, 8px radius
- Typography: Inter

## Handoff

Provide engineering with:

- `dashboard-overview.png` (1600x900)
- `dashboard-charts.png` (1600x900)
- Reusable chart components in the design system