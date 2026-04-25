/**
 * app.integration.test.js
 * Интеграционные тесты для auth.js, home.js, room.js, me.js
 */

describe('Интеграционные тесты приложения бронирования', () => {
  class MockWebSocket {
    static CONNECTING = 0;
    static OPEN = 1;
    static CLOSING = 2;
    static CLOSED = 3;

    constructor(url) {
      this.url = url;
      this.readyState = MockWebSocket.CONNECTING;
      this.listeners = {};
      this.send = jest.fn();
      this.close = jest.fn(() => {
        this.readyState = MockWebSocket.CLOSED;
        this.emit('close', { code: 1000 });
      });
      MockWebSocket.instances.push(this);
    }

    addEventListener(type, cb) {
      if (!this.listeners[type]) this.listeners[type] = [];
      this.listeners[type].push(cb);
    }

    emit(type, payload = {}) {
      const handlers = this.listeners[type] || [];
      handlers.forEach(fn => fn(payload));
    }
  }

  MockWebSocket.instances = [];

  async function flushPromises() {
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();
  }

  function resetGlobals() {
    jest.resetModules();
    jest.useFakeTimers();

    MockWebSocket.instances = [];

    global.fetch = jest.fn();
    global.alert = jest.fn();
    global.confirm = jest.fn();
    global.console = {
      log: jest.fn(),
      warn: jest.fn(),
      error: jest.fn()
    };

    global.WebSocket = MockWebSocket;
    global.WebSocket.OPEN = MockWebSocket.OPEN;
    global.WebSocket.CONNECTING = MockWebSocket.CONNECTING;
    global.WebSocket.CLOSED = MockWebSocket.CLOSED;
    global.WebSocket.CLOSING = MockWebSocket.CLOSING;

    global.IntersectionObserver = class {
      constructor(cb) {
        this.cb = cb;
      }
      observe = jest.fn();
      disconnect = jest.fn();
    };

    global.L = {
      map: jest.fn(() => ({
        setView: jest.fn().mockReturnThis(),
        attributionControl: { setPrefix: jest.fn() }
      })),
      tileLayer: jest.fn(() => ({
        addTo: jest.fn()
      })),
      marker: jest.fn(() => ({
        addTo: jest.fn()
      }))
    };

    delete window.location;
    window.location = {
      protocol: 'http:',
      host: 'localhost:8080',
      href: '',
      reload: jest.fn()
    };
  }

  function buildAuthDOM() {
    document.body.innerHTML = `
      <div id="authMessage"></div>

      <button class="auth-tab active" id="tab-login" data-tab="login">Вход</button>
      <button class="auth-tab" id="tab-register" data-tab="register">Регистрация</button>

      <form id="loginForm" class="auth-form">
        <input id="loginEmail" type="email" />
        <input id="loginPassword" type="password" />
        <button type="submit" id="loginSubmit">Войти</button>
      </form>

      <form id="registerForm" class="auth-form hidden">
        <input id="regUsername" type="text" />
        <input id="regEmail" type="email" />
        <input id="regPassword" type="password" />
        <input id="regConfirm" type="password" />
        <button type="submit" id="registerSubmit">Зарегистрироваться</button>
      </form>
    `;
    window.__USER__ = { auth: false };
  }

  function buildHomeDOM() {
    document.body.innerHTML = `
      <div id="cityName"></div>
      <button id="cityBtn"></button>
      <dialog id="cityModal"></dialog>
      <ul id="cityList">
        <li class="city-item">Москва</li>
        <li class="city-item">Казань</li>
        <li class="city-item">Екатеринбург</li>
      </ul>
      <input id="citySearch" />
      <button id="cityOk"></button>
      <button id="cityCancel"></button>

      <button id="authButton" data-auth="1"></button>
      <a id="brand" href="/">Brand</a>

      <div id="cardsWrapper"></div>
      <div id="statusBar" hidden></div>

      <div id="companyToggleWrap"></div>
      <div id="companyList"></div>

      <input id="priceInput" />
      <input id="capacityInput" />
      <button id="applyFilters"></button>
      <button id="clearFilters"></button>
    `;
    document.body.dataset.initialCity = 'Москва';

    const cityModal = document.getElementById('cityModal');
    cityModal.showModal = jest.fn();
    cityModal.close = jest.fn();
    cityModal.getBoundingClientRect = jest.fn(() => ({
      left: 10, right: 100, top: 10, bottom: 100
    }));
  }

  function buildRoomDOM(auth = true) {
    document.body.innerHTML = `
      <button id="authButton" class="auth-btn" data-auth="${auth ? '1' : '0'}" data-username="slava">Кабинет</button>
      <button id="logoutBtn">Выйти</button>

      <div id="mainImageWrap">
        <img id="mainImage" src="/img/1.png" alt="Комната">
      </div>
      <div id="thumbs">
        <button class="thumb-btn" data-src="/img/2.png"><img src="/img/2.png" alt="thumb"></button>
      </div>

      <button id="bookBtn">Забронировать</button>
      <div id="map"></div>

      <div id="bookingModal" class="hidden" aria-hidden="true">
        <div class="booking-modal__content">
          <button id="bookingClose">x</button>
          <div id="bookingNotice"></div>
          <div id="bookingCalendar"></div>
          <div id="bookingTimes"></div>
          <div id="selectionSummary"></div>
          <button id="bookingCancel">Отмена</button>
          <button id="bookingConfirm">Подтвердить</button>
        </div>
      </div>
    `;

    window.__USER__ = { auth, username: auth ? 'slava' : '' };
    window.__ROOM__ = {
      id: '101',
      title: 'Переговорная',
      lat: 59.93,
      lng: 30.31,
      address: 'Невский проспект'
    };
  }

  function buildMeDOM(auth = true) {
    if (auth) {
      document.body.innerHTML = `
        <div id="col-in_use"></div>
        <div id="col-booked"></div>
        <div id="col-finished"></div>
        <div id="col-canceled"></div>
        <div id="emptyMessage" class="hidden">Ваша история бронирования пуста</div>

        <footer id="meBottomBar">
          <div class="me-bottombar__inner">
            <div class="me-bottombar__center">
              <button id="logoutBtn">Выйти</button>
            </div>
          </div>
        </footer>

        <div id="confirmModal" class="hidden" aria-hidden="true">
          <p id="confirmText">Уверены, что хотите отменить бронь?</p>
          <button id="confirmNo">Нет</button>
          <button id="confirmYes">Отменить</button>
        </div>
      `;
    } else {
      document.body.innerHTML = `
        <div class="not-auth">Вы не авторизованы</div>
        <footer id="meBottomBar">
          <button id="logoutBtn">Выйти</button>
        </footer>

        <div id="confirmModal" class="hidden" aria-hidden="true">
          <p id="confirmText">Уверены, что хотите отменить бронь?</p>
          <button id="confirmNo">Нет</button>
          <button id="confirmYes">Отменить</button>
        </div>
      `;
    }

    window.__USER__ = { auth, username: auth ? 'slava' : '' };
  }

  function loadAuthModule() {
    let authModule;
    jest.isolateModules(() => {
      authModule = require('../static/auth/auth.js');
    });
    return authModule;
  }

  function loadHomeModule() {
    let homeModule;
    jest.isolateModules(() => {
      homeModule = require('../static/home/home.js');
    });
    return homeModule;
  }

  function loadRoomModule() {
    let roomModule;
    jest.isolateModules(() => {
      roomModule = require('../static/room/room.js');
    });
    return roomModule;
  }

  function loadMeModule() {
    let meModule;
    jest.isolateModules(() => {
      meModule = require('../static/me/me.js');
    });
    return meModule;
  }

  beforeEach(() => {
    resetGlobals();
    document.body.innerHTML = '';
  });

  afterEach(() => {
    jest.runOnlyPendingTimers();
    jest.useRealTimers();
    jest.restoreAllMocks();
  });

  test('Интеграция: регистрация нового пользователя и последующий вход в систему', async () => {
    // Техника тест-дизайна: сценарий использования
    buildAuthDOM();
    const authModule = loadAuthModule();
    authModule.initAuthModule();

    global.fetch
      .mockResolvedValueOnce({ ok: true }) // register
      .mockResolvedValueOnce({ ok: true }); // login

    document.getElementById('tab-register').click();
    document.getElementById('regUsername').value = 'slava';
    document.getElementById('regEmail').value = 'slava@test.com';
    document.getElementById('regPassword').value = '123456';
    document.getElementById('regConfirm').value = '123456';

    document.getElementById('registerForm').dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
    await flushPromises();

    expect(document.getElementById('authMessage').textContent)
      .toBe('Регистрация успешна. Пожалуйста, войдите в аккаунт.');

    document.getElementById('loginEmail').value = 'slava@test.com';
    document.getElementById('loginPassword').value = '123456';

    document.getElementById('loginForm').dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
    await flushPromises();

    expect(document.getElementById('authMessage').textContent).toBe('Вход успешен. Перенаправление...');
    jest.advanceTimersByTime(500);
    expect(window.location.href).toBe('/');
  });

  test('Интеграция: вход с ошибкой и последующим успехом', async () => {
    // Техника тест-дизайна: таблица решений
    buildAuthDOM();
    const authModule = loadAuthModule();
    authModule.initAuthModule();

    global.fetch
      .mockResolvedValueOnce({ ok: false, status: 401 })
      .mockResolvedValueOnce({ ok: true });

    document.getElementById('loginEmail').value = 'slava@test.com';
    document.getElementById('loginPassword').value = 'wrong';
    document.getElementById('loginForm').dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
    await flushPromises();

    expect(document.getElementById('authMessage').textContent).toBe('Неверный email или пароль');

    document.getElementById('loginPassword').value = '123456';
    document.getElementById('loginForm').dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
    await flushPromises();

    expect(document.getElementById('authMessage').textContent).toBe('Вход успешен. Перенаправление...');
    jest.advanceTimersByTime(500);
    expect(window.location.href).toBe('/');
  });

  test('Интеграция: выбор города и перезагрузка данных через websocket', () => {
    // Техника тест-дизайна: сценарий использования
    buildHomeDOM();
    const homeModule = loadHomeModule();

    const initialWsCount = MockWebSocket.instances.length;

    document.getElementById('citySearch').value = 'Казань';
    document.getElementById('cityOk').click();

    expect(document.getElementById('cityName').textContent).toBe('Казань');
    expect(document.body.dataset.initialCity).toBe('Казань');
    expect(MockWebSocket.instances.length).toBeGreaterThan(initialWsCount);

    const ws = MockWebSocket.instances[MockWebSocket.instances.length - 1];
    ws.readyState = MockWebSocket.OPEN;
    ws.emit('open');
    ws.emit('message', { data: JSON.stringify(['Компания А']) });
    ws.emit('message', {
      data: JSON.stringify([
        {
          id: '1',
          title: 'Зал 1',
          company: 'Компания А',
          address: 'Адрес 1',
          capacity: 10,
          price: 1000
        }
      ])
    });

    expect(document.querySelectorAll('.card').length).toBe(1);
    expect(document.getElementById('cardsWrapper').textContent).toContain('Зал 1');
    expect(document.querySelectorAll('#companyList input[type="checkbox"]').length).toBe(1);
  });

  test('Интеграция: выбор несуществующего города не меняет состояние страницы', () => {
    // Техника тест-дизайна: негативный сценарий + классы эквивалентности
    buildHomeDOM();
    loadHomeModule();

    document.getElementById('cityName').textContent = 'Москва';
    document.getElementById('citySearch').value = 'Тверь';
    document.getElementById('cityOk').click();

    expect(document.getElementById('cityName').textContent).toBe('Москва');
    expect(document.getElementById('cityError')).not.toBeNull();
    expect(document.getElementById('cityError').textContent).toBe('Такого города не существует');
  });

  test('Интеграция: загрузка компаний и фильтрация помещений по цене, вместимости и компании', () => {
    // Техника тест-дизайна: попарное тестирование
    buildHomeDOM();
    loadHomeModule();

    const ws = MockWebSocket.instances[0];
    ws.readyState = MockWebSocket.OPEN;
    ws.emit('open');
    ws.emit('message', { data: JSON.stringify(['Компания А', 'Компания Б']) });
    ws.emit('message', {
      data: JSON.stringify([
        { id: '1', title: 'Зал 1', company: 'Компания А', address: 'A', capacity: 2, price: 1000 },
        { id: '2', title: 'Зал 2', company: 'Компания Б', address: 'B', capacity: 8, price: 1400 },
        { id: '3', title: 'Зал 3', company: 'Компания Б', address: 'C', capacity: 3, price: 5000 }
      ])
    });

    document.getElementById('priceInput').value = '1500';
    document.getElementById('capacityInput').value = '5';
    const checkboxes = document.querySelectorAll('#companyList input[type="checkbox"]');
    checkboxes[1].checked = true;

    document.getElementById('applyFilters').click();

    expect(document.querySelectorAll('.card').length).toBe(1);
    expect(document.getElementById('cardsWrapper').textContent).toContain('Зал 2');
    expect(document.getElementById('cardsWrapper').textContent).not.toContain('Зал 1');
    expect(document.getElementById('cardsWrapper').textContent).not.toContain('Зал 3');
  });

  test('Интеграция: открытие комнаты и получение слотов через websocket', () => {
    // Техника тест-дизайна: сценарий использования
    buildRoomDOM(true);
    const roomModule = loadRoomModule();

    roomModule.openBookingModal();

    expect(document.getElementById('bookingModal').classList.contains('hidden')).toBe(false);
    expect(MockWebSocket.instances.length).toBe(1);

    const ws = MockWebSocket.instances[0];
    ws.emit('open');

    const firstDayBtn = document.querySelector('#bookingCalendar .day');
    const ymd = firstDayBtn.dataset.ymd;

    ws.emit('message', {
      data: JSON.stringify({
        dates: [
          { date: ymd, available_times: ['10:00', '11:00'] }
        ]
      })
    });

    expect(document.getElementById('bookingNotice').textContent).toBe('Выберите дату и слоты');
    expect(document.querySelectorAll('#bookingTimes .slot').length).toBe(2);
  });

  test('Интеграция: попытка бронирования без выбора даты и слотов', async () => {
    // Техника тест-дизайна: негативный сценарий + классы эквивалентности
    buildRoomDOM(true);
    const roomModule = loadRoomModule();

    await roomModule.confirmBooking();

    expect(global.alert).toHaveBeenCalledWith('Выберите дату и хотя бы один слот');
    expect(global.fetch).not.toHaveBeenCalled();
  });

  test('Интеграция: успешное бронирование помещения', async () => {
    // Техника тест-дизайна: сценарий использования
    buildRoomDOM(true);
    const roomModule = loadRoomModule();
    global.fetch.mockResolvedValue({ ok: true });

    roomModule.openBookingModal();

    const ws = MockWebSocket.instances[0];
    ws.emit('open');

    const firstDayBtn = document.querySelector('#bookingCalendar .day');
    const ymd = firstDayBtn.dataset.ymd;

    ws.emit('message', {
      data: JSON.stringify({
        dates: [
          { date: ymd, available_times: ['10:00', '11:00'] }
        ]
      })
    });

    roomModule.onSlotClick('10:00');
    roomModule.onSlotClick('11:00');

    await roomModule.confirmBooking();

    expect(global.fetch).toHaveBeenCalledWith('/api/booking/new', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'same-origin',
      body: JSON.stringify({
        room_id: 101,
        date: ymd,
        slots: ['10:00', '11:00']
      })
    });

    expect(document.querySelector('#bookingModal .booking-modal__content').textContent)
      .toContain('Бронирование успешно');
  });

  test('Интеграция: просмотр истории бронирований в кабинете через websocket', () => {
    // Техника тест-дизайна: сценарий использования
    buildMeDOM(true);
    loadMeModule();

    document.dispatchEvent(new Event('DOMContentLoaded'));

    expect(MockWebSocket.instances.length).toBe(1);

    const ws = MockWebSocket.instances[0];
    ws.emit('message', {
      data: JSON.stringify([
        {
          id: '1',
          room_id: 'r1',
          title: 'Используется',
          image_url: '1.png',
          date: '2026-04-24',
          start_time: '10:00',
          end_time: '11:00',
          total_price: 100,
          status: 'in_use'
        },
        {
          id: '2',
          room_id: 'r2',
          title: 'Забронирована',
          image_url: '2.png',
          date: '2026-04-24',
          start_time: '11:00',
          end_time: '12:00',
          total_price: 200,
          status: 'booked'
        }
      ])
    });

    expect(document.getElementById('col-in_use').querySelectorAll('.me-card').length).toBe(1);
    expect(document.getElementById('col-booked').querySelectorAll('.me-card').length).toBe(1);
    expect(document.getElementById('emptyMessage').classList.contains('hidden')).toBe(true);
  });

  test('Интеграция: отмена брони из личного кабинета', async () => {
    // Техника тест-дизайна: сценарий использования
    buildMeDOM(true);
    loadMeModule();
    global.fetch.mockResolvedValue({ ok: true });
    window.location.reload = jest.fn();

    document.dispatchEvent(new Event('DOMContentLoaded'));

    const ws = MockWebSocket.instances[0];
    ws.emit('message', {
      data: JSON.stringify([
        {
          id: 'booking-1',
          room_id: 'r1',
          title: 'Зал 1',
          image_url: '1.png',
          date: '2026-04-24',
          start_time: '10:00',
          end_time: '11:00',
          total_price: 100,
          status: 'booked'
        }
      ])
    });

    const cancelBtn = document.querySelector('.cancel-btn');
    cancelBtn.click();

    expect(document.getElementById('confirmModal').classList.contains('hidden')).toBe(false);

    document.getElementById('confirmYes').click();
    await flushPromises();

    expect(global.fetch).toHaveBeenCalledWith('/api/booking/stop', {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ booking_id: 'booking-1' })
    });
    expect(window.location.reload).toHaveBeenCalled();
  });
});