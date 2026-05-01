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

  catalogRoomId: process.env.E2E_CATALOG_ROOM_ID || '1',
  catalogRoomTitle: 'E2E каталог переговорная',

  bookingRoomId: process.env.E2E_BOOKING_ROOM_ID || '2',
  cancelBookingRoomId: process.env.E2E_CANCEL_BOOKING_ROOM_ID || '3',

  draftRoomId: process.env.E2E_DRAFT_ROOM_ID || '4',
  pendingApproveRoomId: process.env.E2E_PENDING_APPROVE_ROOM_ID || '5',
  pendingRejectRoomId: process.env.E2E_PENDING_REJECT_ROOM_ID || '6',

  adminBookingRoomId: process.env.E2E_ADMIN_BOOKING_ROOM_ID || '7',
  roomWithFutureBookingId: process.env.E2E_ROOM_WITH_FUTURE_BOOKING_ID || '8',
  roomWithoutFutureBookingsId: process.env.E2E_ROOM_WITHOUT_FUTURE_BOOKINGS_ID || '9'
};

const dialogMessagesByPage = new WeakMap();

async function setupDialogs(page) {
  const messages = [];

  dialogMessagesByPage.set(page, messages);

  page.on('dialog', async dialog => {
    messages.push(dialog.message());

    try {
      await dialog.accept();
    } catch {
      // Диалог мог быть уже закрыт.
    }
  });

  return messages;
}

function getDialogMessages(page) {
  return dialogMessagesByPage.get(page) || [];
}

async function gotoApp(page, path) {
  await page.goto(appUrl(path), {
    waitUntil: 'domcontentloaded'
  });
}

async function clearSession(page) {
  await page.context().clearCookies();

  await gotoApp(page, '/');

  await page.evaluate(() => {
    localStorage.clear();
    sessionStorage.clear();
  }).catch(() => {});
}

