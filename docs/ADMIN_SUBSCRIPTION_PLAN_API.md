# Admin Subscription Plan API - Frontend Integration Guide

> **Single Source of Truth** for Admin Subscription Plan Management  
> Generated: 2025-12-21

---

## Overview

This document describes all admin endpoints for managing subscription plans. The frontend should use these endpoints to build a complete plan management UI.

---

## Quick Reference

| Action | Method | Endpoint |
|--------|--------|----------|
| List all plans | GET | `/admin/plans` |
| Create a plan | POST | `/admin/plans` |
| Update a plan | PUT | `/admin/plans/:id` |
| Delete a plan | DELETE | `/admin/plans/:id` |
| List plan display features | GET | `/admin/plans/:id/features` |
| Add display feature | POST | `/admin/plans/:id/features` |
| Update display feature | PUT | `/admin/plans/features/:featureId` |
| Delete display feature | DELETE | `/admin/plans/features/:featureId` |

> All endpoints require `Authorization: Bearer <admin_token>` header.

---

## Understanding the Data Model

```
┌─────────────────────────────────────────────────────────────┐
│                    subscription_plans                        │
├─────────────────────────────────────────────────────────────┤
│ Basic Info: name, slug, description, tagline, price         │
│ AI Features: ai_chat_enabled, semantic_search_enabled       │
│ Limits: max_notebooks, max_notes_per_notebook               │
│ Daily Limits: ai_chat_daily_limit, semantic_search_daily_limit │
│ Display: is_most_popular, is_active, sort_order             │
└───────────────────────────┬─────────────────────────────────┘
                            │ 1:N relationship
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                      plan_features                           │
├─────────────────────────────────────────────────────────────┤
│ Custom bullet points for pricing modal                       │
│ Example: "✓ Unlimited notes", "✓ Priority support"          │
└─────────────────────────────────────────────────────────────┘
```

### Important Distinction

| Concept | Where Stored | What It Does |
|---------|--------------|--------------|
| **AI Features** | Columns in `subscription_plans` | Controls actual system behavior (can user use AI chat?) |
| **Display Features** | Rows in `plan_features` table | Marketing text shown in pricing modal |

---

## Endpoints

### 1. GET /admin/plans

Returns all subscription plans.

**Response:**
```json
{
  "success": true,
  "code": 200,
  "message": "Subscription plans",
  "data": [
    {
      "id": "ce8abb84-10e9-4f54-9f7c-4be693b18ea5",
      "name": "Pro Plan",
      "slug": "pro",
      "description": "Unlock AI features and more storage",
      "tagline": "Best for power users",
      "price": 50000,
      "tax_rate": 0,
      "billing_period": "monthly",
      "is_most_popular": true,
      "is_active": true,
      "sort_order": 1,
      "features": {
        "max_notebooks": 10,
        "max_notes_per_notebook": 100,
        "semantic_search": true,
        "ai_chat": true,
        "ai_chat_daily_limit": 50,
        "semantic_search_daily_limit": 20
      }
    }
  ]
}
```

---

### 2. PUT /admin/plans/:id

Update a subscription plan. All fields are optional - only include what you want to change.

**Request Body:**
```json
{
  "name": "Pro Plan",
  "description": "Full access to all AI features",
  "tagline": "Most popular choice",
  "price": 50000,
  "tax_rate": 0.11,
  "is_most_popular": true,
  "is_active": true,
  "sort_order": 1,
  "features": {
    "max_notebooks": 10,
    "max_notes_per_notebook": 100,
    "semantic_search": true,
    "ai_chat": true,
    "ai_chat_daily_limit": 50,
    "semantic_search_daily_limit": 20
  }
}
```

**Field Reference:**

| Field | Type | Editable | Description |
|-------|------|----------|-------------|
| `name` | string | ✅ | Display name (e.g., "Pro Plan") |
| `slug` | string | ❌ | Set at creation only |
| `description` | string | ✅ | Full description text |
| `tagline` | string | ✅ | Subtitle shown in pricing modal |
| `price` | number | ✅ | Price in smallest currency unit |
| `tax_rate` | number | ✅ | Tax rate (0-1) |
| `billing_period` | string | ❌ | Set at creation only |
| `is_most_popular` | boolean | ✅ | Show "Most Popular" badge |
| `is_active` | boolean | ✅ | Show/hide in pricing modal |
| `sort_order` | integer | ✅ | Display order (lower = first) |
| `features.max_notebooks` | integer | ✅ | Max folders (-1 = unlimited) |
| `features.max_notes_per_notebook` | integer | ✅ | Max notes per folder (-1 = unlimited) |
| `features.semantic_search` | boolean | ✅ | Enable semantic search |
| `features.ai_chat` | boolean | ✅ | Enable AI chat |
| `features.ai_chat_daily_limit` | integer | ✅ | Daily chat limit (0 = disabled, -1 = unlimited) |
| `features.semantic_search_daily_limit` | integer | ✅ | Daily search limit (0 = disabled, -1 = unlimited) |

