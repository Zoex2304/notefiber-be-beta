# Refund API Documentation
> **For Frontend Team** - Complete API specification for refund feature implementation
## Overview
The refund system follows a **request → approval** workflow:
1. **User** submits refund request via user-side endpoint
2. **Admin** reviews pending refunds in admin panel
3. **Admin** approves refund (manually transfers money outside system)
4. **System** updates subscription status automatically
---
## User-Side Endpoints
### 1. Request Refund
Create a new refund request for a subscription.
| Property | Value |
|----------|-------|
| **Endpoint** | `POST /api/user/refund/request` |
| **Auth** | User JWT Required |
| **Content-Type** | `application/json` |
#### Request Body
```json
{
  "subscription_id": "uuid-string (required)",
  "reason": "string (required) - User's reason for refund"
}
```
#### Success Response (200)
```json
{
  "success": true,
  "message": "Refund request submitted",
  "data": {
    "refund_id": "uuid-string",
    "status": "pending",
    "message": "Your refund request has been submitted and is awaiting admin review."
  }
}
```
#### Error Responses
| Status | Condition | Response |
|--------|-----------|----------|
| 400 | Missing fields | `{"success": false, "error": {"code": 400, "message": "subscription_id and reason are required"}}` |
| 404 | Subscription not found | `{"success": false, "error": {"code": 404, "message": "Subscription not found"}}` |
| 400 | Already refunded | `{"success": false, "error": {"code": 400, "message": "Subscription already refunded"}}` |
| 400 | Not eligible | `{"success": false, "error": {"code": 400, "message": "Subscription not eligible for refund"}}` |
---
### 2. Get My Refund Requests (Optional Future)
List user's own refund requests with status.
| Property | Value |
|----------|-------|
| **Endpoint** | `GET /api/user/refunds` |
| **Auth** | User JWT Required |
#### Success Response (200)
```json
{
  "success": true,
  "message": "User refund requests",
  "data": [
    {
      "id": "uuid-string",
      "subscription_id": "uuid-string",
      "plan_name": "Premium",
      "amount": 99000.00,
      "reason": "No longer need the service",
      "status": "pending",
      "created_at": "2025-12-21T00:00:00Z"
    }
  ]
}
```
---
## Admin-Side Endpoints
### 1. List Refund Requests
Get list of refund requests with filtering by status.
| Property | Value |
|----------|-------|
| **Endpoint** | `GET /admin/refunds` |
| **Auth** | Admin JWT Required |
| **Query Params** | `status`, `page`, `limit` |
#### Query Parameters
| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `status` | string | `"pending"` | Filter: `pending`, `approved`, `rejected`, or empty for all |
| `page` | int | `1` | Page number |
| `limit` | int | `10` | Items per page |
#### Success Response (200)
```json
{
  "success": true,
  "message": "Refund requests",
  "data": [
    {
      "id": "uuid-string",
      "user": {
        "id": "uuid-string",
        "email": "user@example.com",
        "full_name": "John Doe"
      },
      "subscription": {
        "id": "uuid-string",
        "plan_name": "Premium",
        "amount_paid": 99000.00,
        "payment_date": "2025-12-01T00:00:00Z"
      },
      "amount": 99000.00,
      "reason": "No longer need the service",
      "status": "pending",
      "created_at": "2025-12-21T00:00:00Z"
    }
  ]
}
```
---
### 2. Approve/Process Refund
Approve a pending refund request. This will:
- Update refund status to `approved`
- Cancel the user's subscription
- Mark subscription payment as `refunded`
> **Important**: Since Midtrans doesn't support automated refunds, the admin must manually transfer money to the user outside the system.
| Property | Value |
|----------|-------|
| **Endpoint** | `POST /admin/refunds/:id/approve` |
| **Auth** | Admin JWT Required |
| **Content-Type** | `application/json` |
#### URL Parameters
| Param | Type | Description |
|-------|------|-------------|
| `id` | uuid | Refund request ID |
#### Request Body (Optional)
```json
{
  "admin_notes": "string (optional) - Notes for audit trail"
}
```
#### Success Response (200)
```json
{
  "success": true,
  "message": "Refund approved",
  "data": {
    "refund_id": "uuid-string",
    "status": "approved",
    "refunded_amount": 99000.00,
    "processed_at": "2025-12-21T10:30:00Z"
  }
}
```
#### Error Responses
| Status | Condition | Response |
|--------|-----------|----------|
| 404 | Refund not found | `{"success": false, "error": {"code": 404, "message": "Refund request not found"}}` |
| 400 | Already processed | `{"success": false, "error": {"code": 400, "message": "Refund already processed"}}` |
---
### 3. Reject Refund (Optional Future)
Reject a pending refund request with reason.
| Property | Value |
|----------|-------|
| **Endpoint** | `POST /admin/refunds/:id/reject` |
| **Auth** | Admin JWT Required |
#### Request Body
```json
{
  "rejection_reason": "string (required)"
}
```
---
## Data Models
### RefundStatus Enum
```typescript
type RefundStatus = "pending" | "approved" | "rejected";
```
### Refund Object
```typescript
interface Refund {
  id: string;                    // UUID
  subscription_id: string;       // UUID
  user_id: string;               // UUID
  amount: number;                // Refund amount
  reason: string;                // User's reason
  status: RefundStatus;          // Current status
  admin_notes?: string;          // Admin notes (if approved/rejected)
  created_at: string;            // ISO 8601 timestamp
  processed_at?: string;         // When approved/rejected
}
```
---
## Frontend Pages to Create
### 1. User Side: Refund Request Page
**Route**: `/settings/refund` or `/subscription/refund`
**UI Components**:
- Subscription selector (dropdown of active paid subscriptions)
- Reason textarea (required, min 10 characters)
- Submit button
- Success/error toast notifications
**Flow**:
1. User selects subscription
2. User enters reason
3. Submit → Call `POST /api/user/refund/request`
4. Show success message: "Refund request submitted. We'll review within 3 business days."
---
### 2. Admin Side: Refund Management Page
**Route**: `/admin/refunds`
**UI Components**:
- Status filter tabs: All | Pending | Approved | Rejected
- Data table with columns:
  - User (name, email)
  - Plan Name
  - Amount
  - Reason (truncated, expandable)
  - Status (badge: pending=yellow, approved=green, rejected=red)
  - Date
  - Actions
