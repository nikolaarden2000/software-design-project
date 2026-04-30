# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: booking.playwright.spec.js >> E2E-12. Суперпользователь отклоняет помещение, администратор редактирует и повторно отправляет
- Location: booking.playwright.spec.js:277:1

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: locator('[data-room-id="18"]')
Expected: visible
Timeout: 5000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" with timeout 5000ms
  - waiting for locator('[data-room-id="18"]')

```

# Page snapshot

```yaml
- generic [active] [ref=e1]:
  - banner [ref=e2]:
    - link "Своя Бронь" [ref=e4] [cursor=pointer]:
      - /url: /
    - generic [ref=e5]: Панель суперпользователя
    - button "Выйти" [ref=e7] [cursor=pointer]
  - main [ref=e8]:
    - generic [ref=e9]:
      - generic [ref=e11]:
        - heading "Управление платформой" [level=1] [ref=e12]
        - paragraph [ref=e13]: Создание компаний, локаций, администраторов и модерация помещений.
      - generic [ref=e14]:
        - generic [ref=e15]:
          - heading "Компании" [level=2] [ref=e16]
          - generic [ref=e17]:
            - textbox "Название компании" [ref=e18]
            - textbox "Описание компании" [ref=e19]
            - button "Создать компанию" [ref=e20] [cursor=pointer]
          - generic [ref=e21]:
            - generic [ref=e22]:
              - generic [ref=e23]: ООО «Коворкинг Стандарт»
              - generic [ref=e24]: Сеть коворкингов и переговорных комнат
              - generic [ref=e25]: "Локаций: 5"
            - generic [ref=e26]:
              - generic [ref=e27]: ИП Иванов
              - generic [ref=e28]: Частные офисные пространства
              - generic [ref=e29]: "Локаций: 2"
            - generic [ref=e30]:
              - generic [ref=e31]: ЗАО «Бизнес-Пространство»
              - generic [ref=e32]: Бизнес-центры и конференц-залы
              - generic [ref=e33]: "Локаций: 2"
            - button "Показать все (11)" [ref=e34] [cursor=pointer]
        - generic [ref=e35]:
          - heading "Локации" [level=2] [ref=e36]
          - generic [ref=e37]:
            - combobox [ref=e38]:
              - option "Выберите компанию" [selected]
              - option "ООО «Коворкинг Стандарт»"
              - option "ИП Иванов"
              - option "ЗАО «Бизнес-Пространство»"
              - option "ООО «Деловые Линии»"
              - option "АО «Городские Офисы»"
              - option "ООО «Профис»"
              - option "ИП Петров"
              - option "ООО «Элит-Бизнес»"
              - option "ЗАО «Столичный Коворкинг»"
              - option "ООО «Офис-Хаб»"
              - option "asdas"
            - textbox "Город" [ref=e39]
            - textbox "Адрес" [ref=e40]
            - spinbutton [ref=e41]
            - spinbutton [ref=e42]
            - textbox "Таймзона" [ref=e43]: Europe/Moscow
            - button "Создать локацию" [ref=e44] [cursor=pointer]
          - generic [ref=e45]:
            - generic [ref=e46]:
              - generic [ref=e47]: ООО «Коворкинг Стандарт»
              - generic [ref=e48]: Москва, Москва, Тверская 15
              - generic [ref=e49]: Europe/Moscow
            - generic [ref=e50]:
              - generic [ref=e51]: ООО «Коворкинг Стандарт»
              - generic [ref=e52]: Санкт-Петербург, Санкт-Петербург, Невский проспект 22
              - generic [ref=e53]: Europe/Moscow
            - generic [ref=e54]:
              - generic [ref=e55]: ООО «Коворкинг Стандарт»
              - generic [ref=e56]: Самара, Самара, Ленинградская 43
              - generic [ref=e57]: Europe/Moscow
            - button "Показать все (23)" [ref=e58] [cursor=pointer]
        - generic [ref=e59]:
          - heading "Администраторы" [level=2] [ref=e60]
          - generic [ref=e61]:
            - textbox "Имя администратора" [ref=e62]
            - textbox "Email" [ref=e63]
            - textbox "Пароль" [ref=e64]
            - button "Создать администратора" [ref=e65] [cursor=pointer]
          - generic [ref=e66]:
            - generic [ref=e67]:
              - generic [ref=e68]: Администратор ABC
              - generic [ref=e69]: admin-abc@mail.com
              - generic [ref=e70]:
                - text: "Локации:"
                - text: ООО «Коворкинг Стандарт», Москва, Тверская 15
                - text: ООО «Коворкинг Стандарт», Санкт-Петербург, Невский проспект 22
                - text: ИП Иванов, Москва, Арбат 9
            - generic [ref=e71]:
              - generic [ref=e72]: superuse1232r@mail.com
              - generic [ref=e73]: sadasdasd@mail.ru
              - generic [ref=e74]:
                - text: "Локации:"
                - text: Локации не назначены
        - generic [ref=e75]:
          - heading "Привязка локаций" [level=2] [ref=e76]
          - generic [ref=e77]:
            - combobox [ref=e78]:
              - option "Выберите администратора" [selected]
              - option "Администратор ABC — admin-abc@mail.com"
              - option "superuse1232r@mail.com — sadasdasd@mail.ru"
            - generic [ref=e80] [cursor=pointer]:
              - generic [ref=e81]: Выбрать локацию
              - generic [ref=e82]: ›
            - button "Привязать выбранные локации" [ref=e83] [cursor=pointer]
        - generic [ref=e84]:
          - heading "Модерация помещений" [level=2] [ref=e85]
          - generic [ref=e87]: Нет помещений на модерации
