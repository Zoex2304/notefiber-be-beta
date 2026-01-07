# Usage Limits API Documentation v2.0
> **For Frontend Team** - Single Source of Truth for the dynamic usage limits system
> 
> **Last Updated:** 2025-12-21  
> **Document Version:** 2.0 (VERIFIED against actual backend code)

---

## ⚠️ Breaking Changes from Previous Docs

| Previous Documentation | Actual Backend Route | Status |
|------------------------|---------------------|--------|
| `GET /api/user/subscription/status` | **Does NOT exist** | Use `/api/payment/status` instead |

---

## Table of Contents
1. [Overview](#overview)
2. [Public Endpoints](#public-endpoints)
3. [User Endpoints (Authenticated)](#user-endpoints-authenticated)
4. [Admin Endpoints](#admin-endpoints)
5. [Limit Checking Flow](#limit-checking-flow)
6. [Error Handling](#error-handling)
7. [Seed Data SQL](#seed-data-sql)

---

## Overview

The system enforces usage limits based on subscription plans:

| Resource | Free Plan | Pro Plan |
|----------|-----------|----------|
| Max Notebooks (Folders) | 3 | 20 |
| Max Notes per Notebook | 10 | 50 |
| AI Chat / Day | 0 (Disabled) | 50 |
| Semantic Search / Day | 0 (Disabled) | 30 |

### Key Concepts
- **Storage Limits** (notebooks, notes): Cumulative, don't reset
- **Daily Limits** (AI chat, search): Reset at midnight
- **-1 = Unlimited**, **0 = Disabled**

---

## Public Endpoints

### GET /api/plans
Fetch all active subscription plans for pricing modal.

**Authentication:** None required

**Response:**
```json
{
  "success": true,
  "code": 200,
  "message": "Plans retrieved",
  "data": [
    {
      "id": "uuid",
      "name": "Free Plan",
      "slug": "free",
      "tagline": "Get started with basic note-taking",
      "price": 0,
      "billing_period": "monthly",
      "is_most_popular": false,
      "limits": {
        "max_notebooks": 3,
        "max_notes_per_notebook": 10,
        "ai_chat_daily": 0,
        "semantic_search_daily": 0
      },
      "features": [
        { "key": "basic_notes", "text": "Basic Note Taking", "is_enabled": true },
        { "key": "semantic_search", "text": "Semantic Search", "is_enabled": false },
        { "key": "ai_chat", "text": "AI Chat Assistant", "is_enabled": false }
      ]
    },
    {
      "id": "uuid",
      "name": "Pro Plan",
      "slug": "pro",
      "tagline": "Unlock AI Chat and Semantic Search",
      "price": 50000,
      "billing_period": "yearly",
      "is_most_popular": true,
      "limits": {
        "max_notebooks": 20,
        "max_notes_per_notebook": 50,
        "ai_chat_daily": 50,
        "semantic_search_daily": 30
      },
      "features": [
        { "key": "basic_notes", "text": "Basic Note Taking", "is_enabled": true },
        { "key": "semantic_search", "text": "Semantic Search", "is_enabled": true },
        { "key": "ai_chat", "text": "AI Chat Assistant", "is_enabled": true }
      ]
    }
  ]
}
```

---

## User Endpoints (Authenticated)

All endpoints require:
```http
Authorization: Bearer <user_jwt_token>
```

### GET /api/user/usage-status
Check current usage vs limits before performing actions.

**Response:**
```json
{
  "success": true,
  "code": 200,
  "message": "Usage status retrieved",
  "data": {
    "plan": {
      "id": "uuid",
      "name": "Pro Plan",
      "slug": "pro"
    },
    "storage": {
      "notebooks": {
        "used": 2,
        "limit": 3,
        "can_use": true
      },
      "notes": {
        "used": 0,
        "limit": 10,
        "can_use": true
      }
    },
    "daily": {
      "ai_chat": {
        "used": 12,
        "limit": 50,
        "can_use": true,
        "resets_at": "2025-12-22T00:00:00Z"
      },
      "semantic_search": {
        "used": 5,
        "limit": 30,
        "can_use": true,
        "resets_at": "2025-12-22T00:00:00Z"
      }
    },
    "upgrade_available": false
  }
}
```

---

### GET /api/payment/status
> ⚠️ **Note:** Previous docs incorrectly said `/api/user/subscription/status`. Use this endpoint instead.

Get current subscription status.

**Response:**
```json
{
  "success": true,
  "code": 200,
  "message": "Subscription status",
  "data": {
    "subscription_id": "uuid",
    "plan_name": "Pro Plan",
    "status": "active",
    "current_period_end": "2026-12-21T00:00:00Z",
    "ai_chat_daily_limit": 50,
    "semantic_search_daily_limit": 30,
    "is_active": true,
    "features": {
      "ai_chat": true,
      "semantic_search": true,
      "max_notebooks": 20,
      "max_notes_per_notebook": 50
    }
  }
}
```

---

### GET /api/user/profile
Get user profile information.

**Response:**
```json
{
  "success": true,
  "code": 200,
  "message": "User profile",
  "data": {
    "id": "uuid",
    "email": "user@example.com",
    "full_name": "John Doe",
    "role": "user",
    "status": "active",
    "avatar_url": "https://...",
    "ai_daily_usage": 12,
    "created_at": "2025-01-01T00:00:00Z"
  }
}
```

---

### POST /api/user/refund/request
Request a refund for a subscription.

**Request:**
```json
{
  "subscription_id": "uuid",
  "reason": "Reason for refund request (min 10 characters)"
}
```

**Response:**
```json
{
  "success": true,
  "code": 200,
  "message": "Refund request submitted",
  "data": {
    "refund_id": "uuid",
    "status": "pending"
  }
}
```

---

## Admin Endpoints

All endpoints require:
```http
Authorization: Bearer <admin_jwt_token>
```

### POST /api/admin/login
Admin login (Public).

**Request:**
```json
{
  "email": "admin@example.com",
  "password": "password"
}
```

---

### Plan Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/admin/plans` | Get all plans |
| POST | `/api/admin/plans` | Create new plan |
| PUT | `/api/admin/plans/:id` | Update plan |
| DELETE | `/api/admin/plans/:id` | Delete plan |

#### POST /api/admin/plans (Create Plan)
```json
{
  "name": "Pro Plan",
  "slug": "pro",
  "price": 50000,
  "tax_rate": 0.11,
  "billing_period": "yearly",
  "features": {
    "max_notebooks": 20,
    "max_notes_per_notebook": 50,
    "semantic_search": true,
    "ai_chat": true,
    "ai_chat_daily_limit": 50,
    "semantic_search_daily_limit": 30
  }
}
```

---

### Plan Feature Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/admin/plans/:id/features` | Get plan features |
| POST | `/api/admin/plans/:id/features` | Create feature |
| PUT | `/api/admin/plans/features/:featureId` | Update feature |
| DELETE | `/api/admin/plans/features/:featureId` | Delete feature |

#### POST /api/admin/plans/:id/features
```json
{
  "feature_key": "custom_feature",
  "display_text": "Custom Feature Name",
  "is_enabled": true,
  "sort_order": 4
}
```

**Response:**
```json
{
  "success": true,
  "message": "Feature created",
  "data": {
    "id": "uuid",
    "plan_id": "uuid",
    "feature_key": "custom_feature",
    "display_text": "Custom Feature Name",
    "is_enabled": true,
    "sort_order": 4
  }
}
```

---

### Refund Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/admin/refunds` | List refunds (query: `page`, `limit`, `status`) |
| POST | `/api/admin/refunds/:id/approve` | Approve refund |

---

## Limit Checking Flow

### When to Check Limits

| Action | Check Field |
|--------|-------------|
| Create Folder/Notebook | `storage.notebooks.can_use` |
| Create Note | `storage.notes.can_use` |
| Send AI Chat Message | `daily.ai_chat.can_use` |
| Perform Semantic Search | `daily.semantic_search.can_use` |

### Frontend Pre-Check Pattern
```typescript
// Before creating a notebook
async function canCreateNotebook(): Promise<boolean> {
  const response = await fetch('/api/user/usage-status', {
    headers: { 'Authorization': `Bearer ${token}` }
  });
  const { data } = await response.json();
  return data.storage.notebooks.can_use;
}

// Usage
if (await canCreateNotebook()) {
  await createNotebook();
} else {
  showPricingModal();
}
```

---

## Error Handling

### Limit Exceeded (HTTP 429)
```json
{
  "success": false,
  "code": 429,
  "message": "Daily AI chat limit reached",
  "data": {
    "limit": 50,
    "used": 50,
    "reset_after": "2025-12-22T00:00:00Z",
    "show_modal_pricing": true
  }
}
```

### Feature Requires Upgrade (HTTP 403)
```json
{
  "success": false,
  "code": 403,
  "message": "Access denied: Feature requires upgrade"
}
```

### Frontend Error Handler
```typescript
async function handleApiResponse(response: Response) {
  const data = await response.json();
  
  if (response.status === 429) {
    if (data.data?.show_modal_pricing) {
      showPricingModal();
    } else {
      showLimitWarning({
        used: data.data.used,
        limit: data.data.limit,
        resetsAt: data.data.reset_after
      });
    }
    return null;
  }
  
  if (response.status === 403) {
    showPricingModal();
    return null;
  }
  
  return data;
}
```

---

## Seed Data SQL

Run this SQL to set up initial plans with correct limits:

```sql
-- 1. Update Free plan
UPDATE subscription_plans SET
    tagline = 'Get started with basic note-taking',
    max_notebooks = 3,
    max_notes_per_notebook = 10,
    ai_chat_daily_limit = 0,
    semantic_search_daily_limit = 0,
    is_active = true,
    sort_order = 0
WHERE slug = 'free' OR price = 0;

-- 2. Update Pro plan
UPDATE subscription_plans SET
    tagline = 'Unlock AI Chat and Semantic Search',
    max_notebooks = 20,
    max_notes_per_notebook = 50,
    ai_chat_daily_limit = 50,
    semantic_search_daily_limit = 30,
    is_most_popular = true,
    is_active = true,
    sort_order = 1
WHERE slug = 'pro' OR name ILIKE '%pro%';

-- 3. Insert features for Free plan
INSERT INTO plan_features (plan_id, feature_key, display_text, is_enabled, sort_order)
SELECT id, 'basic_notes', 'Basic Note Taking', true, 1 
FROM subscription_plans WHERE slug = 'free' OR price = 0
ON CONFLICT DO NOTHING;

INSERT INTO plan_features (plan_id, feature_key, display_text, is_enabled, sort_order)
SELECT id, 'semantic_search', 'Semantic Search', false, 2 
FROM subscription_plans WHERE slug = 'free' OR price = 0
ON CONFLICT DO NOTHING;

INSERT INTO plan_features (plan_id, feature_key, display_text, is_enabled, sort_order)
SELECT id, 'ai_chat', 'AI Chat Assistant', false, 3 
FROM subscription_plans WHERE slug = 'free' OR price = 0
ON CONFLICT DO NOTHING;

-- 4. Insert features for Pro plan
INSERT INTO plan_features (plan_id, feature_key, display_text, is_enabled, sort_order)
SELECT id, 'basic_notes', 'Basic Note Taking', true, 1 
FROM subscription_plans WHERE slug = 'pro' OR name ILIKE '%pro%'
ON CONFLICT DO NOTHING;

INSERT INTO plan_features (plan_id, feature_key, display_text, is_enabled, sort_order)
SELECT id, 'semantic_search', 'Semantic Search', true, 2 
FROM subscription_plans WHERE slug = 'pro' OR name ILIKE '%pro%'
ON CONFLICT DO NOTHING;

INSERT INTO plan_features (plan_id, feature_key, display_text, is_enabled, sort_order)
SELECT id, 'ai_chat', 'AI Chat Assistant', true, 3 
FROM subscription_plans WHERE slug = 'pro' OR name ILIKE '%pro%'
ON CONFLICT DO NOTHING;
```

---

## API Endpoint Summary

### Public Endpoints
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/plans` | Get all plans for pricing modal |

### User Endpoints (Authenticated)
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/user/usage-status` | Get current usage vs limits |
| GET | `/api/payment/status` | Get subscription status |
| GET | `/api/user/profile` | Get user profile |
| PUT | `/api/user/profile` | Update profile |
| DELETE | `/api/user/account` | Delete account |
| POST | `/api/user/avatar` | Upload avatar |
| POST | `/api/user/refund/request` | Request refund |

### Admin Endpoints
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/admin/login` | Admin login |
| GET | `/api/admin/dashboard` | Dashboard stats |
| GET | `/api/admin/users` | List users |
| GET | `/api/admin/plans` | List plans |
| POST | `/api/admin/plans` | Create plan |
| PUT | `/api/admin/plans/:id` | Update plan |
| DELETE | `/api/admin/plans/:id` | Delete plan |
| GET | `/api/admin/plans/:id/features` | Get plan features |
| POST | `/api/admin/plans/:id/features` | Create feature |
| PUT | `/api/admin/plans/features/:featureId` | Update feature |
| DELETE | `/api/admin/plans/features/:featureId` | Delete feature |
| GET | `/api/admin/refunds` | List refunds |
| POST | `/api/admin/refunds/:id/approve` | Approve refund |

---

## Notes for Frontend Team

1. **Always pre-check limits** before allowing create/chat/search actions
2. **Cache usage status** locally and refresh periodically (every 30s or on action)
3. **Handle 429 gracefully** by showing the pricing modal
4. **Feature text is dynamic** - use the `features` array from `/api/plans`, not hardcoded text
5. **Daily limits reset at midnight** - show countdown using `resets_at` field
6. **Use `/api/payment/status`** NOT `/api/user/subscription/status` (doesn't exist)
PS D:\notetaker\notefiber-BE> .\test-usage-api.ps1 -BaseUrl "http://localhost:3000" -Token eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NjYzOTY1OTYsInJvbGUiOiJ1c2VyIiwidXNlcl9pZCI6ImEyYjk0ZjRjLWI2NzQtNDMzYi05MGJlLTY1YTkxYTM3ZTdhMyJ9.yP3Cp7R74xoPjnM7PUSTd21QcNfjM-Jlw0qXcER3pOg                                                                                                                                                                                                    ========== TEST 1: GET /api/plans (Public) ==========                                                                  [INFO] Testing: GET http://localhost:3000/api/plans                                                                    [PASS] GET /api/plans returned successfully
[INFO]   Found 4 plan(s)
[INFO]   - Free Plan (slug: free, price: 0)
[INFO]     Limits: notebooks=3, notes=10
[INFO]   - Enterprise Plan (slug: enterprise, price: 500000)
[INFO]     Limits: notebooks=3, notes=10
[INFO]   - Integration Plan (slug: integration-plan-dea2b270-3f19-4970-a07f-f47e77c434d1, price: 10)
[INFO]     Limits: notebooks=3, notes=10
[INFO]   - Pro Plan (slug: pro, price: 50000)                                                                          [INFO]     Limits: notebooks=3, notes=10                                                                                                                                                                                                      ========== TEST 2: GET /api/user/usage-status (Authenticated) ==========                                               
[INFO] Testing: GET http://localhost:3000/api/user/usage-status
[PASS] GET /api/user/usage-status returned successfully
[INFO]   Plan: Pro Plan (pro)
[INFO]   Storage:
[INFO]     - Notebooks: 2/3 (can_use: True)
[INFO]     - Notes: 0/10 (can_use: True)
[INFO]   Daily Limits:
[INFO]     - AI Chat: 2/0 (can_use: False)
[INFO]     - Semantic Search: 0/0 (can_use: False)
[INFO]   Upgrade Available: False

========== TEST 3: GET /api/payment/status (Subscription Status) ==========
[INFO] Testing: GET http://localhost:3000/api/payment/status
[PASS] GET /api/payment/status returned successfully
[INFO]   Subscription ID: 29512c7c-918f-4bb5-a1ab-935d409156fa
[INFO]   Plan: Pro Plan                                                                                                [INFO]   Status: active                                                                                                [INFO]   Is Active: True                                                                                               [INFO]   Features:                                                                                                     
[INFO]     - AI Chat: True
[INFO]     - Semantic Search: True
[INFO]     - Max Notebooks: 3
[INFO]     - Max Notes/Notebook: 10

========== TEST 4: GET /api/user/subscription/status (Should be 404) ==========
[INFO] Testing: GET http://localhost:3000/api/user/subscription/status
[INFO] GET /api/user/subscription/status returned 404 as expected
[INFO]   Note: This endpoint is documented but not implemented
[INFO]   Use /api/payment/status instead

========== TEST 5: GET /api/user/profile ==========

========== TEST 5: GET /api/user/profile ==========
[INFO] Testing: GET http://localhost:3000/api/user/profile
[PASS] GET /api/user/profile returned successfully
[INFO]   User: zikri@students.amikom.ac.id

========== SUMMARY ==========
[INFO] Test completed. Check the results above for any [FAIL] messages.
[INFO]
[INFO] Endpoints tested:
[INFO]   - GET /api/plans (public)
[INFO]   - GET /api/user/usage-status (authenticated)
[INFO]   User: zikri@students.amikom.ac.id

========== SUMMARY ==========
[INFO] Test completed. Check the results above for any [FAIL] messages.
[INFO]
[INFO] Endpoints tested:
[INFO]   - GET /api/plans (public)
[INFO]   - GET /api/user/usage-status (authenticated)

========== SUMMARY ==========
[INFO] Test completed. Check the results above for any [FAIL] messages.
[INFO]
[INFO] Endpoints tested:
[INFO]   - GET /api/plans (public)
[INFO]   - GET /api/user/usage-status (authenticated)
========== SUMMARY ==========
[INFO] Test completed. Check the results above for any [FAIL] messages.
[INFO]
[INFO] Endpoints tested:
[INFO]   - GET /api/plans (public)
[INFO]   - GET /api/user/usage-status (authenticated)
[INFO] Test completed. Check the results above for any [FAIL] messages.
[INFO]
[INFO] Endpoints tested:
[INFO]   - GET /api/plans (public)
[INFO]   - GET /api/user/usage-status (authenticated)
[INFO]
[INFO] Endpoints tested:
[INFO]   - GET /api/plans (public)
[INFO]   - GET /api/user/usage-status (authenticated)
[INFO] Endpoints tested:
[INFO]   - GET /api/plans (public)
[INFO]   - GET /api/user/usage-status (authenticated)
[INFO]   - GET /api/plans (public)
[INFO]   - GET /api/user/usage-status (authenticated)
[INFO]   - GET /api/user/usage-status (authenticated)
[INFO]   - GET /api/payment/status (authenticated)
[INFO]   - GET /api/user/subscription/status (documented but 404)
[INFO]   - GET /api/user/profile (authenticated)
[INFO]
[INFO] Known Issues:
[INFO]   1. Documentation says /api/user/subscription/status but actual route is /api/payment/status
PS D:\notetaker\notefiber-BE>

