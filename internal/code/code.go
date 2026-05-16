package code

type ErrorCode int

const (
	// 通用错误
	ErrUnknown          ErrorCode = iota + 100000 // ErrUnknown - 500: Internal server error.
	ErrInvalidArg                                 // ErrInvalidArg - 400: Invalid argument.
	ErrUnauthorized                               // ErrUnauthorized - 401: Unauthorized.
	ErrForbidden                                  // ErrForbidden - 403: Forbidden.
	ErrNotFound                                   // ErrNotFound - 404: Not found.
	ErrInternal                                   // ErrInternal - 500: Internal server error.
	ErrDeadlineExceeded                           // ErrDeadlineExceeded - 408: Deadline exceeded.
)

const (
	// 用户相关
	ErrUserAlreadyExists ErrorCode = iota + 101000 // ErrUserAlreadyExists - 409: User already exists.
	ErrUserNotFound                                // ErrUserNotFound - 404: User not found.
	ErrUserPasswordWrong                           // ErrUserPasswordWrong - 401: Username or password is wrong.
	ErrUserTokenInvalid                            // ErrUserTokenInvalid - 401: Token is invalid or expired.
	ErrUserRefreshTokenInvalid                     // ErrUserRefreshTokenInvalid - 401: Refresh token is invalid or expired.
)

const (
	// 钱包相关
	ErrWalletChainNotSupported ErrorCode = iota + 102000 // ErrWalletChainNotSupported - 400: Chain not supported.
	ErrWalletAddressInvalid                              // ErrWalletAddressInvalid - 400: Invalid wallet address.
	ErrWalletInsufficientBalance                         // ErrWalletInsufficientBalance - 400: Insufficient balance.
	ErrWalletDailyLimitExceeded                          // ErrWalletDailyLimitExceeded - 400: Daily withdrawal limit exceeded.
	ErrWalletDuplicateWithdraw                           // ErrWalletDuplicateWithdraw - 429: Duplicate withdrawal request.
	ErrWalletBroadcastFailed                             // ErrWalletBroadcastFailed - 500: Transaction broadcast failed.
	ErrWalletPendingReview                               // ErrWalletPendingReview - 202: Withdrawal submitted, pending admin review.
	ErrWalletInvalidStatus                               // ErrWalletInvalidStatus - 400: Invalid withdrawal status for this operation.
)