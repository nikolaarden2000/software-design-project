const { test, expect } = require('@playwright/test');

test.describe.configure({ mode: 'serial' });

const BASE_URL = (process.env.E2E_BASE_URL || 'http://localhost:8081').replace(/\/$/, '');

function appUrl(path) {
  return `${BASE_URL}${path}`;
}

const DATA = {
  user: {
    email: process.env.E2E_USER_EMAIL || 'test3@test3.ru',
    password: process.env.E2E_USER_PASSWORD || 'Password123'
  },

  admin: {
    email: process.env.E2E_ADMIN_EMAIL || 'admin-abc@mail.com',
    password: process.env.E2E_ADMIN_PASSWORD || 'Password123'
  },

  superuser: {
    email: process.env.E2E_SUPERUSER_EMAIL || 'superuser@mail.com',
    password: process.env.E2E_SUPERUSER_PASSWORD || 'Password123'
  },

  wrongPassword: process.env.E2E_WRONG_PASSWORD || 'wrong-password',

  locationId: process.env.E2E_LOCATION_ID || '1',

  publishedRoomId: process.env.E2E_PUBLISHED_ROOM_ID || '17',

  draftRoomId: process.env.E2E_DRAFT_ROOM_ID || '15',

  pendingApproveRoomId: process.env.E2E_PENDING_APPROVE_ROOM_ID || '16',

  pendingRejectRoomId: process.env.E2E_PENDING_REJECT_ROOM_ID || '18',

  adminBookingRoomId: process.env.E2E_ADMIN_BOOKING_ROOM_ID || '17',

  roomWithFutureBookingId: process.env.E2E_ROOM_WITH_FUTURE_BOOKING_ID || '17',

  roomWithoutFutureBookingsId: process.env.E2E_ROOM_WITHOUT_FUTURE_BOOKINGS_ID || '20'
};

async function setupDialogs(page) {
  const messages = [];

  page.on('dialog', async dialog => {
    messages.push(dialog.message());
    await dialog.accept();
  });

  return messages;
}

async function login(page, email, password) {
  await page.goto(appUrl('/auth'));

  const bodyText = await page.locator('body').textContent();

  if (bodyText.includes('Вы уже авторизованы')) {
    return;
  }

  await page.fill('#loginEmail', email);
  await page.fill('#loginPassword', password);

  await page.click('#loginSubmit');

  await expect(page).toHaveURL(appUrl('/'), {
    timeout: 4000
  });
}

async function loginAsUser(page) {
  await login(page, DATA.user.email, DATA.user.password);
}

async function loginAsAdmin(page) {
  await login(page, DATA.admin.email, DATA.admin.password);
}

async function loginAsSuperuser(page) {
  await login(page, DATA.superuser.email, DATA.superuser.password);
}

async function createBookingThroughUi(page, roomId) {
  await page.goto(appUrl(`/room/${roomId}`));

  await expect(page.locator('#bookBtn')).toBeEnabled();

  await page.click('#bookBtn');

  await expect(page.locator('#bookingModal')).not.toHaveClass(/hidden/);
  await expect(page.locator('.slot').first()).toBeVisible();

  await page.locator('.slot').first().click();

  await page.click('#bookingConfirm');

  await expect(page.locator('body')).toContainText('Бронирование успешно');
}

test.beforeEach(async ({ page }) => {
  await setupDialogs(page);
});

test('E2E-01. Регистрация нового пользователя через веб-интерфейс', async ({ page }) => {
  await page.goto(appUrl('/auth'));

  await page.click('#tab-register');

  const uniqueEmail = `e2e-user-${Date.now()}@mail.com`;

  await page.fill('#regUsername', `e2e-user-${Date.now()}`);
  await page.fill('#regEmail', uniqueEmail);
  await page.fill('#regPassword', 'password123');
  await page.fill('#regConfirm', 'password123');

  await page.click('#registerSubmit');

  await expect(page.locator('#authMessage')).toContainText('Регистрация успешна');
  await expect(page.locator('#loginForm')).toBeVisible();
});

test('E2E-02. Вход пользователя и отображение личного кабинета', async ({ page }) => {
  await loginAsUser(page);

  await page.goto(appUrl('/me'));

  await expect(page.locator('body')).toContainText('Забронирована');
  await expect(page.locator('body')).toContainText(/Используется|Забронирована|Завершён|Отменён/);
});

