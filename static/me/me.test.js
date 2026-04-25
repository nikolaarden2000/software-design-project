describe('me.js', () => {
  let meModule;
  let wsInstances;

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
      });
      wsInstances.push(this);
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

  function buildAuthDOM() {
    document.body.innerHTML = `
      <header class="topbar">
        <a href="/" class="brand" id="brand">Своя Бронь</a>
      </header>

      <main class="me-container">
        <section class="me-grid-wrap" aria-live="polite">
          <div class="me-grid">
            <div class="col" data-status="in_use">
              <div class="col-body" id="col-in_use" aria-label="Используется"></div>
            </div>

            <div class="col" data-status="booked">
              <div class="col-body" id="col-booked" aria-label="Забронирована"></div>
            </div>

            <div class="col" data-status="finished">
              <div class="col-body" id="col-finished" aria-label="Завершён"></div>
            </div>

            <div class="col" data-status="canceled">
              <div class="col-body" id="col-canceled" aria-label="Отменён"></div>
            </div>
          </div>

          <div id="emptyMessage" class="empty-message hidden">Ваша история бронирования пуста</div>
        </section>

        <footer class="me-bottombar" id="meBottomBar" role="contentinfo" aria-hidden="false">
          <div class="me-bottombar__inner">
            <div class="me-bottombar__left"></div>
            <div class="me-bottombar__center">
              <button id="logoutBtn" class="btn">Выйти</button>
            </div>
            <div class="me-bottombar__right"></div>
          </div>
        </footer>
      </main>

      <div id="confirmModal" class="modal-overlay hidden" aria-hidden="true" role="dialog" aria-modal="true">
        <div class="modal-content small">
          <div class="modal-body">
            <p id="confirmText">Уверены, что хотите отменить бронь?</p>
            <div class="modal-actions">
              <button id="confirmNo" class="btn">Нет</button>
              <button id="confirmYes" class="btn primary">Отменить</button>
            </div>
          </div>
        </div>
      </div>
    `;
  }

  function buildNotAuthDOM() {
    document.body.innerHTML = `
      <main class="me-container">
        <div class="not-auth" role="status" aria-live="polite">
          <h2>Вы не авторизованы</h2>
          <p>Вам необходимо авторизоваться</p>
          <a class="btn primary" href="/auth">Войти</a>
        </div>

        <footer class="me-bottombar" id="meBottomBar" role="contentinfo" aria-hidden="false">
          <div class="me-bottombar__inner">
            <div class="me-bottombar__center">
              <button id="logoutBtn" class="btn">Выйти</button>
            </div>
          </div>
        </footer>
      </main>

      <div id="confirmModal" class="modal-overlay hidden" aria-hidden="true" role="dialog" aria-modal="true">
        <div class="modal-content small">
          <div class="modal-body">
            <p id="confirmText">Уверены, что хотите отменить бронь?</p>
            <div class="modal-actions">
              <button id="confirmNo" class="btn">Нет</button>
              <button id="confirmYes" class="btn primary">Отменить</button>
            </div>
          </div>
        </div>
      </div>
    `;
  }

  function loadModule() {
    jest.isolateModules(() => {
      meModule = require('./me.js');
    });
  }

  beforeEach(() => {
    jest.resetModules();
    jest.useFakeTimers();

    wsInstances = [];

    global.fetch = jest.fn();
    global.alert = jest.fn();
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

    delete window.location;
    window.location = {
      protocol: 'http:',
      host: 'localhost:8080',
      href: '',
      reload: jest.fn()
    };

    document.body.innerHTML = '';
    window.__USER__ = { auth: true, username: 'slava' };
    buildAuthDOM();

    loadModule();
  });

  afterEach(() => {
    jest.runOnlyPendingTimers();
    jest.useRealTimers();
    jest.restoreAllMocks();
  });

  test('getWsUrl: возвращает ws URL для http протокола', () => {
    // Техника тест-дизайна: классы эквивалентности
    window.location.protocol = 'http:';
    window.location.host = 'example.com';

    expect(meModule.getWsUrl()).toBe('ws://example.com/ws/me');
  });

  test('getWsUrl: возвращает wss URL для https протокола', () => {
    // Техника тест-дизайна: классы эквивалентности
    window.location.protocol = 'https:';
    window.location.host = 'secure.example.com';

    expect(meModule.getWsUrl()).toBe('wss://secure.example.com/ws/me');
  });

  test('clearColumns: очищает все колонки бронирований', () => {
    // Техника тест-дизайна: сценарий использования
    document.getElementById('col-in_use').innerHTML = '<div>1</div>';
    document.getElementById('col-booked').innerHTML = '<div>2</div>';
    document.getElementById('col-finished').innerHTML = '<div>3</div>';
    document.getElementById('col-canceled').innerHTML = '<div>4</div>';

    meModule.clearColumns();

    expect(document.getElementById('col-in_use').innerHTML).toBe('');
    expect(document.getElementById('col-booked').innerHTML).toBe('');
    expect(document.getElementById('col-finished').innerHTML).toBe('');
    expect(document.getElementById('col-canceled').innerHTML).toBe('');
  });

  test('openConfirmCancel: открывает модальное окно и показывает текст подтверждения', () => {
    // Техника тест-дизайна: переходы состояний
    meModule.openConfirmCancel('booking-1');

    const modal = document.getElementById('confirmModal');
    const text = document.getElementById('confirmText');

    expect(modal.classList.contains('hidden')).toBe(false);
    expect(modal.getAttribute('aria-hidden')).toBe('false');
    expect(text.textContent).toBe('Уверены, что хотите отменить бронь?');
  });

  test('closeConfirmCancel: закрывает модальное окно и скрывает его', () => {
    // Техника тест-дизайна: переходы состояний
    meModule.openConfirmCancel('booking-1');
    meModule.closeConfirmCancel();

    const modal = document.getElementById('confirmModal');

    expect(modal.classList.contains('hidden')).toBe(true);
    expect(modal.getAttribute('aria-hidden')).toBe('true');
  });

  test('createCard: создаёт карточку с данными брони и кнопкой отмены для статуса booked', () => {
    // Техника тест-дизайна: классы эквивалентности
    const card = meModule.createCard({
      id: 'b1',
      room_id: 'r1',
      title: 'Переговорная',
      image_url: 'img.png',
      date: '2026-04-24',
      start_time: '10:00',
      end_time: '11:00',
      total_price: 1500,
      status: 'booked'
    });

    expect(card.className).toBe('me-card');
    expect(card.dataset.bookingId).toBe('b1');
    expect(card.querySelector('.me-card__title').textContent).toBe('Переговорная');
    expect(card.querySelector('.me-card__date').textContent).toBe('2026-04-24');
    expect(card.querySelector('.me-card__time').textContent).toBe('10:00 — 11:00');
    expect(card.querySelector('.me-card__price').textContent).toBe('1500 ₽');
    expect(card.querySelector('.cancel-btn')).not.toBeNull();
  });

  test('createCard: для статуса не booked не создаёт кнопку отмены', () => {
    // Техника тест-дизайна: классы эквивалентности
    const card = meModule.createCard({
      id: 'b2',
      room_id: 'r2',
      title: 'Кабинет',
      image_url: 'img.png',
      date: '2026-04-24',
      start_time: '12:00',
      end_time: '13:00',
      total_price: 900,
      status: 'finished'
    });

    expect(card.querySelector('.cancel-btn')).toBeNull();
  });

  test('createCard: при отсутствии image_url подставляет placeholder', () => {
    // Техника тест-дизайна: классы эквивалентности
    const card = meModule.createCard({
      id: 'b3',
      room_id: 'r3',
      title: 'Комната',
      date: '2026-04-24',
      start_time: '14:00',
      end_time: '15:00',
      total_price: 500,
      status: 'finished'
    });

    expect(card.querySelector('img').src).toContain('/static/placeholders/room-placeholder.svg');
  });

  test('createCard: при клике по карточке переходит на страницу комнаты', () => {
    // Техника тест-дизайна: сценарий использования
    const card = meModule.createCard({
      id: 'b4',
      room_id: 'room-42',
      title: 'Комната',
      date: '2026-04-24',
      start_time: '14:00',
      end_time: '15:00',
      total_price: 500,
      status: 'finished'
    });

    card.click();

    expect(window.location.href).toBe('/room/room-42');
  });

  test('createCard: клик по кнопке отмены не ведёт на страницу комнаты, а открывает модальное окно', () => {
    // Техника тест-дизайна: таблица решений
    const card = meModule.createCard({
      id: 'b5',
      room_id: 'room-77',
      title: 'Комната',
      date: '2026-04-24',
      start_time: '16:00',
      end_time: '17:00',
      total_price: 800,
      status: 'booked'
    });

    const cancelBtn = card.querySelector('.cancel-btn');
    cancelBtn.click();

    expect(window.location.href).toBe('');
    expect(document.getElementById('confirmModal').classList.contains('hidden')).toBe(false);
  });

  test('renderBookings: при пустом массиве показывает сообщение о пустой истории', () => {
    // Техника тест-дизайна: классы эквивалентности
    meModule.renderBookings([]);

    expect(document.getElementById('emptyMessage').classList.contains('hidden')).toBe(false);
  });

  test('renderBookings: при не-массиве показывает сообщение о пустой истории', () => {
    // Техника тест-дизайна: классы эквивалентности
    meModule.renderBookings(null);

    expect(document.getElementById('emptyMessage').classList.contains('hidden')).toBe(false);
  });

  test('renderBookings: раскладывает бронирования по соответствующим колонкам', () => {
    // Техника тест-дизайна: сценарий использования
    meModule.renderBookings([
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
      },
      {
        id: '3',
        room_id: 'r3',
        title: 'Завершена',
        image_url: '3.png',
        date: '2026-04-24',
        start_time: '12:00',
        end_time: '13:00',
        total_price: 300,
        status: 'finished'
      },
      {
        id: '4',
        room_id: 'r4',
        title: 'Отменена',
        image_url: '4.png',
        date: '2026-04-24',
        start_time: '13:00',
        end_time: '14:00',
        total_price: 400,
        status: 'canceled'
      }
    ]);

    expect(document.getElementById('col-in_use').querySelectorAll('.me-card').length).toBe(1);
    expect(document.getElementById('col-booked').querySelectorAll('.me-card').length).toBe(1);
    expect(document.getElementById('col-finished').querySelectorAll('.me-card').length).toBe(1);
    expect(document.getElementById('col-canceled').querySelectorAll('.me-card').length).toBe(1);
    expect(document.getElementById('emptyMessage').classList.contains('hidden')).toBe(true);
  });

  test('renderBookings: неизвестный статус кладёт карточку в finished', () => {
    // Техника тест-дизайна: предугадывание ошибок
    meModule.renderBookings([
      {
        id: '10',
        room_id: 'r10',
        title: 'Неизвестный статус',
        image_url: '10.png',
        date: '2026-04-24',
        start_time: '15:00',
        end_time: '16:00',
        total_price: 1000,
        status: 'unexpected_status'
      }
    ]);

    expect(document.getElementById('col-finished').querySelectorAll('.me-card').length).toBe(1);
  });

  test('cancelBooking: при ok вызывает reloadPage', async () => {
    // Техника тест-дизайна: классы эквивалентности
    global.fetch.mockResolvedValue({ ok: true });
    const reloadSpy = jest.spyOn(meModule.uiHelpers, 'reloadPage').mockImplementation(() => {});

    await meModule.cancelBooking('booking-1');

    expect(global.fetch).toHaveBeenCalledWith('/api/booking/stop', {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ booking_id: 'booking-1' })
    });
    expect(reloadSpy).toHaveBeenCalled();
  });

  test('cancelBooking: при пустом bookingId ничего не делает', async () => {
    // Техника тест-дизайна: классы эквивалентности
    await meModule.cancelBooking('');

    expect(global.fetch).not.toHaveBeenCalled();
  });

  test('cancelBooking: при 400 показывает сообщение', async () => {
    // Техника тест-дизайна: таблица решений
    global.fetch.mockResolvedValue({ ok: false, status: 400 });

    await meModule.cancelBooking('booking-2');

    expect(global.alert).toHaveBeenCalledWith('Некорректные данные');
  });

  test('cancelBooking: при 403 показывает сообщение', async () => {
    // Техника тест-дизайна: таблица решений
    global.fetch.mockResolvedValue({ ok: false, status: 403 });

    await meModule.cancelBooking('booking-3');

    expect(global.alert).toHaveBeenCalledWith('Попытка отменить чужую бронь');
  });

  test('cancelBooking: при 404 показывает сообщение', async () => {
    // Техника тест-дизайна: таблица решений
    global.fetch.mockResolvedValue({ ok: false, status: 404 });

    await meModule.cancelBooking('booking-4');

    expect(global.alert).toHaveBeenCalledWith('Бронь не найдена');
  });

  test('cancelBooking: при 409 показывает сообщение и вызывает reloadPage', async () => {
    // Техника тест-дизайна: таблица решений
    global.fetch.mockResolvedValue({ ok: false, status: 409 });
    const reloadSpy = jest.spyOn(meModule.uiHelpers, 'reloadPage').mockImplementation(() => {});

    await meModule.cancelBooking('booking-5');

    expect(global.alert).toHaveBeenCalledWith('Бронь уже используется или завершена — обновляем страницу');
    expect(reloadSpy).toHaveBeenCalled();
  });

  test('cancelBooking: при 500 показывает сообщение об ошибке сервера', async () => {
    // Техника тест-дизайна: таблица решений
    global.fetch.mockResolvedValue({ ok: false, status: 500 });

    await meModule.cancelBooking('booking-6');

    expect(global.alert).toHaveBeenCalledWith('Ошибка доступа к серверу. Повторите попытку.');
  });

  test('cancelBooking: при сетевой ошибке показывает сообщение об ошибке соединения', async () => {
    // Техника тест-дизайна: предугадывание ошибок
    global.fetch.mockRejectedValue(new Error('Network error'));

    await meModule.cancelBooking('booking-7');

    expect(global.alert).toHaveBeenCalledWith('Ошибка соединения');
    expect(console.error).toHaveBeenCalled();
  });

  test('connectWs: создаёт websocket с корректным URL', () => {
    // Техника тест-дизайна: сценарий использования
    meModule.connectWs();

    expect(wsInstances.length).toBe(1);
    expect(wsInstances[0].url).toBe('ws://localhost:8080/ws/me');
  });

