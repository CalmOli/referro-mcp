# Mobile App — Design Spec

## Overview

iOS companion application for quick task capture and review on the go.

## Platform

- iOS 17+, SwiftUI
- iPhone and iPad adaptive layouts
- Light and dark mode support

## Screens

### Onboarding
- 3-step flow: Welcome, Permissions, Notifications
- Progress indicator at top
- Skip button on each step

### Home
- Task list grouped by due date
- Quick-add button (floating action)
- Pull-to-refresh

### Settings
- Account profile
- Notification preferences
- Theme toggle (light/dark/system)

## Navigation

- Tab bar: Home, Search, Profile
- Home is the default tab

## Design tokens

- Primary: #0EA5E9 (sky 500)
- Background: system background
- Radius: 12px cards
- Typography: SF Pro

## Handoff

Provide engineering with:

- `mobile-onboarding.png` (1170x2532)
- `mobile-home.png` (1170x2532)
- `mobile-settings.png` (1170x2532)
- SwiftUI component library in Figma