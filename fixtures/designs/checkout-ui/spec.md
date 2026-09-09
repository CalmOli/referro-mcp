# Checkout UI — Component Spec

## Overview

Reusable checkout UI components used across the web app and mobile app.

## Components

### Payment card form
- Card number field with auto-formatting (spaces every 4 digits)
- Expiry field (MM/YY)
- CVC field (3 digits)
- Inline validation with error messages

### Address form
- Street, city, state, postal code, country
- Country selector drives state/province options
- Autocomplete for street address

### Payment method selection
- Saved cards list
- Add new card option
- Radio selection with card brand icons

## States

Each field supports: default, focus, valid, error, disabled.

## Accessibility

- All fields have labels and aria-describedby
- Full keyboard navigation
- Focus ring visible on all interactive elements

## Design tokens

- Primary: #10B981 (emerald 500)
- Error: #EF4444 (red 500)
- Radius: 8px inputs
- Typography: Inter

## Handoff

Provide engineering with:

- `checkout-card.png` (800x600)
- `checkout-address.png` (800x600)
- Storybook stories for each component state