async function login(page, email, password) {
  await clearSession(page);

  await gotoApp(page, '/auth');

  await expect(page.locator('#loginEmail')).toBeVisible();

  await page.fill('#loginEmail', email);
  await page.fill('#loginPassword', password);

  await page.click('#loginSubmit');

  await expect(page).toHaveURL(appUrl('/'), {
    timeout: 12000
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
  await gotoApp(page, `/room/${roomId}`);

  await expect(page.locator('.room-info h1')).toBeVisible();
  await expect(page.locator('#bookBtn')).toBeEnabled();

  const roomTitle = (await page.locator('.room-info h1').innerText()).trim();

  await page.click('#bookBtn');

  await expect(page.locator('#bookingModal')).not.toHaveClass(/hidden/);

  const firstSlot = page.locator('.slot').first();

  await expect(firstSlot).toBeVisible({
    timeout: 12000
  });

  const selectedDate = await page.locator('.day.selected').getAttribute('data-ymd');
  const slotText = (await firstSlot.innerText()).trim();

  await firstSlot.click();

  await page.click('#bookingConfirm');

  await expect(page.locator('body')).toContainText('Бронирование успешно');

  return {
    roomTitle,
    date: selectedDate || '',
    slotText
  };
}

function findBookedCard(page, booking) {
  return page
    .locator('#col-booked .me-card')
    .filter({ hasText: booking.roomTitle })
    .filter({ hasText: booking.date })
    .first();
}

test.beforeEach(async ({ page }) => {
  await setupDialogs(page);
});

test('E2E-01. Пользователь регистрируется через веб-интерфейс', async ({ page }) => {
  await gotoApp(page, '/auth');

  await page.click('#tab-register');

  const unique = Date.now();
  const uniqueEmail = `e2e-user-${unique}@mail.com`;

  await page.fill('#regUsername', `e2e-user-${unique}`);
  await page.fill('#regEmail', uniqueEmail);
  await page.fill('#regPassword', 'Password123');
  await page.fill('#regConfirm', 'Password123');

  await page.click('#registerSubmit');

  await expect(page.locator('#authMessage')).toContainText('Регистрация успешна');
  await expect(page.locator('#loginForm')).toBeVisible();
});

test('E2E-02. Авторизованный пользователь открывает личный кабинет', async ({ page }) => {
  await loginAsUser(page);

  await gotoApp(page, '/me');

  await expect(page.locator('body')).toContainText('Используется');
  await expect(page.locator('body')).toContainText('Забронирована');
  await expect(page.locator('body')).toContainText('Завершён');
  await expect(page.locator('body')).toContainText('Отменён');
  await expect(page.locator('#logoutBtn')).toBeVisible();
});

test('E2E-03. Пользователь видит ошибку при входе с неверным паролем', async ({ page }) => {
  await gotoApp(page, '/auth');

  await page.fill('#loginEmail', DATA.user.email);
  await page.fill('#loginPassword', DATA.wrongPassword);

  await page.click('#loginSubmit');

  await expect(page.locator('#authMessage')).toContainText('Неверный email или пароль');
  await expect(page).toHaveURL(/\/auth/);
});

test('E2E-04. Пользователь открывает каталог, карточку помещения и окно бронирования', async ({ page }) => {
  await loginAsUser(page);

  await gotoApp(page, '/');

  const card = page
    .locator('.card')
    .filter({ hasText: DATA.catalogRoomTitle })
    .first();

  await expect(card).toBeVisible();

  await card.click();

  await expect(page).toHaveURL(new RegExp(`/room/${DATA.catalogRoomId}$`));

  await expect(page.locator('body')).toContainText('Описание');
  await expect(page.locator('body')).toContainText('Вместимость');
  await expect(page.locator('body')).toContainText('₽/ч');
  await expect(page.locator('#mainImage')).toBeVisible();

  await page.click('#bookBtn');

  await expect(page.locator('#bookingModal')).not.toHaveClass(/hidden/);
  await expect(page.locator('.slot').first()).toBeVisible();
});

test('E2E-05. Пользователь бронирует опубликованное помещение и видит бронь в личном кабинете', async ({ page }) => {
  await loginAsUser(page);

  const booking = await createBookingThroughUi(page, DATA.bookingRoomId);

  await gotoApp(page, '/me');

  const bookedCard = findBookedCard(page, booking);

  await expect(bookedCard).toBeVisible();
  await expect(bookedCard).toContainText(booking.roomTitle);
  await expect(bookedCard).toContainText(booking.date);
});

test('E2E-06. Пользователь отменяет созданную бронь в личном кабинете', async ({ page }) => {
  await loginAsUser(page);

  const booking = await createBookingThroughUi(page, DATA.cancelBookingRoomId);

  await gotoApp(page, '/me');

  const bookedCard = findBookedCard(page, booking);

  await expect(bookedCard).toBeVisible();

  await bookedCard.locator('.cancel-btn').click();

  await expect(page.locator('#confirmModal')).not.toHaveClass(/hidden/);

  await page.click('#confirmYes');

  const canceledCard = page
    .locator('#col-canceled .me-card')
    .filter({ hasText: booking.roomTitle })
    .filter({ hasText: booking.date })
    .first();

  await expect(canceledCard).toBeVisible();
});

test('E2E-07. Обычный пользователь не получает доступ к административной панели', async ({ page }) => {
  await loginAsUser(page);

  await gotoApp(page, '/admin');

  await expect(page.locator('body')).toContainText(/Нет доступа|Доступ запрещён|Вход|Авторизация/i);
});

test('E2E-08. Администратор открывает список своих локаций', async ({ page }) => {
  await loginAsAdmin(page);

  await gotoApp(page, '/admin');

  await expect(page.locator('body')).toContainText('Мои локации');
  await expect(page.locator(`[data-location-id="${DATA.locationId}"]`)).toBeVisible();
});

test('E2E-09. Администратор создаёт помещение в статусе черновика', async ({ page }) => {
  await loginAsAdmin(page);

  await gotoApp(page, `/admin/room/new?location_id=${DATA.locationId}`);

  const title = `E2E новый черновик ${Date.now()}`;

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

test('E2E-09A. Администратор фильтрует помещения локации по статусу', async ({ page }) => {
  await loginAsAdmin(page);

  await gotoApp(page, `/admin/location/${DATA.locationId}`);

  await expect(page.locator('#statusFilter')).toBeVisible();

  await page.selectOption('#statusFilter', 'draft');

  await expect(page.locator(`[data-room-id="${DATA.draftRoomId}"]`)).toBeVisible();
  await expect(page.locator(`[data-room-id="${DATA.pendingApproveRoomId}"]`)).toHaveCount(0);

  await page.selectOption('#statusFilter', 'pending');

  await expect(page.locator(`[data-room-id="${DATA.pendingApproveRoomId}"]`)).toBeVisible();
  await expect(page.locator(`[data-room-id="${DATA.draftRoomId}"]`)).toHaveCount(0);
});

test('E2E-10. Администратор отправляет черновик помещения на модерацию', async ({ page }) => {
  await loginAsAdmin(page);

  await gotoApp(page, `/admin/room/${DATA.draftRoomId}`);

  await expect(page.locator('body')).toContainText('Черновик');

  await page.click('#submitRoomBtn');

  await expect(page.locator('body')).toContainText('На модерации');
  await expect(page.locator('#roomTitle')).not.toBeEditable();
});

test('E2E-11. Суперпользователь одобряет помещение, находящееся на модерации', async ({ page }) => {
  await loginAsSuperuser(page);

  await gotoApp(page, '/superuser');

  const roomCard = page.locator(`[data-room-id="${DATA.pendingApproveRoomId}"]`);

  await expect(roomCard).toBeVisible();

  await roomCard.locator('[data-action="approve"]').click();

  await expect(roomCard).toHaveCount(0);

  await gotoApp(page, `/room/${DATA.pendingApproveRoomId}`);

  await expect(page.locator('body')).toContainText('E2E pending для одобрения');
  await expect(page.locator('body')).toContainText('Описание');
  await expect(page.locator('body')).toContainText('₽/ч');
});

test('E2E-12. Суперпользователь отклоняет помещение, после чего администратор исправляет его и повторно отправляет на модерацию', async ({ browser }) => {
  const superContext = await browser.newContext();
  const superPage = await superContext.newPage();

  await setupDialogs(superPage);
  await loginAsSuperuser(superPage);

  await gotoApp(superPage, '/superuser');

  const roomCard = superPage.locator(`[data-room-id="${DATA.pendingRejectRoomId}"]`);

  await expect(roomCard).toBeVisible();

  const reason = 'Причина E2E: нужно исправить описание';

  await superPage.fill(`#rejectReason-${DATA.pendingRejectRoomId}`, reason);

  await roomCard.locator('[data-action="reject"]').click();

  await expect(roomCard).toHaveCount(0);

  await superContext.close();

  const adminContext = await browser.newContext();
  const adminPage = await adminContext.newPage();

  await setupDialogs(adminPage);
  await loginAsAdmin(adminPage);

  await gotoApp(adminPage, `/admin/room/${DATA.pendingRejectRoomId}`);

  await expect(adminPage.locator('body')).toContainText('Причина отклонения');
  await expect(adminPage.locator('body')).toContainText(reason);

  await adminPage.fill('#roomDescription', 'Исправленное описание после отклонения E2E');

  await adminPage.click('#saveRoomBtn');

  await expect(adminPage.locator('body')).toContainText('Черновик');

  await adminPage.click('#submitRoomBtn');

  await expect(adminPage.locator('body')).toContainText('На модерации');

  await adminContext.close();
});

test('E2E-13. Администратор просматривает бронирования помещения и отменяет бронь', async ({ page }) => {
  await loginAsAdmin(page);

  await gotoApp(page, `/admin/room/${DATA.adminBookingRoomId}`);

  await expect(page.locator('body')).toContainText('Бронирования помещения');

  const row = page
    .locator('#bookingsTable tbody tr')
    .filter({ hasText: DATA.user.email })
    .first();

  await expect(row).toBeVisible();
  await expect(row.locator('[data-cancel-booking]')).toBeVisible();

  await row.locator('[data-cancel-booking]').click();

  const updatedRow = page
    .locator('#bookingsTable tbody tr')
    .filter({ hasText: DATA.user.email })
    .first();

  await expect(updatedRow).toContainText('Отменено');
});

test('E2E-14. Администратор не может немедленно архивировать помещение с будущей бронью', async ({ page }) => {
  await loginAsAdmin(page);

  await gotoApp(page, `/admin/room/${DATA.roomWithFutureBookingId}`);

  await expect(page.locator('body')).toContainText('Архивирование');

  const archiveBtn = page.locator('#archiveImmediateBtn');

  await expect(archiveBtn).toBeVisible();

  if (await archiveBtn.isDisabled()) {
    await expect(page.locator('body')).toContainText(/активные|будущие|бронирования/i);
  } else {
    const beforeDialogsCount = getDialogMessages(page).length;

    await archiveBtn.click();

    await page.waitForTimeout(700);

    const newDialogs = getDialogMessages(page).slice(beforeDialogsCount);

    expect(
      newDialogs.some(message =>
        /бронир|будущ|действующ|активн|нельзя|невозможно/i.test(message)
      )
    ).toBe(true);
  }

  await page.reload({
    waitUntil: 'domcontentloaded'
  });

  await expect(page.locator('body')).not.toContainText('Помещение уже архивировано');
  await expect(page.locator('body')).not.toContainText('Архивировано');
});

test('E2E-15. Администратор архивирует опубликованное помещение без будущих бронирований', async ({ page }) => {
  await loginAsAdmin(page);

  await gotoApp(page, `/admin/room/${DATA.roomWithoutFutureBookingsId}`);

  await expect(page.locator('body')).toContainText('Архивирование');

  const roomTitle = await page.locator('.dashboard-header h1').innerText();

  await expect(page.locator('#archiveImmediateBtn')).toBeEnabled();

  await page.click('#archiveImmediateBtn');

  await expect(page.locator('body')).toContainText('Архивировано');
  await expect(page.locator('body')).toContainText('Помещение уже архивировано');

  await gotoApp(page, `/room/${DATA.roomWithoutFutureBookingsId}`);

  await expect(page.locator('body')).toContainText(/Комната не найдена|Не удалось загрузить комнату/i);
  await expect(page.locator('body')).not.toContainText(roomTitle);
});