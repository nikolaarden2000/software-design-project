# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: booking.playwright.spec.js >> E2E-10. Администратор отправляет помещение на модерацию
- Location: booking.playwright.spec.js:245:1

# Error details

```
Error: expect(locator).toContainText(expected) failed

Locator: locator('body')
Timeout: 5000ms
- Expected substring  -  1
+ Received string     + 31

- Черновик
+
+   
+     
+       Своя Бронь
+     
+
+     
+       Панель администратора
+     
+
+     
+       Выйти
+     
+   
+
+   
+     
+       Доступ запрещён
+       Эта страница доступна только администраторам локаций.
+       На главную
+     
+
+     
+       
+     
+   
+
+   
+   
+
+

Call log:
  - Expect "toContainText" with timeout 5000ms
  - waiting for locator('body')
    9 × locator resolved to <body>…</body>
      - unexpected value "
  
    
      Своя Бронь
    

    
      Панель администратора
    

    
      Выйти
    
  

  
    
      Доступ запрещён
      Эта страница доступна только администраторам локаций.
      На главную
    

    
      
    
  

  
  

"

```

# Page snapshot

```yaml
- generic [active] [ref=e1]:
  - banner [ref=e2]:
    - link "Своя Бронь" [ref=e4] [cursor=pointer]:
      - /url: /
    - generic [ref=e5]: Панель администратора
    - button "Выйти" [ref=e7] [cursor=pointer]
  - main [ref=e8]
```

# Test source

```ts
  150 | 
  151 |   await page.goto(appUrl('/'));
  152 | 
  153 |   await expect(page.locator('.card').first()).toBeVisible();
  154 | 
  155 |   await page.locator('.card').first().click();
  156 | 
  157 |   await expect(page).toHaveURL(/\/room\/\d+/);
  158 | 
  159 |   await expect(page.locator('body')).toContainText('Описание');
  160 |   await expect(page.locator('body')).toContainText('Вместимость');
  161 |   await expect(page.locator('body')).toContainText('₽/ч');
  162 |   await expect(page.locator('#mainImage')).toBeVisible();
  163 | 
  164 |   await page.click('#bookBtn');
  165 | 
  166 |   await expect(page.locator('#bookingModal')).not.toHaveClass(/hidden/);
  167 |   await expect(page.locator('.slot').first()).toBeVisible();
  168 | });
  169 | 
  170 | test('E2E-05. Бронирование помещения пользователем', async ({ page }) => {
  171 |   await loginAsUser(page);
  172 | 
  173 |   await createBookingThroughUi(page, DATA.publishedRoomId);
  174 | 
  175 |   await page.goto(appUrl('/me'));
  176 | 
  177 |   await expect(page.locator('body')).toContainText('Забронирована');
  178 | });
  179 | 
  180 | test('E2E-06. Отмена бронирования пользователем', async ({ page }) => {
  181 |   await loginAsUser(page);
  182 | 
  183 |   await createBookingThroughUi(page, DATA.publishedRoomId);
  184 | 
  185 |   await page.goto(appUrl('/me'));
  186 | 
  187 |   await expect(page.locator('#col-booked .me-card').first()).toBeVisible();
  188 | 
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
> 250 |   await expect(page.locator('body')).toContainText('Черновик');
      |                                      ^ Error: expect(locator).toContainText(expected) failed
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
  289 |   await expect(roomCard).toBeVisible();
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
```