```

# Test source

```ts
  189 |   await page.locator('#col-booked .cancel-btn').first().click();
  190 | 
  191 |   await expect(page.locator('#confirmModal')).not.toHaveClass(/hidden/);
  192 | 
  193 |   await page.click('#confirmYes');
  194 | 
  195 |   await expect(page.locator('#col-canceled')).toContainText(/₽|Переговорная|Помещение|Комната/i);
  196 | });
  197 | 
  198 | test('E2E-07. Доступ обычного пользователя к admin-странице запрещён', async ({ page }) => {
  199 |   await loginAsUser(page);
  200 | 
  201 |   await page.goto(appUrl('/admin'));
  202 | 
  203 |   await expect(page.locator('body')).toContainText(/Доступ запрещён|Авторизация|Вход/);
  204 | });
  205 | 
  206 | test('E2E-08. Администратор просматривает свои локации', async ({ page }) => {
  207 |   await loginAsAdmin(page);
  208 | 
  209 |   await page.goto(appUrl('/admin'));
  210 | 
  211 |   await expect(page.locator('body')).toContainText('Мои локации');
  212 |   await expect(page.locator('[data-location-id]').first()).toBeVisible();
  213 | });
  214 | 
  215 | test('E2E-09. Администратор создаёт помещение как черновик', async ({ page }) => {
  216 |   await loginAsAdmin(page);
  217 | 
  218 |   await page.goto(appUrl('/admin'));
  219 | 
  220 |   await page.click(`[data-location-id="${DATA.locationId}"]`);
  221 | 
  222 |   await expect(page).toHaveURL(appUrl(`/admin/location/${DATA.locationId}`));
  223 | 
  224 |   await page.click(`a[href="/admin/room/new?location_id=${DATA.locationId}"]`);
  225 | 
  226 |   await expect(page).toHaveURL(/\/admin\/room\/new/);
  227 | 
  228 |   const title = `E2E черновик ${Date.now()}`;
  229 | 
  230 |   await page.fill('#roomTitle', title);
  231 |   await page.fill('#roomDescription', 'Описание помещения, созданного Playwright E2E-тестом');
  232 |   await page.fill('#roomPrice', '1800');
  233 |   await page.fill('#roomCapacity', '10');
  234 |   await page.fill('#roomAvailableFrom', '09:00');
  235 |   await page.fill('#roomAvailableTo', '21:00');
  236 |   await page.fill('#roomImages', '/shared/placeholders/room-placeholder.svg');
  237 | 
  238 |   await page.click('#saveDraftBtn');
  239 | 
  240 |   await expect(page).toHaveURL(/\/admin\/room\/\d+/);
  241 |   await expect(page.locator('body')).toContainText(title);
  242 |   await expect(page.locator('body')).toContainText('Черновик');
  243 | });
  244 | 
  245 | test('E2E-10. Администратор отправляет помещение на модерацию', async ({ page }) => {
  246 |   await loginAsAdmin(page);
  247 | 
  248 |   await page.goto(appUrl(`/admin/room/${DATA.draftRoomId}`));
  249 | 
  250 |   await expect(page.locator('body')).toContainText('Черновик');
  251 | 
  252 |   await page.click('#submitRoomBtn');
  253 | 
  254 |   await expect(page.locator('body')).toContainText('На модерации');
  255 |   await expect(page.locator('#roomTitle')).not.toBeEditable();
  256 | });
  257 | 
  258 | test('E2E-11. Суперпользователь одобряет помещение', async ({ page }) => {
  259 |   await loginAsSuperuser(page);
  260 | 
  261 |   await page.goto(appUrl('/superuser'));
  262 | 
  263 |   const roomCard = page.locator(`[data-room-id="${DATA.pendingApproveRoomId}"]`);
  264 | 
  265 |   await expect(roomCard).toBeVisible();
  266 | 
  267 |   await roomCard.locator('[data-action="approve"]').click();
  268 | 
  269 |   await expect(roomCard).toHaveCount(0);
  270 | 
  271 |   await page.goto(appUrl(`/room/${DATA.pendingApproveRoomId}`));
  272 | 
  273 |   await expect(page.locator('body')).toContainText('Описание');
  274 |   await expect(page.locator('body')).toContainText('₽/ч');
  275 | });
  276 | 
  277 | test('E2E-12. Суперпользователь отклоняет помещение, администратор редактирует и повторно отправляет', async ({ browser }) => {
  278 |   const superContext = await browser.newContext();
  279 |   const superPage = await superContext.newPage();
  280 | 
  281 |   await setupDialogs(superPage);
  282 | 
  283 |   await loginAsSuperuser(superPage);
  284 | 
  285 |   await superPage.goto(appUrl('/superuser'));
  286 | 
  287 |   const roomCard = superPage.locator(`[data-room-id="${DATA.pendingRejectRoomId}"]`);
  288 | 
> 289 |   await expect(roomCard).toBeVisible();
      |                          ^ Error: expect(locator).toBeVisible() failed
  290 | 
  291 |   await superPage.fill(
  292 |     `#rejectReason-${DATA.pendingRejectRoomId}`,
  293 |     'Причина E2E: нужно исправить описание'
  294 |   );
  295 | 
  296 |   await roomCard.locator('[data-action="reject"]').click();
  297 | 
  298 |   await expect(roomCard).toHaveCount(0);
  299 | 
  300 |   await superContext.close();
  301 | 
  302 |   const adminContext = await browser.newContext();
  303 |   const adminPage = await adminContext.newPage();
  304 | 
  305 |   await setupDialogs(adminPage);
  306 | 
  307 |   await loginAsAdmin(adminPage);
  308 | 
  309 |   await adminPage.goto(appUrl(`/admin/room/${DATA.pendingRejectRoomId}`));
  310 | 
  311 |   await expect(adminPage.locator('body')).toContainText('Причина отклонения');
  312 |   await expect(adminPage.locator('body')).toContainText('Причина E2E: нужно исправить описание');
  313 | 
  314 |   await adminPage.fill('#roomDescription', 'Исправленное описание после отклонения E2E');
  315 | 
  316 |   await adminPage.click('#saveRoomBtn');
  317 | 
  318 |   await expect(adminPage.locator('body')).toContainText('Черновик');
  319 | 
  320 |   await adminPage.click('#submitRoomBtn');
  321 | 
  322 |   await expect(adminPage.locator('body')).toContainText('На модерации');
  323 | 
  324 |   await adminContext.close();
  325 | });
  326 | 
  327 | test('E2E-13. Администратор просматривает бронирования помещения и отменяет бронь', async ({ page }) => {
  328 |   await loginAsAdmin(page);
  329 | 
  330 |   await page.goto(appUrl(`/admin/room/${DATA.adminBookingRoomId}`));
  331 | 
  332 |   await expect(page.locator('body')).toContainText('Бронирования помещения');
  333 | 
  334 |   await expect(page.locator('[data-cancel-booking]').first()).toBeVisible();
  335 | 
  336 |   await page.locator('[data-cancel-booking]').first().click();
  337 | 
  338 |   await expect(page.locator('body')).toContainText('Отменено');
  339 | });
  340 | 
  341 | test('E2E-14. Администратор не может немедленно архивировать помещение с будущими бронями', async ({ page }) => {
  342 |   await loginAsAdmin(page);
  343 | 
  344 |   await page.goto(appUrl(`/admin/room/${DATA.roomWithFutureBookingId}`));
  345 | 
  346 |   await expect(page.locator('body')).toContainText('Архивирование');
  347 | 
  348 |   const archiveBtn = page.locator('#archiveImmediateBtn');
  349 | 
  350 |   await expect(archiveBtn).toBeVisible();
  351 | 
  352 |   const disabled = await archiveBtn.isDisabled();
  353 | 
  354 |   if (disabled) {
  355 |     await expect(page.locator('body')).toContainText(/активные|будущие|бронирования/i);
  356 |   } else {
  357 |     await archiveBtn.click();
  358 | 
  359 |     await expect(page.locator('body')).not.toContainText('Помещение уже архивировано');
  360 |   }
  361 | });
  362 | 
  363 | test('E2E-15. Администратор архивирует помещение без будущих броней', async ({ page }) => {
  364 |   await loginAsAdmin(page);
  365 | 
  366 |   await page.goto(appUrl(`/admin/room/${DATA.roomWithoutFutureBookingsId}`));
  367 | 
  368 |   await expect(page.locator('body')).toContainText('Архивирование');
  369 | 
  370 |   const roomTitle = await page.locator('.dashboard-header h1').innerText();
  371 | 
  372 |   await expect(page.locator('#archiveImmediateBtn')).toBeEnabled();
  373 | 
  374 |   await page.click('#archiveImmediateBtn');
  375 | 
  376 |   await expect(page.locator('body')).toContainText('Архивировано');
  377 | 
  378 |   await page.goto(appUrl('/'));
  379 | 
  380 |   await expect(page.locator('body')).not.toContainText(roomTitle);
  381 | });
```