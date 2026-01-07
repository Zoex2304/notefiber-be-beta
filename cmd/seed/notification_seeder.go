package main

import (
	"log"

	"ai-notetaking-be/internal/model"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// SeedNotificationTypes populates the database with default notification types.
func SeedNotificationTypes(db *gorm.DB) {
	types := []model.NotificationType{
		{
			Code:        "USER_LOGIN",
			DisplayName: "Login Activity",
			Template:    "You logged in from {device} at {time}",
			TargetType:  "SELF",
			Priority:    "LOW",
			IsActive:    true,
			Channels:    datatypes.JSON([]byte(`["web"]`)),
		},
		{
			Code:        "NOTE_CREATED",
			DisplayName: "Note Created",
			Template:    "You created a note: \"{title}\"",
			TargetType:  "SELF",
			Priority:    "LOW",
			IsActive:    true,
			Channels:    datatypes.JSON([]byte(`["web"]`)),
		},
		{
			Code:        "NOTE_UPDATED",
			DisplayName: "Note Updated",
			Template:    "You updated note: \"{title}\"",
			TargetType:  "SELF",
			Priority:    "LOW",
			IsActive:    true,
			Channels:    datatypes.JSON([]byte(`["web"]`)),
		},
		{
			Code:        "NOTE_DELETED",
			DisplayName: "Note Deleted",
			Template:    "You deleted note: \"{title}\"",
			TargetType:  "SELF",
			Priority:    "LOW",
			IsActive:    true,
			Channels:    datatypes.JSON([]byte(`["web"]`)),
		},
		{
			Code:        "TEST_EVENT",
			DisplayName: "Test Notification",
			Template:    "This is a test notification: {message}",
			TargetType:  "SELF",
			Priority:    "MEDIUM",
			IsActive:    true,
			Channels:    datatypes.JSON([]byte(`["web"]`)),
		},
		// --- Administrative & System Notifications ---
		{
			Code:        "USER_REGISTERED",
			DisplayName: "New User Registration",
			Template:    "New user registered: {email} ({user_id})",
			TargetType:  "ADMIN", // Send to all admins
			Priority:    "MEDIUM",
			Channels:    datatypes.JSON([]byte(`["web", "email"]`)),
			IsActive:    true,
		},
		{
			Code:        "USER_DELETED",
			DisplayName: "User Account Deleted",
			Template:    "User deleted account: {user_id}",
			TargetType:  "ADMIN",
			Priority:    "MEDIUM",
			Channels:    datatypes.JSON([]byte(`["web", "email"]`)),
			IsActive:    true,
		},
		{
			Code:        "SUBSCRIPTION_CREATED",
			DisplayName: "New Subscription",
			Template:    "New subscription: {plan_name} for user {user_id}",
			TargetType:  "ADMIN",
			Priority:    "HIGH",
			Channels:    datatypes.JSON([]byte(`["web", "email"]`)),
			IsActive:    true,
		},
		{
			Code:        "REFUND_REQUESTED",
			DisplayName: "Refund Requested",
			Template:    "Refund requested by {user_id} for subscription {subscription_id}. Reason: {reason}",
			TargetType:  "ADMIN",
			Priority:    "HIGH",
			Channels:    datatypes.JSON([]byte(`["web", "email"]`)),
			IsActive:    true,
		},
		{
			Code:        "REFUND_APPROVED",
			DisplayName: "Refund Approved",
			Template:    "Your refund request for subscription {subscription_id} has been processed.",
			TargetType:  "SELF", // Send to the requesting user
			Priority:    "HIGH",
			Channels:    datatypes.JSON([]byte(`["web", "email"]`)),
			IsActive:    true,
		},
		{
			Code:        "REFUND_REJECTED",
			DisplayName: "Refund Rejected",
			Template:    "Your refund request for subscription {subscription_id} has been rejected. Reason: {reason}",
			TargetType:  "SELF", // Send to the requesting user
			Priority:    "HIGH",
			Channels:    datatypes.JSON([]byte(`["web", "email"]`)),
			IsActive:    true,
		},
		{
			Code:        "SYSTEM_BROADCAST",
			DisplayName: "System Announcement",
			Template:    "{message}",
			TargetType:  "BROADCAST", // Special type for all users
			Priority:    "HIGH",
			Channels:    datatypes.JSON([]byte(`["web"]`)),
			IsActive:    true,
		},
		{
			Code:        "AI_LIMIT_UPDATED",
			DisplayName: "AI Limit Updated",
			Template:    "Your AI daily limit has been updated to {limit_description} by an administrator.",
			TargetType:  "SELF", // Send to the affected user
			Priority:    "MEDIUM",
			Channels:    datatypes.JSON([]byte(`["web"]`)),
			IsActive:    true,
		},
		{
			Code:        "SUBSCRIPTION_CANCELLATION_REQUESTED",
			DisplayName: "Cancellation Requested",
			Template:    "Subscription cancellation requested by {user_id} for subscription {subscription_id}. Reason: {reason}",
			TargetType:  "ADMIN",
			Priority:    "HIGH",
			Channels:    datatypes.JSON([]byte(`["web", "email"]`)),
			IsActive:    true,
		},
		{
			Code:        "SUBSCRIPTION_CANCELLATION_PROCESSED",
			DisplayName: "Cancellation Processed",
			Template:    "Your subscription cancellation request for {plan_name} has been {status}.",
			TargetType:  "SELF",
			Priority:    "HIGH",
			Channels:    datatypes.JSON([]byte(`["web", "email"]`)),
			IsActive:    true,
		},
	}

	for _, t := range types {
		// PostgreSQL specific ON CONFLICT to avoid duplicates
		err := db.Where("code = ?", t.Code).FirstOrCreate(&t).Error
		if err != nil {
			log.Printf("Error seeding notification type %s: %v", t.Code, err)
		} else {
			// Optional: Update if exists? For now, just ensure existence.
			// log.Printf("Seeded notification type: %s", t.Code)
		}
	}
	log.Println("✅ Notification types seeded successfully.")
}
