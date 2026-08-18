package app

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"syscall"

	"mc-server-checker/internal/domain"
	"mc-server-checker/internal/protocol/java"
)

func MapError(ctx context.Context, err error) *domain.AppError {
	if err == nil {
		return nil
	}
	var appError *domain.AppError
	if errors.As(err, &appError) {
		return appError
	}
	if errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, context.Canceled) {
		return classifiedError(domain.ErrorCancelled, "確認を中止しました。", err, false)
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		return classifiedError(domain.ErrorTimeout, "時間内に応答がありませんでした。", err, true)
	}
	var networkError net.Error
	if errors.As(err, &networkError) && networkError.Timeout() {
		return classifiedError(domain.ErrorTimeout, "時間内に応答がありませんでした。", err, true)
	}
	var dnsError *net.DNSError
	if errors.As(err, &dnsError) {
		return classifiedError(domain.ErrorDNS, "ホスト名を解決できません。", err, true)
	}
	if errors.Is(err, syscall.ECONNREFUSED) {
		return classifiedError(domain.ErrorRefused, "接続が拒否されました。ポートを確認してください。", err, true)
	}
	if isPayloadError(err) {
		return classifiedError(domain.ErrorPayload, "サーバー応答が不正または大きすぎます。", err, false)
	}
	if isProtocolError(err) {
		return classifiedError(domain.ErrorProtocol, "Minecraft Status応答を解釈できません。", err, false)
	}
	return classifiedError(domain.ErrorNetwork, "ネットワークに接続できませんでした。", err, true)
}

func isPayloadError(err error) bool {
	var syntaxError *json.SyntaxError
	var typeError *json.UnmarshalTypeError
	return errors.Is(err, java.ErrPacketTooLarge) ||
		errors.Is(err, java.ErrStringTooLarge) ||
		errors.Is(err, java.ErrInvalidUTF8) ||
		errors.Is(err, java.ErrInvalidPlayerCount) ||
		errors.Is(err, java.ErrInvalidMOTDType) ||
		errors.Is(err, java.ErrMOTDTooComplex) ||
		errors.As(err, &syntaxError) ||
		errors.As(err, &typeError)
}

func isProtocolError(err error) bool {
	return errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, java.ErrVarIntTooLong) ||
		errors.Is(err, java.ErrVarIntOutOfRange) ||
		errors.Is(err, java.ErrNegativeLength) ||
		errors.Is(err, java.ErrEmptyPacket) ||
		errors.Is(err, java.ErrUnexpectedPacket) ||
		errors.Is(err, java.ErrTrailingPacketData) ||
		errors.Is(err, java.ErrPongNonceMismatch)
}

func classifiedError(kind domain.ErrorKind, message string, cause error, retryable bool) *domain.AppError {
	return &domain.AppError{Kind: kind, Message: message, Cause: cause, Retryable: retryable}
}

func statusForError(err *domain.AppError) domain.Status {
	switch err.Kind {
	case domain.ErrorDNS, domain.ErrorTimeout, domain.ErrorRefused, domain.ErrorNetwork:
		return domain.StatusOffline
	case domain.ErrorCancelled:
		return domain.StatusUnknown
	default:
		return domain.StatusError
	}
}