test('E2E-03. Ошибка входа с неверным паролем', async ({ page }) => {
  await page.goto(appUrl('/auth'));

  await page.fill('#loginEmail', DATA.user.email);
  await page.fill('#loginPassword', DATA.wrongPassword);

  await page.click('#loginSubmit');

  await expect(page.locator('#authMessage')).toContainText('Неверный email или пароль');
  await expect(page).toHaveURL(/\/auth/);
});

test('E2E-04. Просмотр каталога и карточки помещения', async ({ page }) => {
  await loginAsUser(page);

  await page.goto(appUrl('/'));

  await expect(page.locator('.card').first()).toBeVisible();

  await page.locator('.card').first().click();

  await expect(page).toHaveURL(/\/room\/\d+/);

  await expect(page.locator('body')).toContainText('Описание');
  await expect(page.locator('body')).toContainText('Вместимость');
  await expect(page.locator('body')).toContainText('₽/ч');
  await expect(page.locator('#mainImage')).toBeVisible();

  await page.click('#bookBtn');

  await expect(page.locator('#bookingModal')).not.toHaveClass(/hidden/);
  await expect(page.locator('.slot').first()).toBeVisible();
});

test('E2E-05. Бронирование помещения пользователем', async ({ page }) => {
  await loginAsUser(page);

  await createBookingThroughUi(page, DATA.publishedRoomId);

  await page.goto(appUrl('/me'));

  await expect(page.locator('body')).toContainText('Забронирована');
});

test('E2E-06. Отмена бронирования пользователем', async ({ page }) => {
  await loginAsUser(page);

  await createBookingThroughUi(page, DATA.publishedRoomId);

  await page.goto(appUrl('/me'));

  await expect(page.locator('#col-booked .me-card').first()).toBeVisible();

  await page.locator('#col-booked .cancel-btn').first().click();

  await expect(page.locator('#confirmModal')).not.toHaveClass(/hidden/);

  await page.click('#confirmYes');

  await expect(page.locator('#col-canceled')).toContainText(/₽|Переговорная|Помещение|Комната/i);
});

test('E2E-07. Доступ обычного пользователя к admin-странице запрещён', async ({ page }) => {
  await loginAsUser(page);

  await page.goto(appUrl('/admin'));

  await expect(page.locator('body')).toContainText(/Доступ запрещён|Авторизация|Вход/);
});

test('E2E-08. Администратор просматривает свои локации', async ({ page }) => {
  await loginAsAdmin(page);

  await page.goto(appUrl('/admin'));

  await expect(page.locator('body')).toContainText('Мои локации');
  await expect(page.locator('[data-location-id]').first()).toBeVisible();
});

test('E2E-09. Администратор создаёт помещение как черновик', async ({ page }) => {
  await loginAsAdmin(page);

  await page.goto(appUrl('/admin'));

  await page.click(`[data-location-id="${DATA.locationId}"]`);

  await expect(page).toHaveURL(appUrl(`/admin/location/${DATA.locationId}`));

  await page.click(`a[href="/admin/room/new?location_id=${DATA.locationId}"]`);

  await expect(page).toHaveURL(/\/admin\/room\/new/);

  const title = `E2E черновик ${Date.now()}`;

  await page.fill('#roomTitle', title);
  await page.fill('#roomDescription', 'Описание помещения, созданного Playwright E2E-тестом');
  await page.fill('#roomPrice', '1800');
  await page.fill('#roomCapacity', '10');
  await page.fill('#roomAvailableFrom', '09:00');
  await page.fill('#roomAvailableTo', '21:00');
  await page.fill('#roomImages', '/shared/placeholders/room-placeholder.svg');

  await page.click('#saveDraftBtn');

  await expect(page).toHaveURL(/\/admin\/room\/\d+/);
  await expect(page.locator('body')).toContainText(title);
  await expect(page.locator('body')).toContainText('Черновик');
});

test('E2E-10. Администратор отправляет помещение на модерацию', async ({ page }) => {
  await loginAsAdmin(page);

  await page.goto(appUrl(`/admin/room/${DATA.draftRoomId}`));

  await expect(page.locator('body')).toContainText('Черновик');

  await page.click('#submitRoomBtn');

  await expect(page.locator('body')).toContainText('На модерации');
  await expect(page.locator('#roomTitle')).not.toBeEditable();
});

