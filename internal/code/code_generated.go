package code

import "fmt"

func (e ErrorCode) Error() string {
	return fmt.Sprintf("error code: %d", int(e))
}

func (e ErrorCode) Message() string {
	switch e {
	case ErrUnknown:
		return "Internal server error"
	case ErrInvalidArg:
		return "Invalid argument"
	case ErrUnauthorized:
		return "Unauthorized"
	case ErrForbidden:
		return "Forbidden"
	case ErrNotFound:
		return "Not found"
	case ErrInternal:
		return "Internal server error"
	case ErrDeadlineExceeded:
		return "Deadline exceeded"
	case ErrUserAlreadyExists:
		return "User already exists"
	case ErrUserNotFound:
		return "User not found"
	case ErrUserPasswordWrong:
		return "Username or password is wrong"
	case ErrUserTokenInvalid:
		return "Token is invalid or expired"
	case ErrUserRefreshTokenInvalid:
		return "Refresh token is invalid or expired"
	case ErrWalletChainNotSupported:
		return "Chain not supported"
	case ErrWalletAddressInvalid:
		return "Invalid wallet address"
	case ErrWalletInsufficientBalance:
		return "Insufficient balance"
	case ErrWalletDailyLimitExceeded:
		return "Daily withdrawal limit exceeded"
	case ErrWalletDuplicateWithdraw:
		return "Duplicate withdrawal request"
	case ErrWalletBroadcastFailed:
		return "Transaction broadcast failed"
	case ErrWalletPendingReview:
		return "Withdrawal submitted, pending admin review"
	case ErrWalletInvalidStatus:
		return "Invalid withdrawal status for this operation"
	default:
		return "Unknown error"
	}
}

func (e ErrorCode) HTTPStatus() int {
	switch e {
	case ErrInvalidArg, ErrWalletChainNotSupported, ErrWalletAddressInvalid,
		ErrWalletInsufficientBalance, ErrWalletDailyLimitExceeded, ErrWalletInvalidStatus:
		return 400
	case ErrUnauthorized, ErrUserPasswordWrong, ErrUserTokenInvalid, ErrUserRefreshTokenInvalid:
		return 401
	case ErrForbidden:
		return 403
	case ErrNotFound, ErrUserNotFound:
		return 404
	case ErrWalletPendingReview:
		return 202
	case ErrDeadlineExceeded:
		return 408
	case ErrUserAlreadyExists:
		return 409
	case ErrWalletDuplicateWithdraw:
		return 429
	default:
		return 500
	}
}