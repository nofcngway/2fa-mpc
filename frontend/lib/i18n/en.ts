import type { Translations } from "./ru";

export const en: Translations = {
  common: {
    loading: "Loading...",
    error: "Something went wrong. Please try again.",
    or: "or",
    back: "Back",
    backToDashboard: "Back to Dashboard",
    copied: "Copied",
    copyFailed: "Failed to copy",
  },

  auth: {
    welcomeBack: "Welcome back",
    signInSubtitle: "Sign in to your account",
    signIn: "Sign in",
    createAccount: "Create account",
    createAccountSubtitle: "Get started with MPC-2FA",
    noAccount: "Don't have an account?",
    hasAccount: "Already have an account?",
    createOne: "Create one",
    signInLink: "Sign in",
    email: "Email",
    emailPlaceholder: "you@example.com",
    password: "Password",
    passwordPlaceholder: "Enter your password",
    confirmPassword: "Confirm password",
    confirmPasswordPlaceholder: "Repeat your password",
  },

  validation: {
    emailRequired: "Email is required",
    emailInvalid: "Invalid email format",
    passwordRequired: "Password is required",
    passwordInvalid: "Password does not meet requirements",
    confirmRequired: "Please confirm your password",
    confirmMismatch: "Passwords do not match",
  },

  passwordStrength: {
    weak: "Weak",
    fair: "Fair",
    good: "Good",
    strong: "Strong",
    minLength: "At least 12 characters",
    hasLowercase: "One lowercase letter",
    hasUppercase: "One uppercase letter",
    hasDigit: "One digit",
    hasSpecial: "One special character",
    noSequences: "No 4+ character sequences",
  },

  apiErrors: {
    invalidInput: "Invalid input. Please check your data.",
    notFound: "Account not found.",
    alreadyExists: "This email is already registered.",
    preconditionFailed: "Action not allowed. Please try again later.",
    unauthenticated: "Invalid credentials.",
    sessionExpired: "Session expired",
    generic: "Something went wrong. Please try again.",
  },

  dashboard: {
    title: "Dashboard",
    account: "Account",
    logoutAll: "Logout all devices",
    logoutAllFailed: "Failed to logout",
  },

  navbar: {
    logout: "Logout",
  },

  twofa: {
    title: "Two-Factor Authentication",
    enabled: "Enabled",
    disabled: "Disabled",
    pending: "Pending",
    enabledSince: "Enabled since",
    addSecurity: "Add an extra layer of security to your account",
    enable: "Enable",
    disable: "Disable",
    loadingStatus: "Loading 2FA status...",
  },

  setup: {
    title: "Setup Two-Factor Authentication",
    stepScan: "Scan",
    stepVerify: "Verify",
    stepBackup: "Backup",
    settingUp: "Setting up 2FA...",

    scanTitle: "Scan QR Code",
    scanDescription: "Open your authenticator app and scan this code",
    cantScan: "Can't scan? Enter code manually",
    scanned: "I've scanned the code",

    verifyTitle: "Verify your code",
    verifyDescription: "Enter the 6-digit code from your authenticator app",
    verify: "Verify",
    invalidCode: "Invalid code. Please try again.",
    verificationFailed: "Verification failed. Try again.",
    enabled: "Two-factor authentication enabled!",

    saveBackupCodes: "Save your backup codes",
    backupWarning: "These codes won't be shown again. Store them in a safe place. Each code can only be used once.",
    copyAll: "Copy all",
    download: "Download",
    savedCodes: "I've saved my codes",
    backupFileTitle: "MPC-2FA Backup Codes",
    backupFileWarning: "Keep these codes in a safe place.",
    backupFileNote: "Each code can only be used once.",
    backupFileName: "mpc-2fa-backup-codes.txt",
    backupCodesCopied: "Backup codes copied",
  },

  disableModal: {
    title: "Disable Two-Factor Authentication",
    otpPrompt: "Enter the 6-digit code from your authenticator app.",
    backupPrompt: "Enter one of your backup codes.",
    disableButton: "Disable 2FA",
    useBackup: "Use a backup code instead",
    useOtp: "Use authenticator app",
    enterOtp: "Enter your 6-digit code",
    enterBackup: "Enter your backup code",
    success: "Two-factor authentication disabled",
    failed: "Failed to disable 2FA. Try again.",
  },

  verifyLogin: {
    title: "Two-Factor Verification",
    subtitle: "Your account has 2FA enabled. Enter the code to continue.",
    description: "Enter the 6-digit code from your authenticator app",
    backupDescription: "Enter one of your backup codes",
  },

  theme: {
    light: "Switch to light mode",
    dark: "Switch to dark mode",
  },

  lang: {
    ru: "Русский",
    en: "English",
  },
};