- Pagination
**Actions per row**:
- **Approve** button (opens confirmation modal)
- **View Details** button (opens detail modal)
---
### 3. Admin Side: Refund Detail Modal
**Contents**:
- User info: Name, Email
- Subscription info: Plan, Amount Paid, Start Date
- Refund request: Reason, Date Submitted
- Action buttons: Approve, Reject (future)
**Approve Confirmation**:
```
Are you sure you want to approve this refund?
Amount: Rp 99,000
User: John Doe (john@example.com)
Note: You must manually transfer the refund amount to the user.
[ Cancel ] [ Approve Refund ]
```
---
## Status Workflow Diagram
```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   PENDING   │────▶│  APPROVED   │     │  REJECTED   │
│  (yellow)   │     │   (green)   │     │    (red)    │
└─────────────┘     └─────────────┘     └─────────────┘
       │                                       ▲
       └───────────────────────────────────────┘
                    (future feature)
```
---
## Notes for Frontend Team
1. **No auto-refund**: Money is NOT automatically transferred. Admin must manually process outside the system.
2. **Subscription eligibility**: Only subscriptions with `status: "active"` and `payment_status: "paid"` can request refunds.
3. **One refund per subscription**: Users cannot request multiple refunds for the same subscription.
4. **Reason is required**: Minimum character limit recommended (10+ chars).
5. **Real-time updates**: Consider WebSocket or polling for status updates on user's refund page.
