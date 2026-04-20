export const ru = {
  // --- Common ---
  common: {
    loading: "Загрузка...",
    error: "Произошла ошибка. Попробуйте снова.",
    or: "или",
    back: "Назад",
    backToDashboard: "Вернуться на главную",
    copied: "Скопировано",
    copyFailed: "Не удалось скопировать",
  },

  // --- Auth ---
  auth: {
    welcomeBack: "С возвращением",
    signInSubtitle: "Войдите в свой аккаунт",
    signIn: "Войти",
    createAccount: "Создать аккаунт",
    createAccountSubtitle: "Начните работу с MPC-2FA",
    noAccount: "Нет аккаунта?",
    hasAccount: "Уже есть аккаунт?",
    createOne: "Создать",
    signInLink: "Войти",
    email: "Электронная почта",
    emailPlaceholder: "you@example.com",
    password: "Пароль",
    passwordPlaceholder: "Введите пароль",
    confirmPassword: "Подтвердите пароль",
    confirmPasswordPlaceholder: "Повторите пароль",
  },

  // --- Validation ---
  validation: {
    emailRequired: "Введите электронную почту",
    emailInvalid: "Неверный формат почты",
    passwordRequired: "Введите пароль",
    passwordInvalid: "Пароль не соответствует требованиям",
    confirmRequired: "Подтвердите пароль",
    confirmMismatch: "Пароли не совпадают",
  },

  // --- Password Strength ---
  passwordStrength: {
    weak: "Слабый",
    fair: "Средний",
    good: "Хороший",
    strong: "Надёжный",
    minLength: "Минимум 12 символов",
    hasLowercase: "Одна строчная буква",
    hasUppercase: "Одна заглавная буква",
    hasDigit: "Одна цифра",
    hasSpecial: "Один спецсимвол",
    noSequences: "Без последовательностей из 4+ символов",
  },

  // --- API Errors ---
  apiErrors: {
    invalidInput: "Неверные данные. Проверьте введённую информацию.",
    notFound: "Аккаунт не найден.",
    alreadyExists: "Этот email уже зарегистрирован.",
    preconditionFailed: "Действие недоступно. Попробуйте позже.",
    unauthenticated: "Неверные учётные данные.",
    sessionExpired: "Сессия истекла",
    generic: "Что-то пошло не так. Попробуйте снова.",
  },

  // --- Dashboard ---
  dashboard: {
    title: "Главная",
    account: "Аккаунт",
    logoutAll: "Выйти со всех устройств",
    logoutAllFailed: "Не удалось выйти",
  },

  // --- Navbar ---
  navbar: {
    logout: "Выйти",
  },

  // --- 2FA Status ---
  twofa: {
    title: "Двухфакторная аутентификация",
    enabled: "Включена",
    disabled: "Отключена",
    pending: "Ожидание",
    enabledSince: "Включена с",
    addSecurity: "Добавьте дополнительный уровень защиты",
    enable: "Включить",
    disable: "Отключить",
    loadingStatus: "Загрузка статуса 2FA...",
  },

  // --- 2FA Setup ---
  setup: {
    title: "Настройка двухфакторной аутентификации",
    stepScan: "Сканирование",
    stepVerify: "Проверка",
    stepBackup: "Резервные коды",
    settingUp: "Настройка 2FA...",

    // QR step
    scanTitle: "Отсканируйте QR-код",
    scanDescription: "Откройте приложение-аутентификатор и отсканируйте код",
    cantScan: "Не получается отсканировать? Введите код вручную",
    scanned: "Я отсканировал код",

    // Verify step
    verifyTitle: "Проверьте код",
    verifyDescription: "Введите 6-значный код из приложения-аутентификатора",
    verify: "Проверить",
    invalidCode: "Неверный код. Попробуйте снова.",
    verificationFailed: "Ошибка проверки. Попробуйте снова.",
    enabled: "Двухфакторная аутентификация включена!",

    // Backup step
    saveBackupCodes: "Сохраните резервные коды",
    backupWarning: "Эти коды больше не будут показаны. Сохраните их в надёжном месте. Каждый код можно использовать только один раз.",
    copyAll: "Копировать все",
    download: "Скачать",
    savedCodes: "Я сохранил коды",
    backupFileTitle: "MPC-2FA Резервные коды",
    backupFileWarning: "Храните эти коды в надёжном месте.",
    backupFileNote: "Каждый код можно использовать только один раз.",
    backupFileName: "mpc-2fa-резервные-коды.txt",
    backupCodesCopied: "Резервные коды скопированы",
  },

  // --- 2FA Disable ---
  disableModal: {
    title: "Отключить двухфакторную аутентификацию",
    otpPrompt: "Введите 6-значный код из приложения-аутентификатора.",
    backupPrompt: "Введите один из резервных кодов.",
    disableButton: "Отключить 2FA",
    useBackup: "Использовать резервный код",
    useOtp: "Использовать приложение",
    enterOtp: "Введите 6-значный код",
    enterBackup: "Введите резервный код",
    success: "Двухфакторная аутентификация отключена",
    failed: "Не удалось отключить 2FA. Попробуйте снова.",
  },

  // --- 2FA Verify (login) ---
  verifyLogin: {
    title: "Двухфакторная проверка",
    subtitle: "На вашем аккаунте включена 2FA. Введите код для продолжения.",
    description: "Введите 6-значный код из приложения-аутентификатора",
    backupDescription: "Введите один из резервных кодов",
  },

  // --- Theme ---
  theme: {
    light: "Переключить на светлую тему",
    dark: "Переключить на тёмную тему",
  },

  // --- Language ---
  lang: {
    ru: "Русский",
    en: "English",
  },
};

type DeepStringify<T> = {
  [K in keyof T]: T[K] extends string ? string : DeepStringify<T[K]>;
};

export type Translations = DeepStringify<typeof ru>;
