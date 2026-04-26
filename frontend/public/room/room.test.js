/**
 * room.test.js
 * Тесты для room.js
 */

describe('room.js', () => {
  let roomModule;
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
        this.emit('close');
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

  function buildDOM() {
    document.body.innerHTML = `
      <header class="topbar">
        <div class="topbar__left">
          <a href="/" class="brand">Своя Бронь</a>
        </div>
        <div class="topbar__right">
          <div class="auth-container">
            <button id="authButton" class="auth-btn" data-auth="1" data-username="slava">Кабинет</button>
            <div id="cabinetMenu" class="cabinet-menu hidden"></div>
            <button id="logoutBtn" class="cabinet-btn">Выйти</button>
          </div>
        </div>
      </header>

      <main class="container room-page">
        <section class="room-head">
          <div>
            <div id="mainImageWrap" class="main-image">
              <img id="mainImage" src="/img/1.png" alt="Комната">
            </div>

            <div id="thumbs" class="thumbs">
              <button class="thumb-btn" data-src="/img/2.png" aria-label="Изображение 1">
                <img src="/img/2.png" alt="thumb-1">
              </button>
              <button class="thumb-btn" data-src="/img/3.png" aria-label="Изображение 2">
                <img src="/img/3.png" alt="thumb-2">
              </button>
            </div>
          </div>

          <aside class="room-info">
            <button id="bookBtn" class="btn primary">Забронировать</button>
          </aside>
        </section>

        <section class="room-body">
          <aside class="sidebar">
            <div class="map-card">
              <div id="map"></div>
            </div>
          </aside>
        </section>
      </main>

      <div id="bookingModal" class="booking-modal hidden" aria-hidden="true">
        <div class="booking-modal__content" role="document">
          <button class="modal-close" id="bookingClose" aria-label="Закрыть">&times;</button>
          <h3 id="bookingTitle">Бронирование</h3>
          <div id="bookingNotice"></div>
          <div class="booking-calendar" id="bookingCalendar"></div>
          <div class="booking-times" id="bookingTimes"></div>
          <div class="booking-footer">
            <div id="selectionSummary"></div>
            <button id="bookingCancel" class="btn">Отмена</button>
            <button id="bookingConfirm" class="btn primary">Подтвердить</button>
          </div>
        </div>
      </div>
    `;
  }

  function loadModule() {
    jest.isolateModules(() => {
      roomModule = require('./room.js');
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

    document.body.innerHTML = '';
    buildDOM();

    window.__USER__ = { auth: true, username: 'slava' };
    window.__ROOM__ = {
      id: '101',
      title: 'Переговорная',
      lat: 59.93,
      lng: 30.31,
      address: 'Невский проспект'
    };

    loadModule();
  });

  afterEach(() => {
    jest.runOnlyPendingTimers();
    jest.useRealTimers();
    jest.restoreAllMocks();
  });

  test('parseTimeToMinutes: корректно переводит время в минуты', () => {
    // Техника тест-дизайна: классы эквивалентности
    expect(roomModule.parseTimeToMinutes('10:30')).toBe(630);
  });

  test('parseTimeToMinutes: бросает ошибку для невалидного формата', () => {
    // Техника тест-дизайна: классы эквивалентности
    expect(() => roomModule.parseTimeToMinutes('1030')).toThrow('Invalid time format');
  });

  test('nextHourLabel: корректно добавляет 1 час', () => {
    // Техника тест-дизайна: классы эквивалентности
    expect(roomModule.nextHourLabel('10:00')).toBe('11:00');
  });

  test('nextHourLabel: корректно переходит через полночь', () => {
    // Техника тест-дизайна: граничные условия
    expect(roomModule.nextHourLabel('23:30')).toBe('00:30');
  });

  test('formatYMD: форматирует дату в YYYY-MM-DD', () => {
    // Техника тест-дизайна: классы эквивалентности
    const d = new Date(2026, 3, 24);
    expect(roomModule.formatYMD(d)).toBe('2026-04-24');
  });

  test('weekdayRu: возвращает русское сокращение дня недели', () => {
    // Техника тест-дизайна: классы эквивалентности
    const d = new Date(2026, 3, 24);
    expect(roomModule.weekdayRu(d)).toBe('Пт');
  });

 test('formatDayLabel: возвращает день в формате D.MM', () => {
  // Техника тест-дизайна: классы эквивалентности
  const d = new Date(2026, 3, 5);
  expect(roomModule.formatDayLabel(d)).toBe('5.04');
});

  test('nextDay: возвращает следующую дату', () => {
    // Техника тест-дизайна: классы эквивалентности
    const d = new Date(2026, 3, 24);
    const next = roomModule.nextDay(d, 2);

    expect(roomModule.formatYMD(next)).toBe('2026-04-26');
  });

  test('navigate: меняет window.location.href', () => {
    // Техника тест-дизайна: классы эквивалентности
    roomModule.navigate('/auth');
    expect(window.location.href).toBe('/auth');
  });

  test('detectAuthEarly: читает auth из data-auth кнопки', () => {
    // Техника тест-дизайна: классы эквивалентности
    document.getElementById('authButton').dataset.auth = '1';
    expect(roomModule.detectAuthEarly()).toBe(true);
  });

  test('detectAuthEarly: при отсутствии data-auth берёт auth из window.__USER__', () => {
    // Техника тест-дизайна: классы эквивалентности
    document.getElementById('authButton').remove();
    window.__USER__ = { auth: true, username: 'slava' };

    expect(roomModule.detectAuthEarly()).toBe(true);
  });

  test('readServerAuth: читает auth и username из authButton', () => {
    // Техника тест-дизайна: классы эквивалентности
    document.getElementById('authButton').dataset.auth = '1';
    document.getElementById('authButton').dataset.username = 'slava';

    expect(roomModule.readServerAuth()).toEqual({ auth: true, username: 'slava' });
  });

  test('showBookHint: добавляет подсказку для неавторизованного пользователя', () => {
    // Техника тест-дизайна: сценарий использования
    roomModule.showBookHint(true);

    const hint = document.getElementById('bookHint');
    expect(hint).not.toBeNull();
    expect(hint.textContent).toBe('Вам необходимо войти в систему для бронирования');
  });

  test('showBookHint: удаляет подсказку при show=false', () => {
    // Техника тест-дизайна: переходы состояний
    roomModule.showBookHint(true);
    roomModule.showBookHint(false);

    expect(document.getElementById('bookHint')).toBeNull();
  });

  test('buildCalendarDays: создаёт 7 дней календаря', () => {
    // Техника тест-дизайна: классы эквивалентности
    roomModule.resetState();
    roomModule.buildCalendarDays();
    roomModule.renderCalendar();

    expect(document.querySelectorAll('#bookingCalendar .day').length).toBe(7);
  });

  test('updateSummary: без даты и слотов показывает сообщение о пустом выборе', () => {
    // Техника тест-дизайна: классы эквивалентности
    roomModule.resetState();
    roomModule.updateSummary();

    expect(document.getElementById('selectionSummary').textContent).toBe('Слоты: не выбраны');
  });

  test('renderTimes: без выбранной даты показывает предложение выбрать дату', () => {
    // Техника тест-дизайна: классы эквивалентности
    roomModule.resetState();
    roomModule.renderTimes([]);

    expect(document.getElementById('bookingTimes').textContent).toContain('Выберите дату');
  });

  test('renderCalendar: отрисовывает 7 кнопок дней', () => {
    // Техника тест-дизайна: сценарий использования
    roomModule.resetState();
    roomModule.buildCalendarDays();
    roomModule.renderCalendar();

    expect(document.querySelectorAll('#bookingCalendar .day').length).toBe(7);
  });

test('renderTimes: показывает доступные слоты', () => {
  // Техника тест-дизайна: сценарий использования
  roomModule.resetState();
  roomModule.openBookingModal();

  const ws = wsInstances[0];
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

  const slots = document.querySelectorAll('#bookingTimes .slot');
  expect(slots.length).toBe(2);
  expect(slots[0].textContent).toBe('10:00 — 11:00');
  expect(slots[1].textContent).toBe('11:00 — 12:00');
});
  test('renderTimes: при пустом массиве слотов показывает сообщение', () => {
    // Техника тест-дизайна: классы эквивалентности
    roomModule.resetState();
    roomModule.buildCalendarDays();
    roomModule.renderCalendar();

    const firstDayBtn = document.querySelector('#bookingCalendar .day');
    if (firstDayBtn) firstDayBtn.click();

    roomModule.renderTimes([]);

    expect(document.getElementById('bookingTimes').textContent).toContain('Нет доступных слотов');
  });

  test('onSlotClick: первый клик выбирает один слот', () => {
    // Техника тест-дизайна: классы эквивалентности
    roomModule.resetState();
    roomModule.buildCalendarDays();
    roomModule.openBookingModal();

    const ws = wsInstances[0];
    ws.emit('open');
    ws.emit('message', {
      data: JSON.stringify({
        dates: [
          { date: document.querySelector('#bookingCalendar .day')?.dataset.ymd, available_times: ['10:00', '11:00', '12:00'] }
        ]
      })
    });

    roomModule.onSlotClick('10:00');

    expect(document.getElementById('selectionSummary').textContent).toContain('10:00');
  });

  test('onSlotClick: смежный слот расширяет последовательность', () => {
    // Техника тест-дизайна: попарное тестирование
    roomModule.resetState();
    roomModule.openBookingModal();

    const ws = wsInstances[0];
    ws.emit('open');

    const ymd = document.querySelector('#bookingCalendar .day')?.dataset.ymd;
    ws.emit('message', {
      data: JSON.stringify({
        dates: [
          { date: ymd, available_times: ['10:00', '11:00', '12:00'] }
        ]
      })
    });

    roomModule.onSlotClick('10:00');
    roomModule.onSlotClick('11:00');

    expect(document.getElementById('selectionSummary').textContent).toContain('10:00, 11:00');
  });

  test('onSlotClick: несмежный слот сбрасывает выбор к одному слоту', () => {
    // Техника тест-дизайна: попарное тестирование
    roomModule.resetState();
    roomModule.openBookingModal();

    const ws = wsInstances[0];
    ws.emit('open');

    const ymd = document.querySelector('#bookingCalendar .day')?.dataset.ymd;
    ws.emit('message', {
      data: JSON.stringify({
        dates: [
          { date: ymd, available_times: ['10:00', '11:00', '13:00'] }
        ]
      })
    });

    roomModule.onSlotClick('10:00');
    roomModule.onSlotClick('13:00');

    expect(document.getElementById('selectionSummary').textContent).toContain('13:00');
    expect(document.getElementById('selectionSummary').textContent).not.toContain('10:00, 13:00');
  });

  test('getBookingWsUrl: возвращает ws URL для http', () => {
    // Техника тест-дизайна: классы эквивалентности
    window.location.protocol = 'http:';
    window.location.host = 'example.com';

    expect(roomModule.getBookingWsUrl()).toBe('ws://example.com/ws/booking');
  });

  test('getBookingWsUrl: возвращает wss URL для https', () => {
    // Техника тест-дизайна: классы эквивалентности
    window.location.protocol = 'https:';
    window.location.host = 'secure.example.com';

    expect(roomModule.getBookingWsUrl()).toBe('wss://secure.example.com/ws/booking');
  });

  test('openWsBooking: создаёт websocket и при open отправляет room_id', () => {
    // Техника тест-дизайна: сценарий использования
    roomModule.resetState();
    roomModule.openWsBooking();

    expect(wsInstances.length).toBe(1);
    expect(wsInstances[0].url).toBe('ws://localhost:8080/ws/booking');

    wsInstances[0].emit('open');

    expect(wsInstances[0].send).toHaveBeenCalledWith(JSON.stringify({ room_id: 101 }));
  });

  test('openWsBooking: сообщение с датами обновляет календарь и слоты', () => {
    // Техника тест-дизайна: сценарий использования
    roomModule.resetState();
    roomModule.buildCalendarDays();
    roomModule.renderCalendar();
    roomModule.openWsBooking();

    const ws = wsInstances[0];
    ws.emit('open');

    const dayButtons = document.querySelectorAll('#bookingCalendar .day');
    const firstYmd = dayButtons[0].dataset.ymd;

    ws.emit('message', {
      data: JSON.stringify({
        dates: [
          { date: firstYmd, available_times: ['10:00', '11:00'] }
        ]
      })
    });

    expect(document.getElementById('bookingTimes').textContent).toContain('10:00 — 11:00');
    expect(document.getElementById('bookingNotice').textContent).toBe('Выберите дату и слоты');
  });

  test('openWsBooking: невалидный JSON не выбрасывает исключение и пишет warning', () => {
    // Техника тест-дизайна: предугадывание ошибок
    roomModule.resetState();
    roomModule.openWsBooking();

    const ws = wsInstances[0];

    expect(() => {
      ws.emit('message', { data: 'not-json' });
    }).not.toThrow();

    expect(console.warn).toHaveBeenCalled();
  });

  test('closeBookingModal: скрывает модалку и закрывает websocket', () => {
    // Техника тест-дизайна: переходы состояний
    roomModule.resetState();
    roomModule.openBookingModal();

    const ws = wsInstances[0];
    roomModule.closeBookingModal();

    expect(document.getElementById('bookingModal').classList.contains('hidden')).toBe(true);
    expect(document.getElementById('bookingModal').getAttribute('aria-hidden')).toBe('true');
    expect(ws.close).toHaveBeenCalled();
  });

  test('openBookingModal: для неавторизованного пользователя ведёт на /auth', () => {
    // Техника тест-дизайна: таблица решений
    document.getElementById('authButton').dataset.auth = '0';

    roomModule.openBookingModal();

    expect(window.location.href).toBe('/auth');
  });

  test('openBookingModal: для авторизованного пользователя открывает модалку', () => {
    // Техника тест-дизайна: сценарий использования
    document.getElementById('authButton').dataset.auth = '1';

    roomModule.openBookingModal();

    const modal = document.getElementById('bookingModal');
    expect(modal.classList.contains('hidden')).toBe(false);
    expect(modal.getAttribute('aria-hidden')).toBe('false');
    expect(document.body.style.overflow).toBe('hidden');
  });

  test('confirmBooking: без даты и слотов показывает alert', async () => {
    // Техника тест-дизайна: классы эквивалентности
    roomModule.resetState();

    await roomModule.confirmBooking();

    expect(global.alert).toHaveBeenCalledWith('Выберите дату и хотя бы один слот');
  });



  test('confirmBooking: при успешном бронировании показывает успешное содержимое модалки', async () => {
    // Техника тест-дизайна: сценарий использования
    global.fetch.mockResolvedValue({ ok: true });

    roomModule.resetState();
    roomModule.openBookingModal();

    const ws = wsInstances[0];
    ws.emit('open');

    const ymd = document.querySelector('#bookingCalendar .day')?.dataset.ymd;
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

  test('confirmBooking: при ответе сервера с ошибкой показывает alert', async () => {
    // Техника тест-дизайна: таблица решений
    global.fetch.mockResolvedValue({ ok: false });

    roomModule.resetState();
    roomModule.openBookingModal();

    const ws = wsInstances[0];
    ws.emit('open');

    const ymd = document.querySelector('#bookingCalendar .day')?.dataset.ymd;
    ws.emit('message', {
      data: JSON.stringify({
        dates: [
          { date: ymd, available_times: ['10:00'] }
        ]
      })
    });

    roomModule.onSlotClick('10:00');

    await roomModule.confirmBooking();

    expect(global.alert).toHaveBeenCalledWith('Ошибка бронирования');
  });

  test('confirmBooking: при сетевой ошибке показывает alert', async () => {
    // Техника тест-дизайна: предугадывание ошибок
    global.fetch.mockRejectedValue(new Error('Network error'));

    roomModule.resetState();
    roomModule.openBookingModal();

    const ws = wsInstances[0];
    ws.emit('open');

    const ymd = document.querySelector('#bookingCalendar .day')?.dataset.ymd;
    ws.emit('message', {
      data: JSON.stringify({
        dates: [
          { date: ymd, available_times: ['10:00'] }
        ]
      })
    });

    roomModule.onSlotClick('10:00');

    await roomModule.confirmBooking();

    expect(global.alert).toHaveBeenCalledWith('Ошибка соединения');
  });

  test('DOMContentLoaded: клик по thumbnail меняет главное изображение', () => {
    // Техника тест-дизайна: сценарий использования
    document.dispatchEvent(new Event('DOMContentLoaded'));

    document.querySelector('.thumb-btn').click();

    expect(document.getElementById('mainImage').src).toContain('/img/2.png');
  });

  test('DOMContentLoaded: при наличии координат инициализирует карту Leaflet', () => {
    // Техника тест-дизайна: сценарий использования
    document.dispatchEvent(new Event('DOMContentLoaded'));

    expect(global.L.map).toHaveBeenCalled();
    expect(global.L.tileLayer).toHaveBeenCalled();
    expect(global.L.marker).toHaveBeenCalled();
  });

  test('DOMContentLoaded: при отсутствии координат показывает сообщение', () => {
    // Техника тест-дизайна: классы эквивалентности
    window.__ROOM__.lat = null;
    window.__ROOM__.lng = null;

    document.dispatchEvent(new Event('DOMContentLoaded'));

    expect(document.getElementById('map').textContent).toContain('Координаты отсутствуют');
  });

  test('DOMContentLoaded: для неавторизованного пользователя отключает кнопку бронирования и показывает подсказку', () => {
    // Техника тест-дизайна: классы эквивалентности
    document.getElementById('authButton').dataset.auth = '0';

    document.dispatchEvent(new Event('DOMContentLoaded'));

    expect(document.getElementById('bookBtn').disabled).toBe(true);
    expect(document.getElementById('bookHint')).not.toBeNull();
  });


test('openBookingModal: для неавторизованного пользователя ведёт на /auth', () => {
  // Техника тест-дизайна: таблица решений
  document.getElementById('authButton').dataset.auth = '0';

  roomModule.openBookingModal();

  expect(window.location.href).toBe('/auth');
});
  test('DOMContentLoaded: bookingClose закрывает модалку', () => {
    // Техника тест-дизайна: переходы состояний
    document.dispatchEvent(new Event('DOMContentLoaded'));
    roomModule.openBookingModal();

    document.getElementById('bookingClose').click();

    expect(document.getElementById('bookingModal').classList.contains('hidden')).toBe(true);
  });

  test('DOMContentLoaded: bookingCancel закрывает модалку', () => {
    // Техника тест-дизайна: переходы состояний
    document.dispatchEvent(new Event('DOMContentLoaded'));
    roomModule.openBookingModal();

    document.getElementById('bookingCancel').click();

    expect(document.getElementById('bookingModal').classList.contains('hidden')).toBe(true);
  });

  test('DOMContentLoaded: клик по overlay bookingModal закрывает модалку', () => {
    // Техника тест-дизайна: сценарий использования
    document.dispatchEvent(new Event('DOMContentLoaded'));
    roomModule.openBookingModal();

    const modal = document.getElementById('bookingModal');
    modal.dispatchEvent(new MouseEvent('click', { bubbles: true }));

    expect(modal.classList.contains('hidden')).toBe(true);
  });

  test('DOMContentLoaded: authButton ведёт в /me для авторизованного пользователя', () => {
    // Техника тест-дизайна: таблица решений
    document.getElementById('authButton').dataset.auth = '1';

    document.dispatchEvent(new Event('DOMContentLoaded'));
    document.getElementById('authButton').click();

    expect(window.location.href).toBe('/me');
  });

  test('DOMContentLoaded: authButton ведёт в /auth для неавторизованного пользователя', () => {
    // Техника тест-дизайна: таблица решений
    document.getElementById('authButton').dataset.auth = '0';

    document.dispatchEvent(new Event('DOMContentLoaded'));
    document.getElementById('authButton').click();

    expect(window.location.href).toBe('/auth');
  });

  test('DOMContentLoaded: logout при ok вызывает reload', async () => {
    // Техника тест-дизайна: сценарий использования
    global.fetch.mockResolvedValue({ ok: true });

    document.dispatchEvent(new Event('DOMContentLoaded'));
    document.getElementById('logoutBtn').click();

    await Promise.resolve();
    await Promise.resolve();

    expect(global.fetch).toHaveBeenCalledWith('/api/logout', {
      method: 'POST',
      credentials: 'same-origin'
    });
    expect(window.location.reload).toHaveBeenCalled();
  });

  test('DOMContentLoaded: logout при сетевой ошибке показывает alert', async () => {
    // Техника тест-дизайна: предугадывание ошибок
    global.fetch.mockRejectedValue(new Error('Network error'));

    document.dispatchEvent(new Event('DOMContentLoaded'));
    document.getElementById('logoutBtn').click();

    await Promise.resolve();
    await Promise.resolve();

    expect(global.alert).toHaveBeenCalledWith('Ошибка соединения');
  });
});