# Design Document: Consumer Optimization & Graceful Shutdown Fixes

This document outlines the design and implementation details for optimizing RabbitMQ consumers and their entry points in the `bluebell` project.

## 1. Objectives

- **Fix Double Acknowledgment Bug**: Modify `es_consumer.go` so that the message acknowledgement is not handled inside `handleDelivery` but rather delegated entirely to the caller loop in `Start`.
- **Handle String Parsing Errors**: Modify `vote_consumer.go` to check error returns from string parsing to prevent processing messages with default values.
- **Implement Graceful Shutdown**: Update the `main.go` entrypoints for both the `sync` and `vote` consumers to wait for running operations to complete before terminating.

---

## 2. Detailed Modifications

### 2.1 ES Consumer (`internal/infrastructure/mq/es_consumer.go`)
- Remove calls to `d.Ack(false)` and `d.Nack(false, false)` inside `handleDelivery`.
- If `json.Unmarshal` fails, return a wrapped error: `fmt.Errorf("es_consumer: 反序列化消息失败: %w", err)`.
- Allow the `Start` loop to perform Ack on success (`err == nil`) and Nack on failure (`err != nil`).

### 2.2 Vote Consumer (`internal/infrastructure/mq/vote_consumer.go`)
- Check the error returned from `strconv.ParseInt` for `msg.UserID` and `msg.PostID`.
- Return an error from `handleDelivery` if parsing fails, ensuring the message is properly Nacked.

### 2.3 Sync/Vote Main Entrypoints (`cmd/consumer/sync/main.go` and `cmd/consumer/vote/main.go`)
- Use a `sync.WaitGroup` to track the worker goroutine executing the consumer.
- When a shutdown signal is received:
  1. Invoke the context `cancel()` function.
  2. Call `wg.Wait()` to block the main thread until the consumer loop returns.
  3. Close resources (channels, connections, databases) in deferred functions.
- Downgrade shutdown-induced context-cancellation consumer exit errors from `zap.L().Error` to `zap.L().Info`.

---

## 3. Verification Plan

- Run `go test ./...` to verify there are no compilation errors.
- Validate that both consumer entrypoints compile successfully.
