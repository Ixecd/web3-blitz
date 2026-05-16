package audit

import (
	"encoding/json"
	"log/slog"
	"os"
	"sync"
	"time"
)

type Event struct {
	Timestamp string `json:"timestamp"`
	Action    string `json:"action"`
	UserID    string `json:"user_id"`
	Chain     string `json:"chain,omitempty"`
	Amount    string `json:"amount,omitempty"`
	ToAddress string `json:"to_address,omitempty"`
	TxID      string `json:"tx_id,omitempty"`
	Result    string `json:"result"`
	Detail    string `json:"detail,omitempty"`
}

type Logger struct {
	mu   sync.Mutex
	file *os.File
}

var defaultLogger *Logger

func Init(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defaultLogger = &Logger{file: f}
	return nil
}

func (l *Logger) Log(ev Event) {
	ev.Timestamp = time.Now().UTC().Format(time.RFC3339)
	l.mu.Lock()
	json.NewEncoder(l.file).Encode(ev)
	l.mu.Unlock()
	slog.Info("audit", "action", ev.Action, "user_id", ev.UserID, "result", ev.Result)
}

// 便捷方法
func WithdrawSubmitted(userID, chain, amount, toAddr string) {
	if defaultLogger == nil {
		return
	}
	defaultLogger.Log(Event{
		Action:    "withdraw.submitted",
		UserID:    userID,
		Chain:     chain,
		Amount:    amount,
		ToAddress: toAddr,
		Result:    "pending",
	})
}

func WithdrawCompleted(userID, chain, amount, txID string) {
	if defaultLogger == nil {
		return
	}
	defaultLogger.Log(Event{
		Action: "withdraw.completed",
		UserID: userID,
		Chain:  chain,
		Amount: amount,
		TxID:   txID,
		Result: "completed",
	})
}

func WithdrawFailed(userID, chain, amount, detail string) {
	if defaultLogger == nil {
		return
	}
	defaultLogger.Log(Event{
		Action: "withdraw.failed",
		UserID: userID,
		Chain:  chain,
		Amount: amount,
		Result: "failed",
		Detail: detail,
	})
}

func WithdrawApproved(userID, chain, amount, txID, adminID string) {
	if defaultLogger == nil {
		return
	}
	defaultLogger.Log(Event{
		Action:    "withdraw.approved",
		UserID:    userID,
		Chain:     chain,
		Amount:    amount,
		TxID:      txID,
		Result:    "approved",
		Detail:    "admin:" + adminID,
	})
}

func WithdrawRejected(userID, chain, amount, adminID, reason string) {
	if defaultLogger == nil {
		return
	}
	defaultLogger.Log(Event{
		Action: "withdraw.rejected",
		UserID: userID,
		Chain:  chain,
		Amount: amount,
		Result: "rejected",
		Detail: "admin:" + adminID + " reason:" + reason,
	})
}

func SweepExecuted(chain, amount, txID, coldAddr string) {
	if defaultLogger == nil {
		return
	}
	defaultLogger.Log(Event{
		Action:    "sweep.executed",
		UserID:    "system",
		Chain:     chain,
		Amount:    amount,
		ToAddress: coldAddr,
		TxID:      txID,
		Result:    "success",
	})
}

func SweepFailed(chain, amount, detail string) {
	if defaultLogger == nil {
		return
	}
	defaultLogger.Log(Event{
		Action: "sweep.failed",
		UserID: "system",
		Chain:  chain,
		Amount: amount,
		Result: "failed",
		Detail: detail,
	})
}

func Close() {
	if defaultLogger != nil {
		defaultLogger.file.Close()
	}
}
