// --- Auth Models ---

export interface TokenPair {
  accessToken: string;
  refreshToken: string;
}

export interface User {
  id: string;
  email: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  tokens: TokenPair;
  user: User;
}

export interface RegisterRequest {
  email: string;
  password: string;
}

export interface RegisterResponse {
  tokens: TokenPair;
  user: User;
}

export interface RefreshTokenRequest {
  refreshToken: string;
}

export interface RefreshTokenResponse {
  tokens: TokenPair;
}

export interface ValidateTokenRequest {
  accessToken: string;
}

export interface ValidateTokenResponse {
  userId: string;
  email: string;
}

export interface LogoutRequest {
  refreshToken: string;
}

export interface LogoutAllRequest {
  userId: string;
}

// --- 2FA Models ---

export interface Setup2FARequest {
  userId: string;
  email: string;
}

export interface Setup2FAResponse {
  provisioningUri: string;
  backupCodes: string[];
}

export interface Verify2FARequest {
  userId: string;
  otpCode: string;
}

export interface Verify2FAResponse {
  valid: boolean;
  isNewlyEnabled: boolean;
}

export interface Get2FAStatusResponse {
  isEnabled: boolean;
  createdAt: string;
}

export interface Disable2FARequest {
  userId: string;
  otpCode: string;
}

// --- API Error ---

export interface ApiError {
  code: number;
  message: string;
  details?: Array<{ "@type": string; [key: string]: unknown }>;
}
