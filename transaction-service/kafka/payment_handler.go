package kafka

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"time"
)

// === payload dari payment-service ===
type PaymentPaidEvent struct {
	EventType string `json:"event_type"`
	Data      struct {
		PaymentID     uint32 `json:"payment_id"`
		TransactionID uint32 `json:"transaction_id"`
		UserID        uint32 `json:"user_id"`
		Amount        int64  `json:"amount"`
		PaidAt        string `json:"paid_at"`
	} `json:"data"`
}

// === handler factory ===
func PaymentPaidHandler(db *sql.DB) func([]byte) {
	return func(msg []byte) {
		log.Printf("📥 payment.paid received: %s", string(msg))

		var event PaymentPaidEvent
		if err := json.Unmarshal(msg, &event); err != nil {
			log.Printf("❌ invalid payment.paid payload: %v", err)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		query := `
			UPDATE transactions
			SET status='paid', paid_at=$1
			WHERE id=$2 AND status!='paid'
		`

		res, err := db.ExecContext(
			ctx,
			query,
			event.Data.PaidAt,
			event.Data.TransactionID,
		)
		if err != nil {
			log.Printf("❌ failed update transaction %d: %v",
				event.Data.TransactionID, err)
			return
		}

		rows, _ := res.RowsAffected()
		if rows == 0 {
			log.Printf("⚠ transaction %d already paid / not found",
				event.Data.TransactionID)
			return
		}

		log.Printf(
			"✅ transaction %d marked PAID (payment_id=%d)",
			event.Data.TransactionID,
			event.Data.PaymentID,
		)
	}
}