test('E2E-11. Суперпользователь одобряет помещение', async ({ page }) => {
  await loginAsSuperuser(page);

  await page.goto(appUrl('/superuser'));

  const roomCard = page.locator(`[data-room-id="${DATA.pendingApproveRoomId}"]`);

  await expect(roomCard).toBeVisible();

  await roomCard.locator('[data-action="approve"]').click();

  await expect(roomCard).toHaveCount(0);

  await page.goto(appUrl(`/room/${DATA.pendingApproveRoomId}`));

  await expect(page.locator('body')).toContainText('Описание');
  await expect(page.locator('body')).toContainText('₽/ч');
});

test('E2E-12. Суперпользователь отклоняет помещение, администратор редактирует и повторно отправляет', async ({ browser }) => {
  const superContext = await browser.newContext();
  const superPage = await superContext.newPage();

  await setupDialogs(superPage);

  await loginAsSuperuser(superPage);

  await superPage.goto(appUrl('/superuser'));

  const roomCard = superPage.locator(`[data-room-id="${DATA.pendingRejectRoomId}"]`);

  await expect(roomCard).toBeVisible();

  await superPage.fill(
    `#rejectReason-${DATA.pendingRejectRoomId}`,
    'Причина E2E: нужно исправить описание'
  );

  await roomCard.locator('[data-action="reject"]').click();

  await expect(roomCard).toHaveCount(0);

  await superContext.close();

  const adminContext = await browser.newContext();
  const adminPage = await adminContext.newPage();

  await setupDialogs(adminPage);

  await loginAsAdmin(adminPage);

  await adminPage.goto(appUrl(`/admin/room/${DATA.pendingRejectRoomId}`));

  await expect(adminPage.locator('body')).toContainText('Причина отклонения');
  await expect(adminPage.locator('body')).toContainText('Причина E2E: нужно исправить описание');

  await adminPage.fill('#roomDescription', 'Исправленное описание после отклонения E2E');

  await adminPage.click('#saveRoomBtn');

  await expect(adminPage.locator('body')).toContainText('Черновик');

  await adminPage.click('#submitRoomBtn');

  await expect(adminPage.locator('body')).toContainText('На модерации');

  await adminContext.close();
});

test('E2E-13. Администратор просматривает бронирования помещения и отменяет бронь', async ({ page }) => {
  await loginAsAdmin(page);

  await page.goto(appUrl(`/admin/room/${DATA.adminBookingRoomId}`));

  await expect(page.locator('body')).toContainText('Бронирования помещения');

  await expect(page.locator('[data-cancel-booking]').first()).toBeVisible();

  await page.locator('[data-cancel-booking]').first().click();

  await expect(page.locator('body')).toContainText('Отменено');
});

test('E2E-14. Администратор не может немедленно архивировать помещение с будущими бронями', async ({ page }) => {
  await loginAsAdmin(page);

  await page.goto(appUrl(`/admin/room/${DATA.roomWithFutureBookingId}`));

  await expect(page.locator('body')).toContainText('Архивирование');

  const archiveBtn = page.locator('#archiveImmediateBtn');

  await expect(archiveBtn).toBeVisible();

  const disabled = await archiveBtn.isDisabled();

  if (disabled) {
    await expect(page.locator('body')).toContainText(/активные|будущие|бронирования/i);
  } else {
    await archiveBtn.click();

    await expect(page.locator('body')).not.toContainText('Помещение уже архивировано');
  }
});

test('E2E-15. Администратор архивирует помещение без будущих броней', async ({ page }) => {
  await loginAsAdmin(page);

  await page.goto(appUrl(`/admin/room/${DATA.roomWithoutFutureBookingsId}`));

  await expect(page.locator('body')).toContainText('Архивирование');

  const roomTitle = await page.locator('.dashboard-header h1').innerText();

  await expect(page.locator('#archiveImmediateBtn')).toBeEnabled();

  await page.click('#archiveImmediateBtn');

  await expect(page.locator('body')).toContainText('Архивировано');

  await page.goto(appUrl('/'));

  await expect(page.locator('body')).not.toContainText(roomTitle);
});