---

### 3. POST /admin/plans

Create a new subscription plan.

**Request Body:**
```json
{
  "name": "Enterprise Plan",
  "slug": "enterprise",
  "price": 500000,
  "tax_rate": 0.11,
  "billing_period": "yearly",
  "features": {
    "max_notebooks": -1,
    "max_notes_per_notebook": -1,
    "semantic_search": true,
    "ai_chat": true,
    "ai_chat_daily_limit": -1,
    "semantic_search_daily_limit": -1
  }
}
```

> **Note:** `slug` and `billing_period` cannot be changed after creation.

---

### 4. DELETE /admin/plans/:id

Delete a subscription plan.

**Response:**
```json
{
  "success": true,
  "code": 200,
  "message": "Plan deleted",
  "data": null
}
```

---

## Display Features (Pricing Modal Bullet Points)

These are **separate** from AI features. Use these to add custom marketing text to the pricing modal.

### 5. GET /admin/plans/:id/features

Get all display features for a plan.

**Response:**
```json
{
  "success": true,
  "code": 200,
  "message": "Plan features",
  "data": [
    {
      "id": "297c11c8-ea62-4a8b-ba9f-42383de63bc1",
      "plan_id": "05c035ac-2117-40ec-b38f-ab6b912fa114",
      "feature_key": "unlimited_notes",
      "display_text": "Unlimited notes per notebook",
      "is_enabled": true,
      "sort_order": 1
    }
  ]
}
```

---

### 6. POST /admin/plans/:id/features

Add a display feature to a plan.

**Request Body:**
```json
{
  "feature_key": "priority_support",
  "display_text": "Priority email support",
  "is_enabled": true,
  "sort_order": 2
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `feature_key` | string | ✅ | Internal key (no spaces, e.g., "priority_support") |
| `display_text` | string | ✅ | Text shown in UI (e.g., "Priority email support") |
| `is_enabled` | boolean | ❌ | Show checkmark (true) or X (false), default: true |
| `sort_order` | integer | ❌ | Display order, default: 0 |

---

### 7. PUT /admin/plans/features/:featureId

Update a display feature.

**Request Body:**
```json
{
  "display_text": "24/7 Priority support",
  "is_enabled": true,
  "sort_order": 1
}
```

---

### 8. DELETE /admin/plans/features/:featureId

Delete a display feature.

---

## Frontend Implementation Checklist

### Plan List Page
- [ ] Fetch plans with `GET /admin/plans`
- [ ] Display all fields including `description`, `tagline`, `is_most_popular`, `is_active`
- [ ] Sort plans by `sort_order`

### Plan Edit Form
- [ ] Allow editing: `name`, `description`, `tagline`, `price`, `tax_rate`
- [ ] Add toggles for: `is_most_popular`, `is_active`
- [ ] Add number input for: `sort_order`
- [ ] Show `slug` and `billing_period` as **read-only** (cannot change after creation)
- [ ] Include AI Features section with toggles and daily limit inputs
- [ ] Submit changes with `PUT /admin/plans/:id`

### Display Features Tab
- [ ] Fetch features with `GET /admin/plans/:id/features`
- [ ] Show "Add Feature" button that opens a form
- [ ] Allow editing `display_text`, `is_enabled`, `sort_order` inline or in modal
- [ ] Support drag-and-drop reordering (update `sort_order`)
- [ ] Support delete with confirmation

### Pricing Modal (Public)
- [ ] Use `GET /api/plans` (public endpoint) to fetch plans for display
- [ ] Filter by `is_active = true`
- [ ] Sort by `sort_order`
- [ ] Show `tagline` under plan name
- [ ] Show "Most Popular" badge if `is_most_popular = true`
- [ ] Render display features from `features` array with checkmarks/X marks

---

## Example API Calls

### Update tagline and description
```bash
curl -X PUT http://localhost:3000/admin/plans/ce8abb84-10e9-4f54-9f7c-4be693b18ea5 \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "tagline": "Best for teams",
    "description": "Full collaboration features with AI"
  }'
```

### Set plan as most popular
```bash
curl -X PUT http://localhost:3000/admin/plans/ce8abb84-10e9-4f54-9f7c-4be693b18ea5 \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"is_most_popular": true}'
```

### Add a display feature
```bash
curl -X POST http://localhost:3000/admin/plans/ce8abb84-10e9-4f54-9f7c-4be693b18ea5/features \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "feature_key": "unlimited_storage",
    "display_text": "Unlimited cloud storage",
    "is_enabled": true,
    "sort_order": 1
  }'
```
