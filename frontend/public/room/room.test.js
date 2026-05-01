
const roomModule = require('./room.js');

describe('room.js', () => {
  async function flushPromises() {
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();
  }

  function buildRoomDOM() {
    document.body.innerHTML = `
      <header class="topbar">
        <div class="topbar__left">
          <a href="/" class="brand">Своя Бронь</a>
        </div>

        <div class="topbar__right">
          <div class="auth-container">
            <button id="authButton" class="auth-btn" data-auth="0">Войти</button>
            <div id="cabinetMenu" class="cabinet-menu hidden" aria-hidden="true"></div>
            <span id="userGreeting">Привет, </span>
            <button id="historyBtn" type="button">История броней</button>
            <button id="logoutBtn" type="button">Выйти</button>
          </div>
        </div>
      </header>

      <main class="container room-page">
        <section class="room-head">
          <div>
            <div id="mainImageWrap" class="main-image">
              <img id="mainImage" src="" alt="">
            </div>

            <div id="thumbs" class="thumbs"></div>
          </div>

          <aside class="room-info">
            <h1></h1>
            <div class="company-name"></div>
            <div class="address"></div>

            <div class="price-capacity">
              <div class="price"></div>
              <div class="capacity"></div>
            </div>

            <div class="availability">
              <div class="label">Время работы</div>
              <div class="value"></div>
            </div>

            <div class="book-button">
              <button id="bookBtn" class="btn primary">Забронировать</button>
            </div>
          </aside>
        </section>

        <section class="room-body">
          <article class="description-card">
            <h2>Описание</h2>
            <div id="description"></div>
          </article>

          <aside class="sidebar">
            <div class="map-card">
              <h3>Местоположение</h3>
              <div id="map"></div>
            </div>

            <div class="details-card">
              <h3>Дополнительно</h3>
              <ul></ul>
            </div>
          </aside>
        </section>
      </main>

      <div id="bookingModal" class="booking-modal hidden" aria-hidden="true">
        <div class="booking-modal__content">
          <button id="bookingClose" type="button">×</button>
          <div id="bookingNotice"></div>
          <div id="bookingCalendar"></div>
          <div id="bookingTimes"></div>
          <div id="selectionSummary"></div>
          <button id="bookingCancel" type="button">Отмена</button>
          <button id="bookingConfirm" type="button">Подтвердить</button>
        </div>
      </div>
    `;
  }

  function makeApi(overrides = {}) {
    return {
      getMe: jest.fn().mockResolvedValue({
        authenticated: false,
        user: null
      }),
      getRoom: jest.fn().mockResolvedValue({
        room: {
          id: 101,
          title: 'Переговорная',
          company_name: 'Компания А',
          address: 'Адрес 1',
          images: ['/img/1.png', '/img/2.png'],
          price: 1500,
          capacity: 8,
          max_capacity: 10,
          available_from: '09:00',
          available_to: '21:00',
          description: 'Описание комнаты',
          lat: 55.75,
          lng: 37.61
        }
      }),
      getRoomAvailability: jest.fn().mockResolvedValue({
        dates: []
      }),
      createBooking: jest.fn().mockResolvedValue({}),
      logoutUser: jest.fn().mockResolvedValue({}),
      ...overrides
    };
  }

  beforeEach(() => {
    jest.clearAllMocks();

    global.alert = jest.fn();

    global.console = {
      log: jest.fn(),
      warn: jest.fn(),
      error: jest.fn()
    };

    global.L = {
      map: jest.fn(() => ({
        setView: jest.fn().mockReturnThis(),
        attributionControl: { setPrefix: jest.fn() },
        remove: jest.fn()
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
      href: '',
      pathname: '/room/101',
      reload: jest.fn()
    };

    document.body.innerHTML = '';
    document.body.dataset.roomId = '';
    buildRoomDOM();

    window.Api = makeApi();

    roomModule.__resetStateForTests();
  });

  test('getRoomIdFromUrl: берёт id из pathname /room/:id', () => {
    // Техника тест-дизайна: классы эквивалентности
    window.location.pathname = '/room/123';

    expect(roomModule.getRoomIdFromUrl()).toBe(123);
  });


  test('toNumberOrNull: возвращает число для корректного значения', () => {
    // Техника тест-дизайна: классы эквивалентностаи
    expect(roomModule.toNumberOrNull('55.75')).toBe(55.75);
  });

  test('toNumberOrNull: возвращает null для некорректного значения', () => {
    // Техника тест-дизайна: классы эквивалентности
    expect(roomModule.toNumberOrNull('abc')).toBeNull();
  });

  test('parseTimeToMinutes: корректно переводит время в минуты', () => {
    // Техника тест-дизайна: классы эквивалентности
    expect(roomModule.parseTimeToMinutes('10:30')).toBe(630);
  });

  test('parseTimeToMinutes: бросает ошибку при неверном формате', () => {
    // Техника тест-дизайна: классы эквивалентности
    expect(() => roomModule.parseTimeToMinutes('1030')).toThrow('Invalid time format');
  });

  test('nextHourLabel: корректно вычисляет следующий час', () => {
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

  test('escapeHtml: экранирует HTML-символы', () => {
    // Техника тест-дизайна: классы эквивалентности
    expect(roomModule.escapeHtml('<script>"x"&\'y\'</script>'))
      .toBe('&lt;script&gt;&quot;x&quot;&amp;&#039;y&#039;&lt;/script&gt;');
  });

  test('navigate: меняет window.location.href', () => {
    // Техника тест-дизайна: классы эквивалентности
    roomModule.navigate('/auth');
    expect(window.location.href).toBe('/auth');
  });

  test('setText: записывает текст в найденный элемент', () => {
    // Техника тест-дизайна: классы эквивалентности
    roomModule.setText('.company-name', 'Компания А');
    expect(document.querySelector('.company-name').textContent).toBe('Компания А');
  });

  test('loadCurrentUser: при успешном getMe сохраняет пользователя', async () => {
    // Техника тест-дизайна: сценарий использования
    window.Api = makeApi({
      getMe: jest.fn().mockResolvedValue({
        authenticated: true,
        user: { username: 'slava' }
      })
    });

    await roomModule.loadCurrentUser();

    expect(roomModule.__getCurrentUserForTests()).toEqual({
      authenticated: true,
      user: { username: 'slava' }
    });
  });

  test('loadCurrentUser: при ошибке getMe устанавливает неавторизованного пользователя', async () => {
    // Техника тест-дизайна: предугадывание ошибок
    window.Api = makeApi({
      getMe: jest.fn().mockRejectedValue(new Error('network'))
    });

    await roomModule.loadCurrentUser();

    expect(roomModule.__getCurrentUserForTests()).toEqual({
      authenticated: false,
      user: null
    });
    expect(console.warn).toHaveBeenCalled();
  });

  test('isAuthenticated: возвращает true для authenticated пользователя', () => {
    // Техника тест-дизайна: классы эквивалентности
    roomModule.__setCurrentUserForTests({
      authenticated: true,
      user: { username: 'slava' }
    });

    expect(roomModule.isAuthenticated()).toBe(true);
  });

  test('updateAuthUi: для неавторизованного пользователя настраивает кнопку "Войти"', () => {
    // Техника тест-дизайна: классы эквивалентности
    roomModule.__setCurrentUserForTests({
      authenticated: false,
      user: null
    });

    roomModule.updateAuthUi();

    const authButton = document.getElementById('authButton');
    expect(authButton.dataset.auth).toBe('0');
    expect(authButton.textContent).toBe('Войти');
  });

  test('updateAuthUi: для авторизованного пользователя показывает "Кабинет" и приветствие', () => {
    // Техника тест-дизайна: сценарий использования
    roomModule.__setCurrentUserForTests({
      authenticated: true,
      user: { username: 'slava' }
    });

    roomModule.updateAuthUi();

    expect(document.getElementById('authButton').dataset.auth).toBe('1');
    expect(document.getElementById('authButton').dataset.username).toBe('slava');
    expect(document.getElementById('authButton').textContent).toBe('Кабинет');
    expect(document.getElementById('userGreeting').textContent).toBe('Привет, slava');
  });

  test('updateAuthUi: authButton ведёт в /auth для гостя', () => {
    // Техника тест-дизайна: таблица решений
    roomModule.__setCurrentUserForTests({
      authenticated: false,
      user: null
    });

    roomModule.updateAuthUi();
    document.getElementById('authButton').click();

    expect(window.location.href).toBe('/auth');
  });

  test('updateAuthUi: authButton ведёт в /me для авторизованного пользователя', () => {
    // Техника тест-дизайна: таблица решений
    roomModule.__setCurrentUserForTests({
      authenticated: true,
      user: { username: 'slava' }
    });

    roomModule.updateAuthUi();
    document.getElementById('authButton').click();

    expect(window.location.href).toBe('/me');
  });

  test('updateAuthUi: historyBtn ведёт в /me', () => {
    // Техника тест-дизайна: сценарий использования
    roomModule.__setCurrentUserForTests({
      authenticated: true,
      user: { username: 'slava' }
    });

    roomModule.updateAuthUi();
    document.getElementById('historyBtn').click();

    expect(window.location.href).toBe('/me');
  });

  test('updateAuthUi: успешный logout вызывает reload', async () => {
    // Техника тест-дизайна: сценарий использования
    roomModule.__setCurrentUserForTests({
      authenticated: true,
      user: { username: 'slava' }
    });

    roomModule.updateAuthUi();
    document.getElementById('logoutBtn').click();
    await flushPromises();

    expect(window.Api.logoutUser).toHaveBeenCalled();
    expect(window.location.reload).toHaveBeenCalled();
  });

  test('updateAuthUi: ошибка logout показывает alert', async () => {
    // Техника тест-дизайна: предугадывание ошибок
    window.Api = makeApi({
      logoutUser: jest.fn().mockRejectedValue(new Error('Ошибка выхода'))
    });

    roomModule.__setCurrentUserForTests({
      authenticated: true,
      user: { username: 'slava' }
    });

    roomModule.updateAuthUi();
    document.getElementById('logoutBtn').click();
    await flushPromises();

    expect(global.alert).toHaveBeenCalledWith('Ошибка выхода');
  });

  test('bindHeaderActions: клик по brand ведёт на главную', () => {
    // Техника тест-дизайна: сценарий использования
    roomModule.bindHeaderActions();

    document.querySelector('.brand').click();

    expect(window.location.href).toBe('/');
  });

  test('normalizeRoom: корректно нормализует room из data.room', () => {
    // Техника тест-дизайна: классы эквивалентности
    const normalized = roomModule.normalizeRoom({
      room: {
        id: '101',
        name: 'Комната',
        company_name: 'Компания А',
        address: 'Адрес 1',
        images: ['/img/1.png'],
        price: 1000,
        max_capacity: 12,
        availableFrom: '09:00',
        availableTo: '21:00',
        description: 'Описание',
        latitude: '55.75',
        longitude: '37.61'
      }
    });

    expect(normalized).toEqual({
      id: 101,
      title: 'Комната',
      company: 'Компания А',
      address: 'Адрес 1',
      images: ['/img/1.png'],
      price: 1000,
      currency: 'RUB',
      capacity: 12,
      max_capacity: 12,
      available_from: '09:00',
      available_to: '21:00',
      description: 'Описание',
      lat: 55.75,
      lng: 37.61
    });
  });

  test('renderRoom: заполняет основную информацию о комнате', () => {
    // Техника тест-дизайна: сценарий использования
    const room = {
      id: 101,
      title: 'Переговорная',
      company: 'Компания А',
      address: 'Адрес 1',
      images: ['/img/1.png'],
      price: 1500,
      currency: 'RUB',
      capacity: 8,
      max_capacity: 10,
      available_from: '09:00',
      available_to: '21:00',
      description: 'Описание комнаты',
      lat: 55.75,
      lng: 37.61
    };

    roomModule.renderRoom(room);

    expect(document.body.dataset.roomId).toBe('101');
    expect(document.title).toBe('Переговорная — Компания А');
    expect(document.querySelector('.room-info h1').textContent).toBe('Переговорная');
    expect(document.querySelector('.company-name').textContent).toBe('Компания А');
    expect(document.querySelector('.room-info .address').textContent).toBe('Адрес 1');
    expect(document.querySelector('.price').textContent).toBe('1500 ₽/ч');
    expect(document.querySelector('.capacity').textContent).toBe('Вместимость: 8 чел.');
    expect(document.querySelector('.availability .value').textContent).toBe('c 09:00 до 21:00');
    expect(document.getElementById('description').textContent).toBe('Описание комнаты');
  });

  test('renderImages: при отсутствии изображений подставляет placeholder', () => {
    // Техника тест-дизайна: классы эквивалентности
    roomModule.renderImages({
      title: 'Комната',
      images: []
    });

    expect(document.getElementById('mainImage').src).toContain('/shared/placeholders/room-placeholder.svg');
    expect(document.querySelectorAll('#thumbs .thumb-btn').length).toBe(1);
  });

  test('renderImages: клик по миниатюре меняет основное изображение', () => {
    // Техника тест-дизайна: сценарий использования
    roomModule.renderImages({
      title: 'Комната',
      images: ['/img/1.png', '/img/2.png']
    });

    document.querySelectorAll('#thumbs .thumb-btn')[1].click();

    expect(document.getElementById('mainImage').src).toContain('/img/2.png');
  });

  test('renderDetails: заполняет блок дополнительных деталей', () => {
    // Техника тест-дизайна: сценарий использования
    roomModule.renderDetails({
      id: 101,
      address: 'Адрес 1',
      capacity: 8,
      max_capacity: 10
    });

    const items = document.querySelectorAll('.details-card ul li');
    expect(items.length).toBe(3);
    expect(items[0].textContent).toBe('Макс. вместимость: 10');
    expect(items[1].textContent).toBe('Адрес: Адрес 1');
    expect(items[2].textContent).toBe('ID помещения: 101');
  });

  test('renderMap: при отсутствии координат показывает сообщение', () => {
    // Техника тест-дизайна: классы эквивалентности
    roomModule.renderMap({
      lat: null,
      lng: null
    });

    expect(document.getElementById('map').textContent).toContain('Координаты отсутствуют');
  });

  test('renderMap: при наличии координат инициализирует Leaflet-карту', () => {
    // Техника тест-дизайна: сценарий использования
    roomModule.renderMap({
      lat: 55.75,
      lng: 37.61
    });

    expect(global.L.map).toHaveBeenCalled();
    expect(global.L.tileLayer).toHaveBeenCalled();
    expect(global.L.marker).toHaveBeenCalled();
  });

  test('renderMap: при существующем mapInstance удаляет старую карту перед созданием новой', () => {
    // Техника тест-дизайна: переходы состояний
    const oldMap = { remove: jest.fn() };
    roomModule.__setMapInstanceForTests(oldMap);

    roomModule.renderMap({
      lat: 55.75,
      lng: 37.61
    });

    expect(oldMap.remove).toHaveBeenCalled();
  });

  test('renderMap: при ошибке Leaflet показывает fallback-сообщение', () => {
    // Техника тест-дизайна: предугадывание ошибок
    global.L.map = jest.fn(() => {
      throw new Error('Leaflet failed');
    });

    roomModule.renderMap({
      lat: 55.75,
      lng: 37.61
    });

    expect(document.getElementById('map').textContent).toContain('Не удалось загрузить карту');
    expect(console.warn).toHaveBeenCalled();
  });

  test('showBookHint: добавляет подсказку для гостя', () => {
    // Техника тест-дизайна: сценарий использования
    roomModule.showBookHint(true);

    expect(document.getElementById('bookHint')).not.toBeNull();
    expect(document.getElementById('bookHint').textContent)
      .toBe('Вам необходимо войти в систему для бронирования');
  });

  test('showBookHint: удаляет подсказку при show=false', () => {
    // Техника тест-дизайна: переходы состояний
    roomModule.showBookHint(true);
    roomModule.showBookHint(false);

    expect(document.getElementById('bookHint')).toBeNull();
  });

  test('configureBookingButton: для гостя показывает подсказку', () => {
    // Техника тест-дизайна: классы эквивалентности
    roomModule.__setCurrentUserForTests({
      authenticated: false,
      user: null
    });

    roomModule.configureBookingButton();

    expect(document.getElementById('bookHint')).not.toBeNull();
  });

  test('configureBookingButton: клик гостя по кнопке бронирования ведёт на /auth', () => {
    // Техника тест-дизайна: таблица решений
    roomModule.__setCurrentUserForTests({
      authenticated: false,
      user: null
    });

    roomModule.configureBookingButton();
    document.getElementById('bookBtn').click();

    expect(window.location.href).toBe('/auth');
  });

  test('configureBookingButton: для авторизованного пользователя скрывает подсказку', () => {
    // Техника тест-дизайна: классы эквивалентности
    roomModule.__setCurrentUserForTests({
      authenticated: true,
      user: { username: 'slava' }
    });

    roomModule.configureBookingButton();

    expect(document.getElementById('bookHint')).toBeNull();
  });

  test('bindBookingModalActions: bookingClose закрывает модалку', () => {
    // Техника тест-дизайна: переходы состояний
    document.getElementById('bookingModal').classList.remove('hidden');
    document.getElementById('bookingModal').setAttribute('aria-hidden', 'false');

    roomModule.bindBookingModalActions();
    document.getElementById('bookingClose').click();

    expect(document.getElementById('bookingModal').classList.contains('hidden')).toBe(true);
    expect(document.getElementById('bookingModal').getAttribute('aria-hidden')).toBe('true');
  });

  test('bindBookingModalActions: bookingCancel закрывает модалку', () => {
    // Техника тест-дизайна: переходы состояний
    document.getElementById('bookingModal').classList.remove('hidden');
    document.getElementById('bookingModal').setAttribute('aria-hidden', 'false');

    roomModule.bindBookingModalActions();
    document.getElementById('bookingCancel').click();

    expect(document.getElementById('bookingModal').classList.contains('hidden')).toBe(true);
  });

  test('bindBookingModalActions: клик по overlay закрывает модалку', () => {
    // Техника тест-дизайна: сценарий использования
    document.getElementById('bookingModal').classList.remove('hidden');
    document.getElementById('bookingModal').setAttribute('aria-hidden', 'false');

    roomModule.bindBookingModalActions();
    document.getElementById('bookingModal')
      .dispatchEvent(new MouseEvent('click', { bubbles: true }));

    expect(document.getElementById('bookingModal').classList.contains('hidden')).toBe(true);
  });

  test('openBookingModal: без currentRoom показывает alert', async () => {
    // Техника тест-дизайна: классы эквивалентности
    roomModule.__setCurrentRoomForTests(null);
    roomModule.__setCurrentUserForTests({
      authenticated: true,
      user: { username: 'slava' }
    });

    await roomModule.openBookingModal();

    expect(global.alert).toHaveBeenCalledWith('Комната не загружена');
  });

  test('openBookingModal: для гостя ведёт на /auth', async () => {
    // Техника тест-дизайна: таблица решений
    roomModule.__setCurrentRoomForTests({ id: 101 });
    roomModule.__setCurrentUserForTests({
      authenticated: false,
      user: null
    });

    await roomModule.openBookingModal();

    expect(window.location.href).toBe('/auth');
  });

  test('openBookingModal: успешно открывает модалку и загружает availability', async () => {
    // Техника тест-дизайна: сценарий использования
    const today = new Date();
    const ymd = roomModule.formatYMD(today);

    window.Api = makeApi({
      getRoomAvailability: jest.fn().mockResolvedValue({
        dates: [
          { date: ymd, available_times: ['10:00', '11:00'] }
        ]
      })
    });

    roomModule.__setCurrentRoomForTests({ id: 101 });
    roomModule.__setCurrentUserForTests({
      authenticated: true,
      user: { username: 'slava' }
    });

    await roomModule.openBookingModal();

    expect(window.Api.getRoomAvailability).toHaveBeenCalledWith(101, 7);
    expect(document.getElementById('bookingModal').classList.contains('hidden')).toBe(false);
    expect(document.getElementById('bookingNotice').textContent).toBe('Выберите дату и слоты');
    expect(document.querySelectorAll('#bookingCalendar .day').length).toBe(7);
  });

  test('openBookingModal: ошибка availability показывает сообщение в bookingNotice', async () => {
    // Техника тест-дизайна: предугадывание ошибок
    window.Api = makeApi({
      getRoomAvailability: jest.fn().mockRejectedValue(new Error('Ошибка загрузки слотов'))
    });

    roomModule.__setCurrentRoomForTests({ id: 101 });
    roomModule.__setCurrentUserForTests({
      authenticated: true,
      user: { username: 'slava' }
    });

    await roomModule.openBookingModal();

    expect(document.getElementById('bookingNotice').textContent).toBe('Ошибка загрузки слотов');
    expect(console.error).toHaveBeenCalled();
  });

  test('resetBookingState: сбрасывает состояние бронирования', () => {
    // Техника тест-дизайна: переходы состояний
    roomModule.__setServerDatesMapForTests({ a: ['10:00'] });
    roomModule.__setAvailableDatesSetForTests(new Set(['2026-04-30']));
    roomModule.__setSelectedDateForTests('2026-04-30');
    roomModule.__setSelectedSlotsForTests(['10:00']);
    roomModule.__setCalendarDaysForTests([{ ymd: '2026-04-30' }]);

    roomModule.resetBookingState();

    expect(roomModule.__getServerDatesMapForTests()).toEqual({});
    expect(roomModule.__getAvailableDatesSetForTests()).toEqual(new Set());
    expect(roomModule.__getSelectedDateForTests()).toBeNull();
    expect(roomModule.__getSelectedSlotsForTests()).toEqual([]);
    expect(roomModule.__getCalendarDaysForTests()).toEqual([]);
  });

  test('buildCalendarDays: создаёт 7 дней календаря', () => {
    // Техника тест-дизайна: классы эквивалентности
    roomModule.buildCalendarDays();

    expect(roomModule.__getCalendarDaysForTests().length).toBe(7);
  });

  test('applyAvailability: применяет даты и выбирает первую доступную дату', () => {
    // Техника тест-дизайна: сценарий использования
    roomModule.buildCalendarDays();
    const firstYmd = roomModule.__getCalendarDaysForTests()[0].ymd;

    roomModule.applyAvailability({
      dates: [
        { date: firstYmd, available_times: ['10:00', '11:00'] }
      ]
    });

    expect(roomModule.__getSelectedDateForTests()).toBe(firstYmd);
    expect(roomModule.__getServerDatesMapForTests()[firstYmd]).toEqual(['10:00', '11:00']);
    expect(roomModule.__getAvailableDatesSetForTests().has(firstYmd)).toBe(true);
  });

  test('renderCalendar: отрисовывает 7 дней и делает недоступные даты disabled', () => {
    // Техника тест-дизайна: сценарий использования
    roomModule.buildCalendarDays();
    const firstYmd = roomModule.__getCalendarDaysForTests()[0].ymd;
    roomModule.__setAvailableDatesSetForTests(new Set([firstYmd]));

    roomModule.renderCalendar();

    const days = document.querySelectorAll('#bookingCalendar .day');
    expect(days.length).toBe(7);
    expect(days[0].classList.contains('disabled')).toBe(false);
    expect(days[1].classList.contains('disabled')).toBe(true);
  });

  test('renderCalendar: клик по доступной дате выбирает её', () => {
    // Техника тест-дизайна: сценарий использования
    roomModule.buildCalendarDays();
    const firstYmd = roomModule.__getCalendarDaysForTests()[0].ymd;
    const secondYmd = roomModule.__getCalendarDaysForTests()[1].ymd;

    roomModule.__setAvailableDatesSetForTests(new Set([firstYmd, secondYmd]));
    roomModule.__setServerDatesMapForTests({
      [firstYmd]: ['10:00'],
      [secondYmd]: ['11:00']
    });
    roomModule.__setSelectedDateForTests(firstYmd);

    roomModule.renderCalendar();
    document.querySelectorAll('#bookingCalendar .day')[1].click();

    expect(roomModule.__getSelectedDateForTests()).toBe(secondYmd);
    expect(roomModule.__getSelectedSlotsForTests()).toEqual([]);
  });

  test('renderTimes: без выбранной даты показывает подсказку "Выберите дату"', () => {
    // Техника тест-дизайна: классы эквивалентности
    roomModule.__setSelectedDateForTests(null);

    roomModule.renderTimes([]);

    expect(document.getElementById('bookingTimes').textContent).toContain('Выберите дату');
  });

  test('renderTimes: при отсутствии слотов показывает сообщение', () => {
    // Техника тест-дизайна: классы эквивалентности
    roomModule.__setSelectedDateForTests('2026-04-30');

    roomModule.renderTimes([]);

    expect(document.getElementById('bookingTimes').textContent).toContain('Нет доступных слотов');
  });

  test('renderTimes: отображает доступные слоты', () => {
    // Техника тест-дизайна: сценарий использования
    roomModule.__setSelectedDateForTests('2026-04-30');
    roomModule.__setSelectedSlotsForTests([]);

    roomModule.renderTimes(['10:00', '11:00']);

    const slots = document.querySelectorAll('#bookingTimes .slot');
    expect(slots.length).toBe(2);
    expect(slots[0].textContent).toBe('10:00 — 11:00');
    expect(slots[1].textContent).toBe('11:00 — 12:00');
  });

  test('onSlotClick: первый клик выбирает один слот', () => {
    // Техника тест-дизайна: классы эквивалентности
    roomModule.__setSelectedDateForTests('2026-04-30');
    roomModule.__setSelectedSlotsForTests([]);
    roomModule.__setServerDatesMapForTests({
      '2026-04-30': ['10:00', '11:00', '12:00']
    });

    roomModule.onSlotClick('10:00');

    expect(roomModule.__getSelectedSlotsForTests()).toEqual(['10:00']);
  });

  test('onSlotClick: смежный слот расширяет последовательность', () => {
    // Техника тест-дизайна:классы эквивалентности
    roomModule.__setSelectedDateForTests('2026-04-30');
    roomModule.__setSelectedSlotsForTests(['10:00']);
    roomModule.__setServerDatesMapForTests({
      '2026-04-30': ['10:00', '11:00', '12:00']
    });

    roomModule.onSlotClick('11:00');

    expect(roomModule.__getSelectedSlotsForTests()).toEqual(['10:00', '11:00']);
  });

  test('onSlotClick: несмежный слот сбрасывает выбор к одному слоту', () => {
    // Техника тест-дизайна: классы эквивалентности
    roomModule.__setSelectedDateForTests('2026-04-30');
    roomModule.__setSelectedSlotsForTests(['10:00']);
    roomModule.__setServerDatesMapForTests({
      '2026-04-30': ['10:00', '11:00', '13:00']
    });

    roomModule.onSlotClick('13:00');

    expect(roomModule.__getSelectedSlotsForTests()).toEqual(['13:00']);
  });

  test('onSlotClick: повторный клик по единственному слоту снимает выбор', () => {
    // Техника тест-дизайна: переходы состояний
    roomModule.__setSelectedDateForTests('2026-04-30');
    roomModule.__setSelectedSlotsForTests(['10:00']);
    roomModule.__setServerDatesMapForTests({
      '2026-04-30': ['10:00']
    });

    roomModule.onSlotClick('10:00');

    expect(roomModule.__getSelectedSlotsForTests()).toEqual([]);
  });

  test('updateSummary: без выбора показывает "Слоты: не выбраны"', () => {
    // Техника тест-дизайна: классы эквивалентности
    roomModule.__setSelectedDateForTests(null);
    roomModule.__setSelectedSlotsForTests([]);

    roomModule.updateSummary();

    expect(document.getElementById('selectionSummary').textContent).toBe('Слоты: не выбраны');
  });

  test('updateSummary: с датой и слотами показывает сводку', () => {
    // Техника тест-дизайна: сценарий использования
    roomModule.__setSelectedDateForTests('2026-04-30');
    roomModule.__setSelectedSlotsForTests(['11:00', '10:00']);

    roomModule.updateSummary();

    expect(document.getElementById('selectionSummary').textContent)
      .toBe('Дата: 2026-04-30, слоты: 10:00, 11:00');
  });

  test('confirmBooking: без даты и слотов показывает alert', async () => {
    // Техника тест-дизайна: классы эквивалентности
    roomModule.__setCurrentRoomForTests({ id: 101 });
    roomModule.__setSelectedDateForTests(null);
    roomModule.__setSelectedSlotsForTests([]);

    await roomModule.confirmBooking();

    expect(global.alert).toHaveBeenCalledWith('Выберите дату и хотя бы один слот');
    expect(window.Api.createBooking).not.toHaveBeenCalled();
  });


  test('confirmBooking: успешное бронирование заменяет содержимое модалки на успешное', async () => {
    // Техника тест-дизайна: сценарий использования
    roomModule.__setCurrentRoomForTests({ id: 101 });
    roomModule.__setSelectedDateForTests('2026-04-30');
    roomModule.__setSelectedSlotsForTests(['11:00', '10:00']);

    await roomModule.confirmBooking();

    expect(window.Api.createBooking).toHaveBeenCalledWith({
      room_id: 101,
      date: '2026-04-30',
      slots: ['10:00', '11:00']
    });

    expect(document.querySelector('#bookingModal .booking-modal__content').textContent)
      .toContain('Бронирование успешно');
  });

  test('confirmBooking: code=slot_already_booked показывает специальный alert', async () => {
    // Техника тест-дизайна: таблица решений
    window.Api = makeApi({
      createBooking: jest.fn().mockRejectedValue({ code: 'slot_already_booked' })
    });

    roomModule.__setCurrentRoomForTests({ id: 101 });
    roomModule.__setSelectedDateForTests('2026-04-30');
    roomModule.__setSelectedSlotsForTests(['10:00']);

    await roomModule.confirmBooking();

    expect(global.alert).toHaveBeenCalledWith('Выбранный слот уже занят');
  });

  test('confirmBooking: code=slots_must_be_consecutive показывает специальный alert', async () => {
    // Техника тест-дизайна: таблица решений
    window.Api = makeApi({
      createBooking: jest.fn().mockRejectedValue({ code: 'slots_must_be_consecutive' })
    });

    roomModule.__setCurrentRoomForTests({ id: 101 });
    roomModule.__setSelectedDateForTests('2026-04-30');
    roomModule.__setSelectedSlotsForTests(['10:00']);

    await roomModule.confirmBooking();

    expect(global.alert).toHaveBeenCalledWith('Слоты должны идти подряд');
  });

  test('confirmBooking: code=unauthorized ведёт на /auth', async () => {
    // Техника тест-дизайна: таблица решений
    window.Api = makeApi({
      createBooking: jest.fn().mockRejectedValue({ code: 'unauthorized' })
    });

    roomModule.__setCurrentRoomForTests({ id: 101 });
    roomModule.__setSelectedDateForTests('2026-04-30');
    roomModule.__setSelectedSlotsForTests(['10:00']);

    await roomModule.confirmBooking();

    expect(window.location.href).toBe('/auth');
  });

  test('confirmBooking: неизвестная ошибка показывает message из err', async () => {
    // Техника тест-дизайна: предугадывание ошибок
    window.Api = makeApi({
      createBooking: jest.fn().mockRejectedValue({ message: 'Ошибка бронирования с сервера' })
    });

    roomModule.__setCurrentRoomForTests({ id: 101 });
    roomModule.__setSelectedDateForTests('2026-04-30');
    roomModule.__setSelectedSlotsForTests(['10:00']);

    await roomModule.confirmBooking();

    expect(global.alert).toHaveBeenCalledWith('Ошибка бронирования с сервера');
  });

  test('renderRoomError: при наличии room-page рендерит ошибку в контейнер', () => {
    // Техника тест-дизайна: сценарий использования
    roomModule.renderRoomError('Комната не найдена');

    expect(document.querySelector('.room-page').textContent).toContain('Комната не найдена');
    expect(document.querySelector('.room-page').textContent).toContain('На главную');
  });

  test('renderRoomError: при отсутствии контейнера вызывает alert', () => {
    // Техника тест-дизайна: предугадывание ошибок
    document.querySelector('.room-page').remove();

    roomModule.renderRoomError('Ошибка страницы');

    expect(global.alert).toHaveBeenCalledWith('Ошибка страницы');
  });


  test('initRoomPage: при успешной загрузке получает room, рендерит её и настраивает UI', async () => {
    // Техника тест-дизайна: сценарий использования
    window.Api = makeApi({
      getMe: jest.fn().mockResolvedValue({
        authenticated: true,
        user: { username: 'slava' }
      }),
      getRoom: jest.fn().mockResolvedValue({
        room: {
          id: 101,
          title: 'Переговорная',
          company_name: 'Компания А',
          address: 'Адрес 1',
          images: ['/img/1.png'],
          price: 1500,
          capacity: 8,
          max_capacity: 10,
          available_from: '09:00',
          available_to: '21:00',
          description: 'Описание комнаты',
          lat: 55.75,
          lng: 37.61
        }
      })
    });

    await roomModule.initRoomPage();

    expect(window.Api.getRoom).toHaveBeenCalledWith(101);
    expect(document.querySelector('.room-info h1').textContent).toBe('Переговорная');
    expect(document.getElementById('authButton').textContent).toBe('Кабинет');
  });

  test('initRoomPage: ошибка getRoom рендерит ошибку в контейнер', async () => {
    // Техника тест-дизайна: предугадывание ошибок
    window.Api = makeApi({
      getRoom: jest.fn().mockRejectedValue(new Error('Не удалось загрузить комнату'))
    });

    await roomModule.initRoomPage();

    expect(document.querySelector('.room-page').textContent).toContain('Не удалось загрузить комнату');
    expect(console.error).toHaveBeenCalled();
  });
});