test('connectWs: сообщение-массив отрисовывает бронирование в колонке', () => {
  // Техника тест-дизайна: сценарий использования
  meModule.connectWs();
  const ws = wsInstances[0];

  ws.emit('message', {
    data: JSON.stringify([
      {
        id: '1',
        room_id: 'r1',
        title: 'Комната',
        date: '2026-04-24',
        start_time: '10:00',
        end_time: '11:00',
        total_price: 100,
        status: 'booked'
      }
    ])
  });

  const bookedCards = document.getElementById('col-booked').querySelectorAll('.me-card');
  expect(bookedCards.length).toBe(1);
  expect(document.getElementById('col-booked').textContent).toContain('Комната');
});

  test('connectWs: если пришёл не массив, пишет warning и не вызывает renderBookings', () => {
    // Техника тест-дизайна: классы эквивалентности
    const renderSpy = jest.spyOn(meModule, 'renderBookings');

    meModule.connectWs();
    const ws = wsInstances[0];

    ws.emit('message', {
      data: JSON.stringify({ id: 'not-array' })
    });

    expect(console.warn).toHaveBeenCalled();
    expect(renderSpy).not.toHaveBeenCalled();
  });

  test('connectWs: при невалидном JSON пишет warning', () => {
    // Техника тест-дизайна: предугадывание ошибок
    meModule.connectWs();
    const ws = wsInstances[0];

    ws.emit('message', {
      data: 'not-json'
    });

    expect(console.warn).toHaveBeenCalled();
  });

  test('connectWs: при ошибке websocket пишет error в консоль', () => {
    // Техника тест-дизайна: предугадывание ошибок
    meModule.connectWs();
    const ws = wsInstances[0];

    ws.emit('error', { message: 'ws failed' });

    expect(console.error).toHaveBeenCalled();
  });

  test('DOMContentLoaded: для неавторизованного пользователя удаляет нижнюю панель и скрывает logout', () => {
    // Техника тест-дизайна: классы эквивалентности
    jest.resetModules();
    wsInstances = [];
    buildNotAuthDOM();
    window.__USER__ = { auth: false, username: '' };

    loadModule();
    document.dispatchEvent(new Event('DOMContentLoaded'));

    expect(document.getElementById('meBottomBar')).toBeNull();
  });

