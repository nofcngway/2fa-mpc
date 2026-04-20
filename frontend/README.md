# Frontend

Web-интерфейс для MPC-2FA — Next.js 16 приложение с Apple-inspired Liquid Glass дизайном.

## Стек

| Компонент | Версия |
|-----------|--------|
| Next.js | 16.2.4 |
| React | 19.2.4 |
| TypeScript | 5 |
| Tailwind CSS | 4 |
| HeroUI | 3.0.3 |
| next-themes | 0.4.6 |

## Архитектура

```
app/
├── (auth)/               # Публичные страницы (login, register, 2fa/verify)
├── (protected)/          # Защищенные страницы (dashboard, 2fa/setup)
├── api/auth/session/     # Cookie API для refresh-токенов
└── layout.tsx            # Root layout + ThemeProvider

components/
├── ui/                   # Атомы — переиспользуемые UI-примитивы (11 компонентов)
└── widgets/              # Виджеты — составные блоки интерфейса (9 компонентов)

hooks/                    # use-auth, use-2fa
lib/                      # API client, auth, types, utils
```

### Компоненты (Atoms)

| Компонент | Назначение |
|-----------|------------|
| `GlassCard` | Стеклянный контейнер (default / elevated / flat) |
| `GlassInput` | Текстовое поле с glass-стилем |
| `GlassButton` | Кнопка с вариантами (primary / secondary / ghost / danger) |
| `PasswordInput` | Поле пароля с toggle visibility |
| `PasswordStrength` | Индикатор сложности пароля |
| `OTPInput` | 6-значный OTP-ввод (HeroUI InputOTP) |
| `BackupCodeInput` | Ввод backup-кода |
| `ThemeToggle` | Переключение light/dark |
| `Logo` | Логотип MPC-2FA |
| `StatusBadge` | Статус 2FA (enabled / disabled) |
| `LoadingSpinner` | Индикатор загрузки |

### Виджеты

| Виджет | Назначение |
|--------|------------|
| `LoginForm` | Форма входа |
| `RegisterForm` | Форма регистрации с валидацией пароля |
| `Navbar` | Навигация с темой и logout |
| `TwoFAStatusCard` | Карточка статуса 2FA |
| `TwoFASetupWizard` | Мастер настройки 2FA (QR -> Verify -> Backup) |
| `DisableTwoFAModal` | Модальное окно отключения 2FA (OTP / backup code) |
| `QRCodeDisplay` | Отображение QR-кода |
| `OTPVerifyForm` | Форма верификации OTP |
| `BackupCodesDisplay` | Отображение backup-кодов |

## Страницы

| Страница | Маршрут | Описание |
|----------|---------|----------|
| Login | `/login` | Вход по email + пароль |
| Register | `/register` | Регистрация |
| 2FA Verify | `/2fa/verify` | Верификация OTP после логина (если 2FA включена) |
| Dashboard | `/dashboard` | Информация об аккаунте + управление 2FA |
| 2FA Setup | `/2fa/setup` | Настройка 2FA: QR-код -> верификация -> backup-коды |

## Потоки

### Аутентификация

```
Login -> POST /api/v1/auth/login -> получить токены
  |-- 2FA выключена -> /dashboard
  '-- 2FA включена -> /2fa/verify -> POST /api/v1/2fa/verify -> /dashboard
```

### Токены

- **Access token** -- хранится в памяти (не в localStorage)
- **Refresh token** -- httpOnly cookie через `/api/auth/session`
- Автоматический refresh при 401

## Дизайн-система

**Liquid Glass** -- полупрозрачные стеклянные поверхности с backdrop-blur, мягкими тенями, световыми акцентами на верхних гранях и фиолетовым accent-цветом.

- Цвета: oklch color space, CSS-переменные (`--glass-bg`, `--accent`, `--glass-blur-*`)
- Темы: light + dark, переключение через next-themes
- Радиусы: `rounded-3xl` (карточки), `rounded-xl` (инпуты/кнопки)
- Типографика: системные шрифты (SF Pro / Segoe UI)

## Разработка

```bash
yarn install
yarn dev          # http://localhost:3000
yarn build        # production build
yarn start        # production server
```

### Переменные окружения

| Переменная | По умолчанию | Описание |
|------------|-------------|----------|
| `NEXT_PUBLIC_API_URL` | `http://localhost:8080` | URL API Gateway |

## Docker

```bash
docker build -t mpc-2fa-frontend .
docker run -p 3000:3000 -e NEXT_PUBLIC_API_URL=http://gateway:8080 mpc-2fa-frontend
```

В составе общего docker-compose:

```bash
# из корня проекта
docker compose up -d frontend
```