test('DOMContentLoaded: для авторизованного пользователя создаёт websocket соединение', () => {
  // Техника тест-дизайна: сценарий использования
  document.dispatchEvent(new Event('DOMContentLoaded'));

  expect(wsInstances.length).toBeGreaterThan(0);
  expect(wsInstances[0].url).toBe('ws://localhost:8080/ws/me');
});
  test('DOMContentLoaded: confirmNo закрывает модальное окно', () => {
    // Техника тест-дизайна: переходы состояний
    document.dispatchEvent(new Event('DOMContentLoaded'));
    meModule.openConfirmCancel('booking-8');

    document.getElementById('confirmNo').click();

    expect(document.getElementById('confirmModal').classList.contains('hidden')).toBe(true);
  });

  test('DOMContentLoaded: клик по confirmModal overlay закрывает модальное окно', () => {
    // Техника тест-дизайна: сценарий использования
    document.dispatchEvent(new Event('DOMContentLoaded'));
    meModule.openConfirmCancel('booking-9');

    const modal = document.getElementById('confirmModal');
    modal.dispatchEvent(new MouseEvent('click', { bubbles: true }));

    expect(modal.classList.contains('hidden')).toBe(true);
  });

  test('DOMContentLoaded: Escape закрывает модальное окно', () => {
    // Техника тест-дизайна: переходы состояний
    document.dispatchEvent(new Event('DOMContentLoaded'));
    meModule.openConfirmCancel('booking-10');

    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }));

    expect(document.getElementById('confirmModal').classList.contains('hidden')).toBe(true);
  });

 test('DOMContentLoaded: confirmYes отправляет отмену выбранной брони и закрывает модальное окно', async () => {
  // Техника тест-дизайна: сценарий использования
  global.fetch.mockResolvedValue({ ok: true });

  document.dispatchEvent(new Event('DOMContentLoaded'));

  meModule.openConfirmCancel('booking-11');
  document.getElementById('confirmYes').click();

  await Promise.resolve();
  await Promise.resolve();

  expect(global.fetch).toHaveBeenCalledWith('/api/booking/stop', {
    method: 'POST',
    credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ booking_id: 'booking-11' })
  });

  expect(document.getElementById('confirmModal').classList.contains('hidden')).toBe(true);
});
  test('DOMContentLoaded: logout при ok выполняет переход на главную', async () => {
    // Техника тест-дизайна: сценарий использования
    global.fetch.mockResolvedValue({ ok: true });
    document.dispatchEvent(new Event('DOMContentLoaded'));

    document.getElementById('logoutBtn').click();
    await Promise.resolve();
    await Promise.resolve();

    expect(window.location.href).toBe('/');
  });

  test('DOMContentLoaded: logout при 400 показывает сообщение', async () => {
    // Техника тест-дизайна: таблица решений
    global.fetch.mockResolvedValue({ ok: false, status: 400 });
    document.dispatchEvent(new Event('DOMContentLoaded'));

    document.getElementById('logoutBtn').click();
    await Promise.resolve();
    await Promise.resolve();

    expect(global.alert).toHaveBeenCalledWith('Сессия отсутствует или истекла');
  });

  test('DOMContentLoaded: logout при 500 показывает сообщение', async () => {
    // Техника тест-дизайна: таблица решений
    global.fetch.mockResolvedValue({ ok: false, status: 500 });
    document.dispatchEvent(new Event('DOMContentLoaded'));

    document.getElementById('logoutBtn').click();
    await Promise.resolve();
    await Promise.resolve();

    expect(global.alert).toHaveBeenCalledWith('Ошибка доступа к серверу. Повторите попытку.');
  });

  test('DOMContentLoaded: logout при сетевой ошибке показывает сообщение об ошибке соединения', async () => {
    // Техника тест-дизайна: предугадывание ошибок
    global.fetch.mockRejectedValue(new Error('Network error'));
    document.dispatchEvent(new Event('DOMContentLoaded'));

    document.getElementById('logoutBtn').click();
    await Promise.resolve();
    await Promise.resolve();

    expect(global.alert).toHaveBeenCalledWith('Ошибка соединения');
    expect(console.error).toHaveBeenCalled();
  });

  test('DOMContentLoaded: если logoutBtn отсутствует, создаётся новый logout-кнопка в bottomBar', () => {
    // Техника тест-дизайна: предугадывание ошибок
    jest.resetModules();
    wsInstances = [];
    buildAuthDOM();
    window.__USER__ = { auth: true, username: 'slava' };

    const oldBtn = document.getElementById('logoutBtn');
    oldBtn.remove();

    loadModule();
    document.dispatchEvent(new Event('DOMContentLoaded'));

    const newBtn = document.getElementById('logoutBtn');
    expect(newBtn).not.toBeNull();
    expect(newBtn.textContent).toBe('Выйти');
  });

  test('динамически созданная logout-кнопка: при ok выполняет переход на главную', async () => {
    // Техника тест-дизайна: сценарий использования
    jest.resetModules();
    wsInstances = [];
    buildAuthDOM();
    window.__USER__ = { auth: true, username: 'slava' };
    global.fetch.mockResolvedValue({ ok: true });

    document.getElementById('logoutBtn').remove();

    loadModule();
    document.dispatchEvent(new Event('DOMContentLoaded'));

    document.getElementById('logoutBtn').click();
    await Promise.resolve();
    await Promise.resolve();

    expect(window.location.href).toBe('/');
  });

  test('динамически созданная logout-кнопка: при ошибке сервера показывает "Ошибка выхода"', async () => {
    // Техника тест-дизайна: таблица решений
    jest.resetModules();
    wsInstances = [];
    buildAuthDOM();
    window.__USER__ = { auth: true, username: 'slava' };
    global.fetch.mockResolvedValue({ ok: false, status: 500 });

    document.getElementById('logoutBtn').remove();

    loadModule();
    document.dispatchEvent(new Event('DOMContentLoaded'));

    document.getElementById('logoutBtn').click();
    await Promise.resolve();
    await Promise.resolve();

    expect(global.alert).toHaveBeenCalledWith('Ошибка выхода');
  });

  test('динамически созданная logout-кнопка: при сетевой ошибке показывает "Ошибка соединения"', async () => {
    // Техника тест-дизайна: предугадывание ошибок
    jest.resetModules();
    wsInstances = [];
    buildAuthDOM();
    window.__USER__ = { auth: true, username: 'slava' };
    global.fetch.mockRejectedValue(new Error('Network error'));

    document.getElementById('logoutBtn').remove();

    loadModule();
    document.dispatchEvent(new Event('DOMContentLoaded'));

    document.getElementById('logoutBtn').click();
    await Promise.resolve();
    await Promise.resolve();

    expect(global.alert).toHaveBeenCalledWith('Ошибка соединения');
    expect(console.error).toHaveBeenCalled();
  });
});